# gRPC API Reference

## Table of Contents

### Services

- [AccessGroupStoreService](#access_groupv1accessgroupstoreservice)
- [AccessLogConfigStoreService](#access_log_configv1accesslogconfigstoreservice)
- [ClusterStoreService](#clusterv1clusterstoreservice)
- [HTTPFilterStoreService](#http_filterv1httpfilterstoreservice)
- [ListenerStoreService](#listenerv1listenerstoreservice)
- [NodeStoreService](#nodev1nodestoreservice)
- [PermissionsService](#permissionsv1permissionsservice)
- [PolicyStoreService](#policyv1policystoreservice)
- [RouteStoreService](#routev1routestoreservice)
- [UtilsService](#utilv1utilsservice)
- [VirtualServiceTemplateStoreService](#virtual_service_templatev1virtualservicetemplatestoreservice)
- [VirtualServiceStoreService](#virtual_servicev1virtualservicestoreservice)

### Messages

- [AccessGroupListItem](#accessgrouplistitem)
- [ListAccessGroupsRequest](#listaccessgroupsrequest)
- [ListAccessGroupsResponse](#listaccessgroupsresponse)
- [AccessLogConfigListItem](#accesslogconfiglistitem)
- [ListAccessLogConfigsRequest](#listaccesslogconfigsrequest)
- [ListAccessLogConfigsResponse](#listaccesslogconfigsresponse)
- [ClusterListItem](#clusterlistitem)
- [ListClustersRequest](#listclustersrequest)
- [ListClustersResponse](#listclustersresponse)
- [ResourceRef](#resourceref)
- [VirtualHost](#virtualhost)
- [HTTPFilterListItem](#httpfilterlistitem)
- [ListHTTPFiltersRequest](#listhttpfiltersrequest)
- [ListHTTPFiltersResponse](#listhttpfiltersresponse)
- [ListListenersRequest](#listlistenersrequest)
- [ListListenersResponse](#listlistenersresponse)
- [ListenerListItem](#listenerlistitem)
- [ListNodesRequest](#listnodesrequest)
- [ListNodesResponse](#listnodesresponse)
- [NodeListItem](#nodelistitem)
- [AccessGroupPermissions](#accessgrouppermissions)
- [ListPermissionsRequest](#listpermissionsrequest)
- [ListPermissionsResponse](#listpermissionsresponse)
- [PermissionsItem](#permissionsitem)
- [ListPoliciesRequest](#listpoliciesrequest)
- [ListPoliciesResponse](#listpoliciesresponse)
- [PolicyListItem](#policylistitem)
- [ListRoutesRequest](#listroutesrequest)
- [ListRoutesResponse](#listroutesresponse)
- [RouteListItem](#routelistitem)
- [DomainVerificationResult](#domainverificationresult)
- [VerifyDomainsRequest](#verifydomainsrequest)
- [VerifyDomainsResponse](#verifydomainsresponse)
- [FillTemplateRequest](#filltemplaterequest)
- [FillTemplateResponse](#filltemplateresponse)
- [ListVirtualServiceTemplatesRequest](#listvirtualservicetemplatesrequest)
- [ListVirtualServiceTemplatesResponse](#listvirtualservicetemplatesresponse)
- [TemplateOption](#templateoption)
- [VirtualServiceTemplateListItem](#virtualservicetemplatelistitem)
- [CreateVirtualServiceRequest](#createvirtualservicerequest)
- [CreateVirtualServiceResponse](#createvirtualserviceresponse)
- [DeleteVirtualServiceRequest](#deletevirtualservicerequest)
- [DeleteVirtualServiceResponse](#deletevirtualserviceresponse)
- [GetVirtualServiceRequest](#getvirtualservicerequest)
- [GetVirtualServiceResponse](#getvirtualserviceresponse)
- [ListVirtualServicesRequest](#listvirtualservicesrequest)
- [ListVirtualServicesResponse](#listvirtualservicesresponse)
- [UpdateVirtualServiceRequest](#updatevirtualservicerequest)
- [UpdateVirtualServiceResponse](#updatevirtualserviceresponse)
- [VirtualServiceListItem](#virtualservicelistitem)

### Enums

- [ListenerType](#listenertype)
- [TemplateOptionModifier](#templateoptionmodifier)

## Services

### AccessGroupStoreService {#access_groupv1accessgroupstoreservice}
Service to manage access groups.

#### ListAccessGroups
**rpc** ListAccessGroups([ListAccessGroupsRequest](#listaccessgroupsrequest)) returns [ListAccessGroupsResponse](#listaccessgroupsresponse)

Lists access groups.

### AccessLogConfigStoreService {#access_log_configv1accesslogconfigstoreservice}
Service for storing and listing access log configurations.

#### ListAccessLogConfigs
**rpc** ListAccessLogConfigs([ListAccessLogConfigsRequest](#listaccesslogconfigsrequest)) returns [ListAccessLogConfigsResponse](#listaccesslogconfigsresponse)

Lists all access log configurations based on the given request.

### ClusterStoreService {#clusterv1clusterstoreservice}
Service for managing clusters in the store.

#### ListCluster
**rpc** ListCluster([ListClustersRequest](#listclustersrequest)) returns [ListClustersResponse](#listclustersresponse)

Lists all the clusters in the store.

### HTTPFilterStoreService {#http_filterv1httpfilterstoreservice}
Service to manage HTTP filters.

#### ListHTTPFilters
**rpc** ListHTTPFilters([ListHTTPFiltersRequest](#listhttpfiltersrequest)) returns [ListHTTPFiltersResponse](#listhttpfiltersresponse)

Lists all HTTP filters for a given access group.

### ListenerStoreService {#listenerv1listenerstoreservice}
Service for managing listeners.

#### ListListeners
**rpc** ListListeners([ListListenersRequest](#listlistenersrequest)) returns [ListListenersResponse](#listlistenersresponse)

Retrieves a list of listeners based on the request.

### NodeStoreService {#nodev1nodestoreservice}
NodeStoreService provides operations for managing nodes.

#### ListNodes
**rpc** ListNodes([ListNodesRequest](#listnodesrequest)) returns [ListNodesResponse](#listnodesresponse)

ListNodes retrieves a list of nodes belonging to the specified access group.

### PermissionsService {#permissionsv1permissionsservice}


#### ListPermissions
**rpc** ListPermissions([ListPermissionsRequest](#listpermissionsrequest)) returns [ListPermissionsResponse](#listpermissionsresponse)

Lists the permissions associated with a specific access group.

### PolicyStoreService {#policyv1policystoreservice}
PolicyStoreService provides operations related to policy management.

#### ListPolicies
**rpc** ListPolicies([ListPoliciesRequest](#listpoliciesrequest)) returns [ListPoliciesResponse](#listpoliciesresponse)

ListPolicies retrieves a list of policies.

### RouteStoreService {#routev1routestoreservice}
Service to manage routes.

#### ListRoutes
**rpc** ListRoutes([ListRoutesRequest](#listroutesrequest)) returns [ListRoutesResponse](#listroutesresponse)

Lists all the routes for the specified access group.

### UtilsService {#utilv1utilsservice}


#### VerifyDomains
**rpc** VerifyDomains([VerifyDomainsRequest](#verifydomainsrequest)) returns [VerifyDomainsResponse](#verifydomainsresponse)

Verifies the SSL certificates of the provided domains.

### VirtualServiceTemplateStoreService {#virtual_service_templatev1virtualservicetemplatestoreservice}
Service to manage virtual service templates.

#### ListVirtualServiceTemplates
**rpc** ListVirtualServiceTemplates([ListVirtualServiceTemplatesRequest](#listvirtualservicetemplatesrequest)) returns [ListVirtualServiceTemplatesResponse](#listvirtualservicetemplatesresponse)

Lists all virtual service templates.
#### FillTemplate
**rpc** FillTemplate([FillTemplateRequest](#filltemplaterequest)) returns [FillTemplateResponse](#filltemplateresponse)

Fills a template with specific configurations and returns the result.

### VirtualServiceStoreService {#virtual_servicev1virtualservicestoreservice}
The VirtualServiceStoreService defines operations for managing virtual services.

#### CreateVirtualService
**rpc** CreateVirtualService([CreateVirtualServiceRequest](#createvirtualservicerequest)) returns [CreateVirtualServiceResponse](#createvirtualserviceresponse)

CreateVirtualService creates a new virtual service.
#### UpdateVirtualService
**rpc** UpdateVirtualService([UpdateVirtualServiceRequest](#updatevirtualservicerequest)) returns [UpdateVirtualServiceResponse](#updatevirtualserviceresponse)

UpdateVirtualService updates an existing virtual service.
#### DeleteVirtualService
**rpc** DeleteVirtualService([DeleteVirtualServiceRequest](#deletevirtualservicerequest)) returns [DeleteVirtualServiceResponse](#deletevirtualserviceresponse)

DeleteVirtualService deletes a virtual service by its UID.
#### GetVirtualService
**rpc** GetVirtualService([GetVirtualServiceRequest](#getvirtualservicerequest)) returns [GetVirtualServiceResponse](#getvirtualserviceresponse)

GetVirtualService retrieves a virtual service by its UID.
#### ListVirtualServices
**rpc** ListVirtualServices([ListVirtualServicesRequest](#listvirtualservicesrequest)) returns [ListVirtualServicesResponse](#listvirtualservicesresponse)

ListVirtualServices retrieves a list of virtual services for the specified access group.



## Messages


### AccessGroupListItem {#accessgrouplistitem}
Represents an access group item.


| Field | Type | Description |
| ----- | ---- | ----------- |
| name | [ string](#string) | The name of the access group. |



### ListAccessGroupsRequest {#listaccessgroupsrequest}
Request message for listing access groups.



### ListAccessGroupsResponse {#listaccessgroupsresponse}
Response message containing a list of access groups.


| Field | Type | Description |
| ----- | ---- | ----------- |
| items | [repeated AccessGroupListItem](#accessgrouplistitem) | The list of access group items. |



### AccessLogConfigListItem {#accesslogconfiglistitem}
Represents an access log configuration item.


| Field | Type | Description |
| ----- | ---- | ----------- |
| uid | [ string](#string) | The unique identifier of the access log configuration. |
| name | [ string](#string) | The name of the access log configuration. |
| description | [ string](#string) | Description is the human-readable description of the resource |



### ListAccessLogConfigsRequest {#listaccesslogconfigsrequest}
Request message for listing access log configurations.


| Field | Type | Description |
| ----- | ---- | ----------- |
| access_group | [ string](#string) | The access group to filter the log configurations. |



### ListAccessLogConfigsResponse {#listaccesslogconfigsresponse}
Response message containing a list of access log configuration items.


| Field | Type | Description |
| ----- | ---- | ----------- |
| items | [repeated AccessLogConfigListItem](#accesslogconfiglistitem) | The list of access log configuration items. |



### ClusterListItem {#clusterlistitem}
Represents a list item in the cluster.


| Field | Type | Description |
| ----- | ---- | ----------- |
| uid | [ string](#string) | The unique identifier of the cluster. |
| name | [ string](#string) | The name of the cluster. |



### ListClustersRequest {#listclustersrequest}
Request message for listing clusters.



### ListClustersResponse {#listclustersresponse}
Response message containing a list of clusters.


| Field | Type | Description |
| ----- | ---- | ----------- |
| items | [repeated ClusterListItem](#clusterlistitem) | The list of cluster items. |



### ResourceRef {#resourceref}
ResourceRef represents a reference to a resource with a UID and name.


| Field | Type | Description |
| ----- | ---- | ----------- |
| uid | [ string](#string) | UID is the unique identifier of the resource. |
| name | [ string](#string) | Name is the human-readable name of the resource. |



### VirtualHost {#virtualhost}
VirtualHost represents a virtual host with a list of domain names.


| Field | Type | Description |
| ----- | ---- | ----------- |
| domains | [repeated string](#string) | The list of domain names associated with the virtual host. |



### HTTPFilterListItem {#httpfilterlistitem}
Represents an individual HTTP filter.


| Field | Type | Description |
| ----- | ---- | ----------- |
| uid | [ string](#string) | Unique identifier of the HTTP filter. |
| name | [ string](#string) | Name of the HTTP filter. |
| description | [ string](#string) | Description is the human-readable description of the resource |



### ListHTTPFiltersRequest {#listhttpfiltersrequest}
Request message for listing HTTP filters.


| Field | Type | Description |
| ----- | ---- | ----------- |
| access_group | [ string](#string) | Name of the access group to filter HTTP filters by. |



### ListHTTPFiltersResponse {#listhttpfiltersresponse}
Response message containing a list of HTTP filters.


| Field | Type | Description |
| ----- | ---- | ----------- |
| items | [repeated HTTPFilterListItem](#httpfilterlistitem) | List of HTTP filter items. |



### ListListenersRequest {#listlistenersrequest}
Request message to list listeners.


| Field | Type | Description |
| ----- | ---- | ----------- |
| access_group | [ string](#string) | The access group to filter the listeners. |



### ListListenersResponse {#listlistenersresponse}
Response message containing a list of listeners.


| Field | Type | Description |
| ----- | ---- | ----------- |
| items | [repeated ListenerListItem](#listenerlistitem) | A list of listener items. |



### ListenerListItem {#listenerlistitem}
Details of a listener.


| Field | Type | Description |
| ----- | ---- | ----------- |
| uid | [ string](#string) | Unique identifier for the listener. |
| name | [ string](#string) | Display name of the listener. |
| type | [ ListenerType](#listenertype) | The type of listener. |
| description | [ string](#string) | Description is the human-readable description of the resource |



### ListNodesRequest {#listnodesrequest}
ListNodesRequest represents the request to list nodes.


| Field | Type | Description |
| ----- | ---- | ----------- |
| access_group | [ string](#string) | The access group to filter the nodes by. |



### ListNodesResponse {#listnodesresponse}
ListNodesResponse represents the response containing the list of nodes.


| Field | Type | Description |
| ----- | ---- | ----------- |
| items | [repeated NodeListItem](#nodelistitem) | The list of nodes items. |



### NodeListItem {#nodelistitem}
NodeListItem represents a node with its unique identifier.


| Field | Type | Description |
| ----- | ---- | ----------- |
| id | [ string](#string) | The unique identifier of the node. |



### AccessGroupPermissions {#accessgrouppermissions}



| Field | Type | Description |
| ----- | ---- | ----------- |
| access_group | [ string](#string) | Access group name |
| permissions | [repeated PermissionsItem](#permissionsitem) | Permission items associated with access group. |



### ListPermissionsRequest {#listpermissionsrequest}
Request message for listing permissions.



### ListPermissionsResponse {#listpermissionsresponse}
Response message containing a list of permission items.


| Field | Type | Description |
| ----- | ---- | ----------- |
| items | [repeated AccessGroupPermissions](#accessgrouppermissions) | The list of permission items. |



### PermissionsItem {#permissionsitem}
Represents a permission item with an action and associated objects.


| Field | Type | Description |
| ----- | ---- | ----------- |
| action | [ string](#string) | The action of the permission. |
| objects | [repeated string](#string) | The objects associated with the permission. |



### ListPoliciesRequest {#listpoliciesrequest}
ListPoliciesRequest is the request message for ListPolicies RPC.



### ListPoliciesResponse {#listpoliciesresponse}
ListPoliciesResponse is the response message for ListPolicies RPC, containing a list of policy items.


| Field | Type | Description |
| ----- | ---- | ----------- |
| items | [repeated PolicyListItem](#policylistitem) | items is a list of PolicyListItem objects. |



### PolicyListItem {#policylistitem}
PolicyListItem represents an individual policy item with a unique identifier and name.


| Field | Type | Description |
| ----- | ---- | ----------- |
| uid | [ string](#string) | uid is the unique identifier for the policy. |
| name | [ string](#string) | name is the name of the policy. |
| description | [ string](#string) | Description is the human-readable description of the resource |



### ListRoutesRequest {#listroutesrequest}
Request message for listing routes.


| Field | Type | Description |
| ----- | ---- | ----------- |
| access_group | [ string](#string) | Access group to filter the routes. |



### ListRoutesResponse {#listroutesresponse}
Response message containing the list of routes.


| Field | Type | Description |
| ----- | ---- | ----------- |
| items | [repeated RouteListItem](#routelistitem) | List of route items. |



### RouteListItem {#routelistitem}
Represents a route in the route list.


| Field | Type | Description |
| ----- | ---- | ----------- |
| uid | [ string](#string) | Unique identifier for the route. |
| name | [ string](#string) | Name of the route. |
| description | [ string](#string) | Description is the human-readable description of the resource |



### DomainVerificationResult {#domainverificationresult}



| Field | Type | Description |
| ----- | ---- | ----------- |
| domain | [ string](#string) | The domain being verified. |
| valid_certificate | [ bool](#bool) | Indicates if the domain has a valid SSL certificate. |
| issuer | [ string](#string) | The issuer of the SSL certificate. |
| expires_at | [ google.protobuf.Timestamp](#googleprotobuftimestamp) | The expiration timestamp of the SSL certificate. |
| matched_by_wildcard | [ bool](#bool) | Indicates if the domain was matched using a wildcard certificate. |
| error | [ string](#string) | Any error messages related to the domain's verification. |



### VerifyDomainsRequest {#verifydomainsrequest}



| Field | Type | Description |
| ----- | ---- | ----------- |
| domains | [repeated string](#string) | A list of domains to verify SSL certificates for. |



### VerifyDomainsResponse {#verifydomainsresponse}



| Field | Type | Description |
| ----- | ---- | ----------- |
| results | [repeated DomainVerificationResult](#domainverificationresult) | A list of the results for each domain verification. |



### FillTemplateRequest {#filltemplaterequest}
Request message for filling a template with specific configurations.


| Field | Type | Description |
| ----- | ---- | ----------- |
| template_uid | [ string](#string) | Unique identifier of the template to fill. |
| listener_uid | [ string](#string) | Unique identifier of the listener to associate with the template. |
| virtual_host | [ common.v1.VirtualHost](#commonv1virtualhost) | The virtual host configuration for the virtual service. |
| [**oneof**](https://developers.google.com/protocol-buffers/docs/proto3#oneof) access_log_config.access_log_config_uid | [ string](#string) | Unique identifier of the access log configuration. |
| additional_http_filter_uids | [repeated string](#string) | Additional HTTP filter unique identifiers. |
| additional_route_uids | [repeated string](#string) | Additional route unique identifiers. |
| [**oneof**](https://developers.google.com/protocol-buffers/docs/proto3#oneof) _use_remote_address.use_remote_address | [optional bool](#bool) | Whether to use the remote address. |
| template_options | [repeated TemplateOption](#templateoption) | Options to modify the template. |
| name | [ string](#string) | Virtual service name |
| description | [ string](#string) | Description is the human-readable description of the resource |



### FillTemplateResponse {#filltemplateresponse}
Response message containing the filled template as a raw string.


| Field | Type | Description |
| ----- | ---- | ----------- |
| raw | [ string](#string) | The raw string representation of the filled template. |



### ListVirtualServiceTemplatesRequest {#listvirtualservicetemplatesrequest}
Request message for listing all virtual service templates.


| Field | Type | Description |
| ----- | ---- | ----------- |
| access_group | [ string](#string) | The access group for filtering templates. |



### ListVirtualServiceTemplatesResponse {#listvirtualservicetemplatesresponse}
Response message containing the list of virtual service templates.


| Field | Type | Description |
| ----- | ---- | ----------- |
| items | [repeated VirtualServiceTemplateListItem](#virtualservicetemplatelistitem) | The list of virtual service templates. |



### TemplateOption {#templateoption}
Represents a single option to be applied to a template.


| Field | Type | Description |
| ----- | ---- | ----------- |
| field | [ string](#string) | The field name of the option. |
| modifier | [ TemplateOptionModifier](#templateoptionmodifier) | The modifier applied to the field. |



### VirtualServiceTemplateListItem {#virtualservicetemplatelistitem}
Details of a virtual service template.


| Field | Type | Description |
| ----- | ---- | ----------- |
| uid | [ string](#string) | Unique identifier of the template. |
| name | [ string](#string) | Name of the template. |
| description | [ string](#string) | Description is the human-readable description of the resource |



### CreateVirtualServiceRequest {#createvirtualservicerequest}
CreateVirtualServiceRequest is the request message for creating a virtual service.


| Field | Type | Description |
| ----- | ---- | ----------- |
| name | [ string](#string) | The name of the virtual service. |
| node_ids | [repeated string](#string) | The node IDs associated with the virtual service. |
| access_group | [ string](#string) | The access group of the virtual service. |
| template_uid | [ string](#string) | The UID of the template used by the virtual service. |
| listener_uid | [ string](#string) | The UID of the listener associated with the virtual service. |
| virtual_host | [ common.v1.VirtualHost](#commonv1virtualhost) | The virtual host configuration for the virtual service. |
| [**oneof**](https://developers.google.com/protocol-buffers/docs/proto3#oneof) access_log_config.access_log_config_uid | [ string](#string) | The UID of the access log configuration. |
| additional_http_filter_uids | [repeated string](#string) | UIDs of additional HTTP filters appended to the virtual service. |
| additional_route_uids | [repeated string](#string) | UIDs of additional routes appended to the virtual service. |
| [**oneof**](https://developers.google.com/protocol-buffers/docs/proto3#oneof) _use_remote_address.use_remote_address | [optional bool](#bool) | Whether to use the remote address for the virtual service. |
| template_options | [repeated virtual_service_template.v1.TemplateOption](#virtual_service_templatev1templateoption) | Template options for the virtual service. |
| description | [ string](#string) | Description is the human-readable description of the resource |



### CreateVirtualServiceResponse {#createvirtualserviceresponse}
CreateVirtualServiceResponse is the response message for creating a virtual service.



### DeleteVirtualServiceRequest {#deletevirtualservicerequest}
DeleteVirtualServiceRequest is the request message for deleting a virtual service.


| Field | Type | Description |
| ----- | ---- | ----------- |
| uid | [ string](#string) | The UID of the virtual service to delete. |



### DeleteVirtualServiceResponse {#deletevirtualserviceresponse}
DeleteVirtualServiceResponse is the response message for deleting a virtual service.



### GetVirtualServiceRequest {#getvirtualservicerequest}
GetVirtualServiceRequest is the request message for retrieving a virtual service.


| Field | Type | Description |
| ----- | ---- | ----------- |
| uid | [ string](#string) | The UID of the virtual service to retrieve. |



### GetVirtualServiceResponse {#getvirtualserviceresponse}
GetVirtualServiceResponse is the response message for retrieving a virtual service.


| Field | Type | Description |
| ----- | ---- | ----------- |
| uid | [ string](#string) | The UID of the virtual service. |
| name | [ string](#string) | The name of the virtual service. |
| node_ids | [repeated string](#string) | The node IDs associated with the virtual service. |
| access_group | [ string](#string) | The access group of the virtual service. |
| template | [ common.v1.ResourceRef](#commonv1resourceref) | A reference to the template used by the virtual service. |
| listener | [ common.v1.ResourceRef](#commonv1resourceref) | A reference to the listener associated with the virtual service. |
| virtual_host | [ common.v1.VirtualHost](#commonv1virtualhost) | The virtual host configuration for the virtual service. |
| [**oneof**](https://developers.google.com/protocol-buffers/docs/proto3#oneof) access_log.access_log_config | [ common.v1.ResourceRef](#commonv1resourceref) | A reference to the access log configuration. |
| [**oneof**](https://developers.google.com/protocol-buffers/docs/proto3#oneof) access_log.access_log_config_raw | [ bytes](#bytes) | Raw configuration for access logs. |
| additional_http_filters | [repeated common.v1.ResourceRef](#commonv1resourceref) | Additional HTTP filters associated with the virtual service. |
| additional_routes | [repeated common.v1.ResourceRef](#commonv1resourceref) | Additional routes associated with the virtual service. |
| [**oneof**](https://developers.google.com/protocol-buffers/docs/proto3#oneof) _use_remote_address.use_remote_address | [optional bool](#bool) | Whether the virtual service uses the remote address. |
| template_options | [repeated virtual_service_template.v1.TemplateOption](#virtual_service_templatev1templateoption) | Template options for the virtual service. |
| is_editable | [ bool](#bool) | Indicates whether the virtual service is editable. |
| description | [ string](#string) | Description is the human-readable description of the resource |



### ListVirtualServicesRequest {#listvirtualservicesrequest}
ListVirtualServicesRequest is the request message for listing virtual services.


| Field | Type | Description |
| ----- | ---- | ----------- |
| access_group | [ string](#string) | The access group for which to list virtual services. |



### ListVirtualServicesResponse {#listvirtualservicesresponse}
ListVirtualServicesResponse is the response message for listing virtual services.


| Field | Type | Description |
| ----- | ---- | ----------- |
| items | [repeated VirtualServiceListItem](#virtualservicelistitem) | The list of virtual services. |



### UpdateVirtualServiceRequest {#updatevirtualservicerequest}
UpdateVirtualServiceRequest is the request message for updating a virtual service.


| Field | Type | Description |
| ----- | ---- | ----------- |
| uid | [ string](#string) | The UID of the virtual service. |
| node_ids | [repeated string](#string) | The node IDs associated with the virtual service. |
| template_uid | [ string](#string) | The UID of the template used by the virtual service. |
| listener_uid | [ string](#string) | The UID of the listener associated with the virtual service. |
| virtual_host | [ common.v1.VirtualHost](#commonv1virtualhost) | The virtual host configuration for the virtual service. |
| [**oneof**](https://developers.google.com/protocol-buffers/docs/proto3#oneof) access_log_config.access_log_config_uid | [ string](#string) | The UID of the access log configuration. |
| additional_http_filter_uids | [repeated string](#string) | UIDs of additional HTTP filters appended to the virtual service. |
| additional_route_uids | [repeated string](#string) | UIDs of additional routes appended to the virtual service. |
| [**oneof**](https://developers.google.com/protocol-buffers/docs/proto3#oneof) _use_remote_address.use_remote_address | [optional bool](#bool) | Whether to use the remote address for the virtual service. |
| template_options | [repeated virtual_service_template.v1.TemplateOption](#virtual_service_templatev1templateoption) | Template options for the virtual service. |
| description | [ string](#string) | Description is the human-readable description of the resource |



### UpdateVirtualServiceResponse {#updatevirtualserviceresponse}
UpdateVirtualServiceResponse is the response message for updating a virtual service.



### VirtualServiceListItem {#virtualservicelistitem}
VirtualServiceListItem represents a single virtual service in a list response.


| Field | Type | Description |
| ----- | ---- | ----------- |
| uid | [ string](#string) | The UID of the virtual service. |
| name | [ string](#string) | The name of the virtual service. |
| node_ids | [repeated string](#string) | The node IDs associated with the virtual service. |
| access_group | [ string](#string) | The access group of the virtual service. |
| template | [ common.v1.ResourceRef](#commonv1resourceref) | A reference to the template used by the virtual service. |
| is_editable | [ bool](#bool) | Indicates whether the virtual service is editable. |
| description | [ string](#string) | Description is the human-readable description of the resource |




## Enums


### ListenerType {#listenertype}
Type of listener available.

| Name | Number | Description |
| ---- | ------ | ----------- |
| LISTENER_TYPE_UNSPECIFIED | 0 | Default value, unspecified listener type. |
| LISTENER_TYPE_HTTP | 1 | HTTP listener. |
| LISTENER_TYPE_HTTPS | 2 | HTTPS listener. |
| LISTENER_TYPE_TCP | 3 | TCP listener. |



### TemplateOptionModifier {#templateoptionmodifier}
Enum describing possible modifiers for template options.

| Name | Number | Description |
| ---- | ------ | ----------- |
| TEMPLATE_OPTION_MODIFIER_UNSPECIFIED | 0 | Unspecified modifier. |
| TEMPLATE_OPTION_MODIFIER_MERGE | 1 | Merge modifier for combining with existing options. |
| TEMPLATE_OPTION_MODIFIER_REPLACE | 2 | Replace modifier to overwrite existing options. |
| TEMPLATE_OPTION_MODIFIER_DELETE | 3 | Delete modifier to remove existing options. |




## Scalar Value Types

| .proto Type | Notes | C++ Type | Java Type | Python Type |
| ----------- | ----- | -------- | --------- | ----------- |
| double |  | double | double | float |
| float |  | float | float | float |
| int32 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint32 instead. | int32 | int | int |
| int64 | Uses variable-length encoding. Inefficient for encoding negative numbers – if your field is likely to have negative values, use sint64 instead. | int64 | long | int/long |
| uint32 | Uses variable-length encoding. | uint32 | int | int/long |
| uint64 | Uses variable-length encoding. | uint64 | long | int/long |
| sint32 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int32s. | int32 | int | int |
| sint64 | Uses variable-length encoding. Signed int value. These more efficiently encode negative numbers than regular int64s. | int64 | long | int/long |
| fixed32 | Always four bytes. More efficient than uint32 if values are often greater than 2^28. | uint32 | int | int |
| fixed64 | Always eight bytes. More efficient than uint64 if values are often greater than 2^56. | uint64 | long | int/long |
| sfixed32 | Always four bytes. | int32 | int | int |
| sfixed64 | Always eight bytes. | int64 | long | int/long |
| bool |  | bool | boolean | boolean |
| string | A string must always contain UTF-8 encoded or 7-bit ASCII text. | string | String | str/unicode |
| bytes | May contain any arbitrary sequence of bytes. | string | ByteString | str |

