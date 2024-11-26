package controller

import (
	"context"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	srev1alpha1 "github.com/jingkaihe/opsmate-operator/api/v1alpha1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
)

// EnvironmentBuildReconciler reconciles a EnvironmentBuild object
type EnvironmentBuildReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

const (
	taskIndexerKey = ".spec.environmentBuildName"
)

// +kubebuilder:rbac:groups=sre.opsmate.io,resources=environmentbuilds,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=sre.opsmate.io,resources=environmentbuilds/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=sre.opsmate.io,resources=environmentbuilds/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/reconcile
func (r *EnvironmentBuildReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var envBuild srev1alpha1.EnvironmentBuild
	if err := r.Get(ctx, req.NamespacedName, &envBuild); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Error(err, "unable to find EnvironmentBuild")
		}
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	var taskList srev1alpha1.TaskList
	if err := r.List(ctx, &taskList, client.InNamespace(req.Namespace), client.MatchingFields{taskIndexerKey: req.Name}); err != nil {
		logger.Error(err, "unable to list Tasks")
		return ctrl.Result{}, err
	}

	logger.Info("Found Tasks", "count", len(taskList.Items))

	if envBuild.Status.TaskCount == len(taskList.Items) {
		return ctrl.Result{}, nil
	}

	envBuild.Status.TaskCount = len(taskList.Items)
	if err := r.Status().Update(ctx, &envBuild); err != nil {
		if apierrors.IsConflict(err) {
			logger.V(1).Info("conflict while updating environment build status")
			return ctrl.Result{Requeue: true}, nil
		}
		logger.Error(err, "Failed to update environment build status")
		return ctrl.Result{}, err
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EnvironmentBuildReconciler) SetupWithManager(mgr ctrl.Manager) error {
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &srev1alpha1.Task{}, taskIndexerKey, func(rawObj client.Object) []string {
		task := rawObj.(*srev1alpha1.Task)
		if task.Spec.EnvironmentBuildName == "" {
			return nil
		}
		return []string{task.Spec.EnvironmentBuildName}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&srev1alpha1.EnvironmentBuild{}).
		Named("environmentbuild").
		Watches(
			&srev1alpha1.Task{},
			handler.EnqueueRequestsFromMapFunc(r.enqueueTaskRequests),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{}),
		).
		Complete(r)
}

func (r *EnvironmentBuildReconciler) enqueueTaskRequests(ctx context.Context, obj client.Object) []reconcile.Request {
	var (
		task srev1alpha1.Task
	)

	obj.DeepCopyObject().(*srev1alpha1.Task).DeepCopyInto(&task)

	return []reconcile.Request{
		{
			NamespacedName: types.NamespacedName{
				Name:      task.Spec.EnvironmentBuildName,
				Namespace: task.GetNamespace(),
			},
		},
	}
}
