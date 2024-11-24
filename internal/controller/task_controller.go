package controller

import (
	"context"
	"errors"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	srev1alpha1 "github.com/jingkaihe/opsmate-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// TaskReconciler reconciles a Task object
type TaskReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
}

func NewTaskReconciler(mgr ctrl.Manager) *TaskReconciler {
	return &TaskReconciler{
		Client:   mgr.GetClient(),
		Scheme:   mgr.GetScheme(),
		Recorder: mgr.GetEventRecorderFor("task-controller"),
	}
}

// +kubebuilder:rbac:groups=sre.opsmate.io,resources=tasks,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=sre.opsmate.io,resources=tasks/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=sre.opsmate.io,resources=tasks/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.19.1/pkg/reconcile
func (r *TaskReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var task srev1alpha1.Task
	if err := r.Get(ctx, req.NamespacedName, &task); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("Task not found")

			task = srev1alpha1.Task{
				ObjectMeta: metav1.ObjectMeta{
					Name:      req.Name,
					Namespace: req.Namespace,
				},
				Status: srev1alpha1.TaskStatus{
					State: srev1alpha1.StateNotFound,
				},
			}
		} else {
			logger.Error(err, "unable to fetch task")
			return ctrl.Result{}, err
		}
	}

	logger = logger.WithValues(
		"task", task.Name,
		"userID", task.Spec.UserID,
		"environmentBuildName", task.Spec.EnvironmentBuildName,
		"instruction", task.Spec.Instruction,
		"context", task.Spec.Context,
	)
	ctx = log.IntoContext(ctx, logger)

	if task.Status.State == "" {
		// status init
		task.Status.State = srev1alpha1.StatePending
		meta.SetStatusCondition(&task.Status.Conditions, metav1.Condition{
			Type:    srev1alpha1.ConditionTaskPodScheduled,
			Status:  metav1.ConditionTrue,
			Reason:  "PodNotScheduled",
			Message: "Task pod is not scheduled",
		})
		meta.SetStatusCondition(&task.Status.Conditions, metav1.Condition{
			Type:    srev1alpha1.ConditionTaskPodRunning,
			Status:  metav1.ConditionFalse,
			Reason:  "PodNotRunning",
			Message: "Task pod is not running",
		})

		return r.updateTaskStatus(ctx, &task)
	}

	switch task.Status.State {
	case srev1alpha1.StatePending:
	case srev1alpha1.StateScheduled:
	case srev1alpha1.StateRunning:
	case srev1alpha1.StateTerminating:
	case srev1alpha1.StateNotFound:
	case srev1alpha1.StateError:
	default:
		logger.Info("unknown task state", "state", task.Status.State)
		return ctrl.Result{}, nil
	}
	return ctrl.Result{}, nil
}

func (r *TaskReconciler) updateTaskStatus(ctx context.Context, task *srev1alpha1.Task) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	err := r.Status().Update(ctx, task)
	if err != nil {
		if apierrors.IsConflict(err) {
			logger.V(1).Info("Task status update conflict")
			return ctrl.Result{Requeue: true}, nil
		}
		logger.Error(err, "unable to update task status")
		r.Recorder.Event(task, corev1.EventTypeWarning, "TaskStatusUpdateFailed", "Failed to update task status")
		return ctrl.Result{}, err
	}

	r.Recorder.Event(task,
		corev1.EventTypeNormal,
		"TaskStatusUpdated",
		fmt.Sprintf("Task %s status updated to %s, elapsed: %f",
			task.Name,
			task.Status.State,
			time.Since(task.CreationTimestamp.Time).Seconds(),
		),
	)

	if task.Status.State == srev1alpha1.StateError {
		r.Recorder.Event(task, corev1.EventTypeWarning, "TaskError", task.Status.Reason)
		return ctrl.Result{Requeue: true}, errors.New(task.Status.Reason)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *TaskReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&srev1alpha1.Task{}).
		Named("task").
		Complete(r)
}
