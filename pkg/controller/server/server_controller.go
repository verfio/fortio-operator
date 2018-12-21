package server

import (
	"context"

	fortiov1alpha1 "github.com/verfio/fortio-operator/pkg/apis/fortio/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	appsv1 "k8s.io/api/apps/v1"
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

var log = logf.Log.WithName("controller_server")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new Server Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileServer{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("server-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource Server
	err = c.Watch(&source.Kind{Type: &fortiov1alpha1.Server{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner Server
	err = c.Watch(&source.Kind{Type: &appsv1.ReplicaSet{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &fortiov1alpha1.Server{},
	})
	if err != nil {
		return err
	}

	err = c.Watch(&source.Kind{Type: &corev1.Service{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &fortiov1alpha1.Server{},
	})
	if err != nil {
		return err
	}
	return nil
}

var _ reconcile.Reconciler = &ReconcileServer{}

// ReconcileServer reconciles a Server object
type ReconcileServer struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Server object and makes changes based on the state read
// and what is in the Server.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *ReconcileServer) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling Server")

	// Fetch the Server instance
	instance := &fortiov1alpha1.Server{}
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

	// Define a new ReplicaSet object
	rs := newReplicaSetForCR(instance)

	// Define a new Service object
	svc := newServiceForCR(instance)

	// Set Server instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, rs, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	if err := controllerutil.SetControllerReference(instance, svc, r.scheme); err != nil {
		return reconcile.Result{}, err
	}
	// Check if  ReplicaSet  exists
	found := &appsv1.ReplicaSet{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: rs.Name, Namespace: rs.Namespace}, found)

	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new ReplicaSet", "ReplicaSet.Namespace", rs.Namespace, "ReplicaSet.Name", rs.Name)
		err = r.client.Create(context.TODO(), rs)
		if err != nil {
			return reconcile.Result{}, err
		}

		// ReplicaSet created successfully - let's check Service now, don't reconcile yet
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}


	// Check if Service already exists
	foundSvc := &corev1.Service{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: svc.Name, Namespace: svc.Namespace}, foundSvc)

	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new Service", "Service.Namespace", svc.Namespace, "Service.Name", svc.Name)
		err = r.client.Create(context.TODO(), svc)
		if err != nil {
			return reconcile.Result{}, err
		}

		// Service created saccessfully - don't requeue
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	// Service already exists - don't requeue
	reqLogger.Info("Skip reconcile: Service and ReplicaSet already exist", "ReplicaSet.Name", found.Name, "Service.Name", foundSvc.Name, )
	return reconcile.Result{}, nil
}

// newReplicaSetForCR returns a fortio ReplicaSet with the same name/namespace as the cr
func newReplicaSetForCR(cr *fortiov1alpha1.Server) *appsv1.ReplicaSet {

	// We'd like to mount configmap to local pod
	configMapDefaulMode := int32(0666)
	configMapVolumeSource := corev1.ConfigMapVolumeSource{
		LocalObjectReference: corev1.LocalObjectReference{
			Name: "fortio-data-dir",
		  },
		DefaultMode: &configMapDefaulMode,
		}
	 mountPath := "/var/lib/fortio"

	labels := map[string]string{
		"app": cr.Name,
	}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-pod",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    "fortio",
					Image:   "fortio/fortio",
					Command: []string{"fortio","server"},
					VolumeMounts: []corev1.VolumeMount{
						{
						     Name:      "fortio-data-dir",
						     MountPath: mountPath,
						},
				        },
				},
			},
			 Volumes: []corev1.Volume{
				{
					Name: "fortio-data-dir",
					VolumeSource: corev1.VolumeSource{
						ConfigMap: &configMapVolumeSource,
						},
				},
			},
		},
	}
	selector := &metav1.LabelSelector {
		MatchLabels: labels,
	}
	rsSpec := appsv1.ReplicaSetSpec{
		Selector: selector,
		Template: corev1.PodTemplateSpec{
			Spec: pod.Spec,
			ObjectMeta: pod.ObjectMeta,
		},

	}

	return &appsv1.ReplicaSet {
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: rsSpec,
	}

}

// newServiceForCR returns a fortio Servicet with the same name/namespace as the cr
func newServiceForCR(cr *fortiov1alpha1.Server) *corev1.Service {
	labels := map[string]string{
		"app": cr.Name,
	}

	svcSpec := corev1.ServiceSpec{
		Selector: labels,
		Ports: []corev1.ServicePort{
				corev1.ServicePort{
					Name: "http",
					Port: int32(8080),
			},
		},
		Type: corev1.ServiceTypeLoadBalancer,
	}

	return &corev1.Service {
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: svcSpec,
	}

}
