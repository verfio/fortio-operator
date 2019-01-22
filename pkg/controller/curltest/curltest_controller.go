package curltest

import (
	"bytes"
	"context"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/go-logr/logr"

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

var log = logf.Log.WithName("controller_curltest")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new CurlTest Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileCurlTest{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("curltest-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource CurlTest
	err = c.Watch(&source.Kind{Type: &fortiov1alpha1.CurlTest{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner CurlTest
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &fortiov1alpha1.CurlTest{},
	})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileCurlTest{}

// ReconcileCurlTest reconciles a CurlTest object
type ReconcileCurlTest struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a CurlTest object and makes changes based on the state read
// and what is in the CurlTest.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileCurlTest) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling CurlTest")

	// Fetch the CurlTest instance
	instance := &fortiov1alpha1.CurlTest{}
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

	// If Result is not empty - stop, we already finished with this CR
	if instance.Status.Condition.Result != "" {
		reqLogger.Info("All work with this CR is completed", "CR.Namespace", instance.Namespace, "CR.Name", instance.Name)
		return reconcile.Result{}, nil
	}

	// Define a new Job object
	job := newJobForCR(instance)

	// Set CurlTest instance as the owner and controller
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
	} else if err != nil {
		return reconcile.Result{}, err
	} else {
		// Job already exists - don't requeue
		reqLogger.Info("Job already exists", "Job.Namespace", found.Namespace, "Job.Name", found.Name)
	}

	// Take logs from succeeded pod
	reqLogger.Info("Verify if it is completed or not", "Job.Namespace", job.Namespace, "Job.Name", job.Name)
	sec := 10
	for true {
		err = r.client.Get(context.TODO(), types.NamespacedName{Name: job.Name, Namespace: job.Namespace}, found)
		if err != nil && errors.IsNotFound(err) {
			reqLogger.Info("Job is not yet created. Waiting for "+strconv.Itoa(sec)+"s.", "Job.Namespace", job.Namespace, "Job.Name", job.Name)
			time.Sleep(time.Second * 10)
			switch sec {
			case 10:
				sec = 30
			case 25:
				sec = 60
			case 60:
				sec = 200
			case 200:
				reqLogger.Info("Waited for 5 minutes, job is not created, retunring error", "Job.Namespace", job.Namespace, "Job.Name", job.Name)
				instance.Status.Condition.Result = "Failure"
				instance.Status.Condition.Error = "Failed to create job in 5 minutes"
				updateStatus(r, instance, reqLogger)
				return reconcile.Result{}, nil
			}
			continue
		} else if err != nil {
			return reconcile.Result{}, err
		}
		if found.Status.Failed == *job.Spec.BackoffLimit+1 {
			reqLogger.Info("All attempts of the job finished in error. Please review logs.", "Job.Namespace", found.Namespace, "Job.Name", found.Name)
			instance.Status.Condition.Result = "Failure"
			instance.Status.Condition.Error = "Job failed. Please review logs."
			updateStatus(r, instance, reqLogger)
			return reconcile.Result{}, nil
		} else if found.Status.Succeeded == 0 {
			reqLogger.Info("Job is still running. Waiting for 10s.", "Job.Namespace", found.Namespace, "Job.Name", found.Name)
			time.Sleep(time.Second * 10)
			continue
		} else if found.Status.Succeeded == 1 { // verify that there is one succeeded pod
			for _, c := range found.Status.Conditions {
				if c.Type == "Complete" && c.Status == "True" {
					reqLogger.Info("Job competed. Fetch for logs in pod", "Job.Namespace", found.Namespace, "Job.Name", found.Name)
					logs, err := getPodLogs(r, found)
					if err != nil {
						reqLogger.Error(err, "Failed to get logs from a pod", "instance.Namespace", instance.Namespace, "instance.Name", instance.Name)
						return reconcile.Result{}, err
					}
					reqLogger.Info("Writing results to status of the CR", "instance.Namespace", instance.Namespace, "instance.Name", instance.Name)
					writeConditionsFromLogs(instance, &logs)
					updateStatus(r, instance, reqLogger)
				}
			}
			break
		}
	}
	reqLogger.Info("Finished reconciling cycle", "instance.Namespace", instance.Namespace, "instance.Name", instance.Name)
	return reconcile.Result{}, nil
}

func getPodLogs(r *ReconcileCurlTest, job *batchv1.Job) (string, error) {
	var logs string
	podList := &corev1.PodList{}
	labelSelector := labels.SelectorFromSet(labelsForJob(job.Name))
	listOps := &client.ListOptions{
		Namespace:     job.Namespace,
		LabelSelector: labelSelector,
	}
	err := r.client.List(context.TODO(), listOps, podList)
	if err != nil {
		return logs, err
	}
	pod := &podList.Items[0]
	logs, err = getLogs(pod)
	if err != nil {
		return logs, err
	}
	return logs, nil
}

func updateStatus(r *ReconcileCurlTest, instance *fortiov1alpha1.CurlTest, reqLogger logr.Logger) {
	statusWriter := r.client.Status()
	err := statusWriter.Update(context.TODO(), instance)
	if err != nil {
		reqLogger.Error(err, "Failed to update Status of the CR using statusWriter, switching back to old way", "instance.Namespace", instance.Namespace, "instance.Name", instance.Name)
		err = r.client.Update(context.TODO(), instance)
		if err != nil {
			reqLogger.Error(err, "Failed to update Status of the CR using old way", "instance.Namespace", instance.Namespace, "instance.Name", instance.Name)
		} else {
			reqLogger.Info("Successfully updated Status of the CR", "instance.Namespace", instance.Namespace, "instance.Name", instance.Name)
		}
	} else {
		reqLogger.Info("Successfully updated Status of the CR", "instance.Namespace", instance.Namespace, "instance.Name", instance.Name)
	}
}

func writeConditionsFromLogs(instance *fortiov1alpha1.CurlTest, logs *string) {
	parsedLogs := strings.Fields(*logs)

	for _, word := range parsedLogs {
		if strings.Contains(word, instance.Spec.LookForString) {
			instance.Status.Condition.Result = "Success"
		}
	}
	if instance.Status.Condition.Result == "" {
		instance.Status.Condition.Result = "Failure"
		instance.Status.Condition.Error = "Failed to find provided string"
	}
}

func getLogs(pod *corev1.Pod) (string, error) {
	podLogOpts := corev1.PodLogOptions{}
	config, err := rest.InClusterConfig()
	if err != nil {
		return "error in getting config", err
	}
	// creates the clientset
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return "error in getting access to K8S", err
	}
	req := clientset.CoreV1().Pods(pod.Namespace).GetLogs(pod.Name, &podLogOpts)
	podLogs, err := req.Stream()
	if err != nil {
		return "error in opening stream" + req.URL().String(), err
	}
	defer podLogs.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, podLogs)
	if err != nil {
		return "error in copy information from podLogs to buf", err
	}
	str := buf.String()

	return str, nil
}

func labelsForJob(name string) map[string]string {
	return map[string]string{"job-name": name}
}

func newJobForCR(cr *fortiov1alpha1.CurlTest) *batchv1.Job {
	backoffLimit := int32(0)
	labels := map[string]string{
		"app": cr.Name,
	}

	command := []string{"fortio", "curl"}
	if cr.Spec.ContentType != "" {
		command = append(command, "-content-type", cr.Spec.ContentType)
	} else if strings.ToLower(cr.Spec.Method) == "post" {
		command = append(command, "-content-type", "text/html")
	}
	if cr.Spec.Headers != nil {
		for i := 0; i < len(cr.Spec.Headers); i++ {
			command = append(command, "-H", cr.Spec.Headers[i])
		}
	}
	if cr.Spec.Payload != "" {
		command = append(command, "-payload", cr.Spec.Payload)
	}
	if cr.Spec.PayloadSize != "" {
		command = append(command, "-payload-size", cr.Spec.PayloadSize)
	}
	if cr.Spec.MaxPayloadSizeKB != "" {
		command = append(command, "-maxpayloadsizekb", cr.Spec.MaxPayloadSizeKB)
	}
	if cr.Spec.PayloadFile != "" {
		command = append(command, "-payload-file", cr.Spec.PayloadFile)
	}
	if cr.Spec.LogLevel != "" {
		command = append(command, "-loglevel", cr.Spec.LogLevel)
	}
	// URL should be the last parameter
	if cr.Spec.URL != "" {
		command = append(command, cr.Spec.URL)
	}

	if cr.Spec.PayloadConfigMap != "" {
		// We'd like to mount configmap to local pod
		configMapDefaulMode := int32(0666)
		configMapVolumeSource := corev1.ConfigMapVolumeSource{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: cr.Spec.PayloadConfigMap,
			},
			DefaultMode: &configMapDefaulMode,
		}
		mountPath := "/var/lib/fortio"

		// Returning job with mounted configMap
		return &batchv1.Job{
			ObjectMeta: metav1.ObjectMeta{
				Name:      strings.ToLower(cr.TypeMeta.Kind) + "-" + cr.Name + "-job",
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
								Command: command,
								VolumeMounts: []corev1.VolumeMount{
									{
										Name:      cr.Spec.PayloadConfigMap,
										MountPath: mountPath,
									},
								},
							},
						},
						RestartPolicy: "Never",
						Volumes: []corev1.Volume{
							{
								Name: cr.Spec.PayloadConfigMap,
								VolumeSource: corev1.VolumeSource{
									ConfigMap: &configMapVolumeSource,
								},
							},
						},
					},
				},
				BackoffLimit: &backoffLimit,
			},
		}
	}

	// Returning job without mounted configMap
	return &batchv1.Job{
		ObjectMeta: metav1.ObjectMeta{
			Name:      strings.ToLower(cr.TypeMeta.Kind) + "-" + cr.Name + "-job",
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
							Command: command,
						},
					},
					RestartPolicy: "Never",
				},
			},
			BackoffLimit: &backoffLimit,
		},
	}
}
