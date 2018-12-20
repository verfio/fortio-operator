package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// CurlTestSpec defines the desired state of CurlTest
type CurlTestSpec struct {
	URL		string  `json:"url"`
	WaitForCode	string  `json:"waitForCode"`
	LookForString	string	`json:"lookForString"`
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
}

// CurlTestStatus defines the observed state of CurlTest
type CurlTestStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CurlTest is the Schema for the curltests API
// +k8s:openapi-gen=true
type CurlTest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CurlTestSpec   `json:"spec,omitempty"`
	Status CurlTestStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CurlTestList contains a list of CurlTest
type CurlTestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CurlTest `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CurlTest{}, &CurlTestList{})
}
