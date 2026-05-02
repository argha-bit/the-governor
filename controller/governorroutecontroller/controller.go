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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// +kubebuilder:rbac:groups=governor.io,resources=governorroutes,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups=governor.io,resources=governorroutes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=gateway.networking.k8s.io,resources=httproutes,verbs=get;list;create;update;delete

type Controller struct {
	client.Client
	Scheme            *runtime.Scheme
	gatewayTranslator usecase.GatewayTranslator
}

func NewGovernorRouteController(client client.Client, scheme *runtime.Scheme, gatewayTranslator usecase.GatewayTranslator,
) controller.GovernorRouteController {
	return &Controller{
		Client:            client,
		Scheme:            scheme,
		gatewayTranslator: gatewayTranslator,
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

	// 4. Translate all routes at once (allows implementations to group by domain)
	log.Printf("Translating %d routes", len(routeConfig.Routes))
	objects, err := c.gatewayTranslator.TranslateAll(ctx, routeConfig.Routes)
	if err != nil {
		log.Printf("ERROR translating routes: %v", err)
		return ctrl.Result{}, c.setStatus(ctx, governorRoute, false,
			fmt.Sprintf("Failed to translate routes: %v", err))
	}

	for _, obj := range objects {
		if err := ctrl.SetControllerReference(governorRoute, obj, c.Scheme); err != nil {
			log.Printf("WARN could not set controller reference on %s/%s: %v", obj.GetNamespace(), obj.GetName(), err)
		}
		if err := utils.ApplyObject(ctx, c.Client, obj); err != nil {
			log.Printf("ERROR applying %s/%s: %v", obj.GetNamespace(), obj.GetName(), err)
			return ctrl.Result{}, c.setStatus(ctx, governorRoute, false,
				fmt.Sprintf("Failed to apply %s: %v", obj.GetName(), err))
		}
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
