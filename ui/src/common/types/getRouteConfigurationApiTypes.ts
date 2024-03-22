export interface IRouteConfigurationResponse {
    routeConfigurations: RouteConfiguration[];
}

export interface RouteConfiguration {
    name:          string;
    virtual_hosts: VirtualHost[];
}

export interface VirtualHost {
    name:                   string;
    domains:                string[];
    routes:                 Route[];
    request_headers_to_add: RequestHeadersToAdd[];
}

export interface RequestHeadersToAdd {
    header:         Header;
    append_action?: number;
}

export interface Header {
    key:   string;
    value: string;
}

export interface Route {
    name:   string;
    match:  Match;
    Action: Action;
}

export interface Action {
    DirectResponse?: DirectResponse;
    Route?:          RouteClass;
}

export interface DirectResponse {
    status: number;
    body?:  Body;
}

export interface Body {
    Specifier: Specifier;
}

export interface Specifier {
    InlineString: string;
}

export interface RouteClass {
    ClusterSpecifier:     ClusterSpecifier;
    HostRewriteSpecifier: HostRewriteSpecifier;
}

export interface ClusterSpecifier {
    Cluster: string;
}

export interface HostRewriteSpecifier {
    HostRewriteLiteral?: string;
    HostRewriteHeader?:  string;
}

export interface Match {
    PathSpecifier: PathSpecifier;
}

export interface PathSpecifier {
    Path?:   string;
    Prefix?: string;
}