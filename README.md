# config-guard

Configuration governance tool for YAML config compliance checking across multi-environment deployments.

## Why config-guard

In multi-service, multi-environment setups, each service has independent configs across environments and regions. This leads to common problems:

- **Inconsistent formats**: image addresses, tag formats, JDBC URLs lack a unified standard
- **Uncontrolled value domains**: fields like `region` can end up with invalid values
- **Hidden cross-environment drift**: no clear definition of which configs should stay consistent and which are allowed to differ
- **Config-instance mismatch**: gateway ports should correspond to regions, but manual maintenance is error-prone

config-guard solves these with declarative rules: define once, scan all services and instances automatically, and report violations.

## Quick Start

### Build

```bash
make build
```

### correctness mode

Check whether a single instance's config satisfies all rules:

```bash
./bin/config-guard scan \
    --config ./config/config.yaml \
    --application svcA \
    --mode correctness \
    --instances uat-aps1
```

### diff mode

Compare config differences between two instances, only reporting diffs not covered by the whitelist:

```bash
./bin/config-guard scan \
    --config ./config/config.yaml \
    --application svcA \
    --mode diff \
    --instances uat-aps1,staging-aps1
```

## Directory Structure

config-guard uses a DirResolver to read configs from the filesystem. `base_path` points to the common parent directory of all services:

```
{base_path}/
├── svcA/
│   ├── uat-aps1/
│   │   ├── values.yaml
│   │   └── application.yaml
│   └── staging-aps1/
│       ├── values.yaml
│       └── application.yaml
└── svcB/
    ├── uat-aps1/
    │   └── values.yaml
    └── staging-aps1/
        └── values.yaml
```

All YAML files under each instance directory are merged and flattened into dot-notation key-value pairs. For example:

```yaml
image:
  repository: registry.example.com/svcA
  tag: v1.2.3
```

becomes:

```
image.repository = registry.example.com/svcA
image.tag = v1.2.3
```

## Labels

Each instance name is automatically parsed into labels using the `{env}-{region}` format:

| Label    | Source                    | Example            |
| -------- | ------------------------- | ------------------ |
| `env`    | Part before `-`           | `uat-aps1` → `uat` |
| `region` | Part after `-`            | `uat-aps1` → `aps1` |
| `app`    | Application directory name | `svcA`            |

These labels can be used in compare rules (`target_label`) and derive rules (`from`).

## Config Files

Config files live under `config/`:

| File            | Purpose                                              |
| --------------- | ---------------------------------------------------- |
| `config.yaml`   | Global config: app list, base_path, rule file path   |
| `rules.yaml`    | Global rules, applied to all applications            |
| `{app}.yaml`    | App-specific rules, merged with global rules (optional) |

### config.yaml

```yaml
base_path: /path/to/configs
applications:
  - svcA
  - svcB
rule_file: rules.yaml
```

## Rule Types

| Type       | Purpose                          | Example                                                              |
| ---------- | -------------------------------- | -------------------------------------------------------------------- |
| `required` | Key must exist                   | `key: image.repository`                                              |
| `enum`     | Value must be in allowed set     | `key: region`, `values: [aps1, use1, euw1]`                          |
| `regex`    | Value must match pattern         | `key: image.tag`, `pattern: '^v[0-9]+\.[0-9]+\.[0-9]+$'`             |
| `equals`   | Value must equal expected        | `key: autoscaling.enabled`, `value: "true"`                          |
| `compare`  | Compare against key/label/literal | `key: autoscaling.maxReplicas`, `op: ">"`, `target: autoscaling.minReplicas` |
| `derive`   | Derive expected value from label | `from: instance.labels.region`, `map: {aps1: 1234, use1: 2234}`     |

compare supports three mutually exclusive right-hand side sources:
- `target`: another config key
- `target_label`: an instance label
- `target_value`: a literal value

See `config/rules.yaml` for a complete example.

## Adding an Application

1. Add the app name to `applications` in `config/config.yaml`:

```yaml
applications:
  - svcA
  - svcB
  - svcC    # new
```

2. Create the directory and config files under `base_path`:

```bash
mkdir -p {base_path}/svcC/uat-aps1
# create values.yaml and other config files
```

3. (Optional) Create `config/svcC.yaml` for app-specific rules.

## Adding Rules

Rules are grouped by type under the `rules:` node.

### Global rules

Add to `config/rules.yaml`, applied to all applications:

```yaml
rules:
  required:
    - id: my-required-rule
      key: some.required.field
  enum:
    - id: my-enum-rule
      key: deploy.strategy
      values: [RollingUpdate, Recreate]
  compare:
    - id: my-compare-rule
      key: autoscaling.maxReplicas
      op: ">"
      target: autoscaling.minReplicas
```

### App-specific rules

Create `config/{app}.yaml` with the same format as `rules.yaml`:

```yaml
rules:
  required:
    - id: app-specific-required
      key: app.specific.field
  enum:
    - id: app-specific-enum
      key: deploy.strategy
      values: [RollingUpdate, Recreate]

whitelist:
  correctness: []
  diff: []
```

App rules are merged with global rules. If both sides define a rule with the same `id`, the app rule overrides the global one.

## Whitelist

### correctness whitelist

Skip specific rules on specific instances:

```yaml
whitelist:
  correctness:
    - rule: image-tag-format
      instance: uat-aps1    # optional, omit to apply to all instances
      reason: "UAT uses special tag format"
```

### diff whitelist

Allow specific keys to differ between instances (supports prefix matching):

```yaml
whitelist:
  diff:
    - key: spring.datasource.url
      mode: allow_diff
      reason: "Database URL differs across environments"
    - key: resources
      mode: allow_diff
      reason: "Resource limits differ across environments"
```

`key: resources` matches all keys starting with `resources`, such as `resources.limits.cpu` and `resources.requests.memory`.

#### Restricting to specific instance pairs (pair)

By default, the whitelist applies to all instance pairs. To allow diffs only between specific instances, use `pair`:

```yaml
whitelist:
  diff:
    - key: autoscaling.minReplicas
      mode: allow_diff
      pair: [uat-aps1, staging-aps1]
      reason: "UAT and staging share the same HPA config, prod differs"
```

Omit `pair` to apply to all instance pairs.

#### allow_missing mode

`allow_diff` allows values to differ but requires the key to exist on both sides. To allow a key to exist on only one side (missing on the other), use `allow_missing`:

```yaml
whitelist:
  diff:
    - key: debug.enabled
      mode: allow_missing
      reason: "Only UAT instances have debug flag"
```

## Exit Codes

| Code | Meaning                                        |
| ---- | ---------------------------------------------- |
| `0`  | Scan completed, no issues found                |
| `1`  | Issues found (issues > 0), or runtime error    |

In CI pipelines, rely on the exit code directly: `exit 1` means the check failed and the pipeline should be blocked.
