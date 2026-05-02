package translator

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"the_governor/constants"
	"the_governor/usecase"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	applyStrategyAnnotation = "governor.io/apply-strategy"
	applyStrategyCreateOnly = "create-only"
	domainLabelKey          = "governor-domain"
)

// GlooEdgeTranslator produces Gloo Edge resources as *unstructured.Unstructured.
//
// Domain ownership model — "first writer wins":
//   - Each route becomes a RouteTable (independently managed, create-or-update).
//   - A VirtualService stub is emitted per domain group annotated with
//     governor.io/apply-strategy=create-only. The apply loop creates it only if
//     it does not yet exist, so whoever runs first owns it. The VS delegates all
//     routing to RouteTables via a label selector — it never needs to be updated.
type GlooEdgeTranslator struct {
	namespace string
}

func NewGlooEdgeTranslator(namespace string) usecase.GatewayTranslator {
	return &GlooEdgeTranslator{namespace: namespace}
}

func (g *GlooEdgeTranslator) SupportsHealthChecks() bool        { return true }
func (g *GlooEdgeTranslator) SupportsHeaderTransformation() bool { return true }
func (g *GlooEdgeTranslator) SupportsRateLimiting() bool         { return false }

func (g *GlooEdgeTranslator) TranslateAll(ctx context.Context, routes []constants.RouteDefinition) ([]client.Object, error) {
	var objects []client.Object
	emittedUpstreams := map[string]bool{}

	// Group routes by sorted hostname set
	type group struct {
		domainLabel string
		vsName      string
		hostnames   []string
		routes      []constants.RouteDefinition
	}
	groupOrder := []string{}
	groups := map[string]*group{}

	for _, route := range routes {
		key := hostnameKey(route.Hostnames)
		if _, ok := groups[key]; !ok {
			groups[key] = &group{
				domainLabel: domainLabel(route.Hostnames),
				vsName:      vsNameFromHostnames(route.Hostnames),
				hostnames:   route.Hostnames,
			}
			groupOrder = append(groupOrder, key)
		}
		groups[key].routes = append(groups[key].routes, route)
	}

	for _, key := range groupOrder {
		grp := groups[key]

		// Emit VS stub once per domain group — create-only so first writer wins
		objects = append(objects, g.buildVirtualServiceStub(grp.vsName, grp.hostnames, grp.domainLabel))

		for _, route := range grp.routes {
			// Upstreams — deduplicated across routes
			var upstreamRefs []map[string]interface{}
			for _, backend := range route.Backend {
				upstream, upstreamName, err := g.buildUpstream(backend)
				if err != nil {
					return nil, fmt.Errorf("building upstream for route %s: %w", route.RouteName, err)
				}
				if !emittedUpstreams[upstreamName] {
					objects = append(objects, upstream)
					emittedUpstreams[upstreamName] = true
				}
				weight := 100
				if backend.Weight != nil {
					weight = *backend.Weight
				}
				upstreamRefs = append(upstreamRefs, map[string]interface{}{
					"name":      upstreamName,
					"namespace": g.namespace,
					"weight":    int64(weight),
				})
			}

			// One RouteTable per route definition — independently create-or-update
			rt, err := g.buildRouteTable(route, upstreamRefs, grp.domainLabel)
			if err != nil {
				return nil, fmt.Errorf("building route table for route %s: %w", route.RouteName, err)
			}
			objects = append(objects, rt)
		}
	}

	return objects, nil
}

// buildVirtualServiceStub creates a VS that delegates all traffic to RouteTables
// selected by the domain label. Annotated create-only — never overwritten.
func (g *GlooEdgeTranslator) buildVirtualServiceStub(vsName string, hostnames []string, label string) *unstructured.Unstructured {
	vs := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "gateway.solo.io/v1",
			"kind":       "VirtualService",
			"metadata": map[string]interface{}{
				"name":      vsName,
				"namespace": g.namespace,
				"annotations": map[string]interface{}{
					applyStrategyAnnotation: applyStrategyCreateOnly,
				},
			},
			"spec": map[string]interface{}{
				"virtualHost": map[string]interface{}{
					"domains": toInterfaceSlice(hostnames),
					"routes": []interface{}{
						map[string]interface{}{
							"matchers": []interface{}{
								map[string]interface{}{"prefix": "/"},
							},
							"delegateAction": map[string]interface{}{
								"selector": map[string]interface{}{
									"namespaces": []interface{}{g.namespace},
									"labels": map[string]interface{}{
										domainLabelKey: label,
									},
								},
							},
						},
					},
				},
			},
		},
	}
	return vs
}

func (g *GlooEdgeTranslator) buildRouteTable(route constants.RouteDefinition, upstreamRefs []map[string]interface{}, label string) (*unstructured.Unstructured, error) {
	glooRoute := map[string]interface{}{
		"matchers":    g.buildMatchers(route),
		"routeAction": g.buildRouteAction(upstreamRefs),
	}
	if opts := g.buildRouteOptions(route); len(opts) > 0 {
		glooRoute["options"] = opts
	}

	rt := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "gateway.solo.io/v1",
			"kind":       "RouteTable",
			"metadata": map[string]interface{}{
				"name":      route.RouteName + "-rt",
				"namespace": g.namespace,
				"labels": map[string]interface{}{
					domainLabelKey: label,
				},
			},
			"spec": map[string]interface{}{
				"routes": []interface{}{glooRoute},
			},
		},
	}
	return rt, nil
}

func (g *GlooEdgeTranslator) buildUpstream(backend constants.BackendRef) (*unstructured.Unstructured, string, error) {
	var upstreamName string
	var spec map[string]interface{}

	switch backend.Type {
	case "kube":
		upstreamName = fmt.Sprintf("%s-%d", backend.ServiceName, backend.Port)
		spec = map[string]interface{}{
			"kube": map[string]interface{}{
				"serviceName":      backend.ServiceName,
				"serviceNamespace": backend.Namespace,
				"servicePort":      int64(backend.Port),
			},
		}
	case "external":
		upstreamName = fmt.Sprintf("%s-%d", backend.Host, backend.Port)
		spec = map[string]interface{}{
			"static": map[string]interface{}{
				"hosts": []interface{}{
					map[string]interface{}{
						"addr": backend.Host,
						"port": int64(backend.Port),
					},
				},
			},
		}
	default:
		return nil, "", fmt.Errorf("unsupported backend type: %s", backend.Type)
	}

	if backend.HealthCheck != nil && backend.HealthCheck.Enabled {
		spec["healthChecks"] = []interface{}{
			map[string]interface{}{
				"timeout":            backend.HealthCheck.Timeout,
				"interval":           backend.HealthCheck.Interval,
				"healthyThreshold":   int64(backend.HealthCheck.HealthyThreshold),
				"unhealthyThreshold": int64(backend.HealthCheck.UnhealthyThreshold),
				"httpHealthCheck": map[string]interface{}{
					"path": backend.HealthCheck.Path,
				},
			},
		}
	}

	if backend.CircuitBreaker != nil {
		spec["circuitBreakers"] = map[string]interface{}{
			"maxConnections":     int64(backend.CircuitBreaker.MaxConnections),
			"maxPendingRequests": int64(backend.CircuitBreaker.MaxPendingRequests),
			"maxRequests":        int64(backend.CircuitBreaker.MaxRequests),
			"maxRetries":         int64(backend.CircuitBreaker.MaxRetries),
		}
	}

	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "gloo.solo.io/v1",
			"kind":       "Upstream",
			"metadata": map[string]interface{}{
				"name":      upstreamName,
				"namespace": g.namespace,
			},
			"spec": spec,
		},
	}, upstreamName, nil
}

func (g *GlooEdgeTranslator) buildMatchers(route constants.RouteDefinition) []interface{} {
	pathKey := "prefix"
	if route.PathType == "Exact" {
		pathKey = "exact"
	} else if route.PathType == "RegularExpression" {
		pathKey = "regex"
	}

	if len(route.Methods) == 0 {
		return []interface{}{map[string]interface{}{pathKey: route.Path}}
	}

	matchers := make([]interface{}, len(route.Methods))
	for i, method := range route.Methods {
		matchers[i] = map[string]interface{}{
			pathKey:   route.Path,
			"methods": []interface{}{method},
		}
	}
	return matchers
}

func (g *GlooEdgeTranslator) buildRouteAction(upstreamRefs []map[string]interface{}) map[string]interface{} {
	if len(upstreamRefs) == 1 {
		return map[string]interface{}{
			"single": map[string]interface{}{
				"upstream": map[string]interface{}{
					"name":      upstreamRefs[0]["name"],
					"namespace": upstreamRefs[0]["namespace"],
				},
			},
		}
	}
	destinations := make([]interface{}, len(upstreamRefs))
	for i, ref := range upstreamRefs {
		destinations[i] = map[string]interface{}{
			"destination": map[string]interface{}{
				"upstream": map[string]interface{}{
					"name":      ref["name"],
					"namespace": ref["namespace"],
				},
			},
			"weight": ref["weight"],
		}
	}
	return map[string]interface{}{
		"multi": map[string]interface{}{"destinations": destinations},
	}
}

func (g *GlooEdgeTranslator) buildRouteOptions(route constants.RouteDefinition) map[string]interface{} {
	if route.Extensions == nil {
		return nil
	}
	ext := route.Extensions
	options := map[string]interface{}{}

	if ext.RequestHeaders != nil || ext.ResponseHeaders != nil {
		hm := map[string]interface{}{}
		if ext.RequestHeaders != nil {
			if len(ext.RequestHeaders.Add) > 0 {
				hm["requestHeadersToAdd"] = buildGlooHeaders(ext.RequestHeaders.Add)
			}
			if len(ext.RequestHeaders.Remove) > 0 {
				hm["requestHeadersToRemove"] = toInterfaceSlice(ext.RequestHeaders.Remove)
			}
		}
		if ext.ResponseHeaders != nil {
			if len(ext.ResponseHeaders.Add) > 0 {
				hm["responseHeadersToAdd"] = buildGlooHeaders(ext.ResponseHeaders.Add)
			}
			if len(ext.ResponseHeaders.Remove) > 0 {
				hm["responseHeadersToRemove"] = toInterfaceSlice(ext.ResponseHeaders.Remove)
			}
		}
		if len(hm) > 0 {
			options["headerManipulation"] = hm
		}
	}
	if ext.Timeout != "" {
		options["timeout"] = ext.Timeout
	}
	if ext.Retries != nil {
		options["retries"] = map[string]interface{}{
			"retryOn":    ext.Retries.RetryOn,
			"numRetries": int64(ext.Retries.Attempts),
		}
	}
	return options
}

func hostnameKey(hostnames []string) string {
	sorted := make([]string, len(hostnames))
	copy(sorted, hostnames)
	sort.Strings(sorted)
	return strings.Join(sorted, ",")
}

func vsNameFromHostnames(hostnames []string) string {
	if len(hostnames) == 0 {
		return "default-vs"
	}
	sorted := make([]string, len(hostnames))
	copy(sorted, hostnames)
	sort.Strings(sorted)
	return strings.ReplaceAll(sorted[0], ".", "-") + "-vs"
}

// domainLabel produces a short stable k8s label value from the hostname set.
func domainLabel(hostnames []string) string {
	if len(hostnames) == 0 {
		return "default"
	}
	sorted := make([]string, len(hostnames))
	copy(sorted, hostnames)
	sort.Strings(sorted)
	return strings.ReplaceAll(sorted[0], ".", "-")
}

func buildGlooHeaders(headers []constants.Header) []interface{} {
	result := make([]interface{}, len(headers))
	for i, h := range headers {
		result[i] = map[string]interface{}{
			"header": map[string]interface{}{
				"key":   h.Name,
				"value": h.Value,
			},
		}
	}
	return result
}

func toInterfaceSlice(s []string) []interface{} {
	result := make([]interface{}, len(s))
	for i, v := range s {
		result[i] = v
	}
	return result
}
