export interface IFilterNameResponse {
    filters: Filter[];
}

export interface Filter {
    stat_prefix:        string;
    RouteSpecifier:     RouteSpecifier;
    http_filters:       AccessLog[];
    access_log:         AccessLog[];
    use_remote_address: UseRemoteAddress;
    StripPortMode:      null;
}

export interface RouteSpecifier {
    Rds: RDS;
}

export interface RDS {
    config_source:     ConfigSource;
    route_config_name: string;
}

export interface ConfigSource {
    ConfigSourceSpecifier: ConfigSourceSpecifier;
    resource_api_version:  number;
}

export interface ConfigSourceSpecifier {
    Ads: Ads;
}

export interface Ads {
}

export interface AccessLog {
    name:       string;
    ConfigType: ConfigType;
}

export interface ConfigType {
    TypedConfig: TypedConfig;
}

export interface TypedConfig {
    type_url: string;
    value?:   string;
}

export interface UseRemoteAddress {
    value: boolean;
}
