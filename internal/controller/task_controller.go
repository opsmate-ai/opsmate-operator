package controller

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	srev1alpha1 "github.com/jingkaihe/opsmate-operator/api/v1alpha1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/reference"
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

var (
	graceRequeResult = ctrl.Result{RequeueAfter: 100 * time.Millisecond}
	ErrContainerExit = errors.New("container exited")
	apiGVStr         = srev1alpha1.GroupVersion.String()
)

const (
	ownerKind                = "Task"
	ownerKey                 = ".metadata.controller"
	resourceProvisionTimeout = 10 * time.Second
	taskTimeout              = 10 * time.Minute
	syncPeriod               = 10 * time.Minute
)

// +kubebuilder:rbac:groups=sre.opsmate.io,resources=tasks,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=sre.opsmate.io,resources=tasks/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=sre.opsmate.io,resources=tasks/finalizers,verbs=update
// +kubebuilder:rbac:groups="",resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=ingresses,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
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
		meta.SetStatusCondition(&task.Status.Conditions, metav1.Condition{
			Type:    srev1alpha1.ConditionTaskServiceUp,
			Status:  metav1.ConditionFalse,
			Reason:  "ServiceNotUp",
			Message: "Task service is not up",
		})
		meta.SetStatusCondition(&task.Status.Conditions, metav1.Condition{
			Type:    srev1alpha1.ConditionTaskIngressReady,
			Status:  metav1.ConditionFalse,
			Reason:  "IngressNotReady",
			Message: "Task ingress is not ready",
		})

		return r.updateTaskStatus(ctx, &task)
	}

	switch task.Status.State {
	case srev1alpha1.StatePending:
		return r.statePending(ctx, &task)
	case srev1alpha1.StateScheduled:
		return r.stateScheduled(ctx, &task)
	case srev1alpha1.StateRunning:
		return r.stateRunning(ctx, &task)
	case srev1alpha1.StateTerminating:
		return r.stateTerminating(ctx, &task)
	case srev1alpha1.StateNotFound:
		return r.stateNotFound(ctx, &task)
	case srev1alpha1.StateError:
		return r.stateError(ctx, &task)
	default:
		logger.Info("unknown task state", "state", task.Status.State)
		return ctrl.Result{}, nil
	}
}

func (r *TaskReconciler) statePending(ctx context.Context, task *srev1alpha1.Task) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	// get the environment build
	var envBuild srev1alpha1.EnvironmentBuild
	if err := r.Get(ctx, client.ObjectKey{
		Name:      task.Spec.EnvironmentBuildName,
		Namespace: task.Namespace,
	}, &envBuild); err != nil {
		if apierrors.IsNotFound(err) {
			return r.markTaskAsError(ctx, task, errors.Wrap(err, "environment build not found"))
		}
		return ctrl.Result{}, err
	}

	podRef, err := r.createPod(ctx, task, &envBuild)
	if err != nil {
		logger.Error(err, "failed to create pod")
		return r.markTaskAsError(ctx, task, errors.Wrap(err, "failed to create pod"))
	}

	serviceRef, err := r.createService(ctx, task, &envBuild)
	if err != nil {
		logger.Error(err, "failed to create service")
		return r.markTaskAsError(ctx, task, errors.Wrap(err, "failed to create service"))
	}

	ingressRef, err := r.createIngress(ctx, task, &envBuild)
	if err != nil {
		logger.Error(err, "failed to create ingress")
		return r.markTaskAsError(ctx, task, errors.Wrap(err, "failed to create ingress"))
	}

	task.Status.Pod = podRef
	task.Status.Service = serviceRef
	task.Status.Ingress = ingressRef
	task.Status.State = srev1alpha1.StateScheduled

	logger.Info("task scheduled", "pod", podRef)

	return r.updateTaskStatus(ctx, task)
}

func (r *TaskReconciler) stateScheduled(ctx context.Context, task *srev1alpha1.Task) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var (
		pod     corev1.Pod
		service corev1.Service
	)

	if err := r.Get(ctx, types.NamespacedName{
		Name:      task.Name,
		Namespace: task.Namespace,
	}, &pod); err != nil {
		if apierrors.IsNotFound(err) {
			if time.Now().After(task.CreationTimestamp.Add(resourceProvisionTimeout)) {
				return r.markTaskAsError(ctx, task, errors.New("pod creation timeout"))
			}
			return graceRequeResult, nil
		}
		return ctrl.Result{}, err
	}

	if err := r.Get(ctx, types.NamespacedName{
		Name:      task.Name,
		Namespace: task.Namespace,
	}, &service); err != nil {
		if apierrors.IsNotFound(err) {
			if time.Now().After(task.CreationTimestamp.Add(resourceProvisionTimeout)) {
				return r.markTaskAsError(ctx, task, errors.New("service creation timeout"))
			}
			return graceRequeResult, nil
		}
		return ctrl.Result{}, err
	}

	if !serviceIsUp(ctx, &service) {
		return graceRequeResult, nil
	}

	var ingress networkingv1.Ingress
	if err := r.Get(ctx, types.NamespacedName{
		Name:      task.Name,
		Namespace: task.Namespace,
	}, &ingress); err != nil {
		if apierrors.IsNotFound(err) {
			if time.Now().After(task.CreationTimestamp.Add(resourceProvisionTimeout)) {
				return r.markTaskAsError(ctx, task, errors.New("ingress creation timeout"))
			}
			return graceRequeResult, nil
		}
		return ctrl.Result{}, err
	}

	switch pod.Status.Phase {
	case corev1.PodPending:
		meta.SetStatusCondition(&task.Status.Conditions, metav1.Condition{
			Type:    srev1alpha1.ConditionTaskPodScheduled,
			Status:  metav1.ConditionTrue,
			Reason:  "PodScheduled",
			Message: "Task pod is scheduled",
		})
		// DO NOT use r.updateTaskStatus here to avoid event spam
		if err := r.Status().Update(ctx, task); err != nil {
			return ctrl.Result{}, err
		}
		return graceRequeResult, nil
	case corev1.PodRunning:
		if anyContainerErrors(ctx, &pod) {
			return r.markTaskAsError(ctx, task, errors.New("pod container error"))
		}
		if !podReadyAndRunning(&pod) {
			return graceRequeResult, nil
		}
		return r.markTaskAsRunning(ctx, task, &pod, &service, &ingress)
	case corev1.PodSucceeded:
		return r.markTaskAsError(ctx, task, errors.New("pod completed prematurely"))
	case corev1.PodFailed, corev1.PodUnknown:
		return r.markTaskAsError(ctx, task, errors.New("pod failed"))
	default:
		logger.Info("unknown pod phase", "phase", pod.Status.Phase)
		return ctrl.Result{}, nil
	}
}

func (r *TaskReconciler) stateRunning(ctx context.Context, task *srev1alpha1.Task) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	var updateState = func(task *srev1alpha1.Task, newState string) {
		if newState != task.Status.State {
			task.Status.State = newState
			_, _ = r.updateTaskStatus(ctx, task)
		}
	}

	var (
		pod = corev1.Pod{}
	)

	if err := r.Get(ctx, types.NamespacedName{
		Name:      task.Name,
		Namespace: task.Namespace,
	}, &pod); err != nil {
		if !apierrors.IsNotFound(err) {
			return ctrl.Result{}, err
		}

		// pod not found
		logger.Info("task is terminating", "state", srev1alpha1.StateTerminating)
		updateState(task, srev1alpha1.StateTerminating)
		return ctrl.Result{}, nil
	}

	if pod.Status.Phase == corev1.PodFailed || pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == srev1alpha1.StateTerminating {
		logger.Info("task is terminating", "state", srev1alpha1.StateTerminating, "reason", pod.Status.Reason)
		updateState(task, srev1alpha1.StateTerminating)
		return ctrl.Result{}, nil
	}

	if time.Now().After(pod.CreationTimestamp.Add(taskTimeout)) {
		logger.Info("task is terminating",
			"state", srev1alpha1.StateTerminating,
			"reason", "task timeout",
			"timeout", taskTimeout,
			"lifetime", time.Since(pod.CreationTimestamp.Time).String(),
		)

		r.Recorder.Event(task, corev1.EventTypeWarning, "TaskTimeout", fmt.Sprintf("Timed out after %s", taskTimeout))
		updateState(task, srev1alpha1.StateTerminating)
		return ctrl.Result{}, nil
	}

	// nothing happened but we still need to requeue
	// for the following reasons:
	// * just in case the controller-runtime somehow missed the task update events
	// * continuously check if the task should be evicted due to timeout.
	return ctrl.Result{RequeueAfter: syncPeriod}, nil
}

func (r *TaskReconciler) stateTerminating(ctx context.Context, task *srev1alpha1.Task) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	if err := r.Delete(ctx, task); err != nil {
		logger.Error(err, "unable to delete task")
		return ctrl.Result{}, err
	}

	return ctrl.Result{RequeueAfter: 1 * time.Second}, nil
}

func (r *TaskReconciler) stateNotFound(_ context.Context, _ *srev1alpha1.Task) (ctrl.Result, error) {
	return ctrl.Result{}, nil
}

func (r *TaskReconciler) stateError(ctx context.Context, task *srev1alpha1.Task) (ctrl.Result, error) {
	task.Status.State = srev1alpha1.StateTerminating
	return r.updateTaskStatus(ctx, task)
}

func (r *TaskReconciler) markTaskAsError(ctx context.Context, task *srev1alpha1.Task, reason error) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	task.Status.State = srev1alpha1.StateError
	task.Status.Reason = reason.Error()

	logger.Error(reason, "task error", "state", task.Status.State)
	return r.updateTaskStatus(ctx, task)
}

func (r *TaskReconciler) markTaskAsRunning(
	ctx context.Context,
	task *srev1alpha1.Task,
	pod *corev1.Pod,
	service *corev1.Service,
	ingress *networkingv1.Ingress,
) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	task.Status.State = srev1alpha1.StateRunning
	meta.SetStatusCondition(&task.Status.Conditions, metav1.Condition{
		Type:    srev1alpha1.ConditionTaskPodRunning,
		Status:  metav1.ConditionTrue,
		Reason:  "PodRunning",
		Message: "Task pod is running",
	})

	task.Status.AllocatedAt = &metav1.Time{Time: time.Now()}

	task.Status.ServiceIP = service.Spec.ClusterIP
	meta.SetStatusCondition(&task.Status.Conditions, metav1.Condition{
		Type:    srev1alpha1.ConditionTaskServiceUp,
		Status:  metav1.ConditionTrue,
		Reason:  "ServiceUp",
		Message: "Task service is up",
	})
	task.Status.InternalIP = pod.Status.PodIP

	meta.SetStatusCondition(&task.Status.Conditions, metav1.Condition{
		Type:    srev1alpha1.ConditionTaskIngressReady,
		Status:  metav1.ConditionTrue,
		Reason:  "IngressReady",
		Message: "Task ingress is ready",
	})

	elapsed := time.Since(task.CreationTimestamp.Time)

	logger.Info("task is running",
		"internalIP", pod.Status.PodIP,
		"podName", pod.Name,
		"serviceName", service.Name,
		"ingressName", ingress.Name,
		"state", task.Status.State,
		"elapsed", elapsed.Seconds(),
	)
	return r.updateTaskStatus(ctx, task)
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
		fmt.Sprintf("Task status updated to %s, elapsed: %f",
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
	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.Pod{}, "spec.ownerReferences", func(o client.Object) []string {
		pod := o.(*corev1.Pod)
		owner := metav1.GetControllerOf(pod)

		if owner == nil {
			return nil
		}
		if owner.Kind != ownerKind || owner.APIVersion != apiGVStr {
			return nil
		}
		return []string{owner.Name}
	}); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &corev1.Service{}, "spec.ownerReferences", func(o client.Object) []string {
		service := o.(*corev1.Service)
		owner := metav1.GetControllerOf(service)
		if owner == nil {
			return nil
		}
		if owner.Kind != ownerKind || owner.APIVersion != apiGVStr {
			return nil
		}
		return []string{owner.Name}
	}); err != nil {
		return err
	}

	if err := mgr.GetFieldIndexer().IndexField(context.Background(), &networkingv1.Ingress{}, "spec.ownerReferences", func(o client.Object) []string {
		ingress := o.(*networkingv1.Ingress)
		owner := metav1.GetControllerOf(ingress)
		if owner == nil {
			return nil
		}
		if owner.Kind != "Task" || owner.APIVersion != apiGVStr {
			return nil
		}
		return []string{owner.Name}
	}); err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&srev1alpha1.Task{}).
		Named("task").
		Owns(&corev1.Pod{}).
		Owns(&corev1.Service{}).
		Owns(&networkingv1.Ingress{}).
		Complete(r)
}

func podReadyAndRunning(pod *corev1.Pod) bool {
	if pod.Status.Phase != corev1.PodRunning {
		return false
	}

	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.ContainersReady && condition.Status == corev1.ConditionTrue {
			return true
		}
	}
	return false
}

// create the pod if not exists
func (r *TaskReconciler) createPod(ctx context.Context, task *srev1alpha1.Task, envBuild *srev1alpha1.EnvironmentBuild) (*corev1.ObjectReference, error) {
	logger := log.FromContext(ctx)

	var pod corev1.Pod
	err := r.Get(ctx, types.NamespacedName{
		Name:      task.Name,
		Namespace: task.Namespace,
	}, &pod)

	if err == nil {
		logger.Info("pod already exists", "pod", pod.Name)
		return reference.GetReference(r.Scheme, &pod)
	}

	if !apierrors.IsNotFound(err) {
		return nil, err
	}

	// construct the pod
	podMeta := envBuild.Spec.Template.ObjectMeta
	podMeta.Name = task.Name
	podMeta.Namespace = task.Namespace
	podMeta.Labels = map[string]string{
		"userID":       task.Spec.UserID,
		"envBuildName": task.Spec.EnvironmentBuildName,
	}

	pod = corev1.Pod{
		ObjectMeta: podMeta,
		Spec:       envBuild.Spec.Template.Spec,
	}

	// prevent pod from restarting
	pod.Spec.RestartPolicy = corev1.RestartPolicyNever

	// add the task label to the pod
	if pod.Labels == nil {
		pod.Labels = make(map[string]string)
	}
	pod.Labels["opsmate.io/task"] = task.Name

	if err := ctrl.SetControllerReference(task, &pod, r.Scheme); err != nil {
		return nil, err
	}

	if err := r.Create(ctx, &pod); err != nil {
		if apierrors.IsAlreadyExists(err) {
			logger.Info("pod already exists likely caused by quick subsequent reconcile")
		} else {
			return nil, err
		}
	}

	podRef, err := reference.GetReference(r.Scheme, &pod)
	if err != nil {
		return nil, err
	}

	return podRef, nil
}

// create service if not exists
func (r *TaskReconciler) createService(ctx context.Context, task *srev1alpha1.Task, envBuild *srev1alpha1.EnvironmentBuild) (*corev1.ObjectReference, error) {
	logger := log.FromContext(ctx)

	var service corev1.Service
	err := r.Get(ctx, types.NamespacedName{
		Name:      task.Name,
		Namespace: task.Namespace,
	}, &service)

	if err == nil {
		logger.Info("service already exists", "service", service.Name)
		return reference.GetReference(r.Scheme, &service)
	}

	if !apierrors.IsNotFound(err) {
		return nil, err
	}

	// construct the service
	service = corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      task.Name,
			Namespace: task.Namespace,
		},
		Spec: envBuild.Spec.Service,
	}

	// add the task label to the service
	if service.Spec.Selector == nil {
		service.Spec.Selector = make(map[string]string)
	}
	service.Spec.Selector["opsmate.io/task"] = task.Name

	if err := ctrl.SetControllerReference(task, &service, r.Scheme); err != nil {
		return nil, err
	}

	if err := r.Create(ctx, &service); err != nil {
		if apierrors.IsAlreadyExists(err) {
			logger.Info("service already exists likely caused by quick subsequent reconcile")
		} else {
			return nil, err
		}
	}

	serviceRef, err := reference.GetReference(r.Scheme, &service)
	if err != nil {
		return nil, err
	}

	return serviceRef, nil
}

func (r *TaskReconciler) createIngress(ctx context.Context, task *srev1alpha1.Task, envBuild *srev1alpha1.EnvironmentBuild) (*corev1.ObjectReference, error) {
	logger := log.FromContext(ctx)

	if task.Spec.DomainName == "" {
		logger.Info("no domain name specified, skipping ingress creation")
		return nil, nil
	}

	var ingress networkingv1.Ingress
	err := r.Get(ctx, types.NamespacedName{
		Name:      task.Name,
		Namespace: task.Namespace,
	}, &ingress)

	if err == nil {
		logger.Info("ingress already exists", "ingress", ingress.Name)
		return reference.GetReference(r.Scheme, &ingress)
	}

	if !apierrors.IsNotFound(err) {
		return nil, err
	}

	prefixType := networkingv1.PathTypePrefix
	ingress = networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        task.Name,
			Namespace:   task.Namespace,
			Annotations: envBuild.Spec.IngressAnnotations,
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: task.Spec.DomainName,
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     "/",
									PathType: &prefixType,
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: task.Name,
											Port: networkingv1.ServiceBackendPort{
												Number: int32(envBuild.Spec.IngressTargetPort),
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	if envBuild.Spec.IngressTLS {
		ingressTLS := networkingv1.IngressTLS{
			Hosts: []string{task.Spec.DomainName},
		}
		if task.Spec.IngressSecretName != "" {
			ingressTLS.SecretName = task.Spec.IngressSecretName
		}
		ingress.Spec.TLS = []networkingv1.IngressTLS{ingressTLS}
	}

	if err := ctrl.SetControllerReference(task, &ingress, r.Scheme); err != nil {
		return nil, err
	}

	if err := r.Create(ctx, &ingress); err != nil {
		if apierrors.IsAlreadyExists(err) {
			logger.Info("ingress already exists likely caused by quick subsequent reconcile")
		} else {
			return nil, err
		}
	}

	ingressRef, err := reference.GetReference(r.Scheme, &ingress)
	if err != nil {
		return nil, err
	}

	return ingressRef, nil
}

func anyContainerErrors(ctx context.Context, pod *corev1.Pod) bool {
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if containerStatus.State.Terminated != nil {
			log.FromContext(ctx).Error(ErrContainerExit, "container error",
				"container", containerStatus.Name,
				"reason", containerStatus.State.Terminated.Reason,
				"exitCode", containerStatus.State.Terminated.ExitCode,
			)
			return true
		}
	}
	return false
}

func serviceIsUp(_ context.Context, service *corev1.Service) bool {
	return service.Spec.ClusterIP != ""
}
