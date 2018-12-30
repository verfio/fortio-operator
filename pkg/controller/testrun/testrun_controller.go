package testrun

import (
	"context"
	"encoding/json"
	"sort"
	"strconv"
	"strings"
	"time"

	fortiov1alpha1 "github.com/verfio/fortio-operator/pkg/apis/fortio/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_testrun")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new TestRun Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileTestRun{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("testrun-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource TestRun
	err = c.Watch(&source.Kind{Type: &fortiov1alpha1.TestRun{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner TestRun
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &fortiov1alpha1.TestRun{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileTestRun{}

// ReconcileTestRun reconciles a TestRun object
type ReconcileTestRun struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a TestRun object and makes changes based on the state read
// and what is in the TestRun.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileTestRun) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling TestRun")

	// Fetch the TestRun instance
	instance := &fortiov1alpha1.TestRun{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Pod already exists - don't requeue
	if instance.Status.Result == "Finished" {
		reqLogger.Info("Skip reconcile: Test already finished", "instance.Namespace", instance.Namespace, "instance.Name", instance.Name)
		return reconcile.Result{}, nil
	}

	// Create a map for holding order number and name of the test
	tests := make(map[int][]byte)

	// Create a slice of order numbers to range over it below
	order := make([]int, 0)

	// Range all curltests and get them into map
	for _, c := range instance.Spec.CurlTests {
		c.Action = "curl"
		i, _ := strconv.Atoi(c.Order)
		tests[i] = c.GetSpec()
		order = append(order, i)
	}

	// Range all loadtests and get them into map
	for _, l := range instance.Spec.LoadTests {
		l.Action = "load"
		i, _ := strconv.Atoi(l.Order)
		tests[i] = l.GetSpec()
		order = append(order, i)
	}

	// Sorting order in increasing order(ASC)
	sort.Ints(order)

	for _, o := range order {
		spec := make(map[string]string)
		err := json.Unmarshal(tests[o], &spec)
		if err != nil {
			reqLogger.Error(err, "Can't unmarshal spec into map")
			break
		}
		if spec["action"] == "curl" {
			test := newCurlTestCR(instance, spec, o)
			// Set TestRun instance as the owner and controller
			if err := controllerutil.SetControllerReference(instance, test, r.scheme); err != nil {
				reqLogger.Info("Error setting ControllerReference", "Test.Namespace", test.Namespace, "Test.Name", test.Name)
				break
			}
			// Check if this CurlTest already exists
			found := &fortiov1alpha1.CurlTest{}
			err = r.client.Get(context.TODO(), types.NamespacedName{Name: test.Name, Namespace: test.Namespace}, found)
			if err != nil && errors.IsNotFound(err) {
				reqLogger.Info("Creating a new CurlTest", "Test.Namespace", test.Namespace, "Test.Name", test.Name)
				err = r.client.Create(context.TODO(), test)
				if err != nil {
					reqLogger.Error(err, "Error creating new test", "Test.Namespace", test.Namespace, "Test.Name", test.Name)
					break
				}
			} else if err != nil {
				reqLogger.Error(err, "Error verifying if test already exist", "Test.Namespace", test.Namespace, "Test.Name", test.Name)
				break
			}
			for true {
				err = r.client.Get(context.TODO(), types.NamespacedName{Name: test.Name, Namespace: test.Namespace}, found)
				if err != nil && errors.IsNotFound(err) {
					reqLogger.Info("Test is not yet created. Waiting for 10s.", "Test.Namespace", test.Namespace, "Test.Name", test.Name)
					time.Sleep(time.Second * 10)
					continue
				} else if err != nil {
					reqLogger.Error(err, "Error verifying if test already exist - during loop", "Test.Namespace", test.Namespace, "Test.Name", test.Name)
					break
				}
				if found.Status.Condition.Result == "" {
					reqLogger.Info("Test is not yet finished. Waiting for 10s.", "Test.Namespace", found.Namespace, "Test.Name", found.Name)
					time.Sleep(time.Second * 10)
					continue
				} else if found.Status.Condition.Result == "Success" {
					reqLogger.Info("Test successfully finished.", "Test.Namespace", found.Namespace, "Test.Name", found.Name)
					break
				} else if found.Status.Condition.Result == "Failure" {
					reqLogger.Info("Test failed.", "Test.Namespace", found.Namespace, "Test.Name", found.Name)
					break
				}
			}
		} else if spec["action"] == "load" {
			test := newLoadTestCR(instance, spec, o)
			// Set TestRun instance as the owner and controller
			if err := controllerutil.SetControllerReference(instance, test, r.scheme); err != nil {
				reqLogger.Info("Error setting ControllerReference", "Test.Namespace", test.Namespace, "Test.Name", test.Name)
				break
			}
			// Check if this LoadTest already exists
			found := &fortiov1alpha1.LoadTest{}
			err = r.client.Get(context.TODO(), types.NamespacedName{Name: test.Name, Namespace: test.Namespace}, found)
			if err != nil && errors.IsNotFound(err) {
				reqLogger.Info("Creating a new LoadTest", "Test.Namespace", test.Namespace, "Test.Name", test.Name)
				err = r.client.Create(context.TODO(), test)
				if err != nil {
					reqLogger.Error(err, "Error creating new test", "Test.Namespace", test.Namespace, "Test.Name", test.Name)
					break
				}
			} else if err != nil {
				reqLogger.Error(err, "Error verifying if test already exist", "Test.Namespace", test.Namespace, "Test.Name", test.Name)
				break
			}
			for true {
				err = r.client.Get(context.TODO(), types.NamespacedName{Name: test.Name, Namespace: test.Namespace}, found)
				if err != nil && errors.IsNotFound(err) {
					reqLogger.Info("Test is not yet created. Waiting for 10s.", "Test.Namespace", test.Namespace, "Test.Name", test.Name)
					time.Sleep(time.Second * 10)
					continue
				} else if err != nil {
					reqLogger.Error(err, "Error verifying if test already exist - during loop", "Test.Namespace", test.Namespace, "Test.Name", test.Name)
					break
				}
				if found.Status.Condition.Result == "" {
					reqLogger.Info("Test is not yet finished. Waiting for 10s.", "Test.Namespace", test.Namespace, "Test.Name", test.Name)
					time.Sleep(time.Second * 10)
					continue
				} else if found.Status.Condition.Result == "Success" {
					reqLogger.Info("Test successfully finished.", "Test.Namespace", test.Namespace, "Test.Name", test.Name)
					break
				} else if found.Status.Condition.Result == "Failure" {
					reqLogger.Info("Test failed.", "Test.Namespace", test.Namespace, "Test.Name", test.Name)
					break
				}
			}
		} else {
			reqLogger.Info("Unrecognized action. Ignoring.")
			continue
		}
	}
	// Finishing after all tests ran
	instance.Status.Result = "Finished"
	statusWriter := r.client.Status()
	err = statusWriter.Update(context.TODO(), instance)
	if err != nil {
		err = r.client.Update(context.TODO(), instance)
		if err != nil {
			reqLogger.Error(err, "Failed to update instance", "instance.Namespace", instance.Namespace, "instance.Name", instance.Name)
		} else {
			reqLogger.Info("Successfully written results to status of the CR", "instance.Namespace", instance.Namespace, "instance.Name", instance.Name)
		}
	} else {
		reqLogger.Info("Successfully written results to status of the CR", "instance.Namespace", instance.Namespace, "instance.Name", instance.Name)
	}
	reqLogger.Info("Finished reconciling cycle", "instance.Namespace", instance.Namespace, "instance.Name", instance.Name)
	return reconcile.Result{}, nil
}

func newCurlTestCR(cr *fortiov1alpha1.TestRun, spec map[string]string, order int) *fortiov1alpha1.CurlTest {
	labels := map[string]string{
		"app": cr.Name,
	}
	o := strconv.Itoa(order)
	curlTestSpec := fortiov1alpha1.CurlTestSpec{}
	if _, ok := spec["url"]; ok {
		curlTestSpec.URL = spec["url"]
	}
	if _, ok := spec["lookForString"]; ok {
		curlTestSpec.LookForString = spec["lookForString"]
	}
	if _, ok := spec["method"]; ok {
		curlTestSpec.Method = spec["method"]
	}
	if _, ok := spec["contentType"]; ok {
		curlTestSpec.ContentType = spec["contentType"]
	}
	if _, ok := spec["payload"]; ok {
		curlTestSpec.Payload = spec["payload"]
	}
	if _, ok := spec["payloadSize"]; ok {
		curlTestSpec.PayloadSize = spec["payloadSize"]
	}
	if _, ok := spec["maxPayloadSizeKB"]; ok {
		curlTestSpec.MaxPayloadSizeKB = spec["maxPayloadSizeKB"]
	}
	if _, ok := spec["payloadFile"]; ok {
		curlTestSpec.Payload = spec["payloadFile"]
	}
	if _, ok := spec["payloadConfigMap"]; ok {
		curlTestSpec.Payload = spec["payloadConfigMap"]
	}
	return &fortiov1alpha1.CurlTest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      strings.ToLower(cr.TypeMeta.Kind) + "-" + cr.Name + "-" + o + "-" + spec["action"] + "-test",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: curlTestSpec,
	}
}

func newLoadTestCR(cr *fortiov1alpha1.TestRun, spec map[string]string, order int) *fortiov1alpha1.LoadTest {
	labels := map[string]string{
		"app": cr.Name,
	}
	o := strconv.Itoa(order)
	loadTestSpec := fortiov1alpha1.LoadTestSpec{}
	if _, ok := spec["url"]; ok {
		loadTestSpec.URL = spec["url"]
	}
	if _, ok := spec["duration"]; ok {
		loadTestSpec.Duration = spec["duration"]
	}
	if _, ok := spec["header"]; ok {
		loadTestSpec.Header = spec["header"]
	}
	if _, ok := spec["user"]; ok {
		loadTestSpec.User = spec["user"]
	}
	if _, ok := spec["password"]; ok {
		loadTestSpec.Password = spec["password"]
	}
	if _, ok := spec["qps"]; ok {
		loadTestSpec.QPS = spec["qps"]
	}
	if _, ok := spec["threads"]; ok {
		loadTestSpec.Threads = spec["threads"]
	}
	if _, ok := spec["method"]; ok {
		loadTestSpec.Method = spec["method"]
	}
	if _, ok := spec["contentType"]; ok {
		loadTestSpec.ContentType = spec["contentType"]
	}
	if _, ok := spec["payload"]; ok {
		loadTestSpec.Payload = spec["payload"]
	}
	if _, ok := spec["payloadSize"]; ok {
		loadTestSpec.PayloadSize = spec["payloadSize"]
	}
	if _, ok := spec["maxPayloadSizeKB"]; ok {
		loadTestSpec.MaxPayloadSizeKB = spec["maxPayloadSizeKB"]
	}
	if _, ok := spec["payloadFile"]; ok {
		loadTestSpec.Payload = spec["payloadFile"]
	}
	if _, ok := spec["payloadConfigMap"]; ok {
		loadTestSpec.Payload = spec["payloadConfigMap"]
	}
	return &fortiov1alpha1.LoadTest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      strings.ToLower(cr.TypeMeta.Kind) + "-" + cr.Name + "-" + o + "-" + spec["action"] + "-test",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: loadTestSpec,
	}
}
