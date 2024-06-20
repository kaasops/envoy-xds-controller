# This is a wrapper to manage helm chart
URL=https://kaasops.github.io/envoy-xds-controller/helm

.PHONY: helm-lint
helm-lint:
	helm lint helm/charts/envoy-xds-controller

.PHONY: helm-package
helm-package:
	helm package helm/charts/* -d helm/packages

.PHONY: helm-index
helm-index:
	helm repo index --url ${URL} ./helm