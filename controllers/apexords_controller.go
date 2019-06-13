/*

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

	"github.com/go-logr/logr"
	//corev1 "k8s.io/api/core/v1"
	//"k8s.io/apimachinery/pkg/api/errors"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	//metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	"k8s.io/apimachinery/pkg/runtime"
	//ref "k8s.io/client-go/tools/reference"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	theapexordsv1 "github.com/HenryXie1/apexords-operator/api/v1"
)

// ApexOrdsReconciler reconciles a ApexOrds object
type ApexOrdsReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=theapexords.apexords-operator,resources=apexords,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=theapexords.apexords-operator,resources=apexords/status,verbs=get;update;patch

func (r *ApexOrdsReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("apexords", req.NamespacedName)

	// your logic here
	var apexords theapexordsv1.ApexOrds

	if err := r.Get(ctx, req.NamespacedName, &apexords); err != nil {
		log.Error(err, "unable to fetch CRD ApexOrds")
		// we'll ignore not-found errors, since they can't be fixed by an immediate
		// requeue (we'll need to wait for a new notification), and we can get them
		// on deleted requests.
		return ctrl.Result{}, ignoreNotFound(err)
	}

	Apexordsdbname := apexords.Spec.Dbname
	if Apexordsdbname == "" {
		// We choose to absorb the error here
		log.Error(nil, "DB name can't be empty")
		return ctrl.Result{}, nil
	}

	Apexordsordsname := apexords.Spec.Ordsname
	if Apexordsordsname == "" {
		// We choose to absorb the error here
		log.Error(nil, "Ords name can't be empty")
		return ctrl.Result{}, nil
	}

	// Get the deployments of ords with the name specified in apexords.spec

	var Ordsdeployment appsv1beta1.DeploymentList
	if err := r.List(ctx, &Ordsdeployment, client.InNamespace(req.Namespace)); err != nil {
		log.Error(err, "unable to list Ords deployments")
		return ctrl.Result{}, err
	}
	for i := 0; i < len(Ordsdeployment.Items); i++ {
		fmt.Printf("Found deployment name is %v\n", Ordsdeployment.Items[i].ObjectMeta.Name)
	}

	return ctrl.Result{}, nil
}

func ignoreNotFound(err error) error {
	if apierrs.IsNotFound(err) {
		return nil
	}
	return err
}

func (r *ApexOrdsReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&theapexordsv1.ApexOrds{}).
		Complete(r)
}
