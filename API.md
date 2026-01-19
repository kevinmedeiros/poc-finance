# API Documentation

## Authentication Endpoints

This document describes the authentication endpoints for the POC Finance application.

### Overview

All authentication endpoints are public (no authentication required) and use form-encoded data. Rate limiting is applied to prevent abuse: 5 requests per second per IP address for POST endpoints.

Security features:
- CSRF protection (header-based for HTMX compatibility)
- Password complexity requirements
- HttpOnly cookies for token storage
- Open redirect protection
- Email enumeration prevention

---

### 1. User Registration

Register a new user account.

**Endpoint:** `POST /register`

**Authentication Required:** No

**Rate Limited:** Yes (5 req/sec per IP)

**Request Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| email | string | Yes | User's email address |
| password | string | Yes | User's password (see requirements below) |
| name | string | Yes | User's display name |

**Password Requirements:**
- Minimum 8 characters
- At least one uppercase letter (A-Z)
- At least one lowercase letter (a-z)
- At least one number (0-9)

**Content-Type:** `application/x-www-form-urlencoded`

**Example Request:**
```http
POST /register HTTP/1.1
Host: localhost:8080
Content-Type: application/x-www-form-urlencoded

email=user@example.com&password=SecurePass123&name=John+Doe
```

**Success Response:**
- **Status Code:** 303 See Other
- **Location Header:** `/login?registered=1`
- **Description:** Redirects to login page with success indicator

**Error Responses:**

| Error Message | Description |
|--------------|-------------|
| "Dados inválidos" | Invalid request format or data binding error |
| "Todos os campos são obrigatórios" | One or more required fields are missing |
| "A senha deve ter pelo menos 8 caracteres" | Password is too short |
| "A senha deve conter letras maiúsculas, minúsculas e números" | Password doesn't meet complexity requirements |
| "Este email já está cadastrado" | Email address is already registered |
| "Erro ao criar conta. Tente novamente." | Server error during registration |

**Error Response Format:**
- **Status Code:** 200 OK
- **Content-Type:** text/html
- **Body:** Rendered registration page with error message and preserved form values

---

### 2. User Login

Authenticate a user and establish a session.

**Endpoint:** `POST /login`

**Authentication Required:** No

**Rate Limited:** Yes (5 req/sec per IP)

**Request Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| email | string | Yes | User's email address |
| password | string | Yes | User's password |
| redirect | string | No | URL to redirect after successful login (must be relative path) |

**Content-Type:** `application/x-www-form-urlencoded`

**Example Request:**
```http
POST /login HTTP/1.1
Host: localhost:8080
Content-Type: application/x-www-form-urlencoded

email=user@example.com&password=SecurePass123&redirect=/dashboard
```

**Success Response:**
- **Status Code:** 303 See Other
- **Location Header:** `{redirect}` or `/` (default)
- **Set-Cookie Headers:**
  - `access_token`: JWT access token (HttpOnly, SameSite=Lax)
  - `refresh_token`: JWT refresh token (HttpOnly, SameSite=Lax)
- **Description:** Sets authentication cookies and redirects to specified URL or home page

**Cookie Details:**
- **access_token:**
  - HttpOnly: true
  - Secure: true (in production)
  - SameSite: Lax
  - Path: /
  - MaxAge: Based on AccessTokenDuration

- **refresh_token:**
  - HttpOnly: true
  - Secure: true (in production)
  - SameSite: Lax
  - Path: /
  - MaxAge: Based on RefreshTokenDuration

**Redirect Protection:**
The redirect parameter is validated to prevent open redirect vulnerabilities:
- Must start with `/`
- Must not start with `//`
- Must not contain `://`

**Error Responses:**

| Error Message | Description |
|--------------|-------------|
| "Dados inválidos" | Invalid request format or data binding error |
| "Email e senha são obrigatórios" | Email or password is missing |
| "Email ou senha incorretos" | Invalid credentials |

**Error Response Format:**
- **Status Code:** 200 OK
- **Content-Type:** text/html
- **Body:** Rendered login page with error message and preserved email

---

### 3. User Logout

Revoke the user's session and clear authentication cookies.

**Endpoint:** `POST /logout`

**Authentication Required:** No (but requires valid cookies to revoke tokens)

**Rate Limited:** No

**CSRF Protection:** Skipped (safe operation, needs to work with expired tokens)

**Request Parameters:** None

**Example Request:**
```http
POST /logout HTTP/1.1
Host: localhost:8080
Cookie: access_token=...; refresh_token=...
```

**Success Response:**
- **Status Code:** 303 See Other
- **Location Header:** `/login`
- **Set-Cookie Headers:**
  - `access_token`: Cleared (MaxAge=-1)
  - `refresh_token`: Cleared (MaxAge=-1)
- **Description:** Revokes refresh token, clears cookies, and redirects to login page

**Token Revocation:**
If a valid refresh_token cookie is present, it will be revoked on the server side before clearing the cookies.

**Error Responses:**
None - logout always succeeds and redirects to login page

---

### 4. Forgot Password

Initiate password reset process by requesting a reset link.

**Endpoint:** `POST /forgot-password`

**Authentication Required:** No

**Rate Limited:** Yes (5 req/sec per IP)

**Request Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| email | string | Yes | User's registered email address |

**Content-Type:** `application/x-www-form-urlencoded`

**Example Request:**
```http
POST /forgot-password HTTP/1.1
Host: localhost:8080
Content-Type: application/x-www-form-urlencoded

email=user@example.com
```

**Success Response:**
- **Status Code:** 200 OK
- **Content-Type:** text/html
- **Body:** Rendered forgot-password page with success message
- **Description:** Always shows success message, regardless of whether email exists (prevents email enumeration)

**Implementation Note:**
Currently, this endpoint does not send emails. In production, it would:
1. Generate a password reset token
2. Send an email with a reset link containing the token
3. Token would expire after a set time period

**Error Responses:**

| Error Message | Description |
|--------------|-------------|
| "Dados inválidos" | Invalid request format |
| "Email é obrigatório" | Email field is empty |

**Error Response Format:**
- **Status Code:** 200 OK
- **Content-Type:** text/html
- **Body:** Rendered forgot-password page with error message

**Security Note:**
The endpoint always returns a success message when a valid email is submitted to prevent attackers from enumerating registered email addresses.

---

### 5. Reset Password

Reset user password using a valid reset token.

**Endpoint:** `POST /reset-password`

**Authentication Required:** No (uses token from email)

**Rate Limited:** Yes (5 req/sec per IP)

**Request Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| token | string | Yes | Password reset token from email link |
| password | string | Yes | New password (see requirements below) |
| password_confirm | string | Yes | Password confirmation (must match password) |

**Password Requirements:**
- Minimum 8 characters
- At least one uppercase letter (A-Z)
- At least one lowercase letter (a-z)
- At least one number (0-9)
- Must match password_confirm

**Content-Type:** `application/x-www-form-urlencoded`

**Example Request:**
```http
POST /reset-password HTTP/1.1
Host: localhost:8080
Content-Type: application/x-www-form-urlencoded

token=abc123def456&password=NewSecurePass123&password_confirm=NewSecurePass123
```

**Success Response:**
- **Status Code:** 303 See Other
- **Location Header:** `/login?reset=1`
- **Description:** Password successfully reset, redirects to login page with success indicator

**Error Responses:**

| Error Message | Description |
|--------------|-------------|
| "Dados inválidos" | Invalid request format or data binding error |
| "Todos os campos são obrigatórios" | One or more required fields are missing |
| "A senha deve ter pelo menos 8 caracteres" | Password is too short |
| "A senha deve conter letras maiúsculas, minúsculas e números" | Password doesn't meet complexity requirements |
| "As senhas não coincidem" | password and password_confirm don't match |
| "Link de recuperação inválido ou expirado" | Token is invalid, expired, or already used |
| "Erro ao redefinir senha. Tente novamente." | Server error during password reset |

**Error Response Format:**
- **Status Code:** 200 OK
- **Content-Type:** text/html
- **Body:** Rendered reset-password page with error message and token

**Token Validation:**
Before showing the reset form, tokens are validated on the GET request. Invalid or expired tokens will display an error message without showing the password form.

---

## Common Error Handling

All authentication endpoints follow these error handling patterns:

### HTTP Status Codes

| Code | Description |
|------|-------------|
| 200 OK | Request processed (may contain validation errors in rendered HTML) |
| 303 See Other | Successful action, redirect to next page |
| 404 Not Found | Endpoint not found |
| 429 Too Many Requests | Rate limit exceeded |

### Error Display

Errors are displayed in the rendered HTML pages, not as JSON responses. The application uses server-side rendering with templates.

### Rate Limiting

POST endpoints are rate-limited to 5 requests per second per IP address. When the limit is exceeded:
- **Status Code:** 429 Too Many Requests
- **Retry-After Header:** Indicates when the client can retry

### CSRF Protection

All POST endpoints (except /logout) require CSRF tokens:
- **Token Location:** `X-CSRF-Token` header or `_csrf` form field
- **Token Source:** Automatically included by HTMX or available in template context
- **Missing Token Response:** 403 Forbidden

---

## Authentication Flow

### Registration Flow
1. User submits registration form → `POST /register`
2. Server validates input and creates account
3. Server redirects to `/login?registered=1`
4. User sees login page with success message

### Login Flow
1. User submits login form → `POST /login`
2. Server validates credentials
3. Server generates access and refresh tokens
4. Server sets HttpOnly cookies with tokens
5. Server redirects to home page or specified redirect URL
6. User is authenticated for subsequent requests

### Logout Flow
1. User initiates logout → `POST /logout`
2. Server revokes refresh token
3. Server clears authentication cookies
4. Server redirects to `/login`

### Password Reset Flow
1. User requests password reset → `POST /forgot-password`
2. Server generates reset token and sends email (when implemented)
3. User clicks link in email → `GET /reset-password?token=...`
4. Server validates token and shows reset form
5. User submits new password → `POST /reset-password`
6. Server resets password and redirects to `/login?reset=1`

---

## Security Considerations

### Password Security
- Passwords are hashed using bcrypt before storage
- Plain text passwords are never logged or stored
- Password complexity requirements enforce strong passwords

### Token Security
- Access and refresh tokens are stored in HttpOnly cookies (prevents XSS)
- Cookies use SameSite=Lax (prevents CSRF)
- Cookies use Secure flag in production (HTTPS only)
- Refresh tokens can be revoked server-side

### Request Security
- CSRF protection on all state-changing operations
- Rate limiting prevents brute force attacks
- Input sanitization prevents XSS (HTML escaping)
- Email trimming and validation
- Open redirect protection on login

### Privacy & Enumeration Prevention
- Forgot password always returns success (prevents email enumeration)
- Generic error messages for login failures
- No distinction between "user not found" and "wrong password"

---

## Related Pages

### GET Endpoints (for rendering forms)

| Endpoint | Description |
|----------|-------------|
| `GET /register` | Display registration form |
| `GET /login` | Display login form |
| `GET /forgot-password` | Display forgot password form |
| `GET /reset-password?token=...` | Display reset password form (requires valid token) |

These endpoints render HTML pages with forms that submit to the corresponding POST endpoints documented above.

---

## Dashboard and Account Endpoints

This section describes the main dashboard and account management endpoints.

### Overview

These endpoints require authentication and return server-rendered HTML pages. They use the same CSRF protection and security headers as authentication endpoints.

---

### 1. Dashboard

Display the main financial dashboard with monthly summaries, projections, and upcoming bills.

**Endpoint:** `GET /`

**Authentication Required:** Yes (JWT token in HttpOnly cookie)

**Rate Limited:** No

**Request Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| account_id | uint | No | Filter dashboard by specific account ID. Use "all" or omit for all accounts |

**Query String Examples:**
```
/?account_id=1         # Filter by account ID 1
/?account_id=all       # Show all accounts (default)
/                      # Show all accounts (default)
```

**Example Request:**
```http
GET /?account_id=1 HTTP/1.1
Host: localhost:8080
Cookie: access_token=...; refresh_token=...
```

**Success Response:**
- **Status Code:** 200 OK
- **Content-Type:** text/html
- **Body:** Rendered dashboard.html template with financial data

**Response Data Includes:**
- **Current Month Summary:**
  - Total gross income
  - Total taxes
  - Total expenses
  - Net balance
- **6-Month Financial Projections:**
  - Monthly income/expense forecasts
  - Uses batch query optimization (5 queries instead of 30)
- **Tax Information:**
  - 12-month revenue calculation
  - Current tax bracket
  - Effective tax rate
  - INSS amount
- **Upcoming Bills (next 30 days):**
  - Fixed expenses with due dates
  - Unpaid bills
  - Card installments
  - Sorted by due date, limited to 10 items
- **Account Filter:**
  - List of user's accounts
  - Currently selected account

**Account Validation:**
If an invalid account_id is provided or the user doesn't have access to the specified account, the endpoint falls back to showing all accounts without returning an error.

**Performance Optimization:**
The dashboard uses batch query optimization for 6-month projections, reducing database queries from 30 to 5 for improved performance.

**Error Responses:**

| Error | Description |
|-------|-------------|
| 401 Unauthorized | Missing or invalid authentication token |
| 404 Not Found | Template not found (server configuration error) |

---

### 2. List Accounts

Display all user accounts with their current balances.

**Endpoint:** `GET /accounts`

**Authentication Required:** Yes (JWT token in HttpOnly cookie)

**Rate Limited:** No

**Request Parameters:** None

**Example Request:**
```http
GET /accounts HTTP/1.1
Host: localhost:8080
Cookie: access_token=...; refresh_token=...
```

**Success Response:**
- **Status Code:** 200 OK
- **Content-Type:** text/html
- **Body:** Rendered accounts.html template with account data

**Response Data Includes:**
- **Accounts List:**
  - Account ID
  - Account name
  - Account type (personal, joint)
  - Current balance
  - Group information (if joint account)
- **Total Balance:**
  - Sum of all account balances

**Balance Calculation:**
Account balances are calculated by summing:
- All income transactions for the account
- Minus all expense transactions
- Minus all credit card transactions
- Including all bill payments

**Account Types:**
- **Personal Accounts:** Created by and owned by a single user
- **Joint Accounts:** Shared accounts within family groups, accessible by all group members

**Error Responses:**

| Error | Description |
|-------|-------------|
| 401 Unauthorized | Missing or invalid authentication token |
| 500 Internal Server Error | Database error fetching accounts (returns HTML error message) |

**Error Response Format:**
- **Status Code:** 500 Internal Server Error
- **Content-Type:** text/html
- **Body:** Plain text error message: "Erro ao buscar contas"

---

## Authentication Middleware

Both dashboard and account endpoints are protected by authentication middleware that:

1. **Validates JWT Token:**
   - Checks for access_token in HttpOnly cookie
   - Validates token signature and expiration
   - Extracts user ID from token claims

2. **Token Refresh:**
   - If access token is expired but refresh token is valid
   - Automatically generates new access token
   - Sets new cookie transparently

3. **Authorization:**
   - Ensures user can only access their own data
   - Validates account ownership for filtered views
   - Blocks access to other users' accounts

4. **Failed Authentication:**
   - Returns 401 Unauthorized
   - Redirects to login page
   - Preserves redirect URL for return after login

---

## Related Endpoints

### Dashboard-Related Pages

| Endpoint | Description |
|----------|-------------|
| `GET /incomes` | Income management page |
| `GET /expenses` | Expense management page |
| `GET /cards` | Credit card management page |
| `GET /settings` | User settings and tax configuration |
| `GET /export` | Export financial data |

These related endpoints follow the same authentication and security patterns as the dashboard endpoint.
