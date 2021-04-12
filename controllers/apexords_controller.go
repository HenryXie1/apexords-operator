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
	"os"
	"strings"
	"time"

	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	ctrlconfig "sigs.k8s.io/controller-runtime/pkg/client/config"

	theapexordsv1 "apexords-operator/api/v1"
	config "apexords-operator/controllers/config"
)

// ApexOrdsReconciler reconciles a ApexOrds object
type ApexOrdsReconciler struct {
	client.Client
	Log        logr.Logger
	Scheme     *runtime.Scheme
	Dbpassword string
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
		return ctrl.Result{}, ignoreNotFound(err)
	}

	if apexords.Spec.Dbname == "" {
		log.Error(nil, "DB name can't be empty")
		return ctrl.Result{}, nil
	}
	if apexords.Spec.Ordsname == "" {
		log.Error(nil, "Ords name can't be empty")
		return ctrl.Result{}, nil
	}
	// set default db port to 1521
	if apexords.Spec.Dbport == "" {
		apexords.Spec.Dbport = "1521"
	}

	//create password for DB
	r.Dbpassword = Autopasswd(apexords.Spec.Dbname + apexords.Spec.Ordsname)

	// Get the deployments of ords with the name specified in apexords.spec
	var Ordsdeployment appsv1.DeploymentList
	if err := r.List(ctx, &Ordsdeployment, client.InNamespace(req.Namespace)); err != nil {
		log.Error(err, "unable to list Ords deployments")
		return ctrl.Result{}, err
	}
	for i := 0; i < len(Ordsdeployment.Items); i++ {
		log.Info("Found deployment name is " + Ordsdeployment.Items[i].ObjectMeta.Name)
	}

	//install DB statefulset
	if err := CreateDbstsOption(r, req, &apexords); err != nil {
		log.Error(err, "unable to create Apex on DB")
	}

	//install apex 19.1 in the db,password would be same as sys
	if err := CreateApexOption(r, req, &apexords); err != nil {
		log.Error(err, "unable to create Apex on DB")
	}

	//install ords and http and load balancer
	if err := CreateOrdsOption(r, req, &apexords); err != nil {
		log.Error(err, "unable to create Http,Ords")
	}

	return ctrl.Result{}, nil
}

func ignoreNotFound(err error) error {
	if apierrs.IsNotFound(err) {
		return nil
	}
	return err
}

//CreateOrdsOption to create http and ords deployments plus load balancer
func CreateOrdsOption(r *ApexOrdsReconciler, req ctrl.Request, apexords *theapexordsv1.ApexOrds) error {
	ctx := context.Background()
	log := r.Log.WithValues("apexords", req.NamespacedName)
	apexordsordsdeployname := apexords.Spec.Ordsname + "-apexords-ords-deployment"
	log.Info("Creating Ords deployment :" + apexordsordsdeployname)
	// complete http and ords deployment  settings

	//update sys apex passwords, dbhost, db service in yaml
	config.Ordsconfigmapyml = strings.ReplaceAll(config.Ordsconfigmapyml, "replacepwdapexordsauto", r.Dbpassword)
	config.Ordsconfigmapyml = strings.ReplaceAll(config.Ordsconfigmapyml, "ordsautodbhost", apexords.Spec.Dbname+"-apexords-db-svc")
	config.Ordsconfigmapyml = strings.ReplaceAll(config.Ordsconfigmapyml, "ordsautodbport", apexords.Spec.Dbport)
	config.Ordsconfigmapyml = strings.ReplaceAll(config.Ordsconfigmapyml, "ordsautodbservice", apexords.Spec.Dbservice)
	config.Ordsconfigmapyml = strings.ReplaceAll(config.Ordsconfigmapyml, "replacepwdsysordsauto", r.Dbpassword)

	//complete ords settings
	decode := scheme.Codecs.UniversalDeserializer().Decode
	obj, _, err := decode([]byte(config.Ordsyml), nil, nil)
	if err != nil {
		log.Error(err, "can't deserialize Ords deployment yaml")
	}
	//Update selector and owner reference
	var ordsselector = map[string]string{
		"ordsauto": apexords.Spec.Ordsname + "-DeploymentSelector",
	}
	var apexordsownerref = []metav1.OwnerReference{{
		Kind:       apexords.TypeMeta.Kind,
		APIVersion: apexords.TypeMeta.APIVersion,
		Name:       apexords.ObjectMeta.Name,
		UID:        apexords.ObjectMeta.UID,
	}}
	ordsdeployment := obj.(*appsv1.Deployment)
	ordsdeployment.ObjectMeta.Name = apexordsordsdeployname
	ordsdeployment.ObjectMeta.Namespace = req.NamespacedName.Namespace
	ordsdeployment.ObjectMeta.OwnerReferences = apexordsownerref // add owner reference, so easy to clean up
	ordsdeployment.Spec.Selector.MatchLabels = ordsselector
	ordsdeployment.Spec.Template.ObjectMeta.Labels = ordsselector
	ordsdeployment.Spec.Template.Spec.Volumes[0].VolumeSource.ConfigMap.LocalObjectReference = corev1.LocalObjectReference{Name: apexords.Spec.Ordsname + "-apexords-http-cm"}
	ordsdeployment.Spec.Template.Spec.Volumes[1].VolumeSource.ConfigMap.LocalObjectReference = corev1.LocalObjectReference{Name: apexords.Spec.Ordsname + "-apexords-ords-cm"}

	//Update LB service name
	obj, _, err = decode([]byte(config.OrdsLBsvcyml), nil, nil)
	if err != nil {
		log.Error(err, "can't deserialize Ords Load balancer service yaml")
	}
	ordssvc := obj.(*corev1.Service)
	ordssvc.ObjectMeta.Name = apexords.Spec.Ordsname + "-apexords-svc"
	ordssvc.ObjectMeta.Namespace = req.NamespacedName.Namespace
	ordssvc.ObjectMeta.OwnerReferences = apexordsownerref // add owner reference, so easy to clean up created related objects
	ordssvc.Spec.Selector = ordsselector

	//Update nodeport service name
	obj, _, err = decode([]byte(config.OrdsNodePortsvcyml), nil, nil)
	if err != nil {
		log.Error(err, "can't deserialize Ords Load balancer service yaml")
	}
	ordsnodeportsvc := obj.(*corev1.Service)
	ordsnodeportsvc.ObjectMeta.Name = apexords.Spec.Ordsname + "-apexords-nodeport-svc"
	ordsnodeportsvc.ObjectMeta.Namespace = req.NamespacedName.Namespace
	ordsnodeportsvc.ObjectMeta.OwnerReferences = apexordsownerref // add owner reference, so easy to clean up created related objects
	ordsnodeportsvc.Spec.Selector = ordsselector

	//complete ords and http configmap settings
	obj, _, err = decode([]byte(config.Ordsconfigmapyml), nil, nil)
	if err != nil {
		log.Error(err, "can't deserialize Ords configmap yaml")
	}
	ordsconfigmap := obj.(*corev1.ConfigMap)
	ordsconfigmap.ObjectMeta.Name = apexords.Spec.Ordsname + "-apexords-ords-cm"
	ordsconfigmap.ObjectMeta.Namespace = req.NamespacedName.Namespace
	ordsconfigmap.ObjectMeta.OwnerReferences = apexordsownerref // add owner reference, so easy to clean up

	obj, _, err = decode([]byte(config.Httpconfigmapyml), nil, nil)
	if err != nil {
		log.Error(err, "can't deserialize Ords configmap yaml")
	}
	httpconfigmap := obj.(*corev1.ConfigMap)
	httpconfigmap.ObjectMeta.Name = apexords.Spec.Ordsname + "-apexords-http-cm"
	httpconfigmap.ObjectMeta.Namespace = req.NamespacedName.Namespace
	httpconfigmap.ObjectMeta.OwnerReferences = apexordsownerref // add owner reference, so easy to clean up

	//create configmap
	log.Info("Creating configmap " + apexords.Spec.Ordsname + "-apexords-ords-cm")
	if err := r.Create(ctx, ordsconfigmap); err != nil {
		log.Error(err, "unable to create Ords confimap")

	}

	log.Info("Creating configmap " + apexords.Spec.Ordsname + "-apexords-http-cm")
	if err := r.Create(ctx, httpconfigmap); err != nil {
		log.Error(err, "unable to create http confimap")

	}

	//create Ords schemas in DB
	//create ords pod
	if err := CreateOrdsPod(r, req, apexords); err != nil {
		log.Error(err, "unable to create Ords pod")

	}
	//run Ords installation sql in ords pod
	log.Info("Create Ords in Target DB....")

	ordstext := "mv /opt/oracle/ords/config/ords/defaults.xml /tmp;cp /mnt/k8s/ords_params.properties /tmp/ords_params.properties;java -jar /opt/oracle/ords/ords.war install --parameterFile /tmp/ords_params.properties simple"
	OrdsCommand := []string{"/bin/sh", "-c", ordstext}
	Podname := "ordspod"
	if err := ExecPodCmd(r, req, Podname, OrdsCommand); err != nil {
		log.Error(err, "Error to run "+strings.Join(OrdsCommand, " ")+" in ordspod")
	}
	//clean ords pod
	if err := DeleteOrdsPod(r, req); err != nil {
		log.Error(err, "unable to delete Ords pod")

	}
	//create ords deployments
	log.Info("Creating ords deployment " + apexordsordsdeployname)
	if err := r.Create(ctx, ordsdeployment); err != nil {
		log.Error(err, "unable to create Ords deployment")

	}
	//create nodeport service
	log.Info("Creating ords nodeport service " + apexords.Spec.Ordsname + "-apexords-nodeport-svc")
	if err := r.Create(ctx, ordsnodeportsvc); err != nil {
		log.Error(err, "unable to create Ords ords nodeport service ")

	}

	//create Load balancer service
	log.Info("Creating ords load balancer service " + apexords.Spec.Ordsname + "-apexords-svc")
	if err := r.Create(ctx, ordssvc); err != nil {
		log.Error(err, "unable to create Ords ords load balancer service ")

	}
	log.Info("DB sys Apex Ords schemas password is: " + r.Dbpassword + " users need to update it later.")
	log.Info("Apex Internal Workspace admin password: " + r.Dbpassword + "Apx1#" + " (Use apxchpwd.sql to change it)")
	return nil
}

//CreateDbstsOption to create db statefulset
func CreateDbstsOption(r *ApexOrdsReconciler, req ctrl.Request, apexords *theapexordsv1.ApexOrds) error {
	ctx := context.Background()
	log := r.Log.WithValues("apexords", req.NamespacedName)

	//check if DB stateful exists, if not create one
	var DBstatefulset appsv1.StatefulSetList
	if err := r.List(ctx, &DBstatefulset, client.InNamespace(req.Namespace), client.MatchingLabels(Apexordsoperatorlabel)); err != nil {
		log.Error(err, "unable to list Apexords operator DB statefulset")
		return err
	}
	if len(DBstatefulset.Items) == 0 {
		log.Info("unable to find Apexords operator DB statefulset Pod,going to create new one..")
		//create a new db statefulset here
		if err := CreateDbOption(r, req, apexords); err != nil {
			log.Error(err, "unable to create Apexords operator DB statefulset.")
			return err
		}
	} else {
		for i := 0; i < len(DBstatefulset.Items); i++ {
			log.Info("Found running Apexords operator DB statefulset name is " + DBstatefulset.Items[i].ObjectMeta.Name)
			if DBstatefulset.Items[i].ObjectMeta.Name == apexords.Spec.Dbname+"-apexords-db-sts" {
				log.Info(apexords.Spec.Dbname + "-apexords-db-sts" + "exists. Do nothing")

			} else {
				log.Info("unable to find " + apexords.Spec.Dbname + "-apexords-db-sts." + "going to create new one..")
				//create a new db statefulset here
				if err := CreateDbOption(r, req, apexords); err != nil {
					log.Error(err, "unable to create Apexords operator DB statefulsets.")
					return err
				}
			}
		}
	}

	//check if DB service exists, if not create one
	var DBsvclist corev1.ServiceList
	if err := r.List(ctx, &DBsvclist, client.InNamespace(req.Namespace), client.MatchingLabels(Apexordsoperatorlabel)); err != nil {
		log.Error(err, "unable to list Apexords operator DB service")
		return err
	}
	if len(DBsvclist.Items) == 0 {
		log.Info("unable to find Apexords operator DB service,going to create new one..")
		//create a new db service here
		if err := CreateDbSvcOption(r, req, apexords); err != nil {
			log.Error(err, "unable to create Apexords operator k8s service for DB")
			return err
		}
	} else {
		for i, DBsvc := range DBsvclist.Items {
			fmt.Printf("\n%v\n", i)
			log.Info("Found running Apexords operator DB service name is " + DBsvc.ObjectMeta.Name)
			if DBsvc.ObjectMeta.Name == apexords.Spec.Dbname+"-apexords-db-svc" {
				log.Info(apexords.Spec.Dbname + "-apexords-db-svc" + " exists. Do nothing")

			} else {
				log.Info("unable to find " + apexords.Spec.Dbname + "-apexords-db-svc." + "going to create new one..")
				//create a new db service here
				if err := CreateDbSvcOption(r, req, apexords); err != nil {
					log.Error(err, "unable to create Apexords operator DB service")
					return err
				}
			}
		}
	}
	return nil
}

//CreateApexOption is to create Apex schema in DB
func CreateApexOption(r *ApexOrdsReconciler, req ctrl.Request, apexords *theapexordsv1.ApexOrds) error {
	ctx := context.Background()
	log := r.Log.WithValues("apexords", req.NamespacedName)

	//verify if DB pod is up and running
	dbpodstatus := &corev1.Pod{}
	verifyPodState := func() bool {
		if err := r.Get(ctx, client.ObjectKey{
			Namespace: req.NamespacedName.Namespace,
			Name:      apexords.Spec.Dbname + "-apexords-db-sts-0",
		}, dbpodstatus); err != nil {
			return false
		}
		if dbpodstatus.Status.Phase == corev1.PodRunning {
			return true
		}
		return false
	}
	//30 min timeout for starting pod
	if !verifyPodState() {
		log.Info("waiting for db pod to start.......sleep for 30 min")
		time.Sleep(30 * time.Minute)

	} else {
		log.Info("db pod is started.......")

	}

	if !verifyPodState() {
		log.Error(nil, "30 Min timeout to start db pod")
		return nil
	}
	//create sqlpluspod
	if err := CreateSqlplusPod(r, req); err != nil {
		log.Error(err, "unable to create Sqlpluspod")

	}
	// if apexords.Spec.Apexruntimeonly is true,run apex runtimeonly installation sql in sqlplispod
	if apexords.Spec.Apexruntimeonly {
		log.Info("Create Apex runtime only in Target DB....")
		sqltext := "sqlplus " + "sys/" + r.Dbpassword + "@" + apexords.Spec.Dbname + "-apexords-db-svc" + ":" + apexords.Spec.Dbport + "/" + apexords.Spec.Dbservice + " as sysdba " + "@createapexruntimeonly.sql"
		SQLCommand := []string{"/bin/sh", "-c", sqltext}
		Podname := "sqlpluspod"
		if err := ExecPodCmd(r, req, Podname, SQLCommand); err != nil {
			log.Error(err, "Error to run "+strings.Join(SQLCommand, " ")+" in Sqlpluspod")
		}
	} else {
		log.Info("Create Apex in Target DB....")
		sqltext := "sqlplus " + "sys/" + r.Dbpassword + "@" + apexords.Spec.Dbname + "-apexords-db-svc" + ":" + apexords.Spec.Dbport + "/" + apexords.Spec.Dbservice + " as sysdba " + "@createapex.sql"
		SQLCommand := []string{"/bin/sh", "-c", sqltext}
		Podname := "sqlpluspod"
		if err := ExecPodCmd(r, req, Podname, SQLCommand); err != nil {
			log.Error(err, "Error to run "+strings.Join(SQLCommand, " ")+" in Sqlpluspod")
		}
	}

	log.Info("Update Apex schema password in Target DB....")
	sqltext := "sqlplus " + "sys/" + r.Dbpassword + "@" + apexords.Spec.Dbname + "-apexords-db-svc" + ":" + apexords.Spec.Dbport + "/" + apexords.Spec.Dbservice + " as sysdba " + "@updatepass.sql " + r.Dbpassword
	SQLCommand := []string{"/bin/sh", "-c", sqltext}
	Podname := "sqlpluspod"
	if err := ExecPodCmd(r, req, Podname, SQLCommand); err != nil {
		log.Error(err, "Error to run "+strings.Join(SQLCommand, " ")+" in Sqlpluspod")
	}

	log.Info("Update Apex workspace Admin password in Target DB.....")
	sqltext = "sqlplus " + "sys/" + r.Dbpassword + "@" + apexords.Spec.Dbname + "-apexords-db-svc" + ":" + apexords.Spec.Dbport + "/" + apexords.Spec.Dbservice + " as sysdba " + "@apxchpwd-silent-admin.sql " + r.Dbpassword + "Apx1#"
	SQLCommand = []string{"/bin/sh", "-c", sqltext}
	Podname = "sqlpluspod"
	if err := ExecPodCmd(r, req, Podname, SQLCommand); err != nil {
		log.Error(err, "Error to run "+strings.Join(SQLCommand, " ")+" in Sqlpluspod")
	}

	//delete sqlpluspod
	if err := DeleteSqlplusPod(r, req); err != nil {
		log.Error(err, "unable to delete Sqlpluspod")
	}

	return nil
}

//DeleteSqlplusPod function is to clean sqlpluspod
func DeleteSqlplusPod(r *ApexOrdsReconciler, req ctrl.Request) error {
	ctx := context.Background()
	log := r.Log.WithValues("apexords", req.NamespacedName)

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: req.NamespacedName.Namespace,
			Name:      "sqlpluspod",
		},
	}
	log.Info("Deleting sqlpluspod .......")
	if err := r.Delete(ctx, pod); err != nil {
		log.Error(err, "unable to delete sqlplus pod")
		return err
	}
	return nil
}

//DeleteOrdsPod function is to clean ordspod
func DeleteOrdsPod(r *ApexOrdsReconciler, req ctrl.Request) error {
	ctx := context.Background()
	log := r.Log.WithValues("apexords", req.NamespacedName)

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: req.NamespacedName.Namespace,
			Name:      "ordspod",
		},
	}
	log.Info("Deleting ordspod .......")
	if err := r.Delete(ctx, pod); err != nil {
		log.Error(err, "unable to delete ords pod")
		return err
	}
	return nil
}

//CreateOrdsPod Function to create ords pod to run installation sql for ords schemas
func CreateOrdsPod(r *ApexOrdsReconciler, req ctrl.Request, apexords *theapexordsv1.ApexOrds) error {
	ctx := context.Background()
	log := r.Log.WithValues("apexords", req.NamespacedName)
	var waitsec int64 = 10

	typeMetadata := metav1.TypeMeta{
		Kind:       "Pod",
		APIVersion: "v1",
	}
	objectMetadata := metav1.ObjectMeta{
		Name:      "ordspod",
		Namespace: req.NamespacedName.Namespace,
	}

	configmapvolume := &corev1.ConfigMapVolumeSource{
		LocalObjectReference: corev1.LocalObjectReference{Name: apexords.Spec.Ordsname + "-apexords-ords-cm"},
	}

	podSpecs := corev1.PodSpec{
		//ImagePullSecrets: []corev1.LocalObjectReference{{
		//	Name: "repo-secret",
		//}},
		Volumes: []corev1.Volume{{
			Name: "ords-config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: configmapvolume,
			},
		}},
		Containers: []corev1.Container{{
			Name:  "ordspod",
			Image: "henryxie/apexords-operator-apexords:v19",
			VolumeMounts: []corev1.VolumeMount{{
				Name:      "ords-config",
				MountPath: "/mnt/k8s",
			}},
		}},
		TerminationGracePeriodSeconds: &waitsec,
	}
	pod := corev1.Pod{
		TypeMeta:   typeMetadata,
		ObjectMeta: objectMetadata,
		Spec:       podSpecs,
	}
	log.Info("Creating ords pod .......")
	if err := r.Create(ctx, &pod); err != nil {
		log.Error(err, "unable to create ords pod")
		return err
	}
	podstatus := &corev1.Pod{}
	verifyPodState := func() bool {
		if err := r.Get(ctx, client.ObjectKey{
			Namespace: req.NamespacedName.Namespace,
			Name:      "ordspod",
		}, podstatus); err != nil {
			return false
		}
		if podstatus.Status.Phase == corev1.PodRunning {
			return true
		}
		return false
	}
	//20 min timeout for starting pod
	for i := 0; i < 120; i++ {
		if !verifyPodState() {
			log.Info("waiting for ordspod to start.......")
			time.Sleep(10 * time.Second)

		} else {
			log.Info("ordspod started.......")
			return nil
		}
	}
	log.Error(nil, "Timeout to start ords pod")
	return nil
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
		Namespace: req.NamespacedName.Namespace,
	}
	podSpecs := corev1.PodSpec{
		//ImagePullSecrets: []corev1.LocalObjectReference{{
		//	Name: "repo-secret",
		//}},
		Containers: []corev1.Container{{
			Name:            "sqlpluspod",
			Image:           "henryxie/apexords-operator-instantclient-apex19:v1",
			ImagePullPolicy: "Always",
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

//ExecPodCmd function is to run cmd ie sqlplus
func ExecPodCmd(r *ApexOrdsReconciler, req ctrl.Request, Podname string, SQLCommand []string) error {
	log := r.Log.WithValues("apexords", req.NamespacedName)
	//get restconfig from kubeconfig or cluster
	cfg, err := ctrlconfig.GetConfig()
	if err != nil {
		log.Error(err, "unable to get kubeconfig")
		return err

	}

	kubeclientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return err
	}
	execReq := kubeclientset.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(Podname).
		Namespace(req.NamespacedName.Namespace).
		SubResource("exec")

	execReq.VersionedParams(&corev1.PodExecOptions{
		Command: SQLCommand,
		Stdin:   true,
		Stdout:  true,
		Stderr:  true,
	}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(cfg, "POST", execReq.URL())
	if err != nil {
		log.Error(err, "error while creating Executor:")
		return err

	}

	err = exec.Stream(remotecommand.StreamOptions{
		Stdin:  os.Stdin,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Tty:    false,
	})
	if err != nil {
		log.Error(err, "error in Stream")
		return err
	} else {
		return nil
	}

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
	var apexordsownerref = []metav1.OwnerReference{{
		Kind:       apexords.TypeMeta.Kind,
		APIVersion: apexords.TypeMeta.APIVersion,
		Name:       apexords.ObjectMeta.Name,
		UID:        apexords.ObjectMeta.UID,
	}}
	oradbsts.Spec.Selector.MatchLabels = oradbselector
	//Update ORACLE_SID ,ORACLE_PDB,ORACLE_PWD
	oradbsts.Spec.Template.Spec.Containers[0].Env[0].Value = strings.ToUpper(apexords.Spec.Dbname)
	oradbsts.Spec.Template.Spec.Containers[0].Env[1].Value = strings.ToUpper(apexords.Spec.Dbservice)
	oradbsts.Spec.Template.Spec.Containers[0].Env[2].Value = r.Dbpassword
	oradbsts.Spec.Template.ObjectMeta.Labels = oradbselector
	oradbsts.ObjectMeta.OwnerReferences = apexordsownerref // add owner reference, so easy to clean up
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

//Autopasswd is to reverse dbname+ordsname for temp password for sys, apex, ords schemas, users need to change it later
func Autopasswd(s string) string {
	chars := []rune(s)
	for i, j := 0, len(chars)-1; i < j; i, j = i+1, j-1 {
		chars[i], chars[j] = chars[j], chars[i]
	}
	return string(chars)

}

//SetupWithManager is to set up ord with controller runtime manager
func (r *ApexOrdsReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&theapexordsv1.ApexOrds{}).
		Complete(r)
}
