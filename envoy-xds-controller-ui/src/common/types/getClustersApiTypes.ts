export interface IClustersResponse {
    clusters: Cluster[];
}

export interface Cluster {
    name:                 string;
    ClusterDiscoveryType: ClusterDiscoveryType;
    connect_timeout:      ConnectTimeout;
    lb_policy:            number;
    load_assignment:      LoadAssignment;
    LbConfig:             null;
    transport_socket:     TransportSocket;
}

export interface ClusterDiscoveryType {
    Type: number;
}

export interface ConnectTimeout {
    seconds: number;
}

export interface LoadAssignment {
    cluster_name: string;
    endpoints:    Endpoint[];
}

export interface Endpoint {
    lb_endpoints: LBEndpoint[];
    LbConfig:     null;
}

export interface LBEndpoint {
    HostIdentifier: HostIdentifier;
}

export interface HostIdentifier {
    Endpoint: EndpointClass;
}

export interface EndpointClass {
    address: Address;
}

export interface Address {
    Address: AddressClass;
}

export interface AddressClass {
    SocketAddress: SocketAddress;
}

export interface SocketAddress {
    address:       string;
    PortSpecifier: PortSpecifier;
}

export interface PortSpecifier {
    PortValue: number;
}

export interface TransportSocket {
    name:       string;
    ConfigType: ConfigType;
}

export interface ConfigType {
    TypedConfig: TypedConfig;
}

export interface TypedConfig {
    type_url: string;
    value:    string;
}