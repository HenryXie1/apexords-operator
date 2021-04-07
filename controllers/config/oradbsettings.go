package config

var (
	OradbExample = `
	# 
	#create oracle db 19c statefulset with label app=apexords-operator details in the OKE cluster
	#so far we support oracle db 19.2 , new db versions can be added later
	kubectl-oradb create -c cdbname -p pdbname -w syspassword 
	# delete oracle db  statefulset with label app=apexords-operator details in the OKE cluster. 
	# Data won't be deleted.PV and PVC are kept in OKE
	kubectl-oradb delete -c cdbname
	# list oracle db statefulset with label app=apexords-operator details in the OKE cluster
	kubectl-oradb list 
	`
	OradbStsyml = `
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: oradbauto
  labels:
    app: apexords-operator
    name: oradbauto
spec:
  selector:
        matchLabels:
           name: oradbauto-db-service
  replicas: 1
  template:
    metadata:
        labels:
           name: oradbauto-db-service
    spec:
      securityContext:
         runAsUser: 54321
         fsGroup: 54321
      containers:
        - image: henryxie/apexords-operator-database:19.2
          name: oradbauto
          ports:
            - containerPort: 1521
              name: oradbauto
          volumeMounts:
            - mountPath: /opt/oracle/oradata
              name: oradbauto-db-pv-storage
          env:
            - name: ORACLE_SID
              value: "autocdb"
            - name: ORACLE_PDB
              value: "autopdb"
            - name:  ORACLE_PWD
              value: "changeit"
  volumeClaimTemplates:
  - metadata:
      name: oradbauto-db-pv-storage
    spec:
      accessModes: [ "ReadWriteOnce" ]
      resources:
        requests:
          storage: 50Gi
`
	OradbSvcyml = `
apiVersion: v1
kind: Service
metadata:
  labels:
     app: apexords-operator
  name: oradbauto-db-svc
  namespace: default
spec:
  ports:
  - port: 1521
    protocol: TCP
    targetPort: 1521
  selector:
     name: oradbauto-db-service
`
	OradbSvcymlnodeport = `
apiVersion: v1
kind: Service
metadata:
  labels:
     app: apexords-operator
     name: oradbauto-db-service
  name: oradbauto-db-svc-nodeport
  namespace: default
spec:
  ports:
  - port: 1521
    protocol: TCP
    targetPort: 1521
  selector:
     name: oradbauto-db-service
  type: NodePort
`
)
