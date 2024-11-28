package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// TaskSpec defines the desired state of Task.
type TaskSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// The user ID that initiated the Task
	// +kubebuilder:validation:Required
	// +kubebuilder:default:="anonymous"
	UserID string `json:"userID"`

	// The environmentBuild that the task will use, must be in the same namespace as the task
	// +kubebuilder:validation:Required
	EnvironmentBuildName string `json:"environmentBuildName"`

	// Description is the description for the task
	// +kubebuilder:validation:Optional
	Description string `json:"description,omitempty"`

	// Context is the execution context for the task
	// +kubebuilder:validation:Required
	Context string `json:"context"`

	// DomainName is the domain name for the task ingress
	// +kubebuilder:validation:Optional
	DomainName string `json:"domainName,omitempty"`

	// +kubebuilder:validation:Optional
	// IngressAnnotations are the annotations for the ingress
	IngressAnnotations map[string]string `json:"ingressAnnotations,omitempty"`

	// IngressSecretName is the name of the secret to use for the ingress TLS
	// +kubebuilder:validation:Optional
	IngressSecretName string `json:"ingressSecretName,omitempty"`
}

// TaskStatus defines the observed state of Task.
type TaskStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:validation:Enum=PENDING;SCHEDULED;RUNNING;TERMINATING;ERROR;NOT_FOUND
	// +kubebuilder:default:=PENDING
	State string `json:"state,omitempty"`

	// Pod is the reference to the pod that is running the task
	// +optional
	Pod *corev1.ObjectReference `json:"pod,omitempty"`

	// Service is the reference to the service that is running the task
	// +optional
	Service *corev1.ObjectReference `json:"service,omitempty"`

	// Ingress is the reference to the ingress that is running the task
	// +optional
	Ingress *corev1.ObjectReference `json:"ingress,omitempty"`

	// Reason for the error
	// +optional
	Reason string `json:"reason,omitempty"`

	// The conditions of the task
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`

	// Internal IP of the Task Pod
	// +optional
	InternalIP string `json:"internalIP,omitempty"`

	// Cluster IP of the service
	// +optional
	ServiceIP string `json:"serviceIP,omitempty"`

	// Ingress Domain
	// +optional
	IngressDomain string `json:"ingressDomain,omitempty"`

	// The time when the task pod is up and running
	// +optional
	AllocatedAt *metav1.Time `json:"allocatedAt,omitempty"`

	// Output of the task
	// +optional
	Output string `json:"output,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// +kubebuilder:printcolumn:name="UserID",type=string,JSONPath=`.spec.userID`
// +kubebuilder:printcolumn:name="EnvironmentBuild",type=string,JSONPath=`.spec.environmentBuildName`
// +kubebuilder:printcolumn:name="Output",type=string,JSONPath=`.status.output`
// +kubebuilder:printcolumn:name="State",type=string,JSONPath=`.status.state`
// +kubebuilder:printcolumn:name="Reason",type=string,JSONPath=`.status.reason`
// +kubebuilder:printcolumn:name="InternalIP",type=string,JSONPath=`.status.internalIP`
// +kubebuilder:printcolumn:name="ServiceIP",type=string,JSONPath=`.status.serviceIP`
// +kubebuilder:printcolumn:name="IngressDomain",type=string,JSONPath=`.status.ingressDomain`
// Task is the Schema for the tasks API.
type Task struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TaskSpec   `json:"spec,omitempty"`
	Status TaskStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// TaskList contains a list of Task.
type TaskList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Task `json:"items"`
}

const (
	StatePending     = "PENDING"     // Task CRD just gets created
	StateScheduled   = "SCHEDULED"   // Pod is created
	StateRunning     = "RUNNING"     // Pod is scheduled and running
	StateTerminating = "TERMINATING" // Pod is terminated
	StateError       = "ERROR"       // Task in Error State
	StateNotFound    = "NOT_FOUND"   // Task not found

	ConditionTaskPodRunning   = "TaskPodRunning"
	ConditionTaskPodScheduled = "TaskPodScheduled"
	ConditionTaskServiceUp    = "TaskServiceUp"
	ConditionTaskIngressReady = "IngressReady"
)

func init() {
	SchemeBuilder.Register(&Task{}, &TaskList{})
}
