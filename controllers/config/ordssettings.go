package config

var (
	OrdsExample = `
  # Requirment: 
  # Install Oracle Apex in DB before running this tool. Refer automation tool for Apex
  #
  #
  # Docker images are based on Oracle GitHub Docker Repo 
  # ords container uses henryxie/apexords-operator-apexords:v19
  # httpd container uses henryxie/apexords-operator-oel-httpd:v4
  # list ords deployment with label app=peordshttp
  # list versions ords and Apex status in DB
	kubectl ords list -d dbhost -p 1521 -s testpdbsvc -w syspassword
	# create ords and http Pod with spefified name, run java ords.war install in the pod
	kubectl ords create -o myordsauto -d dbhost -p 1521 -s testpdbsvc -w syspassword -x apexpassword
	# delete ords deployment and drop ords related schemas in DB
	kubectl ords delete -o myordsauto -d dbhost -p 1521 -s testpdbsvc -w syspassword
	`
	OrdsLBsvcyml = `
apiVersion: v1
kind: Service
metadata:
  labels:
    app: peordsauto
  name: ordsauto-lb-service
spec:
  externalTrafficPolicy: Cluster
  ports:
  - name: http
    port: 80
    protocol: TCP
    targetPort: 80
  - name: https
    port: 443
    protocol: TCP
    targetPort: 80
  selector:
    name: ordsauto-service
  sessionAffinity: None
  type: LoadBalancer
`
	OrdsNodePortsvcyml = `
apiVersion: v1
kind: Service
metadata:
  labels:
    app: peordsauto
  name: ordsauto-nodeport-service
spec:
  externalTrafficPolicy: Cluster
  ports:
  - name: http
    port: 80
    protocol: TCP
    targetPort: 80
  selector:
    name: ordsauto-service
  type: NodePort
`

	Ordsyml = `
apiVersion: apps/v1
kind: Deployment
metadata:
  name: ordsauto-deployment
  labels:
    app: peordshttp
    name: ordsauto-service
spec:
  replicas: 1
  selector:
      matchLabels:
         name: ordsauto-service
  template:
       metadata:
          labels:
             name: ordsauto-service
       spec:
         volumes:
            - name: httpd-config
              configMap:
                 name: httpautoconfig
            - name: ords-config
              configMap:
                 name: ordsautoconfig
         containers:
           - name: ords
             image: henryxie/apexords-operator-apexords:v19
             imagePullPolicy: IfNotPresent
             volumeMounts:
                - name: ords-config
                  mountPath: /mnt/k8s
             ports:
                - containerPort: 8888
           - name: httpd
             image: henryxie/apexords-operator-oel-httpd:v4
             imagePullPolicy: IfNotPresent
             volumeMounts:
                - name: httpd-config
                  mountPath: /mnt/k8s
             ports:
                - containerPort: 80
`
	Ordsconfigmapyml = `
apiVersion: v1
data:
  apex.xml: |
    <?xml version="1.0" encoding="UTF-8" standalone="no"?>
    <!DOCTYPE properties SYSTEM "http://java.sun.com/dtd/properties.dtd">
    <properties>
    <comment>Saved on Tue Jan 09 04:08:24 UTC 2018</comment>
    <entry key="db.password">replacepwdapexordsauto</entry>
    <entry key="db.username">APEX_PUBLIC_USER</entry>
    </properties>
  apex_al.xml: |
    <?xml version="1.0" encoding="UTF-8" standalone="no"?>
    <!DOCTYPE properties SYSTEM "http://java.sun.com/dtd/properties.dtd">
    <properties>
    <comment>Saved on Tue Jan 09 04:08:24 UTC 2018</comment>
    <entry key="db.password">replacepwdapexordsauto</entry>
    <entry key="db.username">APEX_LISTENER</entry>
    </properties>
  apex_pu.xml: |
    <?xml version="1.0" encoding="UTF-8" standalone="no"?>
    <!DOCTYPE properties SYSTEM "http://java.sun.com/dtd/properties.dtd">
    <properties>
    <comment>Saved on Tue Jan 09 04:08:24 UTC 2018</comment>
    <entry key="db.password">replacepwdapexordsauto</entry>
    <entry key="db.username">ORDS_PUBLIC_USER</entry>
    </properties>
  apex_rt.xml: |
    <?xml version="1.0" encoding="UTF-8" standalone="no"?>
    <!DOCTYPE properties SYSTEM "http://java.sun.com/dtd/properties.dtd">
    <properties>
    <comment>Saved on Tue Jan 09 04:08:24 UTC 2018</comment>
    <entry key="db.password">replacepwdapexordsauto</entry>
    <entry key="db.username">APEX_REST_PUBLIC_USER</entry>
    </properties>
  defaults.xml: |
    <?xml version="1.0" encoding="UTF-8" standalone="no"?>
    <!DOCTYPE properties SYSTEM "http://java.sun.com/dtd/properties.dtd">
    <properties>
    <comment>Saved on Tue Jan 09 04:12:05 UTC 2018</comment>
    <entry key="cache.caching">false</entry>
    <entry key="cache.directory">/tmp/apex/cache</entry>
    <entry key="cache.duration">days</entry>
    <entry key="cache.expiration">6</entry>
    <entry key="cache.maxEntries">500</entry>
    <entry key="cache.monitorInterval">60</entry>
    <entry key="cache.procedureNameList"/>
    <entry key="cache.type">lru</entry>
    <entry key="db.hostname">ordsautodbhost</entry>
    <entry key="db.port">ordsautodbport</entry>
    <entry key="db.servicename">ordsautodbservice</entry>
    <entry key="debug.debugger">true</entry>
    <entry key="debug.printDebugToScreen">true</entry>
    <entry key="error.keepErrorMessages">true</entry>
    <entry key="error.maxEntries">50</entry>
    <entry key="jdbc.DriverType">thin</entry>
    <entry key="jdbc.InactivityTimeout">600</entry>
    <entry key="jdbc.InitialLimit">20</entry>
    <entry key="jdbc.MaxConnectionReuseCount">1000</entry>
    <entry key="jdbc.MaxLimit">250</entry>
    <entry key="jdbc.MaxStatementsLimit">10</entry>
    <entry key="jdbc.MinLimit">10</entry>
    <entry key="jdbc.statementTimeout">900</entry>
    <entry key="log.logging">false</entry>
    <entry key="log.maxEntries">50</entry>
    <entry key="misc.compress"/>
    <entry key="misc.defaultPage">apex</entry>
    <entry key="security.crypto.enc.password">5TCNNETNxjE8c0hxSKcuQQ..</entry>
    <entry key="security.crypto.mac.password">LpEAIMYFVy1h5K20rahEHQ..</entry>
    <entry key="security.disableDefaultExclusionList">false</entry>
    <entry key="security.maxEntries">2000</entry>
    <entry key="security.requestValidationFunction">wwv_flow_epg_include_modules.authorize</entry>
    <entry key="security.validationFunctionType">plsql</entry>
    <entry key="misc.enableOldFOP">true</entry>
    <entry key="procedure.postProcess">apex_util.close_open_db_links</entry>
    <entry key="procedure.preProcess">apex_util.close_open_db_links</entry>
    <entry key="security.verifySSL">true</entry>
    <entry key="security.httpsHeaderCheck">X-Forwarded-Proto: https</entry>
    <entry key="apex.excel2collection">true</entry>
    <entry key="apex.excel2collection.onecollection">true</entry>
    <entry key="apex.excel2collection.name">EXCEL_COLLECTION</entry>
    <entry key="apex.excel2collection.useSheetName">true</entry>
    <entry key="security.oauth.tokenLifetime">36000</entry>
    </properties>
  standalone.properties: |
    #Tue Sep 25 07:17:23 GMT 2018
    jetty.port=8888
    standalone.context.path=/apex
    standalone.doc.root=/opt/oracle/ords/config/ords/standalone/doc_root
    standalone.scheme.do.not.prompt=true
    standalone.static.context.path=/i
    standalone.static.path=/opt/oracle/ords/images/
  ords_params.properties: |
    db.hostname=ordsautodbhost
    db.password=replacepwdapexordsauto
    db.port=ordsautodbport
    db.servicename=ordsautodbservice
    db.username=APEX_PUBLIC_USER
    migrate.apex.rest=false
    plsql.gateway.add=true
    rest.services.apex.add=true
    rest.services.ords.add=true
    schema.tablespace.default=SYSAUX
    schema.tablespace.temp=TEMP
    standalone.http.port=8888
    standalone.mode=false
    standalone.static.images=/opt/oracle/ords/images/
    standalone.use.https=false
    user.apex.listener.password=replacepwdapexordsauto
    user.apex.restpublic.password=replacepwdapexordsauto
    user.public.password=replacepwdapexordsauto
    user.tablespace.default=SYSAUX
    user.tablespace.temp=TEMP
    sys.user=SYS
    sys.password=replacepwdsysordsauto
kind: ConfigMap
metadata:
  name: ordsautoconfig
  namespace: default
  labels:
    app: peordshttp
`
	Httpconfigmapyml = `
apiVersion: v1
data:
  httpd.conf: |
    ServerRoot "/etc/httpd"
    Include conf.modules.d/*.conf
    User apache
    Group apache
    ServerAdmin root@localhost
    <Directory />
    AllowOverride none
    Require all denied
    </Directory>

    DocumentRoot "/var/www/html"

    <Directory "/var/www">
    AllowOverride None
    Require all granted
    </Directory>

    <Directory "/var/www/html">
    Options Indexes FollowSymLinks

    AllowOverride None

    Require all granted
    </Directory>

    <IfModule dir_module>
    DirectoryIndex index.html
    </IfModule>

    <Files ".ht*">
    Require all denied
    </Files>

    ErrorLog "logs/error_log"
    LogLevel warn
    <IfModule log_config_module>
    LogFormat "%h %l %u %t \"%r\" %>s %b \"%{Referer}i\" \"%{User-Agent}i\"" combined
    LogFormat "%h %l %u %t \"%r\" %>s %b" common

    <IfModule logio_module>
    LogFormat "%h %l %u %t \"%r\" %>s %b \"%{Referer}i\" \"%{User-Agent}i\" %I %O" combinedio
    </IfModule>

    CustomLog "logs/access_log" combined
    </IfModule>

    <IfModule alias_module>
    ScriptAlias /cgi-bin/ "/var/www/cgi-bin/"
    </IfModule>

    <Directory "/var/www/cgi-bin">
    AllowOverride None
    Options None
    Require all granted
    </Directory>

    <IfModule mime_module>
    TypesConfig /etc/mime.types

    AddType application/x-compress .Z
    AddType application/x-gzip .gz .tgz

    AddType text/html .shtml
    AddOutputFilter INCLUDES .shtml
    </IfModule>

    AddDefaultCharset UTF-8

    <IfModule mime_magic_module>
    MIMEMagicFile conf/magic
    </IfModule>

    EnableSendfile on

    IncludeOptional conf.d/*.conf
    Include /etc/httpd/conf/users-define.conf
  users-define.conf: |
    Listen 80
    <VirtualHost *:80>
    
    DocumentRoot "/var/www/html/"
    Alias /i/ "/var/www/html/images/"
    
    AddType text/xml xbl
    AddType text/x-component htc

    <Directory /var/www/html/>
    AllowOverride none
    Order deny,allow
    Allow from all
    </Directory>

    <Directory /var/www/html/images/>
    Header set X-Frame-Options "deny"
    </Directory>

    RedirectMatch ^/$  /apex
    RewriteEngine On
    ProxyPass "/apex" "http://localhost:8888/apex" retry=60
    ProxyPassReverse /apex http://localhost:8888/apex
    ProxyPreserveHost On
    </VirtualHost>
kind: ConfigMap
metadata:
  name: httpautoconfig
  labels:
    app: peordshttp
`
)
