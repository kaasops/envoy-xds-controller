package errors

var (
	WatchNamespacesNotSet = "watch namespace not set"

	EmptySpecMessage               = "spec could not be empty"
	InvalidSpecMessage             = "invalid config component spec"
	MultipleAccessLogConfigMessage = "only one access log config is allowed"

	UnmarshalMessage        = "cannot unmarshal"
	NodeIDMismatchMessage   = "nodeID mismatch"
	GetDefaultNodeIDMessage = "cannot get default NodeID"

	GetNodeIDForResource               = "cannot get NodeID for xDS cache resource"
	CannotDeleteFromCacheMessage       = "cannot delete from xDS cache"
	CannotUpdateCacheMessage           = "cannot update xDS cache"
	CannotValidateCacheResourceMessage = "cannot validate cache resource"

	GetFromKubernetesMessage  = "cannot get resource from Kubernetes"
	CreateInKubernetesMessage = "cannot create resource in Kubernetes"
	UpdateInKubernetesMessage = "cannot update resource in Kubernetes"
	DeleteInKubernetesMessage = "cannot delete resource in Kubernetes"

	// Validate error
	ValidateStructMessage               = "cannot validate Specification"
	VirtualHostCantBeEmptyMessage       = "virtualHost could not be empty"
	ListenerCannotBeEmptyMessage        = "listener could not be empty"
	AccessLogConfigCannotBeEmptyMessage = "accessLogConfig could not be empty"
	AccessLogConfigDeleteUsedMessage    = "cannot delete accesslogconfig, bc is used in Virtual Services: "
	HTTPFilterCannotBeEmptyMessage      = "httpFilter could not be empty"
	HTTPFilterDeleteUsed                = "cannot delete httpFilter, bc is used in Virtual Services: "
	HTTPFilterUsedInVST                 = "httpFilter is used in Virtual Services templates"
	InvalidHTTPFilter                   = "invalid http_filter"
	InvalidParamsCombination            = "invalid combination of parameters"
	RouteCannotBeEmptyMessage           = "route could not be empty"
	RouteDeleteUsed                     = "cannot delete route, bc is used in Virtual Services: "
	RouteUsedInVST                      = "route is used in Virtual Services templates"
	ClusterCannotBeEmptyMessage         = "cluster could not be empty"
	PolicyCannotBeEmptyMessage          = "policy could not be empty"

	// TLS Errors
	ManyParamMessage = `not supported using more then 1 param for configure TLS.
	You can choose one of 'secretRef', 'certManager', 'autoDiscovery'`
	ZeroParamMessage = `need choose one 1 param for configure TLS. \
	You can choose one of 'secretRef', 'certManager', 'autoDiscovery'.\
	If you don't want use TLS for connection - don't install tlsConfig`
	NodeIDsEmpty                  = "Object don't have any NodeID"
	SecretNotTLSTypeMessage       = "kuberentes Secret is not a type TLS"
	ControlLabelNotExistMessage   = "kuberentes Secret doesn't have control label"
	ControlLabelWrongMessage      = "kubernetes Secret have label, but value not true"
	CertManaferCRDNotExistMessage = "cert Manager CRDs not exist. Perhaps Cert Manager is not installed in the Kubernetes cluster"
	TlsConfigManyParamMessage     = "сannot be installed Issuer and ClusterIssuer in 1 config"

	DiscoverNotFoundMessage  = "the secret with the certificate was not found for the domain"
	CreateCertificateMessage = "cannot create certificate for domain"
	RegexDomainMessage       = "regex domains not supported"
)
