# Security Guide: Envoy XDS Controller

This document provides a comprehensive overview of the security features and mechanisms implemented in the Envoy XDS Controller. For a detailed explanation of the authentication flow and ACL implementation, see the [Authentication and Authorization Implementation](auth.md) document.

## Table of Contents

1. [Authentication](#authentication)
2. [Authorization](#authorization)
3. [Access Control](#access-control)
4. [API Security](#api-security)
5. [Best Practices](#best-practices)
6. [Configuration](#configuration)
7. [Troubleshooting](#troubleshooting)
8. [Support](#support)

## Authentication

### OIDC Authentication

The controller uses OpenID Connect (OIDC) for authentication:

- **Provider Integration**: Supports any OIDC-compliant identity provider
- **Token Validation**: JWT tokens are validated for:
  - Signature verification
  - Expiration checks
  - Issuer validation
  - Audience validation

### Token Management

- **Bearer Token**: Required in Authorization header
- **Token Format**: JWT (JSON Web Token)
- **Claims**: Supports standard OIDC claims:
  - `name`: User identifier
  - `groups`: User group memberships
  - Standard OIDC claims (iss, sub, aud, exp, etc.)

## Authorization

### Role-Based Access Control (RBAC)

The controller implements a comprehensive RBAC system using Casbin:

#### Components

1. **Authorizer**:
   ```go
   type Authorizer struct {
       name     string    // User identifier
       groups   []string  // User groups
       action   string    // Requested action
       enforcer *casbin.Enforcer
   }
   ```

2. **Access Groups**:
   - Domain-specific access control
   - Support for general domain (`_`)
   - Wildcard access (`*`)

#### RBAC Model

The RBAC model is defined in the Helm chart configuration and supports:

1. **Request Definition**:
   ```
   r = sub, dom, obj, act
   ```
   Where:
   - `sub`: Subject (user or group)
   - `dom`: Domain (access group)
   - `obj`: Object (resource)
   - `act`: Action (operation)

2. **Policy Definition**:
   ```
   p = sub, dom, obj, act
   ```

3. **Role Definition**:
   ```
   g = _, _, _
   ```
   Note: The role definition includes three parameters to support domain-specific role assignments.

4. **Policy Effect**:
   ```
   e = some(where (p.eft == allow))
   ```

5. **Matchers**:
   ```
   m = g(r.sub, p.sub, r.dom) && globMatch(r.dom, p.dom) && globMatch(r.obj, p.obj) && r.act == p.act || r.sub == "superuser"
   ```
   The matcher supports:
   - Role-based access control with domain inheritance
   - Glob pattern matching for domains and objects
   - Superuser bypass for all permissions

#### Default Policy Configuration

The default policy configuration includes predefined roles:

1. **Reader Role** (`role:reader`):
   ```csv
   p, role:reader, *, *, list-virtual-services
   p, role:reader, *, *, list-virtual-service-templates
   p, role:reader, *, *, list-listeners
   p, role:reader, *, *, list-nodes
   p, role:reader, *, *, list-access-log-configs
   p, role:reader, *, *, list-http-filters
   p, role:reader, *, *, list-routes
   p, role:reader, *, *, get-virtual-service
   p, role:reader, *, *, fill-template
   ```

2. **Editor Role** (`role:editor`):
   ```csv
   p, role:editor, *, *, list-virtual-services
   p, role:editor, *, *, list-virtual-service-templates
   p, role:editor, *, *, list-listeners
   p, role:editor, *, *, list-nodes
   p, role:editor, *, *, list-access-log-configs
   p, role:editor, *, *, list-http-filters
   p, role:editor, *, *, list-routes
   p, role:editor, *, *, get-virtual-service
   p, role:editor, *, *, fill-template
   p, role:editor, *, *, create-virtual-service
   p, role:editor, *, *, update-virtual-service
   ```

#### Custom Policy Configuration

Custom policies can be added through Helm values:

```yaml
auth:
  enabled: true
  rbacPolicy: |
    p, custom-role, domain1, resource1, action1
    p, custom-role, domain2, *, action2
    g, user1, custom-role, domain1
```

#### Available Actions

```go
const (
    ActionListVirtualServices         = "list-virtual-services"
    ActionGetVirtualService           = "get-virtual-service"
    ActionCreateVirtualService        = "create-virtual-service"
    ActionUpdateVirtualService        = "update-virtual-service"
    ActionDeleteVirtualService        = "delete-virtual-service"
    ActionListAccessLogConfigs        = "list-access-log-configs"
    ActionListVirtualServiceTemplates = "list-virtual-service-templates"
    ActionListNodes                   = "list-nodes"
    ActionListRoutes                  = "list-routes"
    ActionListHTTPFilters             = "list-http-filters"
    ActionListPolicies                = "list-policies"
    ActionListAccessGroups            = "list-access-groups"
    ActionListListeners               = "list-listeners"
    ActionListPermissions             = "list-permissions"
)
```

#### Dynamic Policy Updates

The system supports dynamic policy updates:
- Policy changes are detected through file watching
- Model and policy are reloaded automatically
- Changes take effect without service restart

## Access Control

### Access Group Management

1. **Group Types**:
   - Regular access groups
   - General domain (`general`)
   - Wildcard access (`*`)

2. **Access Levels**:
   - Full access
   - Read-only access
   - Domain-specific access
   - Object-specific access

### Permission Checking

```go
func (a *Authorizer) Authorize(domain string, object any) (bool, error) {
    for _, sub := range a.getSubjects() {
        result, err := a.enforcer.Enforce(sub, domain, object, a.action)
        if err != nil {
            return false, err
        }
        if result {
            return true, nil
        }
    }
    return false, nil
}
```

## API Security

### gRPC Security

1. **Authentication Middleware**:
   ```go
   type AuthMiddleware struct {
       verifier          *oidc.IDTokenVerifier
       wrappedMiddleware *authn.Middleware
       enforcer          *casbin.Enforcer
   }
   ```

2. **Request Flow**:
   - Token extraction
   - Token validation
   - Claims extraction
   - Permission checking
   - Action authorization

### REST API Security

1. **Authentication**:
   - OIDC token validation
   - JWT verification
   - Group membership checking

2. **Authorization**:
   - Resource-level access control
   - Action-based permissions
   - Domain-specific restrictions

## Best Practices

### Configuration

1. **OIDC Configuration**:
   ```yaml
   auth:
     enabled: true
     issuerURL: "https://your-oidc-provider"
     clientID: "your-client-id"
   ```

2. **RBAC Configuration**:
   ```yaml
   auth:
     enabled: true
     rbacPolicy: |
       p, role:custom, domain1, resource1, action1
       g, user1, role:custom, domain1
   ```

### Security Recommendations

1. **Token Management**:
   - Use short-lived tokens
   - Implement token refresh
   - Validate token claims

2. **Access Control**:
   - Follow principle of least privilege
   - Regular audit of permissions
   - Use domain-specific access groups

3. **API Security**:
   - Enable TLS for all connections
   - Implement rate limiting
   - Monitor access patterns

## Configuration

### Environment Variables

```bash
OIDC_ENABLED=true
OIDC_ISSUER_URL=https://your-oidc-provider
OIDC_CLIENT_ID=your-client-id
ACL_CONFIG={"group1":["node1","node2"],"group2":["*"]}
```

### Access Control Configuration

1. **Model Configuration**:
   ```conf
   [request_definition]
   r = sub, dom, obj, act

   [policy_definition]
   p = sub, dom, obj, act

   [role_definition]
   g = _, _, _

   [policy_effect]
   e = some(where (p.eft == allow))

   [matchers]
   m = g(r.sub, p.sub, r.dom) && globMatch(r.dom, p.dom) && globMatch(r.obj, p.obj) && r.act == p.act || r.sub == "superuser"
   ```

2. **Policy Configuration**:
   ```csv
   # Default reader role
   p, role:reader, *, *, list-virtual-services
   p, role:reader, *, *, get-virtual-service

   # Default editor role
   p, role:editor, *, *, create-virtual-service
   p, role:editor, *, *, update-virtual-service

   # Custom role
   p, role:custom, domain1, resource1, action1
   g, user1, role:custom, domain1
   ```

### Monitoring and Auditing

1. **Access Logs**:
   - Authentication attempts
   - Authorization decisions
   - Resource access

2. **Metrics**:
   - Authentication success/failure
   - Authorization success/failure
   - API usage patterns

## Troubleshooting

### Common Issues

1. **Authentication Failures**:
   - Check token validity
   - Verify OIDC configuration
   - Check token claims

2. **Authorization Failures**:
   - Verify user groups
   - Check access group configuration
   - Review RBAC policies

### Debug Mode

Enable debug mode for detailed security logs:
```bash
APP_DEV_MODE=true
```

## Support

For security-related issues or questions:
- GitHub Issues: [Security Issues](https://github.com/kaasops/envoy-xds-controller/issues)
- Documentation: [Security Documentation](https://github.com/kaasops/envoy-xds-controller/tree/main/docs)

## Related Documentation

- [Authentication and Authorization Implementation](auth.md): Detailed explanation of the authentication flow and ACL implementation
- [Configuration Guide](configuration.md): Configuration options for the Envoy XDS Controller
- [Troubleshooting Guide](troubleshooting.md): Help with common issues