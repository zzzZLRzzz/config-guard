package resolver

type ResolvedConfig struct {
	Labels    map[string]string
	Values    map[string]interface{}
	KeySource map[string]string
}

type Resolver interface {
	Resolve(instance string, app string) (*ResolvedConfig, error)
}
