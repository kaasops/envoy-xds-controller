# This is a wrapper to manage helm chart

.PHONY: helm-lint
helm-lint:
	helm lint helm/charts/envoy-xds-controller

.PHONY: helm-package
helm-package:
	cp config/crd/bases/*.yaml helm/charts/envoy-xds-controller/crds/
	helm package helm/charts/* -d helm/packages

.PHONY: helm-index
helm-index:
	helm repo index --url ${URL} ./helm