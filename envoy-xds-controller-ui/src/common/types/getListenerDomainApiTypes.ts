export interface IListenersResponse {
    listeners: Listener[];
}

export interface Listener {
    name:              string;
    address:           Address;
    filter_chains:     FilterChain[];
    listener_filters:  ListenerFilter[];
    ListenerSpecifier: null | any;
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

export interface FilterChain {
    filter_chain_match: FilterChainMatch;
    filters:            TransportSocket[];
    transport_socket:   TransportSocket;
    name:               string;
}

export interface FilterChainMatch {
    server_names: string[];
}

export interface TransportSocket {
    name:       string;
    ConfigType: TransportSocketConfigType;
}

export interface TransportSocketConfigType {
    TypedConfig: PurpleTypedConfig;
}

export interface PurpleTypedConfig {
    type_url: string;
    value:    string;
}

export interface ListenerFilter {
    name:       string;
    ConfigType: ListenerFilterConfigType;
}

export interface ListenerFilterConfigType {
    TypedConfig: FluffyTypedConfig;
}

export interface FluffyTypedConfig {
    type_url: string;
}


export interface IListenersErrorResponse {
    error: string;
    name: string;
}