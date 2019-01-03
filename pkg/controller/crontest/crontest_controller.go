package crontest

import (
	"context"
	"time"

	"github.com/besser/cron"
	fortiov1alpha1 "github.com/verfio/fortio-operator/pkg/apis/fortio/v1alpha1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_crontest")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new CronTest Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileCronTest{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("crontest-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource CronTest
	err = c.Watch(&source.Kind{Type: &fortiov1alpha1.CronTest{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner CronTest
	err = c.Watch(&source.Kind{Type: &fortiov1alpha1.CurlTest{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &fortiov1alpha1.CronTest{},
	})
	if err != nil {
		return err
	}

	return nil
}

var s = cron.New()
var _ reconcile.Reconciler = &ReconcileCronTest{}

// ReconcileCronTest reconciles a CronTest object
type ReconcileCronTest struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
	jobs   map[string]int
}

// Reconcile reads that state of the cluster for a CronTest object and makes changes based on the state read
// and what is in the CronTest.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileCronTest) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling CronTest")

	s.Start()

	if len(r.jobs) == 0 {
		r.jobs = make(map[string]int)
	}

	instance := &fortiov1alpha1.CronTest{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)

	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			reqLogger.Info("deleting CronTest job", "id", r.jobs[request.Name])
			// Delete from cron entries
			s.Remove(cron.EntryID(r.jobs[request.Name]))
			// Delete from jobs map
			delete(r.jobs, request.Name)
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Add function to the scheduler to create required Test by given schedule
	if instance.Status.IsScheduled != true || r.jobs[instance.Name] == 0 {

		reqLogger.Info("new scheduled task", "schedule", instance.Spec.Schedule)
		id, _ := s.AddFunc(instance.Spec.Schedule, func() {
			//Define what we need to do
			t := time.Now()
			labels := map[string]string{
				"app": instance.Name,
			}
			reqLogger.Info("CronTest", "schedule", instance.Spec.Schedule, "instance.spec.croncurltest.url", instance.Spec.CronCurlTest.URL)
			if instance.Spec.CronCurlTest.URL != "" {
				curl := &fortiov1alpha1.CurlTest{
					ObjectMeta: metav1.ObjectMeta{
						Name:      instance.Name + "-" + t.Format("20060102150405"),
						Namespace: instance.Namespace,
						Labels:    labels,
					},
					Spec: instance.Spec.CronCurlTest,
				}

				reqLogger.Info("Creating a new CurlTest", "Curl.Namespace", instance.Namespace, "Curl.Name almost like", instance.Name)
				// Set CronTest instance as the owner and controller
				err := controllerutil.SetControllerReference(instance, curl, r.scheme)
				err = r.client.Create(context.TODO(), curl)
				if err != nil {
					reqLogger.Info("CronTest job error", "error", err)
				}
			}
			if instance.Spec.CronLoadTest.URL != "" {
				load := &fortiov1alpha1.LoadTest{
					ObjectMeta: metav1.ObjectMeta{
						Name:      instance.Name + "-" + t.Format("20060102150405"),
						Namespace: instance.Namespace,
						Labels:    labels,
					},
					Spec: instance.Spec.CronLoadTest,
				}

				reqLogger.Info("Creating a new LoadTest", "Load.Namespace", load.Namespace, "Load.Name", load.Name)
				// Set CronTest instance as the owner and controller
				err := controllerutil.SetControllerReference(instance, load, r.scheme)
				err = r.client.Create(context.TODO(), load)
				if err != nil {
					reqLogger.Info("CronTest job error", "error", err)
				}
			}
			if len(instance.Spec.CronTestRun.LoadTests) != 0 || len(instance.Spec.CronTestRun.CurlTests) != 0 {
				run := &fortiov1alpha1.TestRun{
					ObjectMeta: metav1.ObjectMeta{
						Name:      instance.Name + "-" + t.Format("20060102150405"),
						Namespace: instance.Namespace,
						Labels:    labels,
					},
					Spec: instance.Spec.CronTestRun,
				}

				reqLogger.Info("Creating a new TestRun", "TestRun.Namespace", run.Namespace, "TestRun.Name", run.Name)
				// Set CronTest instance as the owner and controller
				err := controllerutil.SetControllerReference(instance, run, r.scheme)
				err = r.client.Create(context.TODO(), run)
				if err != nil {
					reqLogger.Info("CronTest job error", "error", err)
				}
			}

		})
		// Save the key-value info about the job to remove it when instance was deleted
		r.jobs[instance.Name] = int(id)
		// Mark instance as Scheduled
		instance.Status.IsScheduled = true
		instance.Status.CronId = int(id)
		statusWriter := r.client.Status()
		err = statusWriter.Update(context.TODO(), instance)
		err = r.client.Update(context.TODO(), instance)
		reqLogger.Info("CronTest job", "id", id, "id from map", r.jobs[instance.Name])
	}
	return reconcile.Result{}, nil
}
