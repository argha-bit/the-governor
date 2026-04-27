package constants

type RouteDefinition struct {
	RouteName   string           `json:"route_name" validate:"required"`
	Enabled     bool             `json:"enabled" validate:"required"`
	Description string           `json:"description" validate:"required"`
	Hostnames   []string         `json:"hostnames" validate:"required"`
	Path        string           `json:"path" validate:"required"`
	PathType    string           `json:"path_type" validate:"required"`
	Methods     []string         `json:"methods,omitempty"`
	Backend     []BackendRef     `json:"backend"`
	Extensions  *RouteExtensions `json:"extensions,omitempty"`
}

type RouteBackend struct {
	Type     string       `json:"type"`
	BackEnds []BackendRef `json:"backends"`
}

type Route struct {
	Version         string            `json:"version"`
	ServiceMetadata RouteMetaData     `json:"service_metadata"`
	Routes          []RouteDefinition `json:"routes"`
}

type RouteMetaData struct {
	ServiceName string `json:"service_name"`
	TeamName    string `json:"team_name"`
	Namespace   string `json:"namespace"`
	Owner       string `json:"owner"`
}
type BackendRef struct {
	Type        string `json:"type"`
	Host        string `json:"host"`
	Port        int    `json:"port"`
	Protocol    string `json:"protocol"`
	ServiceName string `json:"service_name"`
	Namespace   string `json:"namespace"`
	Weight      *int   `json:"weight"`
}

type KubernetesServiceRouteBackend struct {
	Type        string `json:"type"`
	ServiceName string `json:"service_name"`
	Port        int    `json:"port"`
	Weight      int    `json:"weight"`
}

type RouteExtensions struct {
	HealthCheck     *HealthCheckConfig  `json:"health_check,omitempty"`
	RequestHeaders  *HeaderModification `json:"request_headers,omitempty"`
	ResponseHeaders *HeaderModification `json:"response_headers,omitempty"`
	Timeout         string              `json:"timeout,omitempty"`
	Retries         *RetryConfig        `json:"retries,omitempty"`
}

type HealthCheckConfig struct {
	Enabled            bool   `json:"enabled"`
	Path               string `json:"path"`
	Interval           string `json:"interval"`
	Timeout            string `json:"timeout"`
	HealthyThreshold   int    `json:"healthy_threshold"`
	UnhealthyThreshold int    `json:"unhealthy_threshold"`
}

type HeaderModification struct {
	Add    []Header `json:"add,omitempty"`
	Remove []string `json:"remove,omitempty"`
}

type Header struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

type RetryConfig struct {
	Attempts int      `json:"attempts"`
	RetryOn  []string `json:"retry_on"`
}
