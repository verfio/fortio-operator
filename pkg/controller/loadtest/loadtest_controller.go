package loadtest

import (
	"context"
	"strings"
	"time"

	"bytes"
	"io"

	fortiov1alpha1 "github.com/verfio/fortio-operator/pkg/apis/fortio/v1alpha1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

var log = logf.Log.WithName("controller_loadtest")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new LoadTest Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileLoadTest{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("loadtest-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource LoadTest
	err = c.Watch(&source.Kind{Type: &fortiov1alpha1.LoadTest{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner LoadTest
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &fortiov1alpha1.LoadTest{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileLoadTest{}

// ReconcileLoadTest reconciles a LoadTest object
type ReconcileLoadTest struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a LoadTest object and makes changes based on the state read
// and what is in the LoadTest.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileLoadTest) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling LoadTest")

	// Fetch the LoadTest instance
	instance := &fortiov1alpha1.LoadTest{}
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

	// Define a new Job object
	job := newJobForCR(instance)

	// Set LoadTest instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, job, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this Job already exists
	found := &batchv1.Job{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: job.Name, Namespace: job.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new Job", "Job.Namespace", job.Namespace, "Job.Name", job.Name)
		err = r.client.Create(context.TODO(), job)
		if err != nil {
			return reconcile.Result{}, err
		}

		// Job created successfully - don't requeue
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	// Job already exists - don't requeue
	reqLogger.Info("Job already exists", "Job.Namespace", found.Namespace, "Job.Name", found.Name)

	// If we already got logs from the succeeded pod - don't take logs - don't requeue
	if found.Status.Succeeded == 1 {
		return reconcile.Result{}, nil
	}

	// Take logs from succeeded pod
	reqLogger.Info("Verify if it completed or not", "Job.Namespace", found.Namespace, "Job.Name", found.Name)
	for true {
		err = r.client.Get(context.TODO(), types.NamespacedName{Name: job.Name, Namespace: job.Namespace}, found)
		if err != nil {
			return reconcile.Result{}, err
		}
		if found.Status.Succeeded == 0 {
			reqLogger.Info("Job is still running. Waiting for 10s.", "Job.Namespace", found.Namespace, "Job.Name", found.Name)
			time.Sleep(time.Second * 10)
			continue
		} else if found.Status.Succeeded == 1 { // verify that there is one succeeded pod
			for _, c := range found.Status.Conditions {
				if c.Type == "Complete" && c.Status == "True" {
					reqLogger.Info("Job competed. Fetch for pod", "Job.Namespace", found.Namespace, "Job.Name", found.Name)
					podList := &corev1.PodList{}
					labelSelector := labels.SelectorFromSet(labelsForJob(found.Name))
					listOps := &client.ListOptions{
						Namespace:     instance.Namespace,
						LabelSelector: labelSelector,
					}
					err = r.client.List(context.TODO(), listOps, podList)
					if err != nil {
						reqLogger.Error(err, "Failed to list pods.", "Job.Namespace", instance.Namespace, "Job.Name", instance.Name)
						return reconcile.Result{}, err
					}
					for _, pod := range podList.Items {
						reqLogger.Info("Found pod name:", "Pod.Namespace", pod.Namespace, "Pod.Name", pod.Name)
						reqLogger.Info("Readings logs from pod:", "Pod.Namespace", pod.Namespace, "Pod.Name", pod.Name)
						logs := getPodLogs(pod)
						if logs == "" {
							reqLogger.Info("Nil logs", "Pod.Namespace", pod.Namespace, "Pod.Name", pod.Name)
						} else {
							reqLogger.Info("Writing results to status of "+instance.Name, "Pod.Namespace", pod.Namespace, "Pod.Name", pod.Name)
							writeConditionsFromLogs(instance, logs)
						}
					}
				}
			}
			break
		}
	}
	err = r.client.Update(context.TODO(), instance)
	if err != nil {
		reqLogger.Error(err, "Failed to update instance", "Job.Namespace", instance.Namespace, "Job.Name", instance.Name)
	}
	reqLogger.Info("Finished reconciling cycle", "Job.Namespace", found.Namespace, "Job.Name", found.Name)
	return reconcile.Result{}, nil
}

func newJobForCR(cr *fortiov1alpha1.LoadTest) *batchv1.Job {
	backoffLimit := int32(4)
	labels := map[string]string{
		"app": cr.Name,
	}
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-job",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: batchv1.JobSpec{
			Template: corev1.PodTemplateSpec{
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:    "fortio",
							Image:   "fortio/fortio",
							Command: []string{"fortio", cr.Spec.Action, "-t", cr.Spec.Duration, cr.Spec.URL},
						},
					},
					RestartPolicy: "Never",
				},
			},
			BackoffLimit: &backoffLimit,
		},
	}
}

func labelsForJob(name string) map[string]string {
	return map[string]string{"job-name": name}
}

func getPodLogs(pod corev1.Pod) string {
	podLogOpts := corev1.PodLogOptions{}
	config, err := rest.InClusterConfig()
	if err != nil {
		return "error in getting config"
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return "error in getting access to K8S"
	}
	req := clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &podLogOpts)
	podLogs, err := req.Stream()
	if err != nil {
		return "error in opening stream" + req.URL().String()
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return "error in copy information from podLogs to buf"
	}
	str := buf.String()

	return str
}

func writeConditionsFromLogs(instance *fortiov1alpha1.LoadTest, logs string) {
	parsedLogs := strings.Fields(logs)
	first50persent := true
	condition := &fortiov1alpha1.LoadTestCondition{}

	for i, word := range parsedLogs {
		switch word {
		case "50%":
			if first50persent == true {
				first50persent = false
			} else {
				condition.Target50 = parsedLogs[i+1]
			}
		case "75%":
			condition.Target75 = parsedLogs[i+1]
		case "90%":
			condition.Target90 = parsedLogs[i+1]
		case "99%":
			condition.Target99 = parsedLogs[i+1]
		case "99.9%":
			condition.Target999 = parsedLogs[i+1]
		}
	}
	instance.Status.Condition = append(instance.Status.Condition, *condition)
}
