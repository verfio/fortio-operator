package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CronTestSpec defines the desired state of CronTest
type CronTestSpec struct {
	Schedule string `json:"schedule"`
	CronLoadTest LoadTestSpec `json:"load"`
	CronCurlTest CurlTestSpec `json:"curl"`
	CronTestRun TestRunSpec `json:"testRun"`
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
}

// CronTestStatus defines the observed state of CronTest
type CronTestStatus struct {
	IsScheduled bool `json:"isScheduled"`
	CronID	int	`json:"cronId"`
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CronTest is the Schema for the crontests API
// +k8s:openapi-gen=true
type CronTest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CronTestSpec   `json:"spec,omitempty"`
	Status CronTestStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CronTestList contains a list of CronTest
type CronTestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CronTest `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CronTest{}, &CronTestList{})
}
