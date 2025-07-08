# Authentication and Authorization Implementation

This document provides an overview of the authentication and authorization implementation in the Envoy XDS Controller. For a comprehensive guide to all security features, see the [Security Guide](security.md).

## Table of Contents

1. [Components](#components)
2. [Authentication Flow](#authentication-flow)
3. [Machine Authentication](#machine-authentication)
4. [API Requests with Access Token](#api-requests-with-access-token)
5. [ACL](#acl)
6. [Libraries and Tools Used](#libraries-and-tools-used)

## Components

The authentication system consists of the following components:
1. [Dex](https://github.com/dexidp/dex) (IdP): An identity provider supporting OpenID Connect (OIDC) and OAuth 2.0 protocols
2. Web UI: A React Single Page Application using react-oidc-context for authentication flows
3. Backend API Server: Handles token verification and authorization

## Authentication Flow

The authentication flow follows the Authorization Code Grant with Proof Key for Code Exchange ([PKCE](https://github.com/authts/oidc-client-ts/blob/main/docs/protocols/authorization-code-grant-with-pkce.md)) protocol, which is recommended for public clients such as SPAs:

1. **User Initiates Login**
   - When a user attempts to access the application, the Web UI detects that the user is not authenticated
   - The Web UI initiates the authentication process using the react-oidc-context library

2. **Redirect to Dex (IdP)**
   - The user is redirected to the Dex IdP login page
   - The authorization request includes a code challenge and code verifier as part of the PKCE extension for enhanced security

3. **User Authentication**
   - The user enters their credentials (username and password) on the Dex login page
   - Dex authenticates the user

4. **Authorization Code Issuance**
   - Upon successful authentication, Dex redirects the user back to the Web UI with an authorization code

5. **Token Exchange**
   - The Web UI uses the authorization code and the code verifier to request tokens from Dex
   - Dex issues an ID Token and an Access Token to the Web UI

6. **User Logged In**
   - The Web UI stores the tokens in Session storage in browser
   - The user's authenticated state is updated in the application

## Machine Authentication

In addition to the web-based authentication flow, Dex supports machine authentication, which allows systems to obtain tokens programmatically without going through the web flow. This is useful for:

- Integration with third-party systems
- Automated scripts and CI/CD pipelines
- Development and testing

### Configuration

To enable machine authentication, add the following configuration to your Dex setup:

```yaml
config:
  # ... other configuration ...
  enablePasswordDB: true
  oauth2:
    passwordConnector: local
  staticPasswords:
    - email: "sa@example.com"
      # bcrypt hash of the string "password": $(echo password | htpasswd -BinC 10 admin | cut -d: -f2)
      hash: "$2a$10$2b2cU8CPhOTaGrs1HRQuAueS7JTT5ZHsHSzYiFPm1leZck7Mc8T4W"
      username: "sa"
      userID: "08a8684b-db88-4b73-90a9-3cd1661f5466"
```

This configuration:
- Enables the password database (`enablePasswordDB: true`)
- Sets the password connector to "local" (`passwordConnector: local`)
- Creates a static user with credentials that can be used for machine authentication

### Obtaining Tokens

Once configured, you can obtain tokens using the Resource Owner Password Credentials grant type:

```bash
curl -L -X POST 'http://localhost:5556/token' \
-u 'envoy-xds-controller:' \
-H 'Content-Type: application/x-www-form-urlencoded' \
--data-urlencode 'grant_type=password' \
--data-urlencode 'scope=openid' \
--data-urlencode 'username=sa@example.com' \
--data-urlencode 'password=password'
```

The response will contain the access token and ID token:

```json
{
  "access_token": "eyJhbGc....",
  "token_type": "bearer",
  "expires_in": 86399,
  "id_token": "eyJhbGciOiJSUzI1N..."
}
```

These tokens can be used in the same way as tokens obtained through the web flow, as described in the [API Requests with Access Token](#api-requests-with-access-token) section.

### Security Considerations

When using machine authentication:
- Use strong, unique passwords for service accounts
- Rotate credentials regularly
- Limit the permissions of service accounts to only what is necessary
- Consider using IP restrictions or other security measures to protect the token endpoint

For more information, see:
- [Dex Token Exchange Guide](https://dexidp.io/docs/guides/token-exchange/)
- [Dex Local Connector Documentation](https://dexidp.io/docs/connectors/local/)

## API Requests with Access Token

When the Web UI needs to make API requests to the backend server:

1. **Include Access Token**
   - The Web UI includes the Access Token in the Authorization header of HTTP requests to the backend API server, using the Bearer scheme:

     ```
     Authorization: Bearer <access_token>
     ```

2. **Backend Token Verification**
   - The backend API server intercepts incoming requests using middleware
   - The middleware extracts the Access Token from the Authorization header

3. **Token Validation**
   - The middleware verifies the Access Token by:
     - Checking the token's signature using the public keys from Dex (e.g., via JSON Web Keys (JWKs) endpoint)
     - Validating the token's claims, such as issuer (iss), audience (aud), expiration time (exp), etc.
     - Ensuring the token has the necessary scopes and permissions

4. **Authorized Access**
   - If the token is valid, the middleware allows the request to proceed to the appropriate handler
   - If the token is invalid or expired, the middleware responds with an appropriate HTTP error (e.g., 401 Unauthorized)

## ACL

In addition to authentication and authorization, the system implements fine-grained access control to entities called nodes.
The access control is configured via a configuration map (configmap), which defines the permissions for different user groups.

### Configuration

The access control settings are defined in a JSON configuration, which is supplied to the backend API server through environment variables.
An example configuration is as follows:
```json
{
  "admins": ["*"],
  "authors": ["node1"],
  "users": ["node1", "node2"]
}
```

- **Groups**: The configuration defines multiple user groups, such as "admins", "authors", and "users"
- **Nodes**: Each group is associated with a list of node identifiers (e.g., "node1", "node2")
- **Special Values**: The special value "*" indicates access to all nodes
- **Default Behavior**: If the configuration is not provided, users are granted access to all nodes by default

### Access Control Mechanism

1. **User Groups from JWT Claims**
   - When the API server receives a request with a JWT Access Token, it extracts the user's group memberships from the token's claims
   - The claims should include a list of groups (e.g., "groups") the user belongs to

2. **Group-to-Node Mapping**
   - The API server maps the user's groups to the nodes they have access to, based on the configuration
   - For example:
     - Users in the "admins" group have access to all nodes ("*")
     - Users in the "authors" group have access to "node1"
     - Users in the "users" group have access to "node1" and "node2"

3. **Data Filtering**
   - When the user requests data, the API server filters the records and returns only those associated with the nodes the user has access to
   - This ensures that users can only access data relevant to their permissions

### Example Scenario

**User A**:
- Belongs to the "admins" group
- Has access to all nodes ("*")

**User B**:
- Belongs to the "users" group
- Has access to "node1" and "node2"

**User C**:
- Belongs to both "authors" and "users" groups
- Has access to "node1" (from both groups) and "node2" (from "users" group)

### Implementation Details

**JWT Claims**:
- The JWT Access Token should include a claim (e.g., "groups") that lists the groups the user belongs to
- This claim is used by the API server to determine access rights

**Configuration Loading**:
- The access control configuration is loaded into the API server from environment variables at startup
- The configuration defines which nodes are accessible to each group

**Access Enforcement**:
- Before processing a request, the API server:
  - Extracts the user's groups from the JWT claims
  - Determines the nodes accessible to those groups based on the configuration
  - Filters the data or restricts actions based on the user's node access permissions

**Default Access**:
- If the access control configuration is not provided, the system defaults to granting users access to all nodes
- This ensures backward compatibility and a fail-open policy if access control is not configured

## Libraries and Tools Used

- Dex: An OpenID Connect identity provider written in Go
- [react-oidc-context](https://github.com/authts/react-oidc-context): A React context provider for managing OIDC authentication
- [oidc-client](https://github.com/authts/oidc-client-ts): A JavaScript library for OpenID Connect (OIDC) and OAuth2 protocol support
- Custom middleware for token verification (used github.com/coreos/go-oidc)
