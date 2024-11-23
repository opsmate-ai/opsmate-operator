package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	srev1alpha1 "github.com/jingkaihe/opsmate-operator/api/v1alpha1"
)

// EnvrionmentBuildReconciler reconciles a EnvrionmentBuild object
type EnvrionmentBuildReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=sre.opsmate.io,resources=envrionmentbuilds,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=sre.opsmate.io,resources=envrionmentbuilds/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=sre.opsmate.io,resources=envrionmentbuilds/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the EnvrionmentBuild object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/reconcile
func (r *EnvrionmentBuildReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// TODO(user): your logic here

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EnvrionmentBuildReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&srev1alpha1.EnvrionmentBuild{}).
		Named("envrionmentbuild").
		Complete(r)
}
