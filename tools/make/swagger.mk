# This is a wrapper to generate swagger documentation

.PHONY: swagger-kube
swagger-kube:
	swag init -o ./docs/kubeRestAPI --instanceName kube --exclude ./pkg/xds/api --parseDependency