package governorroutecontroller

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"the_governor/constants"
	"the_governor/controller"
	"the_governor/usecase"
	"the_governor/utils"
	"time"

	governorv1alpha1 "the_governor/api/v1alpha1"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	gatewayclient "sigs.k8s.io/gateway-api/pkg/client/clientset/versioned"
)

// +kubebuilder:rbac:groups=governor.io,resources=governorroutes,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=governor.io,resources=governorroutes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=httproutes,verbs=get;list;create;update;delete

type Controller struct {
	client.Client
	Scheme          *runtime.Scheme
	routeTranslator usecase.RouteTranslator
}

func NewGovernorRouteController(client client.Client, scheme *runtime.Scheme, routeTranslator usecase.RouteTranslator,
) controller.GovernorRouteController {
	return &Controller{
		Client:          client,
		Scheme:          scheme,
		routeTranslator: routeTranslator,
	}
}

func (c *Controller) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&governorv1alpha1.GovernorRoute{}).
		WithEventFilter(predicate.GenerationChangedPredicate{}).
		Complete(c)
}

func (c *Controller) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log.Printf("Reconciling GovernorRoute: %s/%s", req.Namespace, req.Name)

	// 1. Fetch the CR
	governorRoute := &governorv1alpha1.GovernorRoute{}
	if err := c.Get(ctx, req.NamespacedName, governorRoute); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// 2. Call configEndpoint
	code, body, err := utils.MakeAPICall(http.MethodGet, governorRoute.Spec.ConfigEndpoint, map[string]string{}, nil)
	if err != nil {
		log.Printf("ERROR calling configEndpoint %s: %v", governorRoute.Spec.ConfigEndpoint, err)
		return ctrl.Result{RequeueAfter: 30 * time.Second},
			c.setStatus(ctx, governorRoute, false, fmt.Sprintf("Failed to call configEndpoint: %v", err))
	}
	if code != http.StatusOK {
		log.Printf("ERROR configEndpoint returned status %d", code)
		return ctrl.Result{RequeueAfter: 30 * time.Second},
			c.setStatus(ctx, governorRoute, false, fmt.Sprintf("configEndpoint returned %d", code))
	}

	// 3. Parse routes
	var routeConfig constants.Route
	if err := json.Unmarshal(body, &routeConfig); err != nil {
		log.Printf("ERROR parsing route config: %v", err)
		return ctrl.Result{}, c.setStatus(ctx, governorRoute, false, fmt.Sprintf("Failed to parse route config: %v", err))
	}
	log.Printf("Parsed %d routes from configEndpoint", len(routeConfig.Routes))

	// 4. Get K8s clients
	k8sConfig, err := utils.GetK8sClient()
	if err != nil {
		log.Printf("ERROR getting K8s client: %v", err)
		return ctrl.Result{}, c.setStatus(ctx, governorRoute, false, fmt.Sprintf("Failed to get K8s client: %v", err))
	}
	gatewayClient, err := gatewayclient.NewForConfig(k8sConfig)
	if err != nil {
		log.Printf("ERROR creating gateway client: %v", err)
		return ctrl.Result{}, c.setStatus(ctx, governorRoute, false, fmt.Sprintf("Failed to create gateway client: %v", err))
	}

	// 5. Translate and apply routes — reuses existing translator
	namespace := governorRoute.Spec.Namespace
	for _, routeDef := range routeConfig.Routes {
		log.Printf("Translating route: %s", routeDef.RouteName)
		httpRoute, backendObjects, err := c.routeTranslator.TranslateHTTPRoute(ctx, routeDef)
		if err != nil {
			log.Printf("ERROR translating route %s: %v", routeDef.RouteName, err)
			return ctrl.Result{}, c.setStatus(ctx, governorRoute, false,
				fmt.Sprintf("Failed to translate route %s: %v", routeDef.RouteName, err))
		}

		for _, obj := range backendObjects {
			if _, err := utils.CreateExternalK8sService(obj.(*corev1.Service), namespace, k8sConfig); err != nil {
				log.Printf("WARN failed to create external service: %v", err)
			}
		}

		log.Printf("Applying HTTPRoute %s in namespace %s", httpRoute.Name, namespace)
		existing, err := gatewayClient.GatewayV1().HTTPRoutes(namespace).Get(ctx, httpRoute.Name, metav1.GetOptions{})
		if err != nil {
			if !k8serrors.IsNotFound(err) {
				log.Printf("ERROR getting HTTPRoute %s: %v", httpRoute.Name, err)
				return ctrl.Result{}, c.setStatus(ctx, governorRoute, false,
					fmt.Sprintf("Failed to get HTTPRoute %s: %v", httpRoute.Name, err))
			}
			log.Printf("HTTPRoute %s not found, creating...", httpRoute.Name)
			ctrl.SetControllerReference(governorRoute, httpRoute, c.Scheme)
			_, err = gatewayClient.GatewayV1().HTTPRoutes(namespace).Create(ctx, httpRoute, metav1.CreateOptions{})
		} else {
			log.Printf("HTTPRoute %s exists (rv=%s), updating...", httpRoute.Name, existing.ResourceVersion)
			httpRoute.ResourceVersion = existing.ResourceVersion
			_, err = gatewayClient.GatewayV1().HTTPRoutes(namespace).Update(ctx, httpRoute, metav1.UpdateOptions{})
		}
		if err != nil {
			log.Printf("ERROR applying HTTPRoute %s: %v", httpRoute.Name, err)
			return ctrl.Result{}, c.setStatus(ctx, governorRoute, false,
				fmt.Sprintf("Failed to apply HTTPRoute %s: %v", httpRoute.Name, err))
		}
		log.Printf("HTTPRoute %s applied successfully in namespace %s", httpRoute.Name, namespace)
	}

	// 6. Mark success — requeue every 5 minutes for self-healing
	return ctrl.Result{RequeueAfter: 5 * time.Minute},
		c.setStatus(ctx, governorRoute, true, "HTTPRoutes synced successfully")
}

func (c *Controller) setStatus(ctx context.Context, route *governorv1alpha1.GovernorRoute, success bool, message string) error {
	now := metav1.Now()
	status := metav1.ConditionTrue
	reason := "Synced"
	if !success {
		status = metav1.ConditionFalse
		reason = "SyncFailed"
	}
	route.Status.Message = message
	route.Status.LastSyncedAt = &now
	route.Status.Conditions = []metav1.Condition{
		{
			Type:               "Ready",
			Status:             status,
			Reason:             reason,
			Message:            message,
			LastTransitionTime: now,
		},
	}
	return c.Status().Update(ctx, route)
}
