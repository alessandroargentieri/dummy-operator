/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	dummyv1 "github.com/alessandroargentieri/dummy-operator/api/v1"
	//~~~~~

	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
)

// DummyReconciler reconciles a Dummy object
type DummyReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=apps.alessandroargentieri.com,resources=dummies,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps.alessandroargentieri.com,resources=dummies/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=apps.alessandroargentieri.com,resources=dummies/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=rbac.authorization.k8s.io,resources=roles;rolebindings,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Dummy object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *DummyReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	l := log.FromContext(ctx)
	l.Info("Enter Reconcile", "req", req)
	var err error

	// ~~~~~ fetching the Dummy CR resource object of the reconciling ~~~~~ //
	dummy := &dummyv1.Dummy{}
	err = r.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: req.Namespace}, dummy)
	if err != nil {
		l.Error(err, "Error while fetching Dummy CR: "+err.Error())
		if !errors.IsNotFound(err) {
			l.Error(err, "Dummy "+req.Name+" not found!")
		}
		return ctrl.Result{}, err
	}

	// ~~~~~ fetching the Deployment created under the hood by the DummyOperator ~~~~~ //

	dummyDeployment := &v1.Deployment{}
	deploymentName := req.Name + "-deployment"
	err = r.Get(ctx, types.NamespacedName{Name: deploymentName, Namespace: req.Namespace}, dummyDeployment)
	if err != nil {
		if errors.IsNotFound(err) {
			l.Info("DummyDeployment " + deploymentName + " not found. Creating...")
			dummyDeployment = &v1.Deployment{
				ObjectMeta: metav1.ObjectMeta{
					Name:      deploymentName,
					Namespace: req.Namespace,
					Labels: map[string]string{
						"app.kubernetes.io/name": req.Name,
					},
				},
				Spec: v1.DeploymentSpec{
					Selector: &metav1.LabelSelector{
						MatchLabels: map[string]string{
							"app.kubernetes.io/name": req.Name,
						},
					},
					Replicas: &[]int32{int32(dummy.Spec.DummyDeployment.Replicas)}[0],
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app.kubernetes.io/name": req.Name,
							},
						},
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{
								{
									Name:  req.Name + "-container",
									Image: dummy.Spec.DummyDeployment.Image,
								},
							},
						},
					},
				},
			}
			if err = r.Create(ctx, dummyDeployment); err != nil {
				l.Error(err, fmt.Sprintf("Error while creating the DummyDeployment %v+", dummyDeployment))
			} else {
				l.Info("Deployment " + deploymentName + " correctly created")
			}
		} else {
			l.Error(err, "Error while fetching DummyDeployment "+deploymentName)
			return ctrl.Result{}, err
		}
	}

	// ~~~~~ fetching the Service created under the hood by the DummyOperator ~~~~~ //

	dummyService := &corev1.Service{}
	serviceName := req.Name + "-service"
	err = r.Get(ctx, types.NamespacedName{Name: serviceName, Namespace: req.Namespace}, dummyService)
	if err != nil {
		if errors.IsNotFound(err) {
			l.Info("DummyService " + serviceName + " not found. Creating...")
			dummyService = &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      serviceName,
					Namespace: req.Namespace,
				},
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{
						{
							Port:       int32(dummy.Spec.DummyService.Port),
							TargetPort: intstr.Parse(fmt.Sprintf("%d", dummy.Spec.DummyService.TargetPort)),
							NodePort:   int32(dummy.Spec.DummyService.NodePort),
						},
					},
					Selector: map[string]string{
						// this allows connecting the service to the deployment pods with the same label
						"app.kubernetes.io/name": req.Name,
					},
					Type: toServiceType(dummy.Spec.DummyService.Type),
				},
			}
			if err = r.Create(ctx, dummyService); err != nil {
				l.Error(err, fmt.Sprintf("Error while creating the DummyService %v+", dummyService))
			} else {
				l.Info("Service " + serviceName + " correctly created")
			}
		} else {
			l.Error(err, "Error while fetching DummyService "+serviceName)
			return ctrl.Result{}, err
		}
	}

	if dummyDeployment.Status.AvailableReplicas == int32(dummy.Spec.DummyDeployment.Replicas) {
		l.Info("Setting status to Ready")
		dummy.Status.Status = "Ready"
	} else {
		l.Info(fmt.Sprintf("DummyDeployment available replicas %d", dummyDeployment.Status.AvailableReplicas))
		l.Info(fmt.Sprintf("Dummy requested replicas %d", dummy.Spec.DummyDeployment.Replicas))
		dummy.Status.Status = "WaitingForReplicas"
	}
	r.Status().Update(ctx, dummy)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *DummyReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&dummyv1.Dummy{}).
		Complete(r)
}

func toServiceType(serviceType string) corev1.ServiceType {
	switch serviceType {
	case string(corev1.ServiceTypeClusterIP):
		return corev1.ServiceTypeClusterIP
	case string(corev1.ServiceTypeNodePort):
		return corev1.ServiceTypeNodePort
	case string(corev1.ServiceTypeLoadBalancer):
		return corev1.ServiceTypeLoadBalancer
	}
	return corev1.ServiceTypeClusterIP
}
