package controller

import ctrl "sigs.k8s.io/controller-runtime"

type GovernorRouteController interface {
	SetupWithManager(mgr ctrl.Manager) error
}
