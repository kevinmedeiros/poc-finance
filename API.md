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

---

## Income Endpoints

This section describes the income management endpoints for the POC Finance application.

### Overview

All income endpoints require authentication via JWT tokens (access_token cookie). These endpoints manage income records across user accounts, including automatic tax calculations based on Brazilian progressive tax brackets.

Key features:
- Multi-account support (individual and joint accounts)
- Automatic tax calculation based on 12-month rolling revenue
- HTMX partial responses for dynamic UI updates
- Currency conversion tracking (USD to BRL)
- Account access validation

---

### 1. List Incomes

Retrieve all income records for accounts accessible by the authenticated user.

**Endpoint:** `GET /incomes`

**Authentication Required:** Yes (JWT access_token cookie)

**Rate Limited:** No

**Request Parameters:** None

**Example Request:**
```http
GET /incomes HTTP/1.1
Host: localhost:8080
Cookie: access_token=...
```

**Success Response:**
- **Status Code:** 200 OK
- **Content-Type:** text/html
- **Body:** Rendered income.html page with income list and tax information

**Response Data Includes:**

| Field | Type | Description |
|-------|------|-------------|
| incomes | []Income | Array of income records ordered by date (DESC) |
| accounts | []Account | User's accessible accounts for the account selector |
| revenue12m | float64 | Total revenue for last 12 months across all accounts |
| currentBracket | string | Current tax bracket description |
| effectiveRate | float64 | Effective tax rate as a percentage |
| nextBracketAt | float64 | Revenue amount at which next tax bracket applies |

**Income Record Fields:**

| Field | Type | Description |
|-------|------|-------------|
| ID | uint | Income record ID |
| AccountID | uint | Associated account ID |
| Date | time.Time | Date of income receipt |
| AmountUSD | float64 | Amount in USD |
| ExchangeRate | float64 | USD to BRL exchange rate used |
| AmountBRL | float64 | Amount in BRL (AmountUSD × ExchangeRate) |
| GrossAmount | float64 | Gross amount before taxes |
| TaxAmount | float64 | Calculated tax amount |
| NetAmount | float64 | Net amount after taxes |
| Description | string | Income description |

**Tax Calculation:**
Tax is calculated based on Brazilian progressive tax brackets using the 12-month rolling revenue across all user accounts.

**Error Responses:**

| Error | Description |
|-------|-------------|
| 401 Unauthorized | Missing or invalid authentication token |
| 404 Not Found | Template not found (server configuration error) |

---

### 2. Create Income

Create a new income record with automatic tax calculation.

**Endpoint:** `POST /incomes`

**Authentication Required:** Yes (JWT access_token cookie)

**Rate Limited:** No

**CSRF Protection:** Yes (X-CSRF-Token header required)

**Request Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| account_id | uint | No | Account ID (defaults to user's individual account if not specified) |
| date | string | Yes | Date in YYYY-MM-DD format |
| amount_usd | float64 | Yes | Amount in USD |
| exchange_rate | float64 | Yes | USD to BRL exchange rate |
| description | string | Yes | Description of the income |

**Content-Type:** `application/x-www-form-urlencoded`

**Example Request:**
```http
POST /incomes HTTP/1.1
Host: localhost:8080
Cookie: access_token=...
X-CSRF-Token: ...
Content-Type: application/x-www-form-urlencoded

account_id=1&date=2024-01-15&amount_usd=5000.00&exchange_rate=5.20&description=Client+Project+Payment
```

**Automatic Calculations:**
- `AmountBRL = AmountUSD × ExchangeRate`
- Tax is calculated based on 12-month rolling revenue
- `NetAmount = AmountBRL - TaxAmount`

**Success Response:**
- **Status Code:** 200 OK
- **Content-Type:** text/html
- **HX-Trigger:** Updates UI via HTMX
- **Body:** Rendered `partials/income-list.html` with updated income list

**HTMX Partial Response:**
The response is an HTML fragment containing the updated income list table, suitable for swapping into the page via HTMX.

**Account Validation:**
- If `account_id` is 0 or not specified, the user's individual account is used
- If `account_id` is specified, the system validates the user has access to that account
- Joint accounts are supported if the user is a member

**Error Responses:**

| Status Code | Error Message | Description |
|------------|---------------|-------------|
| 400 | "Dados inválidos" | Invalid request format or data binding error |
| 400 | "Data inválida" | Date format is invalid (must be YYYY-MM-DD) |
| 403 | "Acesso negado à conta selecionada" | User doesn't have access to the specified account |
| 500 | "Conta não encontrada" | Individual account not found for user |
| 500 | "Erro ao criar recebimento" | Database error during creation |

**Error Response Format:**
- **Status Code:** As specified above
- **Content-Type:** text/plain
- **Body:** Error message string

---

### 3. Delete Income

Delete an existing income record.

**Endpoint:** `DELETE /incomes/:id`

**Authentication Required:** Yes (JWT access_token cookie)

**Rate Limited:** No

**CSRF Protection:** Yes (X-CSRF-Token header required)

**URL Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| id | int | Yes | Income record ID to delete |

**Example Request:**
```http
DELETE /incomes/123 HTTP/1.1
Host: localhost:8080
Cookie: access_token=...
X-CSRF-Token: ...
```

**Access Validation:**
The system verifies that the income record belongs to one of the user's accessible accounts before allowing deletion.

**Success Response:**
- **Status Code:** 200 OK
- **Content-Type:** text/html
- **HX-Trigger:** Updates UI via HTMX
- **Body:** Rendered `partials/income-list.html` with updated income list (record removed)

**HTMX Partial Response:**
The response is an HTML fragment containing the updated income list table with the deleted record removed.

**Error Responses:**

| Status Code | Error Message | Description |
|------------|---------------|-------------|
| 404 | "Recebimento não encontrado" | Income record not found or user doesn't have access |
| 500 | "Erro ao deletar" | Database error during deletion |

**Error Response Format:**
- **Status Code:** As specified above
- **Content-Type:** text/plain
- **Body:** Error message string

---

### 4. Calculate Income Preview

Calculate tax preview for a potential income without saving it to the database.

**Endpoint:** `GET /incomes/preview`

**Authentication Required:** Yes (JWT access_token cookie)

**Rate Limited:** No

**Request Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| amount_usd | float64 | Yes | Amount in USD |
| exchange_rate | float64 | Yes | USD to BRL exchange rate |

**Example Request:**
```http
GET /incomes/preview?amount_usd=5000.00&exchange_rate=5.20 HTTP/1.1
Host: localhost:8080
Cookie: access_token=...
```

**Tax Calculation Context:**
The preview calculation uses the user's current 12-month rolling revenue across all accessible accounts to determine the applicable tax bracket and calculate the tax amount.

**Success Response:**
- **Status Code:** 200 OK
- **Content-Type:** application/json

**Response Body:**
```json
{
  "amount_brl": 26000.00,
  "tax": 3900.00,
  "net": 22100.00,
  "effective_rate": 15.0
}
```

**Response Fields:**

| Field | Type | Description |
|-------|------|-------------|
| amount_brl | float64 | Converted amount in BRL (amount_usd × exchange_rate) |
| tax | float64 | Calculated tax amount based on current revenue |
| net | float64 | Net amount after tax deduction |
| effective_rate | float64 | Effective tax rate as a percentage |

**Zero Amount Handling:**
If either `amount_usd` or `exchange_rate` is zero or negative, the response returns all zeros:
```json
{
  "amount_brl": 0,
  "tax": 0,
  "net": 0
}
```

**Use Cases:**
- Real-time preview in income creation forms
- HTMX-powered dynamic tax calculation as user types
- "What-if" scenarios for income planning

**Error Responses:**
This endpoint does not return traditional errors. Invalid or zero amounts result in a successful response with zero values.

---

## Income Endpoint Security

All income endpoints enforce the following security measures:

### Authentication & Authorization
- JWT token validation on every request
- User can only access income records for their own accounts
- Account access validation for multi-user (joint) accounts
- Automatic fallback to individual account if no account specified

### CSRF Protection
- POST and DELETE operations require CSRF token
- Token sent via `X-CSRF-Token` header (HTMX compatible)
- GET operations (List, Preview) do not require CSRF tokens

### Data Validation
- Date format validation (YYYY-MM-DD)
- Numeric amount validation
- Account ownership verification before any operation
- Safe handling of zero/negative amounts

### Tax Calculation Integrity
- Server-side tax calculation (never trusted from client)
- Based on accurate 12-month rolling revenue
- Uses current Brazilian tax brackets
- Consistent calculation across create and preview operations

---

## Income Data Flow

### Creating Income Record
1. User submits income form → `POST /incomes`
2. Server validates account access
3. Server parses and validates date format
4. Server calculates `AmountBRL = AmountUSD × ExchangeRate`
5. Server fetches 12-month revenue for tax calculation
6. Server calculates tax using progressive brackets
7. Server creates income record with all calculated values
8. Server returns updated income list as HTMX partial
9. Client-side HTMX swaps the new list into the page

### Deleting Income Record
1. User clicks delete button → `DELETE /incomes/:id`
2. Server verifies income belongs to user's accounts
3. Server deletes the record
4. Server returns updated income list as HTMX partial
5. Client-side HTMX swaps the new list into the page

### Tax Preview Flow
1. User types in amount/exchange rate fields
2. JavaScript triggers debounced preview request
3. Client sends → `GET /incomes/preview?amount_usd=...&exchange_rate=...`
4. Server calculates tax based on current 12-month revenue
5. Server returns JSON with calculated values
6. Client updates form fields with preview values in real-time

---

## Expense Endpoints

This document section describes the expense management endpoints for the POC Finance application.

### Overview

All expense endpoints require authentication (valid JWT token). They support both fixed and variable expenses, with features for:
- Split expenses across multiple users (joint account expenses)
- Payment tracking for fixed recurring expenses
- Budget limit monitoring with notifications
- Active/inactive status toggling

Security features:
- JWT authentication required for all endpoints
- Account-level access control
- CSRF protection for state-changing operations
- Automatic notifications for split expenses and budget alerts

---

### 1. List Expenses

Retrieve all expenses (fixed and variable) for user's accessible accounts.

**Endpoint:** `GET /expenses`

**Authentication Required:** Yes

**Rate Limited:** No

**Request Parameters:** None (query parameters from URL are ignored)

**Success Response:**
- **Status Code:** 200 OK
- **Content-Type:** text/html
- **Body:** Rendered expenses.html page with expense data

**Response Data Structure:**
```go
{
  "fixedExpenses": []ExpenseWithStatus,    // Fixed expenses with payment status
  "variableExpenses": []Expense,            // Variable expenses
  "accounts": []Account,                    // User's accessible accounts
  "totalFixed": float64,                    // Sum of active fixed expenses
  "totalVariable": float64,                 // Sum of active variable expenses
  "totalPaid": float64,                     // Sum of paid fixed expenses (current month)
  "totalPending": float64,                  // Sum of unpaid fixed expenses (current month)
  "categories": []string,                   // Available expense categories
  "currentMonth": int,                      // Current month (1-12)
  "currentYear": int                        // Current year
}
```

**ExpenseWithStatus Structure:**
- All fields from Expense model
- `IsPaid`: boolean indicating if expense is paid for current month

**Expense Categories:**
- Moradia (Housing)
- Alimentação (Food)
- Transporte (Transportation)
- Saúde (Health)
- Educação (Education)
- Lazer (Leisure)
- Serviços (Services)
- Impostos (Taxes)
- Outros (Others)

**Example Request:**
```http
GET /expenses HTTP/1.1
Host: localhost:8080
Cookie: access_token=...
```

**Error Responses:**

| Status Code | Description |
|------------|-------------|
| 401 | Authentication required (no valid token) |

---

### 2. Create Expense

Create a new expense (fixed or variable) with optional split configuration.

**Endpoint:** `POST /expenses`

**Authentication Required:** Yes

**Rate Limited:** No

**CSRF Protection:** Required

**Request Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| account_id | uint | No | Target account ID (defaults to user's individual account) |
| name | string | Yes | Expense name/description |
| amount | float64 | Yes | Expense amount |
| type | string | Yes | Expense type: "fixed" or "variable" |
| due_day | int | Conditional | Day of month expense is due (1-31, required for fixed expenses) |
| category | string | Yes | Expense category (see categories list above) |
| is_split | bool | No | Whether this is a split expense (default: false) |
| split_user_ids | []uint | Conditional | User IDs for split (required if is_split=true) |
| split_percentages | []float64 | Conditional | Split percentages (required if is_split=true, must sum to 100) |

**Content-Type:** `application/x-www-form-urlencoded`

**Split Expense Rules:**
- Only available for joint accounts
- User IDs must be members of the selected account
- Percentages must sum to exactly 100% (tolerance: ±0.01)
- Each split amount is calculated as: `expense.amount × percentage / 100`
- All group members receive notifications for split expenses

**Example Request (Simple Fixed Expense):**
```http
POST /expenses HTTP/1.1
Host: localhost:8080
Content-Type: application/x-www-form-urlencoded
Cookie: access_token=...
X-CSRF-Token: ...

account_id=1&name=Aluguel&amount=2000.00&type=fixed&due_day=5&category=Moradia
```

**Example Request (Split Variable Expense):**
```http
POST /expenses HTTP/1.1
Host: localhost:8080
Content-Type: application/x-www-form-urlencoded
Cookie: access_token=...
X-CSRF-Token: ...

account_id=2&name=Supermercado&amount=500.00&type=variable&category=Alimentação&is_split=true&split_user_ids=1&split_user_ids=2&split_percentages=60&split_percentages=40
```

**Success Response:**
- **Status Code:** 200 OK
- **Content-Type:** text/html
- **Body:** Rendered partial template with updated expense list
  - Fixed expenses: `partials/fixed-expense-list.html`
  - Variable expenses: `partials/variable-expense-list.html`

**Side Effects:**
- Creates expense record in database
- Creates split records if `is_split=true`
- Sends notifications to group members (joint accounts only)
- Checks budget limit and sends alert if exceeded

**Error Responses:**

| Error Message | Status Code | Description |
|--------------|-------------|-------------|
| "Dados inválidos" | 400 | Invalid request format or data binding error |
| "Conta não encontrada" | 500 | User's individual account not found (fallback failed) |
| "Acesso negado à conta selecionada" | 403 | User doesn't have access to specified account |
| "A soma dos percentuais deve ser 100%" | 400 | Split percentages don't sum to 100% |
| "Erro ao criar despesa" | 500 | Database error creating expense |
| "Erro ao criar divisão" | 500 | Database error creating split record |

---

### 3. Toggle Expense Status

Toggle the active/inactive status of an expense.

**Endpoint:** `POST /expenses/:id/toggle`

**Authentication Required:** Yes

**Rate Limited:** No

**CSRF Protection:** Required

**URL Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| id | uint | Yes | Expense ID to toggle |

**Request Parameters:** None

**Example Request:**
```http
POST /expenses/123/toggle HTTP/1.1
Host: localhost:8080
Cookie: access_token=...
X-CSRF-Token: ...
```

**Success Response:**
- **Status Code:** 200 OK
- **Content-Type:** text/html
- **Body:** Rendered partial template with updated expense list
- **Side Effects:** Toggles `expense.active` field (true ↔ false)

**Error Responses:**

| Error Message | Status Code | Description |
|--------------|-------------|-------------|
| "Despesa não encontrada" | 404 | Expense not found or user doesn't have access |

---

### 4. Mark Expense as Paid

Mark a fixed expense as paid for the current month.

**Endpoint:** `POST /expenses/:id/paid`

**Authentication Required:** Yes

**Rate Limited:** No

**CSRF Protection:** Required

**URL Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| id | uint | Yes | Expense ID to mark as paid |

**Request Parameters:** None

**Example Request:**
```http
POST /expenses/123/paid HTTP/1.1
Host: localhost:8080
Cookie: access_token=...
X-CSRF-Token: ...
```

**Success Response:**
- **Status Code:** 200 OK
- **Content-Type:** text/html
- **Body:** Rendered partial template with updated expense list

**Payment Record Details:**
- Month: Current month (1-12)
- Year: Current year
- PaidAt: Current timestamp
- Amount: Expense amount at time of payment

**Side Effects:**
- Creates `ExpensePayment` record for current month/year
- If payment already exists for current month/year, no duplicate is created

**Error Responses:**

| Error Message | Status Code | Description |
|--------------|-------------|-------------|
| "Despesa não encontrada" | 404 | Expense not found or user doesn't have access |

**Notes:**
- Idempotent operation (safe to call multiple times)
- Only creates one payment record per expense per month/year
- Payment tracking is independent for each month

---

### 5. Mark Expense as Unpaid

Remove the payment record for an expense in the current month.

**Endpoint:** `POST /expenses/:id/unpaid`

**Authentication Required:** Yes

**Rate Limited:** No

**CSRF Protection:** Required

**URL Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| id | uint | Yes | Expense ID to mark as unpaid |

**Request Parameters:** None

**Example Request:**
```http
POST /expenses/123/unpaid HTTP/1.1
Host: localhost:8080
Cookie: access_token=...
X-CSRF-Token: ...
```

**Success Response:**
- **Status Code:** 200 OK
- **Content-Type:** text/html
- **Body:** Rendered partial template with updated expense list

**Side Effects:**
- Deletes `ExpensePayment` record for current month/year
- If no payment record exists, operation succeeds silently

**Error Responses:**

| Error Message | Status Code | Description |
|--------------|-------------|-------------|
| "Despesa não encontrada" | 404 | Expense not found or user doesn't have access |

**Notes:**
- Idempotent operation (safe to call multiple times)
- Only affects payment for current month/year
- Does not delete the expense itself

---

### 6. Delete Expense

Delete an expense and all associated records.

**Endpoint:** `DELETE /expenses/:id`

**Authentication Required:** Yes

**Rate Limited:** No

**CSRF Protection:** Required

**URL Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| id | uint | Yes | Expense ID to delete |

**Request Parameters:** None

**Example Request:**
```http
DELETE /expenses/123 HTTP/1.1
Host: localhost:8080
Cookie: access_token=...
X-CSRF-Token: ...
```

**Success Response:**
- **Status Code:** 200 OK
- **Content-Type:** text/html
- **Body:** Rendered partial template with updated expense list

**Side Effects:**
- Deletes expense record
- Cascading deletes:
  - All associated `ExpensePayment` records
  - All associated `ExpenseSplit` records (if split expense)

**Error Responses:**

| Error Message | Status Code | Description |
|--------------|-------------|-------------|
| "Despesa não encontrada" | 404 | Expense not found or user doesn't have access |

**Notes:**
- Permanent deletion (cannot be undone)
- Removes all historical payment tracking for this expense
- Removes all split configurations

---

### 7. Get Account Members

Retrieve members of an account for split expense configuration.

**Endpoint:** `GET /accounts/:accountId/members`

**Authentication Required:** Yes

**Rate Limited:** No

**CSRF Protection:** Not required (read-only operation)

**URL Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| accountId | uint | Yes | Account ID to get members for |

**Request Parameters:** None

**Example Request:**
```http
GET /accounts/2/members HTTP/1.1
Host: localhost:8080
Cookie: access_token=...
```

**Success Response:**
- **Status Code:** 200 OK
- **Content-Type:** text/html
- **Body:** Rendered `partials/split-members.html` with member data

**Response Data Structure:**
```go
{
  "members": []User,           // Account member list
  "account": Account,          // Account details
  "isJoint": bool              // Whether account is a joint account
}
```

**Error Responses:**

| Error Message | Status Code | Description |
|--------------|-------------|-------------|
| "Acesso negado" | 403 | User doesn't have access to specified account |
| "Conta não encontrada" | 404 | Account ID not found |
| "Erro ao buscar membros" | 500 | Database error retrieving members |

**Notes:**
- Used by frontend to populate split expense form
- Only returns members for joint accounts
- Individual accounts will return single member (account owner)

---

## Expense Endpoint Security

All expense endpoints enforce the following security measures:

### Authentication & Authorization
- JWT token validation on every request
- User can only access expenses for their own accounts
- Account access validation for multi-user (joint) accounts
- Automatic fallback to individual account if no account specified

### CSRF Protection
- POST and DELETE operations require CSRF token
- Token sent via `X-CSRF-Token` header (HTMX compatible)
- GET operations (List, GetAccountMembers) do not require CSRF tokens

### Data Validation
- Account ownership verification before any operation
- Split percentage validation (must sum to 100%)
- User membership verification for split expenses
- Safe handling of zero/negative amounts

### Notifications
- Automatic notifications for split expenses (joint accounts)
- Budget alert notifications when limit reached/exceeded
- Notifications sent to all relevant account members

### Split Expense Rules
- Only available for joint accounts
- All split users must be account members
- Percentages validated server-side
- Split amounts calculated server-side (never trusted from client)

---

## Expense Data Flow

### Creating Fixed Expense
1. User submits expense form → `POST /expenses`
2. Server validates account access
3. Server determines expense type (fixed vs variable)
4. Server creates expense record
5. If split expense:
   - Server validates split users are account members
   - Server validates percentages sum to 100%
   - Server creates split records
6. Server notifies group members (if joint account)
7. Server checks budget limit and sends alert if exceeded
8. Server returns updated expense list as HTMX partial
9. Client-side HTMX swaps the new list into the page

### Marking Expense as Paid
1. User clicks "Mark Paid" button → `POST /expenses/:id/paid`
2. Server verifies expense belongs to user's accounts
3. Server creates payment record for current month/year
4. Server returns updated expense list showing paid status
5. Client-side HTMX updates the UI

### Deleting Expense
1. User clicks delete button → `DELETE /expenses/:id`
2. Server verifies expense belongs to user's accounts
3. Server deletes expense (cascades to payments and splits)
4. Server returns updated expense list as HTMX partial
5. Client-side HTMX swaps the new list into the page

### Split Expense Configuration
1. User selects joint account in expense form
2. JavaScript triggers → `GET /accounts/:accountId/members`
3. Server returns member list HTML
4. Client-side HTMX injects split member inputs
5. User configures split percentages
6. User submits form with split data
7. Server validates and creates expense with splits

---

## Credit Card Endpoints

This document describes the credit card management endpoints for the POC Finance application.

### Overview

Credit card endpoints enable users to manage credit cards and installment purchases. All endpoints require authentication and are protected by CSRF validation. The system supports:

- Multiple credit cards per user
- Installment tracking with automatic monthly calculations
- Card billing cycle management (closing day and due day)
- Credit limit tracking
- Category-based installment organization

**Authentication Required:** Yes (all endpoints)
**CSRF Protection:** Yes (all POST/DELETE endpoints)
**Account Access:** Users can only access cards associated with their individual or joint accounts

---

### 1. List Credit Cards and Installments

Retrieve all credit cards and active installments for the authenticated user's accounts.

**Endpoint:** `GET /cards`

**Authentication Required:** Yes

**Rate Limited:** No

**Request Parameters:** None

**Example Request:**
```http
GET /cards HTTP/1.1
Host: localhost:8080
Cookie: access_token=...; refresh_token=...
```

**Success Response:**
- **Status Code:** 200 OK
- **Content-Type:** text/html
- **Body:** Rendered cards.html page with credit cards, installments, and totals
- **Description:** Returns the credit cards page with all user's cards and active installments

**Response Data:**
The page includes:
- List of all credit cards with their details (name, limit, closing day, due day)
- Current month's total for each card (sum of active installments)
- List of active installments with current installment number
- Available expense categories for categorization

**Installment Filtering:**
Only displays installments that are still active in the current month:
- Calculates months passed since installment start date
- Shows installments where `monthsPassed < totalInstallments`
- Displays current installment number (monthsPassed + 1)

**Error Responses:**
None - authenticated users always receive a valid page (may be empty if no cards exist)

---

### 2. Create Credit Card

Create a new credit card for the authenticated user's individual account.

**Endpoint:** `POST /cards`

**Authentication Required:** Yes

**CSRF Protection:** Yes

**Request Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| name | string | Yes | Display name for the credit card (e.g., "Visa Gold", "Mastercard Black") |
| closing_day | integer | Yes | Day of month when billing cycle closes (1-31) |
| due_day | integer | Yes | Day of month when payment is due (1-31) |
| limit_amount | float | Yes | Credit limit amount (positive number) |

**Content-Type:** `application/x-www-form-urlencoded`

**Example Request:**
```http
POST /cards HTTP/1.1
Host: localhost:8080
Content-Type: application/x-www-form-urlencoded
Cookie: access_token=...; refresh_token=...
X-CSRF-Token: ...

name=Visa+Gold&closing_day=15&due_day=25&limit_amount=5000.00
```

**Success Response:**
- **Status Code:** 200 OK
- **Content-Type:** text/html
- **Body:** HTMX partial - updated card list (partials/card-list.html)
- **Description:** Creates the card and returns updated card list for HTMX to swap into the page

**Card Association:**
- Card is automatically associated with the user's individual account
- Account is retrieved using the authenticated user's ID

**Error Responses:**

| Error Message | Description |
|--------------|-------------|
| "Dados inválidos" | Invalid request format or data binding error |
| "Conta não encontrada" | User's individual account not found |
| "Erro ao criar cartão" | Database error during card creation |

**Error Response Format:**
- **Status Code:** 400/500
- **Content-Type:** text/plain
- **Body:** Error message string

---

### 3. Delete Credit Card

Delete an existing credit card and all its associated installments.

**Endpoint:** `DELETE /cards/:id`

**Authentication Required:** Yes

**CSRF Protection:** Yes

**URL Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| id | integer | Yes | Credit card ID to delete |

**Example Request:**
```http
DELETE /cards/123 HTTP/1.1
Host: localhost:8080
Cookie: access_token=...; refresh_token=...
X-CSRF-Token: ...
```

**Success Response:**
- **Status Code:** 200 OK
- **Content-Type:** text/html
- **Body:** HTMX partial - updated card list (partials/card-list.html)
- **Description:** Deletes the card and all installments, returns updated card list

**Authorization Check:**
- Verifies card belongs to one of the user's accounts before deletion
- Returns 404 if card not found or doesn't belong to user

**Cascade Deletion:**
All installments associated with the card are automatically deleted.

**Error Responses:**

| Error Message | Description |
|--------------|-------------|
| "Cartão não encontrado" | Card ID not found or doesn't belong to user's accounts |

**Error Response Format:**
- **Status Code:** 404
- **Content-Type:** text/plain
- **Body:** Error message string

---

### 4. Create Installment

Create a new installment purchase on an existing credit card.

**Endpoint:** `POST /installments`

**Authentication Required:** Yes

**CSRF Protection:** Yes

**Request Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| credit_card_id | integer | Yes | ID of the credit card for this purchase |
| description | string | Yes | Description of the purchase |
| total_amount | float | Yes | Total purchase amount (will be divided by installments) |
| total_installments | integer | Yes | Number of monthly installments (e.g., 12) |
| start_date | string | Yes | First installment date (format: YYYY-MM-DD) |
| category | string | Yes | Expense category (e.g., "Alimentação", "Transporte") |

**Content-Type:** `application/x-www-form-urlencoded`

**Example Request:**
```http
POST /installments HTTP/1.1
Host: localhost:8080
Content-Type: application/x-www-form-urlencoded
Cookie: access_token=...; refresh_token=...
X-CSRF-Token: ...

credit_card_id=123&description=Notebook+Dell&total_amount=3600.00&total_installments=12&start_date=2024-01-15&category=Eletrônicos
```

**Success Response:**
- **Status Code:** 200 OK
- **Content-Type:** text/html
- **Body:** HTMX partial - updated installment list (partials/installment-list.html)
- **Description:** Creates the installment and returns updated installment list

**Installment Calculation:**
- `installment_amount = total_amount / total_installments`
- Example: $3,600 / 12 = $300 per month
- `current_installment` starts at 1

**Authorization Check:**
- Verifies credit card belongs to one of the user's accounts
- Returns 404 if card not found or doesn't belong to user

**Error Responses:**

| Error Message | Description |
|--------------|-------------|
| "Dados inválidos" | Invalid request format or data binding error |
| "Cartão não encontrado" | Credit card ID not found or doesn't belong to user's accounts |
| "Data inválida" | start_date is not in YYYY-MM-DD format |
| "Erro ao criar parcelamento" | Database error during installment creation |

**Error Response Format:**
- **Status Code:** 400/404/500
- **Content-Type:** text/plain
- **Body:** Error message string

---

### 5. Delete Installment

Delete an existing installment purchase from a credit card.

**Endpoint:** `DELETE /installments/:id`

**Authentication Required:** Yes

**CSRF Protection:** Yes

**URL Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| id | integer | Yes | Installment ID to delete |

**Example Request:**
```http
DELETE /installments/456 HTTP/1.1
Host: localhost:8080
Cookie: access_token=...; refresh_token=...
X-CSRF-Token: ...
```

**Success Response:**
- **Status Code:** 200 OK
- **Content-Type:** text/html
- **Body:** HTMX partial - updated installment list (partials/installment-list.html)
- **Description:** Deletes the installment and returns updated installment list

**Authorization Check:**
- Verifies installment's credit card belongs to one of the user's accounts
- Returns 404 if installment not found or doesn't belong to user

**Error Responses:**

| Error Message | Description |
|--------------|-------------|
| "Parcela não encontrada" | Installment ID not found or card doesn't belong to user's accounts |

**Error Response Format:**
- **Status Code:** 404
- **Content-Type:** text/plain
- **Body:** Error message string

---

## Credit Card Endpoint Security

### Authentication & Authorization

All credit card endpoints require valid JWT authentication via cookie. Authorization is enforced at two levels:
1. **User Level:** User must be authenticated
2. **Account Level:** Cards and installments are filtered to only those associated with user's accounts (individual or joint)

The `AccountService.GetUserAccountIDs()` method is used to retrieve all account IDs the user has access to, ensuring proper data isolation.

### CSRF Protection

All mutating operations (POST, DELETE) are protected by CSRF tokens. Tokens must be included in the `X-CSRF-Token` header for HTMX requests.

### Data Validation

- **Billing Days:** closing_day and due_day should be valid day numbers (1-31)
- **Amounts:** limit_amount and total_amount must be positive numbers
- **Date Format:** start_date must be in YYYY-MM-DD format
- **Installments:** total_installments must be a positive integer

### Card Ownership

Before any delete or create installment operation:
1. Server verifies the card exists
2. Server verifies the card's account_id is in the user's accessible accounts
3. Operation is rejected if verification fails

---

## Credit Card Data Flow

### Creating Credit Card
1. User submits card form via HTMX POST request
2. Server validates user has an individual account
3. Server creates credit card record with account association
4. Server queries all user's cards
5. Server renders updated card list HTML partial
6. Client-side HTMX swaps the new list into the page

### Creating Installment Purchase
1. User submits installment form via HTMX POST request
2. Server verifies credit card belongs to user
3. Server calculates installment_amount (total_amount / total_installments)
4. Server creates installment record
5. Server queries all active installments (filters by current month)
6. Server renders updated installment list HTML partial
7. Client-side HTMX swaps the new list into the page

### Monthly Installment Calculation
The system calculates which installments are active for the current month:
1. For each installment, calculate months between start_date and current date
2. If `monthsPassed < totalInstallments`, the installment is still active
3. Current installment number = `monthsPassed + 1`
4. Sum all active installment amounts to get card's monthly total

**Example:**
- Purchase: $1,200 in 12 installments starting Jan 2024
- Monthly amount: $100
- In March 2024: monthsPassed = 2, currentInstallment = 3, still active
- In January 2025: monthsPassed = 12, installment complete (not shown)

### Deleting Credit Card
1. User clicks delete button → HTMX DELETE request
2. Server verifies card ownership
3. Server deletes all associated installments (cascade)
4. Server deletes credit card record
5. Server returns updated card list as HTMX partial
6. Client-side HTMX swaps the new list into the page

---

## Settings Endpoints

This section describes endpoints for managing application settings such as pro-labore amounts, INSS rates, and tax calculations.

### Overview

Settings endpoints allow users to view and update their financial configuration. Settings are cached for performance and the cache is invalidated on updates. All settings endpoints require authentication.

---

### 1. Get Settings

Retrieve current application settings including pro-labore amount, INSS ceiling, INSS rate, and calculated INSS amount.

**Endpoint:** `GET /settings`

**Authentication Required:** Yes

**CSRF Protection:** No (read-only operation)

**Request Parameters:** None

**Example Request:**
```http
GET /settings HTTP/1.1
Host: localhost:8080
Cookie: access_token=...; refresh_token=...
```

**Success Response:**
- **Status Code:** 200 OK
- **Content-Type:** text/html
- **Body:** Rendered settings page (settings.html)
- **Description:** Returns settings page with current configuration values

**Response Data Structure:**
```go
{
  "settings": {
    "pro_labore": 15000.00,      // Monthly pro-labore amount
    "inss_ceiling": 7507.49,     // INSS contribution ceiling
    "inss_rate": 0.11,           // INSS rate (11%)
    "inss_amount": 825.82        // Calculated INSS amount (read-only)
  }
}
```

**Settings Cache:**
- Settings are cached using the SettingsCacheService
- Cache is loaded from database on first request
- Cache remains valid until explicitly invalidated

**Error Responses:**

| Status Code | Description |
|-------------|-------------|
| 401 | Unauthorized - Missing or invalid authentication token |

---

### 2. Update Settings

Update application settings and invalidate the settings cache.

**Endpoint:** `POST /settings`

**Authentication Required:** Yes

**CSRF Protection:** Yes

**Request Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| pro_labore | float | Yes | Monthly pro-labore amount in BRL |
| inss_ceiling | float | Yes | Maximum INSS contribution ceiling in BRL |
| inss_rate | float | Yes | INSS contribution rate (0.11 = 11%) |

**Content-Type:** `application/x-www-form-urlencoded`

**Example Request:**
```http
POST /settings HTTP/1.1
Host: localhost:8080
Content-Type: application/x-www-form-urlencoded
Cookie: access_token=...; refresh_token=...
X-CSRF-Token: ...

pro_labore=15000.00&inss_ceiling=7507.49&inss_rate=0.11
```

**Success Response:**
- **Status Code:** 200 OK
- **Content-Type:** text/html
- **Body:** HTMX partial - updated settings form (partials/settings-form.html)
- **Description:** Updates settings in database, invalidates cache, and returns form with updated values and success indicator

**Response Data:**
```go
{
  "settings": {
    "pro_labore": 15000.00,
    "inss_ceiling": 7507.49,
    "inss_rate": 0.11,
    "inss_amount": 825.82  // Recalculated based on new values
  },
  "saved": true  // Success indicator for UI feedback
}
```

**Settings Persistence:**
Each setting is stored as a key-value pair in the database:
- Key: `models.SettingProLabore`, `models.SettingINSSCeiling`, `models.SettingINSSRate`
- Value: String representation of the float value
- Settings are created if they don't exist, updated if they do

**Cache Invalidation:**
After successful update, the SettingsCacheService cache is invalidated, forcing a fresh database load on the next request.

**Error Responses:**

| Status Code | Description |
|-------------|-------------|
| 401 | Unauthorized - Missing or invalid authentication token |
| 403 | Forbidden - Missing or invalid CSRF token |
| 400 | Bad Request - Invalid parameter format or values |

**Error Response Format:**
- **Content-Type:** text/plain or text/html
- **Body:** Error message string or rendered error partial

---

## Export Endpoints

This section describes endpoints for exporting financial data to Excel format.

### Overview

Export endpoints allow users to download their financial data as Excel spreadsheets. The export includes multiple sheets with summaries, incomes, expenses, and credit card installments for a specified year.

---

### 1. Export Year Data

Export all financial data for a specific year to an Excel file with multiple sheets.

**Endpoint:** `GET /export`

**Authentication Required:** Yes

**CSRF Protection:** No (read-only operation)

**Request Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| year | integer | No | Year to export (defaults to current year if not provided) |

**Example Request:**
```http
GET /export?year=2024 HTTP/1.1
Host: localhost:8080
Cookie: access_token=...; refresh_token=...
```

**Success Response:**
- **Status Code:** 200 OK
- **Content-Type:** application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
- **Content-Disposition:** attachment; filename=financeiro_2024.xlsx
- **Body:** Binary Excel file (.xlsx format)
- **Description:** Generates and returns Excel file with financial data

**Excel File Structure:**

The exported Excel file contains 4 sheets:

#### Sheet 1: Resumo Mensal (Monthly Summary)
Columns:
- Mês (Month) - Month name in Portuguese
- Receita Bruta (Gross Income) - Total gross income for the month
- Imposto (Tax) - Total tax amount for the month
- Receita Líquida (Net Income) - Total net income (gross - tax)
- Despesas Fixas (Fixed Expenses) - Total fixed expenses
- Despesas Variáveis (Variable Expenses) - Total variable expenses
- Cartões (Cards) - Total credit card installments
- Contas (Bills) - Total bills
- Total Despesas (Total Expenses) - Sum of all expenses
- Saldo (Balance) - Net income minus total expenses

**Styling:**
- Header row with bold text, blue background (#4472C4), centered alignment
- Column width: 15 units
- Data for all 12 months of the specified year

#### Sheet 2: Recebimentos (Incomes)
Columns:
- Data (Date) - Income date in DD/MM/YYYY format
- Valor USD (USD Amount) - Amount in US Dollars
- Taxa Câmbio (Exchange Rate) - USD to BRL exchange rate
- Valor BRL (BRL Amount) - Amount in Brazilian Reais
- Imposto (Tax) - Tax amount
- Líquido (Net) - Net amount after tax
- Descrição (Description) - Income description

**Styling:**
- Header row with bold text, green background (#70AD47)
- Column width: 15 units
- Sorted by date ascending
- Only includes incomes within the specified year (Jan 1 to Dec 31)

#### Sheet 3: Despesas (Expenses)
Columns:
- Nome (Name) - Expense name
- Valor (Amount) - Expense amount
- Tipo (Type) - "Fixa" (Fixed) or "Variável" (Variable)
- Dia Venc. (Due Day) - Due day of month
- Categoria (Category) - Expense category
- Ativa (Active) - "Sim" (Yes) or "Não" (No)

**Styling:**
- Header row with bold text, orange background (#ED7D31)
- Column width: 15 units
- Sorted by type, then name
- Includes all expenses (not filtered by year)

#### Sheet 4: Parcelamentos (Installments)
Columns:
- Cartão (Card) - Credit card name
- Descrição (Description) - Purchase description
- Valor Total (Total Amount) - Total purchase amount
- Parcela (Installment Amount) - Monthly installment amount
- Total Parcelas (Total Installments) - Number of installments
- Parcela Atual (Current Installment) - Current installment number
- Início (Start) - First installment date in DD/MM/YYYY
- Categoria (Category) - Purchase category

**Styling:**
- Header row with bold text, blue background (#5B9BD5)
- Column width: 15 units
- Only includes active installments (not yet completed)
- Filters out installments where all payments are complete

**Data Sources:**
- Monthly summaries: Generated by `services.GetYearlySummaries()`
- Incomes: Filtered by year from `models.Income` table
- Expenses: All records from `models.Expense` table
- Installments: Active records from `models.Installment` table with preloaded credit card data

**File Generation:**
Uses the `github.com/xuri/excelize/v2` library for Excel file creation and manipulation.

**Error Responses:**

| Status Code | Description |
|-------------|-------------|
| 401 | Unauthorized - Missing or invalid authentication token |
| 500 | Internal Server Error - Error generating Excel file |

**Error Response Format:**
- **Content-Type:** text/plain
- **Body:** Error message string

**Notes:**
- If year parameter is invalid or missing, defaults to current year
- The default "Sheet1" created by excelize is automatically deleted
- File is generated in-memory and streamed directly to the response
- All monetary values are formatted as numbers (not strings) for Excel calculations

---

## Settings Endpoint Security

### Authentication & Authorization

All settings endpoints require valid JWT authentication via cookie. Settings are global per user account.

### CSRF Protection

Settings update operations (POST) are protected by CSRF tokens. Tokens must be included in the `X-CSRF-Token` header for HTMX requests.

### Data Validation

- **Amounts:** pro_labore and inss_ceiling should be positive numbers
- **Rates:** inss_rate should be between 0 and 1 (e.g., 0.11 for 11%)

### Settings Cache

The application uses a `SettingsCacheService` to cache settings data:
1. Settings are loaded from database on first access
2. Cache remains valid until explicitly invalidated
3. Cache is invalidated after successful updates
4. This reduces database queries for frequently accessed settings

---

## Export Endpoint Security

### Authentication & Authorization

Export endpoints require valid JWT authentication via cookie. Users can only export their own financial data.

### Data Filtering

- Incomes are filtered to the specified year (Jan 1 to Dec 31)
- Monthly summaries are calculated for all 12 months of the year
- Expenses include all records (not year-filtered as they are recurring)
- Installments are filtered to show only currently active purchases

### Performance Considerations

- Large datasets may take time to generate
- Excel file is generated synchronously and streamed to response
- Consider adding year range validation to prevent excessive data exports

---

## Group Management Endpoints

### Overview

Group management endpoints allow users to create and manage family groups for shared financial tracking. Groups support multiple members with role-based permissions (admin/member) and invite-based joining system.

**Authentication:**
- Most endpoints require JWT authentication via cookie
- Public endpoints: invite viewing and registration
- Admin-only endpoints: invite management, member removal, group deletion

**Roles:**
- **Admin:** Can invite members, remove members, delete group, manage invites
- **Member:** Can view group data, leave group

---

### 1. List Groups

Get all groups the authenticated user is a member of.

**Endpoint:** `GET /groups`

**Authentication Required:** Yes

**Request Parameters:** None

**Success Response:**
- **Status Code:** 200 OK
- **Content-Type:** text/html
- **Body:** Rendered groups.html page with user's groups, members, and joint accounts

**Response Data Includes:**
- List of groups with members
- User's role in each group
- Joint accounts for each group

---

### 2. Create Group

Create a new family group. The creator is automatically added as an admin member.

**Endpoint:** `POST /groups`

**Authentication Required:** Yes

**Request Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| name | string | Yes | Name of the group |
| description | string | No | Optional group description |

**Content-Type:** `application/x-www-form-urlencoded`

**Example Request:**
```http
POST /groups HTTP/1.1
Host: localhost:8080
Content-Type: application/x-www-form-urlencoded
Cookie: access_token=...

name=Family+Budget&description=Our+family+shared+expenses
```

**Success Response:**
- **Status Code:** 200 OK
- **Content-Type:** text/html
- **Body:** Updated group list partial (partials/group-list.html)

**Error Responses:**

| Error Message | Description |
|--------------|-------------|
| "Dados inválidos" | Invalid request format or data binding error |
| "Nome do grupo é obrigatório" | Group name is missing |
| "Erro ao criar grupo" | Server error during group creation |
| "Erro ao adicionar membro" | Server error when adding creator as admin |

---

### 3. Leave Group

Leave a group you're a member of. Last admin cannot leave unless there are no other members.

**Endpoint:** `POST /groups/:id/leave`

**Authentication Required:** Yes

**URL Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| id | integer | Group ID |

**Example Request:**
```http
POST /groups/123/leave HTTP/1.1
Host: localhost:8080
Cookie: access_token=...
```

**Success Response:**
- **Status Code:** 200 OK
- **Content-Type:** text/html
- **Body:** Updated group list partial

**Error Responses:**

| Error Message | Description |
|--------------|-------------|
| "ID do grupo inválido" | Invalid group ID parameter |
| "Você não é membro deste grupo" | User is not a member of the group |
| "Você é o único administrador e não pode sair do grupo" | Last admin cannot leave the group |
| "Erro ao sair do grupo" | Server error during leave operation |

---

### 4. Delete Group

Delete a group permanently. Only group admins can delete groups.

**Endpoint:** `DELETE /groups/:id`

**Authentication Required:** Yes (Admin only)

**URL Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| id | integer | Group ID |

**Example Request:**
```http
DELETE /groups/123 HTTP/1.1
Host: localhost:8080
Cookie: access_token=...
```

**Success Response:**
- **Status Code:** 200 OK
- **Content-Type:** text/html
- **Body:** Updated group list partial

**Error Responses:**

| Error Message | Description |
|--------------|-------------|
| "ID do grupo inválido" | Invalid group ID parameter |
| "Apenas administradores podem excluir o grupo" | User is not a group admin |
| "Erro ao excluir grupo" | Server error during deletion |

---

### 5. Remove Member

Remove a member from a group. Only group admins can remove members.

**Endpoint:** `DELETE /groups/:id/members/:userId`

**Authentication Required:** Yes (Admin only)

**URL Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| id | integer | Group ID |
| userId | integer | User ID of the member to remove |

**Example Request:**
```http
DELETE /groups/123/members/456 HTTP/1.1
Host: localhost:8080
Cookie: access_token=...
```

**Success Response:**
- **Status Code:** 200 OK
- **Content-Type:** text/html
- **Body:** Updated group list partial

**Error Responses:**

| Error Message | Description |
|--------------|-------------|
| "ID do grupo inválido" | Invalid group ID parameter |
| "ID do membro inválido" | Invalid user ID parameter |
| "Apenas administradores podem remover membros" | User is not a group admin |
| "Membro não encontrado" | Member is not part of the group |
| "Erro ao remover membro" | Server error during removal |

---

### 6. Generate Invite Code

Generate a new invite code for a group. Only group admins can generate invites.

**Endpoint:** `POST /groups/:id/invite`

**Authentication Required:** Yes (Admin only)

**URL Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| id | integer | Group ID |

**Example Request:**
```http
POST /groups/123/invite HTTP/1.1
Host: localhost:8080
Cookie: access_token=...
```

**Success Response:**
- **Status Code:** 200 OK
- **Content-Type:** text/html
- **Body:** Rendered invite modal partial with invite code and link

**Response Includes:**
- Invite code
- Full invite URL (e.g., http://localhost:8080/groups/join/ABC123)
- Expiration date (30 days from creation)
- Maximum usage count (unlimited by default)

**Error Responses:**

| Error Message | Description |
|--------------|-------------|
| "ID do grupo inválido" | Invalid group ID parameter |
| "Você não é administrador deste grupo" | User is not a group admin |
| "Erro ao gerar convite" | Server error during invite generation |

---

### 7. List Group Invites

Get all active invites for a group. Only group admins can view invites.

**Endpoint:** `GET /groups/:id/invites`

**Authentication Required:** Yes (Admin only)

**URL Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| id | integer | Group ID |

**Example Request:**
```http
GET /groups/123/invites HTTP/1.1
Host: localhost:8080
Cookie: access_token=...
```

**Success Response:**
- **Status Code:** 200 OK
- **Content-Type:** text/html
- **Body:** Rendered invite list partial with active invites

**Response Includes:**
- All active invites for the group
- Invite codes and full URLs
- Creation dates and expiration dates
- Usage counts
- Creator information

**Error Responses:**

| Error Message | Description |
|--------------|-------------|
| "ID do grupo inválido" | Invalid group ID parameter |
| "Você não é administrador deste grupo" | User is not a group admin |
| "Erro ao buscar convites" | Server error retrieving invites |

---

### 8. View Invite Page (Public)

View details about a group invite. This is a public endpoint that shows invite information to non-authenticated users.

**Endpoint:** `GET /groups/join/:code`

**Authentication Required:** No

**URL Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| code | string | Invite code |

**Example Request:**
```http
GET /groups/join/ABC123XYZ HTTP/1.1
Host: localhost:8080
```

**Success Response:**
- **Status Code:** 200 OK
- **Content-Type:** text/html
- **Body:** Rendered join-group.html page with invite details

**Response Includes:**
- Group name and description
- Inviter information
- Options to login or register

**Behavior:**
- If user is already logged in, checks if already a member
- If already a member, shows appropriate message
- If not logged in, shows login/register options
- If logged in but not a member, allows accepting invite

**Error Responses:**

| Error Message | Description |
|--------------|-------------|
| "Convite inválido ou expirado" | Invite code is invalid, expired, or reached max uses |

---

### 9. Accept Invite

Accept a group invite and join the group. Requires authentication.

**Endpoint:** `POST /groups/join/:code`

**Authentication Required:** Yes

**URL Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| code | string | Invite code |

**Example Request:**
```http
POST /groups/join/ABC123XYZ HTTP/1.1
Host: localhost:8080
Cookie: access_token=...
```

**Success Response:**
- **Status Code:** 200 OK
- **Content-Type:** text/html
- **Body:** Rendered join-group.html page with success message

**Side Effects:**
- User is added as a member of the group with "member" role
- User receives a notification about joining the group
- Invite usage count is incremented

**Error Responses:**

| Error Message | Description |
|--------------|-------------|
| "Convite não encontrado" | Invite code does not exist |
| "Convite expirado" | Invite has passed expiration date |
| "Convite inválido" | Invite is not valid (revoked or deleted) |
| "Convite atingiu o limite de usos" | Invite reached maximum usage count |
| "Você já é membro deste grupo" | User is already a member |
| "Erro ao aceitar convite" | Server error during acceptance |

---

### 10. Revoke Invite

Revoke an invite code, making it invalid. Only group admins can revoke invites.

**Endpoint:** `DELETE /groups/invites/:id`

**Authentication Required:** Yes (Admin only)

**URL Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| id | integer | Invite ID |

**Example Request:**
```http
DELETE /groups/invites/789 HTTP/1.1
Host: localhost:8080
Cookie: access_token=...
```

**Success Response:**
- **Status Code:** 200 OK
- **Body:** Empty response

**Side Effects:**
- Invite is soft-deleted (deleted_at timestamp set)
- Invite code can no longer be used to join the group

**Error Responses:**

| Error Message | Description |
|--------------|-------------|
| "ID do convite inválido" | Invalid invite ID parameter |
| "Você não é administrador deste grupo" | User is not a group admin |
| "Erro ao revogar convite" | Server error during revocation |

---

### 11. Register and Join

Register a new user account and immediately join a group via invite code. This is a public endpoint that combines registration with group joining.

**Endpoint:** `POST /groups/join/:code/register`

**Authentication Required:** No

**URL Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| code | string | Invite code |

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
POST /groups/join/ABC123XYZ/register HTTP/1.1
Host: localhost:8080
Content-Type: application/x-www-form-urlencoded

email=newuser@example.com&password=SecurePass123&name=Jane+Doe
```

**Success Response:**
- **Status Code:** 200 OK
- **Content-Type:** text/html
- **Set-Cookie Headers:**
  - `access_token`: JWT access token
  - `refresh_token`: JWT refresh token
- **Body:** Rendered join-group.html page with success message

**Side Effects:**
1. New user account is created
2. User is authenticated (cookies set)
3. User is added to the group as a member
4. User receives a notification about joining
5. Invite usage count is incremented

**Error Responses:**

| Error Message | Description |
|--------------|-------------|
| "Dados inválidos" | Invalid request format or data binding error |
| "Todos os campos são obrigatórios" | One or more required fields are missing |
| "A senha deve ter pelo menos 8 caracteres" | Password is too short |
| "A senha deve conter letras maiúsculas, minúsculas e números" | Password doesn't meet complexity requirements |
| "Este email já está cadastrado" | Email address is already registered |
| "Erro ao criar conta" | Server error during registration |
| "Convite não encontrado" | Invite code does not exist |
| "Convite expirado" | Invite has passed expiration date |
| "Convite inválido" | Invite is not valid |
| "Convite atingiu o limite de usos" | Invite reached maximum usage count |
| "Erro ao aceitar convite" | Server error during group joining |

---

## Group Management Endpoint Security

### Authentication & Authorization

**Protected Endpoints:**
All group management endpoints except the public invite viewing and registration endpoints require valid JWT authentication via cookie.

**Role-Based Access Control:**
- **Admin-only operations:**
  - Generate invite codes
  - View group invites
  - Revoke invites
  - Remove members
  - Delete group

- **Member operations:**
  - View groups
  - Create new groups (becomes admin)
  - Leave groups (if not last admin)

**Public Endpoints:**
- `GET /groups/join/:code` - View invite details
- `POST /groups/join/:code/register` - Register and join

### Authorization Checks

The GroupService validates:
1. User is a member of the group (for member operations)
2. User has admin role (for admin operations)
3. Last admin cannot leave group
4. Invite codes are valid and not expired

### Data Validation

**Group Creation:**
- Group name is required
- Creator is automatically added as admin

**Invite Generation:**
- Invite codes are 32-character random strings
- Default expiration: 30 days
- Default max uses: unlimited (0)

**Member Removal:**
- Cannot remove yourself (use leave endpoint instead)
- Admin check performed before removal

### Security Considerations

1. **Invite Code Security:**
   - Invite codes are cryptographically random (32 characters)
   - Codes expire after 30 days by default
   - Codes can have maximum usage limits
   - Codes can be revoked by admins

2. **Open Redirect Protection:**
   - All redirects use relative paths

3. **CSRF Protection:**
   - All state-changing endpoints require CSRF token
   - Public registration endpoint has CSRF validation

4. **Notification System:**
   - Users receive notifications when joining groups
   - Notifications include inviter information

### Data Flow

**Group Creation Flow:**
1. User submits group creation form
2. Group record created with user as creator
3. User added as admin member
4. Updated group list returned as HTML partial

**Invite Flow:**
1. Admin generates invite code
2. Invite stored with expiration and usage limits
3. Invite link shared externally
4. Public user views invite page
5. User logs in or registers
6. User accepts invite and joins group
7. Notification sent to new member

**Member Management Flow:**
1. Admin requests member removal
2. Authorization check performed
3. Member record soft-deleted
4. Updated group list returned

---

## Group Joint Account Endpoints

### Overview

Joint accounts are shared bank accounts within a family group. All group members can create transactions in joint accounts, and the balances are tracked collectively. These accounts appear on the group dashboard alongside individual member accounts.

**Authentication:** All endpoints require JWT authentication via cookie

**Authorization:** Users must be members of the group to manage joint accounts

---

### 1. Create Joint Account

Create a new joint account for a group.

**Endpoint:** `POST /groups/:id/accounts`

**Authentication Required:** Yes

**Authorization:** User must be a member of the group

**URL Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| id | integer | Group ID |

**Request Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| name | string | Yes | Name of the joint account |

**Content-Type:** `application/x-www-form-urlencoded`

**Example Request:**
```http
POST /groups/123/accounts HTTP/1.1
Host: localhost:8080
Content-Type: application/x-www-form-urlencoded
Cookie: access_token=...

name=Family+Checking+Account
```

**Success Response:**
- **Status Code:** 200 OK
- **Content-Type:** text/html
- **Body:** Updated joint accounts list partial (partials/joint-accounts-list.html)

**Response Includes:**
- All joint accounts for the group
- Account names and IDs
- Group ID and user ID for client-side rendering

**Error Responses:**

| Error Message | Description |
|--------------|-------------|
| "ID do grupo inválido" | Invalid group ID parameter |
| "Dados inválidos" | Invalid request format or data binding error |
| "Nome da conta é obrigatório" | Account name is missing |
| "Você não é membro deste grupo" | User is not a member of the group |
| "Erro ao criar conta conjunta" | Server error during account creation |

---

### 2. Delete Joint Account

Delete a joint account from a group.

**Endpoint:** `DELETE /groups/:id/accounts/:accountId`

**Authentication Required:** Yes

**Authorization:** User must be a member of the group

**URL Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| id | integer | Group ID |
| accountId | integer | Joint account ID |

**Example Request:**
```http
DELETE /groups/123/accounts/456 HTTP/1.1
Host: localhost:8080
Cookie: access_token=...
```

**Success Response:**
- **Status Code:** 200 OK
- **Content-Type:** text/html
- **Body:** Updated joint accounts list partial (partials/joint-accounts-list.html)

**Response Includes:**
- Remaining joint accounts for the group (after deletion)
- Account names and IDs
- Group ID and user ID for client-side rendering

**Error Responses:**

| Error Message | Description |
|--------------|-------------|
| "ID do grupo inválido" | Invalid group ID parameter |
| "ID da conta inválido" | Invalid account ID parameter |
| "Você não é membro deste grupo" | User is not a member of the group |
| "Conta não encontrada" | Joint account does not exist |
| "Erro ao excluir conta" | Server error during deletion |

**Side Effects:**
- Account is permanently deleted
- All transactions associated with the account remain but are orphaned
- Account balance data is removed

---

## Group Dashboard Endpoint

### Overview

The group dashboard provides a consolidated financial view for a family group, showing both joint account data and holistic family finances (individual + joint accounts combined).

**Features:**
- Monthly income/expense summary for joint accounts
- 6-month financial projection for joint accounts
- Member contribution tracking
- Upcoming bills from joint accounts
- Holistic view of all family finances (individual + joint accounts)

**Authentication:** Requires JWT authentication via cookie

**Authorization:** User must be a member of the group

---

### 1. View Group Dashboard

Display the consolidated dashboard for a group's joint accounts and holistic family view.

**Endpoint:** `GET /groups/:id/dashboard`

**Authentication Required:** Yes

**Authorization:** User must be a member of the group

**URL Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| id | integer | Group ID |

**Example Request:**
```http
GET /groups/123/dashboard HTTP/1.1
Host: localhost:8080
Cookie: access_token=...
```

**Success Response:**
- **Status Code:** 200 OK
- **Content-Type:** text/html
- **Body:** Rendered group-dashboard.html page with comprehensive financial data

**Response Data Includes:**

**Group Information:**
- Group name and description
- List of group members with roles

**Joint Accounts Summary:**
- Current month income and expenses for joint accounts
- 6-month projection of income/expenses/balance
- Individual joint account balances with transaction totals
- Total income, expenses, and balance across all joint accounts

**Member Contributions:**
- Breakdown of each member's contributions to joint accounts
- Income and expense amounts per member

**Upcoming Bills:**
- Fixed expenses due in the current month from joint accounts
- Due dates and amounts

**Holistic Family Summary:**
- Combined data from all accounts (individual + joint)
- Total family income, expenses, and balance
- 6-month holistic projection
- All account balances (individual and joint)

**Error Responses:**

| Error Message | Description |
|--------------|-------------|
| "ID do grupo inválido" | Invalid group ID parameter |
| "Você não é membro deste grupo" | User is not a member of the group |
| "Grupo não encontrado" | Group does not exist |

**Data Calculations:**

1. **Joint Account Summary:**
   - Aggregates income and expenses from joint accounts only
   - Calculates monthly summaries for current and next 5 months
   - Includes recurring transactions in projections

2. **Member Contributions:**
   - Tracks income added by each member to joint accounts
   - Tracks expenses paid by each member from joint accounts

3. **Holistic Summary:**
   - Combines data from personal accounts and joint accounts
   - Provides complete family financial picture
   - Useful for overall family budget planning

---

## Group Summary Endpoints

### Overview

Group summary endpoints generate periodic financial summary notifications for family groups. Summaries are sent to all group members as in-app notifications.

**Types of Summaries:**
- **Weekly:** Covers the last 7 days of financial activity
- **Monthly:** Covers the current calendar month

**Authentication:** Requires JWT authentication via cookie

**Authorization:** User must be a member of the group

---

### 1. Generate Weekly Summary

Generate and send a weekly financial summary notification to all group members.

**Endpoint:** `POST /groups/:id/summary/weekly`

**Authentication Required:** Yes

**Authorization:** User must be a member of the group

**URL Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| id | integer | Group ID |

**Example Request:**
```http
POST /groups/123/summary/weekly HTTP/1.1
Host: localhost:8080
Cookie: access_token=...
```

**Success Response:**
- **Status Code:** 200 OK
- **Content-Type:** text/plain
- **Body:** "Resumo semanal enviado com sucesso!"

**Side Effects:**
- Creates a notification for all group members
- Notification includes:
  - Period label (e.g., "Semana de 13/01 a 20/01")
  - Total income for the week (from joint accounts)
  - Total expenses for the week (from joint accounts)
  - Net balance (income - expenses)
  - Count of income and expense transactions

**Data Calculation:**
- Period: Last 7 days from current date
- Scope: Joint accounts only
- Income: Sum of net amounts from income records in the period
- Expenses: Sum of amounts from active expense records in the period

**Error Responses:**

| Error Message | Description |
|--------------|-------------|
| "ID do grupo inválido" | Invalid group ID parameter |
| "Você não é membro deste grupo" | User is not a member of the group |
| "Grupo não encontrado" | Group does not exist |
| "Erro ao buscar membros do grupo" | Server error retrieving group members |
| "Erro ao criar notificação de resumo" | Server error creating notifications |

---

### 2. Generate Monthly Summary

Generate and send a monthly financial summary notification to all group members.

**Endpoint:** `POST /groups/:id/summary/monthly`

**Authentication Required:** Yes

**Authorization:** User must be a member of the group

**URL Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| id | integer | Group ID |

**Example Request:**
```http
POST /groups/123/summary/monthly HTTP/1.1
Host: localhost:8080
Cookie: access_token=...
```

**Success Response:**
- **Status Code:** 200 OK
- **Content-Type:** text/plain
- **Body:** "Resumo mensal enviado com sucesso!"

**Side Effects:**
- Creates a notification for all group members
- Notification includes:
  - Period label (current month name, e.g., "Janeiro")
  - Total income for the month (from joint accounts)
  - Total expenses for the month (from joint accounts)
  - Net balance (income - expenses)

**Data Calculation:**
- Period: Current calendar month (month and year)
- Scope: Joint accounts only
- Uses monthly summary service to aggregate income and expenses
- Includes recurring transactions and projections

**Error Responses:**

| Error Message | Description |
|--------------|-------------|
| "ID do grupo inválido" | Invalid group ID parameter |
| "Você não é membro deste grupo" | User is not a member of the group |
| "Grupo não encontrado" | Group does not exist |
| "Erro ao buscar membros do grupo" | Server error retrieving group members |
| "Erro ao criar notificação de resumo" | Server error creating notifications |

---

## Group Features Security

### Authentication & Authorization

**All Group Feature Endpoints:**
All group feature endpoints (joint accounts, dashboard, summaries) require valid JWT authentication via cookie.

**Group Membership Validation:**
- All endpoints verify user is a member of the specified group
- Non-members receive "Você não é membro deste grupo" error
- Authorization check performed using GroupService.IsGroupMember()

**No Admin Requirement:**
Unlike group management endpoints, group feature endpoints are accessible to all group members (both admin and regular members).

### Data Access Control

**Joint Accounts:**
- Joint accounts are scoped to a specific group
- Only group members can create/delete joint accounts
- Account ownership is tied to the group, not individual users
- All transactions in joint accounts are visible to all group members

**Dashboard Data:**
- Group dashboard shows financial data only from:
  - Joint accounts (for joint summary)
  - All accounts belonging to group members (for holistic summary)
- Non-members cannot access group dashboard data

**Summary Notifications:**
- Summaries are sent to all active group members
- Notification service validates group membership before sending
- Summary data is calculated only from joint accounts

### Data Validation

**Joint Account Creation:**
- Account name is required
- Account is automatically associated with the group
- Account type is set to "joint"

**Joint Account Deletion:**
- Validates account exists and belongs to the specified group
- Validates user is a member of the group
- Permanently deletes account record

**Summary Generation:**
- Validates group exists
- Validates user is a group member
- Calculates data only from joint accounts belonging to the group

### CSRF Protection

All state-changing endpoints (POST/DELETE) require CSRF token validation via the header-based CSRF middleware.

---

## Goal Endpoints

This section describes the goal management endpoints for tracking financial goals within groups.

### Overview

Goal endpoints allow group members to create, track, and manage financial goals collaboratively. Goals can be linked to joint accounts and support multiple user contributions.

**Authentication:** All endpoints require valid JWT authentication via cookie.

**Authorization:** All endpoints require the user to be a member of the specified group.

**HTMX Support:** List endpoints return HTML partials for dynamic UI updates.

---

### 1. Goals Page

Get the goals page for a group, including all goals and joint accounts.

**Endpoint:** `GET /groups/:id/goals`

**Authentication Required:** Yes (JWT cookie)

**Rate Limited:** No

**URL Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| id | integer | Group ID |

**Example Request:**
```http
GET /groups/123/goals HTTP/1.1
Host: localhost:8080
Cookie: access_token=...
```

**Success Response:**
- **Status Code:** 200 OK
- **Content-Type:** text/html
- **Body:** Rendered goals.html template with:
  - group: Group information
  - goals: List of all goals with progress
  - accounts: Joint accounts for goal association
  - groupID: Group identifier
  - userID: Current user identifier

**Error Responses:**

| Error Message | Description |
|--------------|-------------|
| "ID do grupo inválido" | Invalid group ID parameter |
| "Você não é membro deste grupo" | User is not a member of the group |
| "Grupo não encontrado" | Group does not exist |
| "Erro ao buscar metas" | Server error retrieving goals |

---

### 2. List Goals

Get all goals for a group as an HTML partial (for HTMX updates).

**Endpoint:** `GET /groups/:id/goals` (partial)

**Authentication Required:** Yes (JWT cookie)

**Rate Limited:** No

**URL Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| id | integer | Group ID |

**Example Request:**
```http
GET /groups/123/goals HTTP/1.1
Host: localhost:8080
Cookie: access_token=...
HX-Request: true
```

**Success Response:**
- **Status Code:** 200 OK
- **Content-Type:** text/html
- **Body:** Rendered partials/goal-list.html with:
  - goals: Array of goal objects
  - groupID: Group identifier
  - userID: Current user identifier

**Goal Object Structure:**
- ID: Goal identifier
- Name: Goal name
- Description: Goal description
- TargetAmount: Target amount to reach
- CurrentAmount: Current amount contributed
- TargetDate: Target completion date
- AccountID: Optional linked joint account ID
- GroupID: Parent group ID
- Completed: Whether goal is completed
- Progress: Calculated progress percentage
- Contributions: List of user contributions

**Error Responses:**

| Error Message | Description |
|--------------|-------------|
| "ID do grupo inválido" | Invalid group ID parameter |
| "Você não é membro deste grupo" | User is not a member of the group |
| "Erro ao buscar metas" | Server error retrieving goals |

---

### 3. Create Goal

Create a new financial goal for a group.

**Endpoint:** `POST /groups/:id/goals`

**Authentication Required:** Yes (JWT cookie)

**Rate Limited:** No

**CSRF Protection:** Yes

**URL Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| id | integer | Group ID |

**Request Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| name | string | Yes | Goal name |
| description | string | No | Goal description |
| target_amount | float64 | Yes | Target amount (must be > 0) |
| target_date | string | Yes | Target date (YYYY-MM-DD format) |
| account_id | integer | No | Joint account ID to link (optional) |

**Content-Type:** `application/x-www-form-urlencoded`

**Example Request:**
```http
POST /groups/123/goals HTTP/1.1
Host: localhost:8080
Cookie: access_token=...
Content-Type: application/x-www-form-urlencoded

name=Emergency+Fund&description=3+months+expenses&target_amount=15000.00&target_date=2024-12-31&account_id=5
```

**Success Response:**
- **Status Code:** 200 OK
- **Content-Type:** text/html
- **Body:** Rendered partials/goal-list.html with updated goals list

**Side Effects:**
- Creates new goal record in database
- Goal is associated with the specified group
- If account_id is provided and valid, links goal to joint account
- Current amount is initialized to 0

**Error Responses:**

| Error Message | Description |
|--------------|-------------|
| "ID do grupo inválido" | Invalid group ID parameter |
| "Dados inválidos" | Invalid request format or data binding error |
| "Nome e valor alvo são obrigatórios" | Name or target_amount is missing or invalid |
| "Data alvo inválida" | target_date format is invalid |
| "Você não é membro deste grupo" | User is not a member of the group |
| "Erro ao criar meta" | Server error creating goal |

---

### 4. Delete Goal

Delete a financial goal.

**Endpoint:** `DELETE /groups/:id/goals/:goalId`

**Authentication Required:** Yes (JWT cookie)

**Rate Limited:** No

**CSRF Protection:** Yes

**URL Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| id | integer | Group ID |
| goalId | integer | Goal ID to delete |

**Example Request:**
```http
DELETE /groups/123/goals/456 HTTP/1.1
Host: localhost:8080
Cookie: access_token=...
```

**Success Response:**
- **Status Code:** 200 OK
- **Content-Type:** text/html
- **Body:** Rendered partials/goal-list.html with updated goals list

**Side Effects:**
- Permanently deletes goal record
- Deletes all associated contributions
- Cannot be undone

**Error Responses:**

| Error Message | Description |
|--------------|-------------|
| "ID da meta inválido" | Invalid goal ID parameter |
| "Meta não encontrada" | Goal does not exist |
| "Você não tem permissão para deletar esta meta" | User is not a member of the goal's group |
| "Erro ao deletar meta" | Server error deleting goal |

---

### 5. Add Contribution

Add a monetary contribution to a goal.

**Endpoint:** `POST /groups/:id/goals/:goalId/contribute`

**Authentication Required:** Yes (JWT cookie)

**Rate Limited:** No

**CSRF Protection:** Yes

**URL Parameters:**

| Parameter | Type | Description |
|-----------|------|-------------|
| id | integer | Group ID |
| goalId | integer | Goal ID to contribute to |

**Request Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| amount | float64 | Yes | Contribution amount (must be > 0) |

**Content-Type:** `application/x-www-form-urlencoded`

**Example Request:**
```http
POST /groups/123/goals/456/contribute HTTP/1.1
Host: localhost:8080
Cookie: access_token=...
Content-Type: application/x-www-form-urlencoded

amount=500.00
```

**Success Response:**
- **Status Code:** 200 OK
- **Content-Type:** text/html
- **Body:** Rendered partials/goal-list.html with updated goals list

**Side Effects:**
- Creates contribution record linked to user and goal
- Updates goal's current_amount
- Recalculates goal progress percentage
- If contribution causes goal to reach or exceed target, marks goal as completed

**Contribution Tracking:**
- Each contribution is recorded with:
  - User ID (contributor)
  - Goal ID
  - Amount
  - Timestamp
- Users can see their individual contributions
- Group members can see all contributions to shared goals

**Error Responses:**

| Error Message | Description |
|--------------|-------------|
| "ID da meta inválido" | Invalid goal ID parameter |
| "Valor inválido" | Amount is missing, non-numeric, or <= 0 |
| "Meta não encontrada" | Goal does not exist |
| "Meta já foi concluída" | Goal has already been marked as completed |
| "Você não é membro deste grupo" | User is not a member of the goal's group |
| "Erro ao adicionar contribuição" | Server error adding contribution |

---

## Goal Endpoint Security

### Authentication & Authorization

**All Goal Endpoints:**
All goal endpoints require valid JWT authentication via cookie.

**Group Membership Validation:**
- All endpoints verify user is a member of the goal's group
- Non-members receive "Você não é membro deste grupo" error
- Authorization check performed using GroupService.IsGroupMember()

**No Admin Requirement:**
All group members can create, view, and contribute to goals. Only the operations that modify goals require membership validation.

### Data Access Control

**Goal Visibility:**
- Goals are scoped to a specific group
- Only group members can view goals
- All group members see the same goals
- Goal data includes all contributions from all members

**Goal Creation:**
- Any group member can create goals
- Goals are automatically associated with the group
- Optional joint account linking (if account_id provided)

**Goal Deletion:**
- Any group member can delete goals
- Deletion removes goal and all contributions
- Permanent operation - cannot be undone

**Contributions:**
- Any group member can contribute to any group goal
- Contributions are tracked per user
- Users can see who contributed and how much
- Cannot contribute to completed goals

### Data Validation

**Goal Creation:**
- Name is required (non-empty string)
- Target amount must be greater than 0
- Target date must be valid YYYY-MM-DD format
- Account ID (if provided) must be a valid joint account

**Goal Contribution:**
- Amount must be greater than 0
- Cannot contribute to non-existent goals
- Cannot contribute to completed goals
- Goal membership must be validated

**Goal Progress:**
- Progress calculated as: (current_amount / target_amount) * 100
- Goal marked completed when current_amount >= target_amount
- Completed status prevents further contributions

### Joint Account Integration

**Account Linking:**
- Goals can optionally be linked to a joint account
- Account ID validation ensures account belongs to the group
- Linking is for tracking purposes (doesn't automatically withdraw funds)
- Goals can exist without linked accounts

### CSRF Protection

All state-changing endpoints (POST/DELETE) require CSRF token validation via the header-based CSRF middleware.

---

## Notification Endpoints

This document describes the notification endpoints for the POC Finance application.

### Overview

All notification endpoints require authentication via JWT token. These endpoints provide notification management capabilities including listing notifications, checking unread counts, marking as read, and deleting notifications.

Features:
- Real-time notification badge updates via HTMX
- Dropdown partial for navigation bar integration
- Mark individual or all notifications as read
- Delete notifications
- Unread count tracking

---

### 1. List Notifications

Get all notifications for the current user.

**Endpoint:** `GET /notifications`

**Authentication Required:** Yes (JWT via cookie)

**Rate Limited:** No

**Request Parameters:** None

**Example Request:**
```http
GET /notifications HTTP/1.1
Host: localhost:8080
Cookie: access_token=...
```

**Success Response:**
- **Status Code:** 200 OK
- **Content-Type:** text/html
- **Body:** Rendered notifications page with full notification list and unread count

**Response Data:**
```
{
  "notifications": [...],  // Array of notification objects
  "unreadCount": 5         // Count of unread notifications
}
```

**Error Responses:**

| Status Code | Error Message | Description |
|------------|---------------|-------------|
| 500 | "Erro ao buscar notificações" | Server error fetching notifications |

---

### 2. Get Notification Badge

Get the notification badge count for display in navigation (HTMX partial).

**Endpoint:** `GET /notifications/badge`

**Authentication Required:** Yes (JWT via cookie)

**Rate Limited:** No

**Request Parameters:** None

**Example Request:**
```http
GET /notifications/badge HTTP/1.1
Host: localhost:8080
Cookie: access_token=...
```

**Success Response:**
- **Status Code:** 200 OK
- **Content-Type:** text/html
- **Body:** Rendered notification badge partial (HTML fragment)

**Response Data:**
```
{
  "unreadCount": 5  // Count of unread notifications
}
```

**Usage:**
This endpoint is designed to be called via HTMX for dynamic badge updates without full page reload.

**Error Responses:**

| Status Code | Error Message | Description |
|------------|---------------|-------------|
| 500 | Server error | Error fetching unread count |

---

### 3. Get Notification Dropdown

Get the notification dropdown content with recent notifications (HTMX partial).

**Endpoint:** `GET /notifications/dropdown`

**Authentication Required:** Yes (JWT via cookie)

**Rate Limited:** No

**Request Parameters:** None

**Behavior:**
Returns the 5 most recent notifications for display in navigation dropdown.

**Example Request:**
```http
GET /notifications/dropdown HTTP/1.1
Host: localhost:8080
Cookie: access_token=...
```

**Success Response:**
- **Status Code:** 200 OK
- **Content-Type:** text/html
- **Body:** Rendered notification dropdown partial (HTML fragment)

**Response Data:**
```
{
  "notifications": [...],  // Array of up to 5 recent notifications
  "unreadCount": 5         // Count of unread notifications
}
```

**Usage:**
This endpoint is designed to be called via HTMX for dynamic dropdown updates without full page reload.

**Error Responses:**

| Status Code | Error Message | Description |
|------------|---------------|-------------|
| 500 | "Erro ao buscar notificações" | Server error fetching notifications |

---

### 4. Mark Notification as Read

Mark a single notification as read.

**Endpoint:** `POST /notifications/:id/read`

**Authentication Required:** Yes (JWT via cookie)

**Rate Limited:** No

**CSRF Protection:** Yes

**Path Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| id | integer | Yes | Notification ID |

**Example Request:**
```http
POST /notifications/42/read HTTP/1.1
Host: localhost:8080
Cookie: access_token=...
X-CSRF-Token: ...
```

**Success Response:**
- **Status Code:** 200 OK
- **Content-Type:** text/html
- **Body:** Rendered notification dropdown partial with updated data

**Behavior:**
After marking as read, the endpoint returns the updated dropdown partial for seamless HTMX integration.

**Authorization:**
- Only the notification owner can mark it as read
- Attempting to mark another user's notification returns error

**Error Responses:**

| Status Code | Error Message | Description |
|------------|---------------|-------------|
| 400 | "ID da notificação inválido" | Invalid notification ID format |
| 500 | "Erro ao marcar como lida" | Server error marking notification as read |

---

### 5. Mark All Notifications as Read

Mark all notifications as read for the current user.

**Endpoint:** `POST /notifications/mark-all-read`

**Authentication Required:** Yes (JWT via cookie)

**Rate Limited:** No

**CSRF Protection:** Yes

**Request Parameters:** None

**Example Request:**
```http
POST /notifications/mark-all-read HTTP/1.1
Host: localhost:8080
Cookie: access_token=...
X-CSRF-Token: ...
```

**Success Response:**
- **Status Code:** 200 OK
- **Content-Type:** text/html
- **Body:** Rendered notification dropdown partial with updated data

**Behavior:**
- Marks all notifications for the current user as read
- Returns the updated dropdown partial for seamless HTMX integration

**Error Responses:**

| Status Code | Error Message | Description |
|------------|---------------|-------------|
| 500 | "Erro ao marcar todas como lidas" | Server error marking all notifications as read |

---

### 6. Delete Notification

Delete a single notification.

**Endpoint:** `DELETE /notifications/:id`

**Authentication Required:** Yes (JWT via cookie)

**Rate Limited:** No

**CSRF Protection:** Yes

**Path Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| id | integer | Yes | Notification ID |

**Example Request:**
```http
DELETE /notifications/42 HTTP/1.1
Host: localhost:8080
Cookie: access_token=...
X-CSRF-Token: ...
```

**Success Response:**
- **Status Code:** 200 OK
- **Body:** Empty string

**Behavior:**
Permanently deletes the notification from the database.

**Authorization:**
- Only the notification owner can delete it
- Attempting to delete another user's notification returns error

**Error Responses:**

| Status Code | Error Message | Description |
|------------|---------------|-------------|
| 400 | "ID da notificação inválido" | Invalid notification ID format |
| 500 | "Erro ao excluir notificação" | Server error deleting notification |

---

## Notification Endpoint Security

### Authentication & Authorization

**All Notification Endpoints:**
All notification endpoints require valid JWT authentication via cookie.

**User-Scoped Access:**
- All endpoints automatically scope to the authenticated user
- Users can only access their own notifications
- User ID is extracted from JWT token via middleware
- No cross-user notification access possible

### Data Access Control

**Notification Visibility:**
- Notifications are strictly scoped to the notification owner (user_id)
- Users cannot view other users' notifications
- Service layer enforces user_id filtering on all operations

**Notification Modification:**
- Mark as read: User can only mark their own notifications
- Delete: User can only delete their own notifications
- All operations validate notification ownership via user_id

### HTMX Integration

**Partial Rendering:**
- Badge and dropdown endpoints return HTML fragments for HTMX
- Enables real-time UI updates without full page reload
- After mutation operations (mark read, delete), dropdown partial is returned
- Maintains consistent state between server and client

**Polling Support:**
- Badge endpoint designed for periodic polling
- Dropdown endpoint supports on-demand loading
- Efficient queries with limit parameter for dropdown (5 notifications)

### CSRF Protection

All state-changing endpoints (POST/DELETE) require CSRF token validation via the header-based CSRF middleware.

---

## Recurring Transaction Endpoints

This section describes the recurring transaction management endpoints for the POC Finance application.

### Overview

All recurring transaction endpoints require authentication via JWT tokens (access_token cookie). These endpoints manage recurring transactions that automatically create income or expense records at specified intervals (daily, weekly, monthly, or yearly).

Key features:
- Multi-account support (individual and joint accounts)
- Automatic transaction generation based on schedule
- Flexible frequency options (daily, weekly, monthly, yearly)
- Optional end date for finite recurring transactions
- Active/pause toggle for temporary suspension
- HTMX partial responses for dynamic UI updates

---

### 1. List Recurring Transactions

Retrieve all recurring transactions for accounts accessible by the authenticated user, separated by active and paused status.

**Endpoint:** `GET /recurring`

**Authentication Required:** Yes (JWT access_token cookie)

**Rate Limited:** No

**Request Parameters:** None

**Example Request:**
```http
GET /recurring HTTP/1.1
Host: localhost:8080
Cookie: access_token=...
```

**Success Response:**
- **Status Code:** 200 OK
- **Content-Type:** text/html
- **Body:** Rendered recurring.html page with active and paused recurring transaction lists

**Response Data Includes:**

| Field | Type | Description |
|-------|------|-------------|
| activeRecurringTransactions | []RecurringTransaction | Array of active recurring transactions ordered by next_run_date (ASC) |
| pausedRecurringTransactions | []RecurringTransaction | Array of paused recurring transactions ordered by next_run_date (ASC) |
| accounts | []Account | User's accessible accounts for the account selector |
| transactionTypes | []string | Available transaction types: ["expense", "income"] |
| frequencies | []string | Available frequencies: ["daily", "weekly", "monthly", "yearly"] |

**RecurringTransaction Record Fields:**

| Field | Type | Description |
|-------|------|-------------|
| ID | uint | Recurring transaction record ID |
| AccountID | uint | Associated account ID |
| Account | Account | Preloaded account information |
| TransactionType | string | Type of transaction: "expense" or "income" |
| Frequency | string | Recurrence frequency: "daily", "weekly", "monthly", or "yearly" |
| Amount | float64 | Transaction amount |
| Description | string | Transaction description |
| StartDate | time.Time | Date when recurring transactions begin |
| EndDate | *time.Time | Optional date when recurring transactions end (null for indefinite) |
| NextRunDate | time.Time | Next scheduled execution date |
| Active | bool | Whether the recurring transaction is active (true) or paused (false) |
| Category | string | Transaction category |

**Error Responses:**

| Error | Description |
|-------|-------------|
| 401 Unauthorized | Missing or invalid authentication token |
| 404 Not Found | Template not found (server configuration error) |

---

### 2. Create Recurring Transaction

Create a new recurring transaction that will automatically generate transactions at specified intervals.

**Endpoint:** `POST /recurring`

**Authentication Required:** Yes (JWT access_token cookie)

**Rate Limited:** No

**CSRF Protection:** Yes (X-CSRF-Token header required)

**Request Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| account_id | uint | No | Account ID (defaults to user's individual account if not specified) |
| transaction_type | string | Yes | Transaction type: "expense" or "income" |
| frequency | string | Yes | Recurrence frequency: "daily", "weekly", "monthly", or "yearly" |
| amount | float64 | Yes | Transaction amount (must be greater than zero) |
| description | string | Yes | Description of the recurring transaction |
| start_date | string | Yes | Start date in YYYY-MM-DD format |
| end_date | string | No | Optional end date in YYYY-MM-DD format |
| category | string | No | Transaction category |

**Content-Type:** `application/x-www-form-urlencoded`

**Example Request:**
```http
POST /recurring HTTP/1.1
Host: localhost:8080
Cookie: access_token=...
X-CSRF-Token: ...
Content-Type: application/x-www-form-urlencoded

account_id=1&transaction_type=expense&frequency=monthly&amount=99.99&description=Netflix+Subscription&start_date=2024-01-01&category=Entertainment
```

**Automatic Initialization:**
- `NextRunDate` is automatically set to the `start_date`
- `Active` is automatically set to `true`

**Success Response:**
- **Status Code:** 200 OK
- **Content-Type:** text/html
- **HX-Trigger:** Updates UI via HTMX
- **Body:** Rendered `partials/recurring-list.html` with updated active recurring transaction list

**HTMX Partial Response:**
The response is an HTML fragment containing the updated active recurring transaction list, suitable for swapping into the page via HTMX.

**Account Validation:**
- If `account_id` is 0 or not specified, the user's individual account is used
- If `account_id` is specified, the system validates the user has access to that account
- Joint accounts are supported if the user is a member

**Error Responses:**

| Status Code | Error Message | Description |
|------------|---------------|-------------|
| 400 | "Dados inválidos" | Invalid request format or data binding error |
| 400 | "Tipo de transação inválido" | Transaction type is not "expense" or "income" |
| 400 | "Frequência inválida" | Frequency is not one of: "daily", "weekly", "monthly", "yearly" |
| 400 | "Data de início inválida" | Start date format is invalid (must be YYYY-MM-DD) |
| 400 | "Data de término inválida" | End date format is invalid (must be YYYY-MM-DD) |
| 400 | "Valor deve ser maior que zero" | Amount is zero or negative |
| 403 | "Acesso negado à conta selecionada" | User doesn't have access to the specified account |
| 500 | "Conta não encontrada" | Individual account not found for user |
| 500 | "Erro ao criar transação recorrente" | Database error during creation |

**Error Response Format:**
- **Status Code:** As specified above
- **Content-Type:** text/plain
- **Body:** Error message string

---

### 3. Update Recurring Transaction

Update an existing recurring transaction. All fields are optional - only provided fields will be updated.

**Endpoint:** `POST /recurring/:id`

**Authentication Required:** Yes (JWT access_token cookie)

**Rate Limited:** No

**CSRF Protection:** Yes (X-CSRF-Token header required)

**URL Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| id | int | Yes | Recurring transaction ID to update |

**Request Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| account_id | uint | No | Account ID to change to |
| transaction_type | string | No | Transaction type: "expense" or "income" |
| frequency | string | No | Recurrence frequency: "daily", "weekly", "monthly", or "yearly" |
| amount | float64 | No | Transaction amount (must be greater than zero if provided) |
| description | string | No | Description of the recurring transaction |
| start_date | string | No | Start date in YYYY-MM-DD format |
| end_date | string | No | End date in YYYY-MM-DD format |
| category | string | No | Transaction category |

**Content-Type:** `application/x-www-form-urlencoded`

**Example Request:**
```http
POST /recurring/123 HTTP/1.1
Host: localhost:8080
Cookie: access_token=...
X-CSRF-Token: ...
Content-Type: application/x-www-form-urlencoded

amount=109.99&description=Netflix+Premium+Subscription
```

**Partial Update Behavior:**
Only the fields provided in the request will be updated. Fields not included in the request will retain their current values.

**Access Validation:**
The system verifies that the recurring transaction belongs to one of the user's accessible accounts before allowing updates.

**Success Response:**
- **Status Code:** 200 OK
- **Content-Type:** text/html
- **HX-Trigger:** Updates UI via HTMX
- **Body:** Rendered `partials/recurring-list.html` with updated active recurring transaction list

**HTMX Partial Response:**
The response is an HTML fragment containing the updated active recurring transaction list with the modified record.

**Error Responses:**

| Status Code | Error Message | Description |
|------------|---------------|-------------|
| 400 | "Dados inválidos" | Invalid request format or data binding error |
| 400 | "Tipo de transação inválido" | Transaction type is not "expense" or "income" |
| 400 | "Frequência inválida" | Frequency is not one of: "daily", "weekly", "monthly", "yearly" |
| 400 | "Data de início inválida" | Start date format is invalid (must be YYYY-MM-DD) |
| 400 | "Data de término inválida" | End date format is invalid (must be YYYY-MM-DD) |
| 403 | "Acesso negado à conta selecionada" | User doesn't have access to the specified account (when changing account) |
| 404 | "Transação recorrente não encontrada" | Recurring transaction not found or user doesn't have access |
| 500 | "Erro ao atualizar transação recorrente" | Database error during update |

**Error Response Format:**
- **Status Code:** As specified above
- **Content-Type:** text/plain
- **Body:** Error message string

---

### 4. Delete Recurring Transaction

Delete an existing recurring transaction. This will not affect any transactions that have already been created by this recurring transaction.

**Endpoint:** `DELETE /recurring/:id`

**Authentication Required:** Yes (JWT access_token cookie)

**Rate Limited:** No

**CSRF Protection:** Yes (X-CSRF-Token header required)

**URL Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| id | int | Yes | Recurring transaction ID to delete |

**Example Request:**
```http
DELETE /recurring/123 HTTP/1.1
Host: localhost:8080
Cookie: access_token=...
X-CSRF-Token: ...
```

**Access Validation:**
The system verifies that the recurring transaction belongs to one of the user's accessible accounts before allowing deletion.

**Success Response:**
- **Status Code:** 200 OK
- **Content-Type:** text/html
- **HX-Trigger:** Updates UI via HTMX
- **Body:** Rendered `partials/recurring-list.html` with updated active recurring transaction list (record removed)

**HTMX Partial Response:**
The response is an HTML fragment containing the updated active recurring transaction list with the deleted record removed.

**Important Note:**
Deleting a recurring transaction does NOT delete any transactions that were previously created by it. It only prevents future transactions from being automatically generated.

**Error Responses:**

| Status Code | Error Message | Description |
|------------|---------------|-------------|
| 404 | "Transação recorrente não encontrada" | Recurring transaction not found or user doesn't have access |
| 500 | "Erro ao deletar" | Database error during deletion |

**Error Response Format:**
- **Status Code:** As specified above
- **Content-Type:** text/plain
- **Body:** Error message string

---

### 5. Toggle Recurring Transaction Active Status

Toggle a recurring transaction between active and paused states. When paused, the recurring transaction will not generate new transactions until reactivated.

**Endpoint:** `POST /recurring/:id/toggle`

**Authentication Required:** Yes (JWT access_token cookie)

**Rate Limited:** No

**CSRF Protection:** Yes (X-CSRF-Token header required)

**URL Parameters:**

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| id | int | Yes | Recurring transaction ID to toggle |

**Request Parameters:** None

**Example Request:**
```http
POST /recurring/123/toggle HTTP/1.1
Host: localhost:8080
Cookie: access_token=...
X-CSRF-Token: ...
```

**Toggle Behavior:**
- If `Active` is `true`, it will be set to `false` (paused)
- If `Active` is `false`, it will be set to `true` (active)

**Access Validation:**
The system verifies that the recurring transaction belongs to one of the user's accessible accounts before allowing the toggle operation.

**Success Response:**
- **Status Code:** 200 OK
- **Content-Type:** text/html
- **HX-Trigger:** Updates UI via HTMX
- **Body:** Rendered `partials/recurring-list.html` with updated active recurring transaction list

**HTMX Partial Response:**
The response is an HTML fragment containing the updated active recurring transaction list. Transactions moved to paused status will be removed from this list and appear in the paused transactions list.

**Use Cases:**
- Temporarily pause subscriptions during vacation
- Suspend recurring bills that are on hold
- Disable recurring income during sabbatical
- Reactivate paused recurring transactions

**Error Responses:**

| Status Code | Error Message | Description |
|------------|---------------|-------------|
| 404 | "Transação recorrente não encontrada" | Recurring transaction not found or user doesn't have access |

**Error Response Format:**
- **Status Code:** As specified above
- **Content-Type:** text/plain
- **Body:** Error message string

---

## Recurring Transaction Endpoint Security

### Authentication & Authorization

**All Recurring Transaction Endpoints:**
All recurring transaction endpoints require valid JWT authentication via cookie.

**User-Scoped Access:**
- All endpoints automatically scope to the authenticated user's accessible accounts
- Users can only access recurring transactions for their own accounts (individual or joint)
- User ID is extracted from JWT token via middleware
- Account IDs are fetched using AccountService.GetUserAccountIDs()
- No cross-user recurring transaction access possible

### Data Access Control

**Recurring Transaction Visibility:**
- Recurring transactions are scoped to accounts the user has access to
- Users can view recurring transactions for:
  - Their individual account
  - Joint accounts they are members of
- Query filtering: `WHERE account_id IN (user's accessible account IDs)`

**Recurring Transaction Modification:**
- Create: User can create recurring transactions for accounts they have access to
- Update: User can only update recurring transactions for their accessible accounts
- Delete: User can only delete recurring transactions for their accessible accounts
- Toggle: User can only toggle recurring transactions for their accessible accounts
- All operations validate account ownership before execution

### Account Access Validation

**Account Selection:**
- If no account_id is specified, defaults to user's individual account
- If account_id is specified, validates user has access via AccountService.CanUserAccessAccount()
- Returns 403 Forbidden if user doesn't have access to specified account
- Supports both individual and joint accounts

### HTMX Integration

**Partial Rendering:**
- All mutation endpoints (Create, Update, Delete, Toggle) return HTML fragments
- Enables real-time UI updates without full page reload
- Responses render `partials/recurring-list.html` with updated active transactions list
- Maintains consistent state between server and client

### CSRF Protection

All state-changing endpoints (POST/DELETE) require CSRF token validation via the header-based CSRF middleware.

### Data Validation

**Required Field Validation:**
- Transaction type must be "expense" or "income"
- Frequency must be one of: "daily", "weekly", "monthly", "yearly"
- Amount must be greater than zero
- Start date is required and must be in YYYY-MM-DD format
- End date is optional but must be in YYYY-MM-DD format if provided

**Business Logic Validation:**
- NextRunDate is automatically set to StartDate on creation
- Active flag is automatically set to true on creation
- Account access is validated before any operation
- Transactions are ordered by next_run_date for predictable display

### Automated Transaction Processing

**Background Scheduler:**
Recurring transactions are processed by a background scheduler service that:
- Runs immediately on application startup
- Checks for due transactions daily at midnight
- Only processes transactions where:
  - Active = true
  - NextRunDate <= current date
  - EndDate is null or >= current date
- Automatically creates expense or income records based on TransactionType
- Updates NextRunDate based on Frequency after creating transaction

**Transaction Generation:**
When a recurring transaction is due, the system:
1. Checks if the recurring transaction is active
2. Checks if NextRunDate has arrived
3. Creates the corresponding expense or income transaction
4. Updates NextRunDate based on frequency:
   - Daily: Adds 1 day
   - Weekly: Adds 7 days
   - Monthly: Adds 1 month (preserving day of month)
   - Yearly: Adds 1 year (preserving month and day)
5. If EndDate is reached, may deactivate the recurring transaction

---
