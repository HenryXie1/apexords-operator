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
	"math/rand"
	"strings"
	"time"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	//"k8s.io/apimachinery/pkg/api/errors"
	appsv1beta1 "k8s.io/api/apps/v1beta1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	//ref "k8s.io/client-go/tools/reference"
	"k8s.io/client-go/kubernetes/scheme"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	theapexordsv1 "apexords-operator/api/v1"
	config "apexords-operator/controllers/config"
)

// ApexOrdsReconciler reconciles a ApexOrds object
type ApexOrdsReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=theapexords.apexords-operator,resources=apexords,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=theapexords.apexords-operator,resources=apexords/status,verbs=get;update;patch
var (
	//set label details for related objects
	Apexordsoperatorlabel = map[string]string{
		"app": "apexords-operator",
	}
)

//Reconcile is the core part of this operator
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
		log.Info("Found deployment name is " + Ordsdeployment.Items[i].ObjectMeta.Name)
	}

	//check if DB stateful exists, if not create one
	var DBstatefulset appsv1beta1.StatefulSetList
	if err := r.List(ctx, &DBstatefulset, client.InNamespace(req.Namespace), client.MatchingLabels(Apexordsoperatorlabel)); err != nil {
		log.Error(err, "unable to list Apexords operator DB statefulset")
		return ctrl.Result{}, err
	}
	if len(DBstatefulset.Items) == 0 {
		log.Info("unable to find Apexords operator DB statefulset,going to create new one..")
		//create a new db statefulset here
		if err := CreateDbOption(r, req, &apexords); err != nil {
			log.Error(err, "unable to create Apexords operator DB statefulset.")
		}
	} else {
		for i := 0; i < len(DBstatefulset.Items); i++ {
			log.Info("Found running Apexords operator DB statefulset name is " + DBstatefulset.Items[i].ObjectMeta.Name)
			if DBstatefulset.Items[i].ObjectMeta.Name == Apexordsdbname+"-apexords-db-sts" {
				log.Info(Apexordsdbname + "-apexords-db-sts" + "exists. Do nothing")

			} else {
				log.Info("unable to find " + Apexordsdbname + "-apexords-db-sts." + "going to create new one..")
				//create a new db statefulset here
				if err := CreateDbOption(r, req, &apexords); err != nil {
					log.Error(err, "unable to create Apexords operator DB statefulset.")
				}
			}
		}
	}

	//check if DB service exists, if not create one
	var DBsvclist corev1.ServiceList
	if err := r.List(ctx, &DBsvclist, client.InNamespace(req.Namespace), client.MatchingLabels(Apexordsoperatorlabel)); err != nil {
		log.Error(err, "unable to list Apexords operator DB service")
		return ctrl.Result{}, err
	}
	if len(DBsvclist.Items) == 0 {
		log.Info("unable to find Apexords operator DB service,going to create new one..")
		//create a new db service here
		if err := CreateDbSvcOption(r, req, &apexords); err != nil {
			log.Error(err, "unable to create Apexords operator k8s service for DB")
		}
	} else {
		for i, DBsvc := range DBsvclist.Items {
			fmt.Printf("\n%v\n", i)
			log.Info("Found running Apexords operator DB service name is " + DBsvc.ObjectMeta.Name)
			if DBsvc.ObjectMeta.Name == Apexordsdbname+"-apexords-db-svc" {
				log.Info(Apexordsdbname + "-apexords-db-svc" + " exists. Do nothing")

			} else {
				log.Info("unable to find " + Apexordsdbname + "-apexords-db-svc." + "going to create new one..")
				//create a new db service here
				if err := CreateDbSvcOption(r, req, &apexords); err != nil {
					log.Error(err, "unable to create Apexords operator DB service")
				}
			}
		}
	}

	//check if apex 19.1 is installed in the db, if not, install it
	//create sqlpluspod
	if err := CreateSqlplusPod(r, req); err != nil {
		log.Error(err, "unable to create Sqlpluspod")
	}
	return ctrl.Result{}, nil
}

func ignoreNotFound(err error) error {
	if apierrs.IsNotFound(err) {
		return nil
	}
	return err
}

//CreateSqlplusPod Function to create sqlpluspod to run installation sql
func CreateSqlplusPod(r *ApexOrdsReconciler, req ctrl.Request) error {
	ctx := context.Background()
	log := r.Log.WithValues("apexords", req.NamespacedName)
	var waitsec int64 = 10
	typeMetadata := metav1.TypeMeta{
		Kind:       "Pod",
		APIVersion: "v1",
	}
	objectMetadata := metav1.ObjectMeta{
		Name:      "sqlpluspod",
		Namespace: "default",
	}
	podSpecs := corev1.PodSpec{
		//ImagePullSecrets: []corev1.LocalObjectReference{{
		//	Name: "iad-ocir-secret",
		//}},
		Containers: []corev1.Container{{
			Name:  "sqlpluspod",
			Image: "iad.ocir.io/espsnonprodint/autostg/instantclient-apex19:v1",
		}},
		TerminationGracePeriodSeconds: &waitsec,
	}
	pod := corev1.Pod{
		TypeMeta:   typeMetadata,
		ObjectMeta: objectMetadata,
		Spec:       podSpecs,
	}
	log.Info("Creating sqlpluspod .......")

	if err := r.Create(ctx, &pod); err != nil {
		log.Error(err, "unable to create sqlplus pod")
		return err
	}
	podstatus := &corev1.Pod{}
	verifyPodState := func() bool {
		if err := r.Get(ctx, client.ObjectKey{
			Namespace: req.NamespacedName.Namespace,
			Name:      "sqlpluspod",
		}, podstatus); err != nil {
			return false
		}
		if podstatus.Status.Phase == corev1.PodRunning {
			return true
		}
		return false
	}
	//8 min timeout for starting pod
	for i := 0; i < 96; i++ {
		if !verifyPodState() {
			log.Info("waiting for sqlpluspod to start.......")
			time.Sleep(5 * time.Second)

		} else {
			log.Info("sqlpluspod started.......")
			return nil
		}
	}
	log.Error(nil, "Timeout to start sqlplus pod")
	return nil

}

//CreateDbSvcOption is to create DB service in K8S
func CreateDbSvcOption(r *ApexOrdsReconciler, req ctrl.Request, apexords *theapexordsv1.ApexOrds) error {
	ctx := context.Background()
	log := r.Log.WithValues("apexords", req.NamespacedName)

	log.Info("Creating DB service :" + apexords.Spec.Dbname + "-apexords-db-svc")

	//Update service name
	var oradbsvc *corev1.Service
	var oradbselector = map[string]string{
		"oradbsts": apexords.Spec.Dbname + "-StsSelector",
	}

	var oradbsvcownerref = []metav1.OwnerReference{{
		Kind:       apexords.TypeMeta.Kind,
		APIVersion: apexords.TypeMeta.APIVersion,
		Name:       apexords.ObjectMeta.Name,
		UID:        apexords.ObjectMeta.UID,
	}}

	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode([]byte(config.OradbSvcyml), nil, nil)
	if err != nil {
		log.Error(err, "can't deserialize oradb service yaml")
	}
	oradbsvc = obj.(*corev1.Service)
	oradbsvc.ObjectMeta.Name = apexords.Spec.Dbname + "-apexords-db-svc"
	oradbsvc.ObjectMeta.Namespace = req.NamespacedName.Namespace
	oradbsvc.ObjectMeta.OwnerReferences = oradbsvcownerref // add owner reference, so easy to clean up
	oradbsvc.Spec.Selector = oradbselector

	if err := r.Create(ctx, oradbsvc); err != nil {
		log.Error(err, "unable to create DB service")
		return err
	}

	return nil
}

//CreateDbOption function to DB statefulset
func CreateDbOption(r *ApexOrdsReconciler, req ctrl.Request, apexords *theapexordsv1.ApexOrds) error {
	ctx := context.Background()
	log := r.Log.WithValues("apexords", req.NamespacedName)
	apexordsdbstsname := apexords.Spec.Dbname + "-apexords-db-sts"
	log.Info("Creating DB statefulset :" + apexordsdbstsname)
	oradbpasswd := Autopasswd()
	log.Info("DB sys password is: " + oradbpasswd)

	// complete db statefulset  settings
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode([]byte(config.OradbStsyml), nil, nil)
	if err != nil {
		log.Error(err, "can't deserialize oradb sts yaml")
	}
	oradbsts := obj.(*appsv1.StatefulSet)
	oradbsts.ObjectMeta.Name = apexordsdbstsname
	oradbsts.ObjectMeta.Namespace = req.NamespacedName.Namespace

	//Update selector
	var oradbselector = map[string]string{
		"oradbsts": apexords.Spec.Dbname + "-StsSelector",
	}
	var oradbownerref = []metav1.OwnerReference{{
		Kind:       apexords.TypeMeta.Kind,
		APIVersion: apexords.TypeMeta.APIVersion,
		Name:       apexords.ObjectMeta.Name,
		UID:        apexords.ObjectMeta.UID,
	}}
	oradbsts.Spec.Selector.MatchLabels = oradbselector
	//Update ORACLE_SID ,ORACLE_PDB,ORACLE_PWD
	oradbsts.Spec.Template.Spec.Containers[0].Env[0].Value = strings.ToUpper(apexords.Spec.Dbname)
	oradbsts.Spec.Template.Spec.Containers[0].Env[1].Value = strings.ToUpper(apexords.Spec.Dbservice)
	oradbsts.Spec.Template.Spec.Containers[0].Env[2].Value = oradbpasswd
	oradbsts.Spec.Template.ObjectMeta.Labels = oradbselector
	oradbsts.ObjectMeta.OwnerReferences = oradbownerref // add owner reference, so easy to clean up
	//update volume mouth and template name
	oradbvolname := apexords.Spec.Dbname + "-db-pv-storage"
	oradbsts.Spec.Template.Spec.Containers[0].VolumeMounts[0].Name = oradbvolname
	oradbsts.Spec.VolumeClaimTemplates[0].ObjectMeta.Name = oradbvolname
	//fmt.Printf("%v#\n",o.oradbsts.Spec.VolumeClaimTemplates)

	if err := r.Create(ctx, oradbsts); err != nil {
		log.Error(err, "unable to create DB statefulset")
		return err
	}

	return nil
}

//Autopasswd is to generate random password for sys, apex, ords schemas
func Autopasswd() string {
	rand.Seed(time.Now().UnixNano())
	chars := []rune("ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
		"abcdefghijklmnopqrstuvwxyz" +
		"0123456789")
	length := 8
	var b strings.Builder
	for i := 0; i < length; i++ {
		b.WriteRune(chars[rand.Intn(len(chars))])
	}
	str := b.String()
	return str
}

func (r *ApexOrdsReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&theapexordsv1.ApexOrds{}).
		Complete(r)
}
