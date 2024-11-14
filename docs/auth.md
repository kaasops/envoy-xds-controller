# Authentication and Authorization Implementation Overview

This document provides an overview of the authentication and authorization implementation.
It is intended to help developers understand how the different components interact and how user authentication is handled.

## Components

The system consists of the following components:
1.	[Dex](https://github.com/dexidp/dex) (IdP): An identity provider (IdP) based on Dex, which supports OpenID Connect (OIDC) and OAuth 2.0 protocols for authentication.
2.	Web UI (React SPA): A Single Page Application (SPA) built with React, utilizing the react-oidc-context library (a wrapper around oidc-client) to handle authentication flows.
3.	Backend API Server (Go): An API server written in Go, which provides backend services to the Web UI and handles token verification and authorization.

## Authentication Flow

The authentication flow follows the Authorization Code Grant with Proof Key for Code Exchange ([PKCE](https://github.com/authts/oidc-client-ts/blob/main/docs/protocols/authorization-code-grant-with-pkce.md)) protocol, which is recommended for public clients such as SPAs. Here is how the flow works:
1.	User Initiates Login:
•	When a user attempts to access the application, the Web UI detects that the user is not authenticated.
•	The Web UI initiates the authentication process using the react-oidc-context library.
2.	Redirect to Dex (IdP):
•	The user is redirected to the Dex IdP login page.
•	The authorization request includes a code challenge and code verifier as part of the PKCE extension for enhanced security.
3.	User Authentication:
•	The user enters their credentials (username and password) on the Dex login page.
•	Dex authenticates the user.
4.	Authorization Code Issuance:
•	Upon successful authentication, Dex redirects the user back to the Web UI with an authorization code.
5.	Token Exchange:
•	The Web UI uses the authorization code and the code verifier to request tokens from Dex.
•	Dex issues an ID Token and an Access Token to the Web UI.
6.	User Logged In:
•	The Web UI stores the tokens in Session storage in browser.
•	The user’s authenticated state is updated in the application.

## API Requests with Access Token

When the Web UI needs to make API requests to the backend server:
1.	Include Access Token:  
•	The Web UI includes the Access Token in the Authorization header of HTTP requests to the backend API server, using the Bearer scheme: <br><br>_Authorization: Bearer <access_token>_<br/><br/>

2.	Backend Token Verification:  
•	The backend API server intercepts incoming requests using middleware.  
•	The middleware extracts the Access Token from the Authorization header.  
3.	Token Validation:  
•	The middleware verifies the Access Token by:  
•	Checking the token’s signature using the public keys from Dex (e.g., via JSON Web Keys (JWKs) endpoint).  
•	Validating the token’s claims, such as issuer (iss), audience (aud), expiration time (exp), etc.  
•	Ensuring the token has the necessary scopes and permissions.  
4.	Authorized Access:  
•	If the token is valid, the middleware allows the request to proceed to the appropriate handler.  
•	If the token is invalid or expired, the middleware responds with an appropriate HTTP error (e.g., 401 Unauthorized). 

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
    
-	Groups: The configuration defines multiple user groups, such as "admins", "authors", and "users".
-	Nodes: Each group is associated with a list of node identifiers (e.g., "node1", "node2").
-	The special value "*" indicates access to all nodes.
-	Default Behavior: If the configuration is not provided, users are granted access to all nodes by default.`

### Access Control Mechanism

      1.	User Groups from JWT Claims:
      •	When the API server receives a request with a JWT Access Token, it extracts the user’s group memberships from the token’s claims.
      •	The claims should include a list of groups (e.g., "groups") the user belongs to.
      2.	Group-to-Node Mapping:
      •	The API server maps the user’s groups to the nodes they have access to, based on the configuration.
      •	For example:
      •	Users in the "admins" group have access to all nodes ("*") .
      •	Users in the "authors" group have access to "node1".
      •	Users in the "users" group have access to "node1" and "node2".
      3.	Data Filtering:
      •	When the user requests data, the API server filters the records and returns only those associated with the nodes the user has access to.
      •	This ensures that users can only access data relevant to their permissions.

Example Scenario

User A:
- Belongs to the "admins" group.
- Has access to all nodes ("*").

User B:
- Belongs to the "users" group.
- Has access to "node1" and "node2".

User C:
- Belongs to both "authors" and "users" groups.
- Has access to "node1" (from both groups) and "node2" (from "users" group).

### Implementation Details

JWT Claims:
-	The JWT Access Token should include a claim (e.g., "groups") that lists the groups the user belongs to.
- 	This claim is used by the API server to determine access rights.

Configuration Loading:
-	The access control configuration is loaded into the API server from environment variables at startup.
-	The configuration defines which nodes are accessible to each group.

Access Enforcement. Before processing a request, the API server:
-	Extracts the user’s groups from the JWT claims.
-	Determines the nodes accessible to those groups based on the configuration.  
-	Filters the data or restricts actions based on the user’s node access permissions.

Default Access:
-	If the access control configuration is not provided, the system defaults to granting users access to all nodes.
-	This ensures backward compatibility and a fail-open policy if access control is not configured.

## Libraries and Tools Used

- Dex: An OpenID Connect identity provider written in Go
- [react-oidc-context](https://github.com/authts/react-oidc-context): A React context provider for managing OIDC authentication.
- [oidc-client](https://github.com/authts/oidc-client-ts): A JavaScript library for OpenID Connect (OIDC) and OAuth2 protocol support.
- Custom middleware for token verification (used github.com/coreos/go-oidc).
