package v1alpha1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	//	runtime    "k8s.io/apimachinery/pkg/runtime"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// TestRunSpec defines the desired state of TestRun

//type Test interface {
//	isFailed()  bool
//	GetObkectKind()  schema.ObjectKind
//	DeepCopyObject() Object
//
//}

// TestRunSpec defines the desired state of TestRun
type TestRunSpec struct {
	//	StopOnFailure	bool	`json:"stopOnFailure"`
	//	Items	[]runtime.Object  `json:"items"`
	LoadTests []LoadTestSpec `json:"load"`
	CurlTests []CurlTestSpec `json:"curl"`

	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
}

//type Test struct {
//	Kind	string	`json:"string"`
//
//}

// TestRunStatus defines the observed state of TestRun
type TestRunStatus struct {
	Result string `json:"result"`
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TestRun is the Schema for the testruns API
// +k8s:openapi-gen=true
type TestRun struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TestRunSpec   `json:"spec,omitempty"`
	Status TestRunStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// TestRunList contains a list of TestRun
type TestRunList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TestRun `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TestRun{}, &TestRunList{})
}
