package v1alpha1

import (
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// LoadTestSpec defines the desired state of LoadTest
type LoadTestSpec struct {
	URL         string `json:"url"`
	Duration    string `json:"duration"`
	Header      string `json:"header"`
	User        string `json:"user"`
	Password    string `json:"password"`
	QPS         string `json:"qps"`
	Threads     string `json:"threads"`
	Method      string `json:"method"`
	ContentType string `json:"contentType"`
	// parameters for TestRun
	Action        string `json:"action"`
	Order         string `json:"order"`
	StopOnFailure string `json:"stopOnFailure"` // not implemented yet
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
}

// LoadTestStatus defines the observed state of LoadTest
type LoadTestStatus struct {
	Condition LoadTestCondition `json:"condition"`
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
}

// LoadTestCondition defines one item of Condition in LoadTestStatus
type LoadTestCondition struct {
	Target50  string `json:"50%"`
	Target75  string `json:"75%"`
	Target90  string `json:"90%"`
	Target99  string `json:"99%"`
	Target999 string `json:"99.9%"`
	RespTime  string `json:"avg"`
	QPS       string `json:"qps"`
	Result    string `json:"result"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// LoadTest is the Schema for the loadtests API
// +k8s:openapi-gen=true
type LoadTest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LoadTestSpec   `json:"spec,omitempty"`
	Status LoadTestStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// LoadTestList contains a list of LoadTest
type LoadTestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LoadTest `json:"items"`
}

func init() {
	SchemeBuilder.Register(&LoadTest{}, &LoadTestList{})
}

// GetSpec() returns full LoadTestSpec as json in []byte
func (l *LoadTestSpec) GetSpec() []byte {
	s, err := json.Marshal(l)
	if err != nil {
		return []byte(err.Error())
	}
	return s
}
