package config

var (
	ApexExample = `
	# 
	# list apex version details in the target
	kubectl-apex list -d dbhost -p 1521 -s testpdbsvc -w syspassword 
	# create apex full development apex framework in target db. 
	kubectl-apex create -d dbhost -p 1521 -s testpdbsvc -w syspassword 
	# create apex runtime only framework in target db. 
	kubectl-apex create -d dbhost -p 1521 -s testpdbsvc -w syspassword -r
	# delete apex framework in target db. 
	kubectl-apex delete -d dbhost -p 1521 -s testpdbsvc -w syspassword 
	`
)
