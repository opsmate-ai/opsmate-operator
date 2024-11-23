package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// EnvrionmentBuildSpec defines the desired state of EnvrionmentBuild.
type EnvrionmentBuildSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// +kubebuilder:validation:Required
	// Template is the execution environment definition in form of a pod template
	Template corev1.PodTemplateSpec `json:"template,omitempty"`
}

// EnvrionmentBuildStatus defines the observed state of EnvrionmentBuild.
type EnvrionmentBuildStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Number of tasks uses this environmentBuild
	TaskCount int `json:"taskCount,omitempty"`

	// Number of threads uses this environmentBuild
	ThreadCount int `json:"threadCount,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// EnvrionmentBuild is the Schema for the envrionmentbuilds API.
type EnvrionmentBuild struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EnvrionmentBuildSpec   `json:"spec,omitempty"`
	Status EnvrionmentBuildStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// EnvrionmentBuildList contains a list of EnvrionmentBuild.
type EnvrionmentBuildList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []EnvrionmentBuild `json:"items"`
}

func init() {
	SchemeBuilder.Register(&EnvrionmentBuild{}, &EnvrionmentBuildList{})
}
