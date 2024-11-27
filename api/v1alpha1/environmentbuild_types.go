package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// EnvironmentBuildSpec defines the desired state of EnvironmentBuild.
type EnvironmentBuildSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:validation:Required
	// Template is the execution environment definition in form of a pod template
	Template corev1.PodTemplateSpec `json:"template,omitempty"`

	// +kubebuilder:validation:Required
	// Service is the service definition for the environment build
	Service corev1.ServiceSpec `json:"service,omitempty"`

	// +kubebuilder:validation:Optional
	// IngressAnnotations are the annotations for the ingress
	IngressAnnotations map[string]string `json:"ingressAnnotations,omitempty"`

	// +kubebuilder:validation:Optional
	// Whether to use TLS for the ingress
	IngressTLS bool `json:"ingressTLS,omitempty"`

	// +kubebuilder:validation:Optional
	// IngressTargetPort is the service port for the ingress to target
	IngressTargetPort int `json:"ingressTargetPort,omitempty"`
}

// EnvironmentBuildStatus defines the observed state of EnvironmentBuild.
type EnvironmentBuildStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Number of tasks uses this environmentBuild
	TaskCount int `json:"taskCount,omitempty"`

	// Number of threads uses this environmentBuild
	ThreadCount int `json:"threadCount,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// +kubebuilder:printcolumn:name="TaskCount",type=integer,JSONPath=`.status.taskCount`
// +kubebuilder:printcolumn:name="ThreadCount",type=integer,JSONPath=`.status.threadCount`
// EnvironmentBuild is the Schema for the environmentbuilds API.
type EnvironmentBuild struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EnvironmentBuildSpec   `json:"spec,omitempty"`
	Status EnvironmentBuildStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// EnvironmentBuildList contains a list of EnvironmentBuild.
type EnvironmentBuildList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EnvironmentBuild `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EnvironmentBuild{}, &EnvironmentBuildList{})
}
