URL=https://kaasops.github.io/envoy-xds-controller/helm
HELM_IMAGE = alpine/helm:3.12.1
HELM_DOCS_IMAGE = jnorwood/helm-docs:v1.11.0
KNOWN_TARGETS=helm

# Run linter for helm chart
lint:
	helm lint charts/envoy-xds-controller

# Package chart into zip file
package:
	cp ../config/crd/bases/*.yaml charts/envoy-xds-controller/crds/
	helm package charts/* -d packages

# Create index file (use only for initial setup)
index:
	helm repo index --url ${URL} .