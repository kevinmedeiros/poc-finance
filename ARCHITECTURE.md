# Architecture Overview

**Project**: Personal Finance Management System (poc-finance)
**Status**: ✅ Production Ready
**Created**: 2026-01-19

---

## Project Overview

**poc-finance** is a full-stack personal finance management system built with Go that helps users track income, expenses, credit cards, recurring transactions, and manage family group finances. The application follows a clean, layered architecture pattern with clear separation of concerns.

**Key Capabilities:**
- 💰 Income and expense tracking with categorization
- 💳 Credit card management with installment tracking
- 🔄 Recurring transactions with automated scheduling
- 👥 Family groups with joint accounts and shared expenses
- 🎯 Financial goals with contribution tracking
- 🔔 Real-time notifications system
- 📊 Data export and reporting

---

## Technology Stack

### Backend
- **Language**: Go 1.25.5
- **Web Framework**: Echo v4 (high-performance HTTP router)
- **ORM**: GORM v1.31.1 (database abstraction)
- **Database**: SQLite (via gorm.io/driver/sqlite)
- **Authentication**: JWT (golang-jwt/jwt/v5)
- **Password Hashing**: bcrypt (golang.org/x/crypto)

### Frontend
- **Template Engine**: Go html/template (server-side rendering)
- **Interactivity**: HTMX (dynamic updates without full page reloads)
- **Styling**: Tailwind CSS
- **Icons**: Bootstrap Icons

### Additional Libraries
- **Excel Export**: excelize v2 (github.com/xuri/excelize/v2)
- **Security**: Echo middleware (CSRF, rate limiting, security headers)

---

## Layered Architecture

The application follows a **4-layer architecture** pattern:

```
┌─────────────────────────────────────────────┐
│          Presentation Layer                 │
│  (HTTP Handlers + HTML Templates)           │
│  - Request validation                       │
│  - Response rendering                       │
│  - Session management                       │
└─────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────┐
│          Business Logic Layer               │
│  (Services)                                 │
│  - Complex operations                       │
│  - Transaction orchestration                │
│  - Background jobs (schedulers)             │
└─────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────┐
│          Data Access Layer                  │
│  (Models + GORM)                            │
│  - Database queries                         │
│  - Data validation                          │
│  - Relationships                            │
└─────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────┐
│          Database Layer                     │
│  (SQLite)                                   │
│  - Data persistence                         │
│  - Transactions                             │
└─────────────────────────────────────────────┘
```

### Layer Responsibilities

**1. Presentation Layer (Handlers + Templates)**
- Accepts HTTP requests and extracts parameters
- Validates input data
- Calls appropriate services or models
- Renders HTML responses (full pages or HTMX partials)
- Manages sessions and authentication state

**2. Business Logic Layer (Services)**
- Implements complex business rules
- Coordinates multiple models
- Handles background processing (e.g., recurring transaction scheduler)
- Sends notifications
- Generates reports and summaries

**3. Data Access Layer (Models)**
- Defines database schema using GORM structs
- Provides CRUD operations
- Manages relationships (belongs to, has many, many-to-many)
- Enforces data integrity

**4. Database Layer (SQLite)**
- Stores persistent data
- Provides transactional guarantees
- Handles migrations automatically via GORM AutoMigrate

---

## Directory Structure

```
poc-finance/
│
├── cmd/
│   └── server/
│       └── main.go                    # Application entry point
│                                      # - Database initialization
│                                      # - Route configuration
│                                      # - Server startup
│                                      # - Background scheduler launch
│
├── internal/
│   ├── database/
│   │   └── database.go                # Database connection and migration
│   │
│   ├── models/                        # Data Access Layer (GORM models)
│   │   ├── user.go                    # User authentication model
│   │   ├── account.go                 # Financial accounts (personal/joint)
│   │   ├── income.go                  # Income transactions
│   │   ├── expense.go                 # Expense transactions
│   │   ├── credit_card.go             # Credit card entities
│   │   ├── installment.go             # Credit card installments
│   │   ├── recurring_transaction.go   # Recurring transaction definitions
│   │   ├── group.go                   # Family groups
│   │   ├── expense_split.go           # Shared expense splits
│   │   ├── notification.go            # User notifications
│   │   ├── goal.go                    # Financial goals
│   │   ├── bill.go                    # Bill tracking
│   │   ├── expense_payment.go         # Expense payment records
│   │   ├── settings.go                # User settings
│   │   └── *_test.go                  # Model unit tests
│   │
│   ├── handlers/                      # Presentation Layer (HTTP controllers)
│   │   ├── auth.go                    # Login, register, password reset
│   │   ├── dashboard.go               # Main dashboard view
│   │   ├── income.go                  # Income CRUD operations
│   │   ├── expense.go                 # Expense CRUD operations
│   │   ├── credit_card.go             # Card and installment management
│   │   ├── recurring_transaction.go   # Recurring transaction management
│   │   ├── group.go                   # Family group management
│   │   ├── account.go                 # Account list and balance
│   │   ├── goal.go                    # Goal management
│   │   ├── notification.go            # Notification management
│   │   ├── settings.go                # User settings
│   │   ├── export.go                  # Data export (Excel)
│   │   └── *_test.go                  # Handler integration tests
│   │
│   ├── services/                      # Business Logic Layer
│   │   ├── auth.go                    # JWT token management
│   │   ├── account.go                 # Account balance calculations
│   │   ├── group.go                   # Group operations and invites
│   │   ├── goal.go                    # Goal progress tracking
│   │   ├── notification.go            # Notification creation and delivery
│   │   ├── recurring_scheduler.go     # Background job: process recurring transactions
│   │   ├── summary.go                 # Financial summaries and reports
│   │   ├── tax.go                     # Tax calculations
│   │   └── *_test.go                  # Service unit tests
│   │
│   ├── middleware/
│   │   ├── auth.go                    # JWT authentication middleware
│   │   └── auth_test.go               # Middleware tests
│   │
│   ├── templates/                     # HTML templates
│   │   ├── base.html                  # Base layout with navigation
│   │   ├── dashboard.html             # Dashboard page
│   │   ├── income.html                # Income management page
│   │   ├── expenses.html              # Expense management page
│   │   ├── cards.html                 # Credit card management page
│   │   ├── recurring.html             # Recurring transactions page
│   │   ├── groups.html                # Family groups page
│   │   ├── group-dashboard.html       # Group-specific dashboard
│   │   ├── accounts.html              # Accounts list page
│   │   ├── goals.html                 # Financial goals page
│   │   ├── notifications.html         # Notifications page
│   │   ├── settings.html              # User settings page
│   │   ├── login.html                 # Login page
│   │   ├── register.html              # Registration page
│   │   ├── forgot-password.html       # Password recovery page
│   │   ├── reset-password.html        # Password reset page
│   │   ├── join-group.html            # Group invite acceptance page
│   │   └── partials/                  # HTMX partial templates
│   │       ├── income-list.html       # Income list fragment
│   │       ├── expense-list.html      # Expense list fragment
│   │       ├── notification-*.html    # Notification fragments
│   │       └── ...                    # Other partial fragments
│   │
│   └── testutil/
│       └── testutil.go                # Shared test utilities
│
├── scripts/                           # Utility scripts
│
├── tasks/                             # Task-related files and specs
│
├── go.mod                             # Go module definition
├── go.sum                             # Go dependency checksums
├── Makefile                           # Build and test automation
├── ARCHITECTURE.md                    # This file
├── TESTING_GUIDE.md                   # Testing instructions
└── README.md                          # Project documentation
```

---

## Key Architectural Patterns

### 1. Dependency Injection

Handlers and services receive dependencies through constructors:

```go
// Handler receives database connection
func NewIncomeHandler() *IncomeHandler {
    return &IncomeHandler{
        db: database.DB,
    }
}

// Service receives database connection
func NewRecurringSchedulerService() *RecurringSchedulerService {
    return &RecurringSchedulerService{
        db: database.DB,
    }
}
```

### 2. Middleware Chain

Echo middleware processes requests before reaching handlers:

```
Request → Logger → Recover → BodyLimit → Security Headers →
          CSRF Protection → Rate Limiter → Auth Middleware → Handler
```

**Middleware Responsibilities:**
- **Logger**: Request/response logging
- **Recover**: Panic recovery
- **BodyLimit**: Prevent large payloads (2MB limit)
- **Secure**: Security headers (XSS, CSP, HSTS)
- **CSRF**: Cross-site request forgery protection
- **RateLimiter**: Rate limiting for auth endpoints (5 req/sec)
- **AuthMiddleware**: JWT validation and user context

### 3. Handlers Layer

The Handlers Layer (Presentation Layer) follows a consistent pattern using the Echo framework. Handlers are responsible for HTTP request/response handling, input validation, and template rendering.

#### Handler Structure Pattern

All handlers follow this structure:

```go
// Handler struct with dependencies
type AuthHandler struct {
    authService *services.AuthService
}

// Constructor with dependency injection
func NewAuthHandler() *AuthHandler {
    return &AuthHandler{
        authService: services.NewAuthService(),
    }
}
```

#### Request Handling with echo.Context

Each handler method accepts `echo.Context` and returns `error`:

```go
func (h *AuthHandler) LoginPage(c echo.Context) error {
    // Extract query parameters
    registered := c.QueryParam("registered") == "1"
    redirect := c.QueryParam("redirect")

    // Render template with data
    return c.Render(http.StatusOK, "login.html", map[string]interface{}{
        "registered": registered,
        "redirect":   redirect,
    })
}
```

**echo.Context provides:**
- `c.Bind()` - Bind form/JSON data to structs
- `c.QueryParam()` - Extract query parameters
- `c.Param()` - Extract path parameters
- `c.Cookie()` - Read cookies
- `c.SetCookie()` - Set cookies
- `c.Render()` - Render templates
- `c.Redirect()` - HTTP redirects
- `c.Get()` - Access middleware-injected values (e.g., user from auth middleware)

#### Input Validation and Binding

Handlers validate and sanitize input before processing:

```go
type LoginRequest struct {
    Email    string `form:"email"`
    Password string `form:"password"`
    Redirect string `form:"redirect"`
}

func (h *AuthHandler) Login(c echo.Context) error {
    var req LoginRequest

    // Bind form data
    if err := c.Bind(&req); err != nil {
        return c.Render(http.StatusOK, "login.html", map[string]interface{}{
            "error": "Dados inválidos",
            "email": req.Email,
        })
    }

    // Sanitize input
    req.Email = strings.TrimSpace(req.Email)

    // Validate required fields
    if req.Email == "" || req.Password == "" {
        return c.Render(http.StatusOK, "login.html", map[string]interface{}{
            "error": "Email e senha são obrigatórios",
            "email": req.Email,
        })
    }

    // Process request...
}
```

**Validation Best Practices:**
1. Trim whitespace from string inputs
2. Sanitize HTML to prevent XSS (use `html.EscapeString()`)
3. Check required fields
4. Validate format/complexity (e.g., password requirements)
5. Re-render form with error messages on validation failure
6. Preserve user input in error responses

#### Error Handling Pattern

Handlers handle errors gracefully and provide user-friendly messages:

```go
// Authenticate user
_, accessToken, refreshToken, err := h.authService.Login(req.Email, req.Password)
if err != nil {
    // Generic error for security (don't reveal if user exists)
    return c.Render(http.StatusOK, "login.html", map[string]interface{}{
        "error":    "Email ou senha incorretos",
        "email":    req.Email,
        "redirect": req.Redirect,
    })
}

// Check for specific error types
if errors.Is(err, services.ErrUserExists) {
    return c.Render(http.StatusOK, "register.html", map[string]interface{}{
        "error": "Este email já está cadastrado",
        "email": req.Email,
    })
}
```

#### Cookie Management

Handlers set secure cookies for authentication:

```go
// Set secure HTTP-only cookies
c.SetCookie(&http.Cookie{
    Name:     "access_token",
    Value:    accessToken,
    Path:     "/",
    HttpOnly: true,                                    // Prevent JavaScript access
    Secure:   isProduction(),                          // HTTPS only in production
    SameSite: http.SameSiteLaxMode,                   // CSRF protection
    MaxAge:   int(services.AccessTokenDuration.Seconds()),
})
```

**Cookie Security:**
- `HttpOnly: true` - Prevents XSS attacks
- `Secure: true` - HTTPS only (production)
- `SameSite: Lax` - CSRF protection
- Proper expiration times

#### Redirect with Security

Handlers validate redirect URLs to prevent open redirect vulnerabilities:

```go
// Redirect to specified URL or home (with open redirect protection)
redirectURL := "/"
if req.Redirect != "" &&
    strings.HasPrefix(req.Redirect, "/") &&           // Must start with /
    !strings.HasPrefix(req.Redirect, "//") &&         // Prevent protocol-relative URLs
    !strings.Contains(req.Redirect, "://") {          // Prevent absolute URLs
    redirectURL = req.Redirect
}
return c.Redirect(http.StatusSeeOther, redirectURL)
```

#### Template Rendering

**Full Page Rendering:**
```go
return c.Render(http.StatusOK, "dashboard.html", map[string]interface{}{
    "user":    user,
    "summary": summary,
})
```

**HTMX Partial Rendering:**
```go
// Return only a fragment for HTMX to swap into the page
return c.Render(http.StatusOK, "partials/income-list.html", map[string]interface{}{
    "incomes": incomes,
})
```

#### Complete Handler Example

Here's a complete registration handler demonstrating all patterns:

```go
func (h *AuthHandler) Register(c echo.Context) error {
    var req RegisterRequest

    // 1. Bind and validate
    if err := c.Bind(&req); err != nil {
        return c.Render(http.StatusOK, "register.html", map[string]interface{}{
            "error": "Dados inválidos",
            "email": req.Email,
            "name":  req.Name,
        })
    }

    // 2. Sanitize input
    req.Email = strings.TrimSpace(req.Email)
    req.Name = html.EscapeString(strings.TrimSpace(req.Name))

    // 3. Validate required fields
    if req.Email == "" || req.Password == "" || req.Name == "" {
        return c.Render(http.StatusOK, "register.html", map[string]interface{}{
            "error": "Todos os campos são obrigatórios",
            "email": req.Email,
            "name":  req.Name,
        })
    }

    // 4. Validate password strength
    if valid, errMsg := isValidPassword(req.Password); !valid {
        return c.Render(http.StatusOK, "register.html", map[string]interface{}{
            "error": errMsg,
            "email": req.Email,
            "name":  req.Name,
        })
    }

    // 5. Call service layer
    _, err := h.authService.Register(req.Email, req.Password, req.Name)
    if err != nil {
        if errors.Is(err, services.ErrUserExists) {
            return c.Render(http.StatusOK, "register.html", map[string]interface{}{
                "error": "Este email já está cadastrado",
                "email": req.Email,
                "name":  req.Name,
            })
        }
        return c.Render(http.StatusOK, "register.html", map[string]interface{}{
            "error": "Erro ao criar conta. Tente novamente.",
            "email": req.Email,
            "name":  req.Name,
        })
    }

    // 6. Redirect on success
    return c.Redirect(http.StatusSeeOther, "/login?registered=1")
}
```

#### Handler Route Registration

Handlers are registered in `cmd/server/main.go`:

```go
func main() {
    e := echo.New()

    // Create handlers
    authHandler := handlers.NewAuthHandler()
    incomeHandler := handlers.NewIncomeHandler()

    // Public routes
    e.GET("/login", authHandler.LoginPage)
    e.POST("/login", authHandler.Login)
    e.GET("/register", authHandler.RegisterPage)
    e.POST("/register", authHandler.Register)

    // Protected routes (with auth middleware)
    protected := e.Group("")
    protected.Use(authmw.AuthMiddleware)
    protected.GET("/", dashboardHandler.Dashboard)
    protected.POST("/income", incomeHandler.Create)
    protected.DELETE("/income/:id", incomeHandler.Delete)

    e.Start(":8080")
}
```

### 4. Services Layer

The **Services Layer** (Business Logic Layer) encapsulates complex business operations, coordinates multiple models, and implements domain logic that doesn't belong in handlers or models. Services sit between handlers and models, providing reusable business operations.

#### Purpose and Responsibilities

Services handle:
- **Complex Business Logic**: Multi-step operations, calculations, validations
- **Transaction Orchestration**: Coordinating multiple model operations atomically
- **Cross-Cutting Concerns**: Authentication, notifications, background jobs
- **Business Rule Enforcement**: Implementing domain-specific rules
- **External Integrations**: Third-party APIs, email, file storage

#### Service Structure Pattern

All services follow this structure:

```go
// Custom error definitions for business logic
var (
    ErrInvalidCredentials = errors.New("credenciais inválidas")
    ErrUserExists         = errors.New("email já cadastrado")
    ErrUserNotFound       = errors.New("usuário não encontrado")
)

// Configuration constants
const (
    AccessTokenDuration  = 15 * time.Minute
    RefreshTokenDuration = 7 * 24 * time.Hour
    BcryptCost          = 12
)

// Service struct (may hold dependencies)
type AuthService struct {
    // Dependencies injected via constructor (if needed)
    // db *gorm.DB
    // emailService *EmailService
}

// Constructor for dependency injection
func NewAuthService() *AuthService {
    return &AuthService{}
}
```

**Key Patterns:**
1. **Custom Errors**: Define business-specific errors for meaningful error handling
2. **Constants**: Centralize configuration values (timeouts, limits, defaults)
3. **Struct-based Services**: Organize related operations into a service
4. **Constructor Functions**: Use `NewXxxService()` for initialization

#### Real-World Example: Authentication Service

The `AuthService` demonstrates the services layer pattern with authentication and user management:

**1. Password Security Operations**
```go
// HashPassword creates a bcrypt hash of the password
func (s *AuthService) HashPassword(password string) (string, error) {
    hash, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
    if err != nil {
        return "", err
    }
    return string(hash), nil
}

// CheckPassword compares a password with its hash
func (s *AuthService) CheckPassword(password, hash string) bool {
    err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
    return err == nil
}
```

**Business Logic**: Encapsulates password hashing algorithm and cost factor, providing a consistent interface for password operations.

**2. Token Generation (JWT)**
```go
// GenerateAccessToken creates a new JWT access token
func (s *AuthService) GenerateAccessToken(user *models.User) (string, error) {
    claims := &Claims{
        UserID: user.ID,
        Email:  user.Email,
        RegisteredClaims: jwt.RegisteredClaims{
            ExpiresAt: jwt.NewNumericDate(time.Now().Add(AccessTokenDuration)),
            IssuedAt:  jwt.NewNumericDate(time.Now()),
            Subject:   user.Email,
        },
    }

    token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
    return token.SignedString(JWTSecret)
}
```

**Business Logic**: Handles JWT token creation with appropriate claims, expiration, and signing. Centralizes token generation logic used across authentication flows.

**3. Token Validation**
```go
// ValidateAccessToken validates and parses a JWT access token
func (s *AuthService) ValidateAccessToken(tokenString string) (*Claims, error) {
    token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
        if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
            return nil, ErrTokenInvalid
        }
        return JWTSecret, nil
    })

    if err != nil {
        if errors.Is(err, jwt.ErrTokenExpired) {
            return nil, ErrTokenExpired
        }
        return nil, ErrTokenInvalid
    }

    claims, ok := token.Claims.(*Claims)
    if !ok || !token.Valid {
        return nil, ErrTokenInvalid
    }

    return claims, nil
}
```

**Business Logic**: Validates JWT signature, expiration, and format. Returns custom errors for different failure scenarios.

**4. Coordinating Multiple Models (Transaction Orchestration)**
```go
// Register creates a new user account
func (s *AuthService) Register(email, password, name string) (*models.User, error) {
    // Check if user already exists
    var existingUser models.User
    if err := database.DB.Where("email = ?", email).First(&existingUser).Error; err == nil {
        return nil, ErrUserExists
    }

    // Hash password
    hash, err := s.HashPassword(password)
    if err != nil {
        return nil, err
    }

    // Create user
    user := &models.User{
        Email:        email,
        PasswordHash: hash,
        Name:         name,
    }

    if err := database.DB.Create(user).Error; err != nil {
        return nil, err
    }

    // Auto-create individual account for the user (private data by default)
    account := &models.Account{
        Name:   "Conta Pessoal",
        Type:   models.AccountTypeIndividual,
        UserID: user.ID,
    }

    if err := database.DB.Create(account).Error; err != nil {
        // Rollback user creation if account creation fails
        database.DB.Delete(user)
        return nil, err
    }

    return user, nil
}
```

**Business Logic**:
- **Multi-step Operation**: Check user existence → Hash password → Create user → Create default account
- **Transaction Orchestration**: Coordinates User and Account models
- **Error Handling**: Returns business-specific errors
- **Rollback Logic**: Manually rolls back user if account creation fails
- **Business Rule**: "Every new user gets a personal account automatically"

**5. Refresh Token Management (Database Coordination)**
```go
// GenerateRefreshToken creates a new refresh token and stores it in database
func (s *AuthService) GenerateRefreshToken(user *models.User) (string, error) {
    // Generate random token
    bytes := make([]byte, 32)
    if _, err := rand.Read(bytes); err != nil {
        return "", err
    }
    tokenString := hex.EncodeToString(bytes)

    // Store in database
    refreshToken := &models.RefreshToken{
        UserID:    user.ID,
        Token:     tokenString,
        ExpiresAt: time.Now().Add(RefreshTokenDuration),
    }

    if err := database.DB.Create(refreshToken).Error; err != nil {
        return "", err
    }

    return tokenString, nil
}

// ValidateRefreshToken validates a refresh token from database
func (s *AuthService) ValidateRefreshToken(tokenString string) (*models.User, error) {
    var refreshToken models.RefreshToken
    if err := database.DB.Where("token = ?", tokenString).Preload("User").First(&refreshToken).Error; err != nil {
        return nil, ErrTokenInvalid
    }

    if refreshToken.IsExpired() {
        // Delete expired token
        database.DB.Delete(&refreshToken)
        return nil, ErrTokenExpired
    }

    return &refreshToken.User, nil
}
```

**Business Logic**:
- **Token Generation**: Creates cryptographically secure random tokens
- **Database Coordination**: Stores tokens with expiration tracking
- **Automatic Cleanup**: Deletes expired tokens on validation attempt
- **Preloading**: Efficiently loads related User data

#### Service Layer Benefits

**1. Reusability**
Services can be called from multiple handlers:
```go
// Used in login handler
authService.Login(email, password)

// Used in password reset handler
authService.ResetPassword(token, newPassword)

// Used in token refresh handler
authService.RefreshTokens(refreshToken)
```

**2. Testability**
Services can be unit tested independently of HTTP layer:
```go
func TestAuthService_Register(t *testing.T) {
    service := services.NewAuthService()

    user, err := service.Register("test@example.com", "password123", "Test User")

    assert.NoError(t, err)
    assert.Equal(t, "test@example.com", user.Email)
    // Verify user and account were created...
}
```

**3. Separation of Concerns**
- **Handlers**: HTTP concerns (request/response, validation, rendering)
- **Services**: Business logic (authentication, calculations, orchestration)
- **Models**: Data persistence (CRUD, relationships, queries)

**4. Business Rule Centralization**
All business rules live in one place:
```go
// Business rule: Token expiration times
const AccessTokenDuration = 15 * time.Minute

// Business rule: Password requirements enforced
func (s *AuthService) ValidatePassword(password string) error {
    if len(password) < 8 {
        return errors.New("senha deve ter pelo menos 8 caracteres")
    }
    // Additional rules...
}
```

#### Other Service Examples in the Application

**AccountService** (`internal/services/account.go`)
- Calculates account balances
- Aggregates income/expense data
- Handles account ownership verification

**GroupService** (`internal/services/group.go`)
- Manages group invitations
- Handles member operations (add, remove)
- Coordinates expense splits

**NotificationService** (`internal/services/notification.go`)
- Creates notifications for events
- Handles notification delivery
- Manages notification preferences

**RecurringSchedulerService** (`internal/services/recurring_scheduler.go`)
- Background job processing
- Generates transactions from recurring templates
- Updates next run dates
- Sends automated notifications

**SummaryService** (`internal/services/summary.go`)
- Generates financial summaries
- Calculates category breakdowns
- Produces reports and analytics

#### When to Create a Service

Create a service when:
1. **Operation involves multiple models** (e.g., creating user + account)
2. **Complex business logic** (e.g., tax calculations, balance aggregations)
3. **Background processing** (e.g., schedulers, batch jobs)
4. **External integrations** (e.g., email, payment gateways)
5. **Reusable operations** (e.g., authentication used across many handlers)

Don't create a service for:
- Simple CRUD operations (use models directly)
- Single-model operations (belongs in model)
- Presentation logic (belongs in handlers/templates)

---

### 5. Models Layer and Database Patterns

The **Models Layer** (Data Access Layer) defines the database schema, handles data persistence, and manages relationships between entities. This layer uses GORM as the ORM to abstract database operations and provide a clean, idiomatic Go interface for data access.

#### Database Initialization

The database is initialized in `internal/database/database.go` during application startup:

```go
package database

import (
    "log"

    "gorm.io/driver/sqlite"
    "gorm.io/gorm"
    "gorm.io/gorm/logger"

    "poc-finance/internal/models"
)

var DB *gorm.DB

func Init() error {
    var err error
    DB, err = gorm.Open(sqlite.Open("finance.db"), &gorm.Config{
        Logger: logger.Default.LogMode(logger.Info),
    })
    if err != nil {
        return err
    }

    log.Println("Conectado ao banco de dados SQLite")

    // Auto migrate all models
    err = DB.AutoMigrate(
        &models.User{},
        &models.RefreshToken{},
        &models.PasswordResetToken{},
        &models.Account{},
        &models.Income{},
        &models.Expense{},
        // ... all other models
    )
    if err != nil {
        return err
    }

    // Initialize default settings
    initDefaultSettings()

    log.Println("Migrações executadas com sucesso")
    return nil
}

func GetDB() *gorm.DB {
    return DB
}
```

**Key Patterns:**
1. **Global DB Variable**: `DB` is a package-level variable accessible throughout the application
2. **Logger Configuration**: GORM logger is set to Info mode for query logging
3. **AutoMigrate**: Automatically creates/updates tables based on struct definitions
4. **Default Data Initialization**: Seeds initial configuration data after migration
5. **GetDB() Helper**: Provides access to the database connection

#### Model Structure Pattern

All models follow GORM conventions with consistent struct patterns:

```go
package models

import (
    "time"

    "gorm.io/gorm"
)

// Example: User model with standard GORM patterns
type User struct {
    gorm.Model                                    // Embeds ID, CreatedAt, UpdatedAt, DeletedAt
    Email        string `json:"email" gorm:"uniqueIndex;not null"`
    PasswordHash string `json:"-" gorm:"not null"`
    Name         string `json:"name" gorm:"not null"`
}

// TableName explicitly defines the table name
func (u *User) TableName() string {
    return "users"
}
```

**Standard Model Components:**

1. **gorm.Model Embedding**
   - Provides: `ID uint`, `CreatedAt time.Time`, `UpdatedAt time.Time`, `DeletedAt gorm.DeletedAt`
   - Enables soft deletes (records are marked deleted, not removed)
   - Automatically managed by GORM

2. **Struct Tags**
   - `json:"email"` - JSON serialization field name
   - `json:"-"` - Excludes field from JSON (e.g., password hashes)
   - `gorm:"uniqueIndex"` - Creates unique index on column
   - `gorm:"not null"` - Database NOT NULL constraint
   - `gorm:"index"` - Creates standard index for faster queries
   - `gorm:"default:false"` - Sets database default value
   - `gorm:"foreignKey:UserID"` - Defines foreign key relationship

3. **TableName() Method**
   - Explicitly defines the database table name
   - Overrides GORM's default pluralization
   - Ensures consistent naming across the application

#### Relationship Patterns

GORM supports various relationship types. Here are the patterns used in this application:

**1. Belongs To (Many-to-One)**

A child model references a parent through a foreign key:

```go
type RefreshToken struct {
    gorm.Model
    UserID    uint      `json:"user_id" gorm:"not null;index"`
    User      User      `json:"-" gorm:"foreignKey:UserID"`
    Token     string    `json:"token" gorm:"uniqueIndex;not null"`
    ExpiresAt time.Time `json:"expires_at" gorm:"not null"`
}
```

**Pattern:**
- `UserID uint` - Foreign key field (references User.ID)
- `User User` - Association field (holds the related User object)
- `gorm:"foreignKey:UserID"` - Explicitly defines the FK field
- `gorm:"index"` - Index on FK for faster joins
- `json:"-"` - Excludes user object from JSON (prevents circular refs)

**Usage:**
```go
// Query with preloading
var token RefreshToken
db.Preload("User").First(&token, id)
// token.User is now populated

// Query without preloading
db.First(&token, id)
// token.User is zero value, token.UserID is populated
```

**2. Has Many (One-to-Many)**

Defined implicitly through foreign keys in the child model. Example: User has many RefreshTokens.

```go
// Querying related records
var user User
db.Preload("RefreshTokens").First(&user, userID)
// user has implicit collection of tokens

// Alternative: Manual query
var tokens []RefreshToken
db.Where("user_id = ?", userID).Find(&tokens)
```

**3. Has Many Through (Many-to-Many with Join Table)**

For complex relationships like expense splits:

```go
type Expense struct {
    gorm.Model
    AccountID   uint            `gorm:"not null;index"`
    Amount      float64         `gorm:"not null"`
    Description string          `gorm:"not null"`
    Splits      []ExpenseSplit  `gorm:"foreignKey:ExpenseID"`
}

type ExpenseSplit struct {
    gorm.Model
    ExpenseID uint    `gorm:"not null;index"`
    UserID    uint    `gorm:"not null;index"`
    Amount    float64 `gorm:"not null"`
    Settled   bool    `gorm:"default:false"`
}
```

#### Model Helper Methods

Models can include helper methods for business logic and data validation:

```go
type RefreshToken struct {
    gorm.Model
    UserID    uint      `json:"user_id" gorm:"not null;index"`
    User      User      `json:"-" gorm:"foreignKey:UserID"`
    Token     string    `json:"token" gorm:"uniqueIndex;not null"`
    ExpiresAt time.Time `json:"expires_at" gorm:"not null"`
}

// Helper method for token expiration check
func (r *RefreshToken) IsExpired() bool {
    return time.Now().After(r.ExpiresAt)
}
```

```go
type PasswordResetToken struct {
    gorm.Model
    UserID    uint      `json:"user_id" gorm:"not null;index"`
    User      User      `json:"-" gorm:"foreignKey:UserID"`
    Token     string    `json:"token" gorm:"uniqueIndex;not null"`
    ExpiresAt time.Time `json:"expires_at" gorm:"not null"`
    Used      bool      `json:"used" gorm:"default:false"`
}

// Helper method combining multiple checks
func (p *PasswordResetToken) IsExpired() bool {
    return time.Now().After(p.ExpiresAt)
}
```

**Helper Method Guidelines:**
- Keep methods simple and focused on single responsibility
- Use for data validation, state checks, and computed properties
- Don't include database operations (belongs in services/handlers)
- Return primitive types or simple results

#### Database Query Patterns

**1. Basic CRUD Operations**

```go
// Create
user := &models.User{Email: "test@example.com", Name: "Test"}
database.DB.Create(user)

// Read by ID
var user models.User
database.DB.First(&user, id)

// Read by condition
var user models.User
database.DB.Where("email = ?", email).First(&user)

// Update
database.DB.Model(&user).Update("name", "New Name")
// or
database.DB.Model(&user).Updates(map[string]interface{}{"name": "New Name", "email": "new@example.com"})

// Delete (soft delete with gorm.Model)
database.DB.Delete(&user, id)

// Permanent delete
database.DB.Unscoped().Delete(&user, id)
```

**2. Preloading Relationships**

```go
// Preload single relationship
var token models.RefreshToken
database.DB.Preload("User").First(&token, id)

// Preload nested relationships
var expense models.Expense
database.DB.Preload("Splits").Preload("Splits.User").First(&expense, id)

// Conditional preloading
database.DB.Preload("Splits", "settled = ?", false).First(&expense, id)
```

**3. Complex Queries**

```go
// Filtering with multiple conditions
var expenses []models.Expense
database.DB.Where("account_id = ? AND paid = ?", accountID, true).
    Order("date DESC").
    Limit(10).
    Find(&expenses)

// Date range queries
database.DB.Where("date BETWEEN ? AND ?", startDate, endDate).
    Find(&expenses)

// Aggregations
var total float64
database.DB.Model(&models.Expense{}).
    Where("account_id = ?", accountID).
    Select("SUM(amount)").
    Scan(&total)

// Counting records
var count int64
database.DB.Model(&models.Expense{}).
    Where("account_id = ?", accountID).
    Count(&count)
```

**4. Transaction Management**

```go
// Manual transaction
tx := database.DB.Begin()

user := &models.User{Email: email, Name: name}
if err := tx.Create(user).Error; err != nil {
    tx.Rollback()
    return err
}

account := &models.Account{UserID: user.ID, Name: "Personal"}
if err := tx.Create(account).Error; err != nil {
    tx.Rollback()
    return err
}

tx.Commit()

// Automatic transaction (preferred)
err := database.DB.Transaction(func(tx *gorm.DB) error {
    if err := tx.Create(&user).Error; err != nil {
        return err
    }
    if err := tx.Create(&account).Error; err != nil {
        return err
    }
    return nil
})
```

#### Migration Strategy

The application uses **GORM AutoMigrate** for schema management:

**Benefits:**
- Automatic schema synchronization with model definitions
- Safe migrations (adds columns/tables, doesn't drop data)
- No separate migration files to maintain
- Works across different databases (SQLite, PostgreSQL, MySQL)

**How it Works:**
1. On application startup, `database.Init()` is called
2. AutoMigrate compares model structs to database schema
3. Missing tables are created
4. Missing columns are added
5. Indexes are created/updated
6. Existing data is preserved

**Limitations:**
- Won't remove columns (by design, for safety)
- Won't modify column types automatically
- Complex schema changes require manual SQL

**For Complex Migrations:**
```go
// Add custom migration in database.Init() after AutoMigrate
if !database.DB.Migrator().HasColumn(&models.Expense{}, "new_column") {
    database.DB.Migrator().AddColumn(&models.Expense{}, "new_column")
}
```

#### Default Data Initialization

After migrations, default settings are initialized:

```go
func initDefaultSettings() {
    defaults := map[string]string{
        models.SettingProLabore:   "0",       // Pró-labore não configurado
        models.SettingINSSCeiling: "7786.02", // Teto INSS 2024
        models.SettingINSSRate:    "11",      // 11%
    }

    for key, value := range defaults {
        var setting models.Settings
        result := DB.Where("key = ?", key).First(&setting)
        if result.Error != nil {
            DB.Create(&models.Settings{Key: key, Value: value})
        }
    }
}
```

**Pattern**: Check if record exists before creating (idempotent initialization)

#### Model Organization

Models are organized by domain in `internal/models/`:

```
internal/models/
├── user.go                    # Authentication models (User, RefreshToken, PasswordResetToken)
├── account.go                 # Financial accounts
├── income.go                  # Income transactions
├── expense.go                 # Expense transactions
├── credit_card.go             # Credit card entities
├── installment.go             # Installment payments
├── recurring_transaction.go   # Recurring transaction templates
├── group.go                   # Family groups (FamilyGroup, GroupMember, GroupInvite)
├── expense_split.go           # Expense sharing/splits
├── goal.go                    # Financial goals (GroupGoal, GoalContribution)
├── bill.go                    # Bill tracking
├── notification.go            # User notifications
├── settings.go                # User settings (key-value store)
└── expense_payment.go         # Expense payment records
```

**Organization Principles:**
- One file per primary domain entity
- Related models grouped together (e.g., User, RefreshToken, PasswordResetToken in user.go)
- Test files colocated as `*_test.go`

#### Best Practices

**1. Always Use Struct Tags**
```go
// Good: Explicit constraints and JSON mapping
Email string `json:"email" gorm:"uniqueIndex;not null"`

// Bad: No constraints or JSON control
Email string
```

**2. Define TableName() for Consistency**
```go
// Good: Explicit table name
func (u *User) TableName() string {
    return "users"
}

// Bad: Relying on GORM's pluralization (can be unpredictable)
```

**3. Use json:"-" for Sensitive/Relational Data**
```go
// Good: Hide password and prevent circular JSON
PasswordHash string `json:"-"`
User         User   `json:"-" gorm:"foreignKey:UserID"`

// Bad: Exposes sensitive data or causes circular references
PasswordHash string `json:"password_hash"`
User         User   `json:"user"`
```

**4. Index Foreign Keys**
```go
// Good: Fast joins and queries
UserID uint `gorm:"not null;index"`

// Bad: Slow queries on large tables
UserID uint `gorm:"not null"`
```

**5. Use Preload for Related Data**
```go
// Good: Single optimized query
database.DB.Preload("User").Find(&tokens)

// Bad: N+1 query problem
database.DB.Find(&tokens)
for i := range tokens {
    database.DB.First(&tokens[i].User, tokens[i].UserID)
}
```

---

### 6. HTMX Partial Rendering

The application uses HTMX for dynamic updates without full page reloads:

```
User Action (Click/Submit)
         ↓
HTMX sends HTTP request
         ↓
Handler processes request
         ↓
Handler returns partial HTML (not full page)
         ↓
HTMX swaps partial into DOM
```

**Benefits:**
- Fast, responsive UI
- Reduced bandwidth usage
- Server-side rendering (no complex JavaScript)
- Progressive enhancement

### 6. Background Scheduler

Recurring transactions are processed by a background goroutine:

```
Application Startup
         ↓
Launch scheduler goroutine
         ↓
Run immediately (process overdue transactions)
         ↓
Wait until midnight
         ↓
Run every 24 hours (check for due transactions)
```

**Scheduler Operations:**
1. Find recurring transactions where `next_run_date <= today` and `active = true`
2. Create corresponding expense/income transactions
3. Update `next_run_date` based on frequency
4. Send notifications to users
5. Deactivate if `end_date` is reached

---

## Request Flow Example

**Example: Creating a Recurring Transaction**

```
1. User fills form and clicks "Create" button
   ↓
2. HTMX sends POST /recurring with form data
   ↓
3. Request passes through middleware chain:
   - Logger logs the request
   - CSRF validates token
   - AuthMiddleware validates JWT and loads user
   ↓
4. recurringHandler.Create() is invoked
   - Extracts and validates form data
   - Calls model to create database record
   - Returns HTMX partial HTML (new row in list)
   ↓
5. HTMX receives partial HTML response
   ↓
6. HTMX swaps new row into the transactions list
   ↓
7. User sees new transaction appear (no page reload)
```

---

## Database Schema Overview

### Core Tables

**users**
- Authentication and profile information
- Fields: id, email, password_hash, name, created_at

**accounts**
- Financial accounts (personal or joint)
- Fields: id, name, type, group_id, created_at
- Relationships: belongs to user/group, has many transactions

**incomes**
- Income transactions
- Fields: id, account_id, amount, description, date, category
- Relationships: belongs to account

**expenses**
- Expense transactions
- Fields: id, account_id, amount, description, date, category, paid
- Relationships: belongs to account, has many splits

**credit_cards**
- Credit card entities
- Fields: id, account_id, name, limit, closing_day, due_day
- Relationships: belongs to account, has many installments

**installments**
- Credit card purchases split into monthly payments
- Fields: id, card_id, description, total_amount, installments, current
- Relationships: belongs to credit_card

**recurring_transactions**
- Templates for recurring transactions
- Fields: id, account_id, transaction_type, frequency, amount, description, start_date, end_date, next_run_date, active
- Relationships: belongs to account

**groups**
- Family groups for shared finances
- Fields: id, name, owner_id, invite_code, created_at
- Relationships: has many members, has many accounts, has many goals

**expense_splits**
- Split expenses among group members
- Fields: id, expense_id, user_id, amount, settled
- Relationships: belongs to expense and user

**goals**
- Financial goals
- Fields: id, group_id, name, target_amount, current_amount, deadline
- Relationships: belongs to group

**notifications**
- User notifications
- Fields: id, user_id, title, message, type, read, created_at
- Relationships: belongs to user

---

## Security Features

### Authentication & Authorization
- **JWT Tokens**: Secure, stateless authentication
- **bcrypt**: Password hashing with salt
- **Session Cookies**: HttpOnly, SameSite=Lax
- **Auth Middleware**: Validates JWT on protected routes
- **Account Ownership**: Users can only access their own data

### HTTP Security
- **CSRF Protection**: Header-based tokens for HTMX compatibility
- **Security Headers**: XSS protection, content type sniffing prevention, frame options
- **Content Security Policy**: Restricts resource loading
- **HSTS**: Forces HTTPS (max-age 1 year)
- **Body Size Limit**: 2MB to prevent large payloads
- **Rate Limiting**: 5 requests/second on auth endpoints

### Data Validation
- **Input Sanitization**: Handler-level validation
- **SQL Injection Prevention**: GORM parameterized queries
- **Access Control**: Verify ownership before updates/deletes

---

## Testing Strategy

### Test Levels

**1. Unit Tests (Models)**
- Test CRUD operations
- Test data validation
- Test model methods
- Run with: `go test ./internal/models/*_test.go -v`

**2. Integration Tests (Handlers)**
- Test HTTP endpoints
- Test authentication flows
- Test HTMX responses
- Run with: `go test ./internal/handlers/*_test.go -v`

**3. Service Tests (Business Logic)**
- Test complex operations
- Test scheduler processing
- Test notification generation
- Run with: `go test ./internal/services/*_test.go -v`

**4. End-to-End Tests**
- Test complete user workflows
- Test browser interactions
- Automated scripts in `test_*_e2e.sh`

### Running Tests

```bash
# Run all tests
go test ./... -v

# Run specific package tests
go test ./internal/models -v
go test ./internal/handlers -v
go test ./internal/services -v

# Run with coverage
go test ./... -cover

# Run E2E tests
./test_recurring_e2e.sh
```

---

## Deployment Considerations

### Running the Application

```bash
# Development
go run ./cmd/server

# Build binary
go build -o poc-finance ./cmd/server

# Run production binary
./poc-finance
```

**Server Configuration:**
- Default port: 8080
- Database: finance.db (SQLite file)
- Automatic migrations on startup
- Background scheduler starts automatically

### Environment Setup

**Required:**
- Go 1.25.5 or higher
- SQLite (embedded, no separate installation needed)

**Optional:**
- Reverse proxy (nginx/caddy) for HTTPS
- Process manager (systemd/supervisord) for daemon mode

---

## Future Extensibility

### Adding New Features

**1. New Model (Data Entity)**
```
1. Create ./internal/models/new_feature.go
2. Define GORM struct with relationships
3. Add model to database.AutoMigrate in database.go
4. Write unit tests in new_feature_test.go
```

**2. New Handler (HTTP Endpoint)**
```
1. Create handler in ./internal/handlers/new_feature.go
2. Define routes in cmd/server/main.go
3. Create HTML template in ./internal/templates/
4. Write integration tests
```

**3. New Service (Business Logic)**
```
1. Create service in ./internal/services/new_feature.go
2. Inject dependencies via constructor
3. Call from handlers as needed
4. Write service tests
```

### Scaling Considerations

**Current State**: Single-server SQLite application

**Potential Improvements:**
- **Database**: Migrate to PostgreSQL for multi-user concurrency
- **Caching**: Add Redis for session storage and query caching
- **File Storage**: Use S3 for file uploads (reports, attachments)
- **Background Jobs**: Use queue (RabbitMQ/Redis) for scheduler
- **API Mode**: Add JSON API alongside HTML responses
- **Frontend**: Build SPA (React/Vue) consuming JSON API

---

## Key Design Decisions

### Why Echo Framework?
- High performance (one of the fastest Go web frameworks)
- Clean, expressive routing API
- Rich middleware ecosystem
- Great documentation

### Why GORM?
- Mature, battle-tested ORM
- Automatic migrations
- Relationship management
- Active development and community

### Why HTMX?
- Simplicity: No complex JavaScript build process
- Server-side rendering: Easier to reason about
- Progressive enhancement: Works without JavaScript
- Fast development: Reuse existing templates

### Why SQLite?
- Zero configuration
- Perfect for single-server deployments
- File-based: Easy backups
- Fast for read-heavy workloads
- Easy to migrate to PostgreSQL later

### Why Layered Architecture?
- **Separation of Concerns**: Each layer has clear responsibilities
- **Testability**: Test each layer independently
- **Maintainability**: Easy to locate and modify code
- **Flexibility**: Swap implementations (e.g., switch databases)

---

## Common Operations

### Adding a New Route

```go
// In cmd/server/main.go

// 1. Create handler
newHandler := handlers.NewFeatureHandler()

// 2. Register route (protected or public)
protected.GET("/new-feature", newHandler.List)
protected.POST("/new-feature", newHandler.Create)
```

### Creating a Migration

GORM AutoMigrate handles migrations automatically:

```go
// In internal/database/database.go
DB.AutoMigrate(
    &models.User{},
    &models.NewModel{}, // Add new model here
)
```

### Rendering a Template

```go
// Full page render
return c.Render(http.StatusOK, "page.html", data)

// HTMX partial render
return c.Render(http.StatusOK, "partials/fragment.html", data)
```

---

## Troubleshooting

### Server Won't Start
- Check if port 8080 is in use: `lsof -i :8080`
- Verify database file permissions
- Check for compile errors: `go build ./cmd/server`

### Database Migration Issues
- Delete `finance.db` and restart (dev only)
- Check GORM migration logs in server output
- Verify model struct tags are correct

### Template Not Found
- Ensure template is registered in `loadTemplates()`
- Check file path matches `internal/templates/`
- Verify template name matches render call

### Authentication Issues
- Check JWT token expiration
- Verify CSRF token is included in requests
- Clear browser cookies and re-login

---

## Resources

### Documentation Files
- **ARCHITECTURE.md** (this file) - System architecture overview
- **TESTING_GUIDE.md** - Comprehensive testing instructions
- **README.md** - Project setup and usage guide

### Key Code Files
- **cmd/server/main.go** - Application entry point and route configuration
- **internal/database/database.go** - Database initialization and migrations
- **internal/middleware/auth.go** - Authentication middleware

### External Documentation
- [Echo Framework](https://echo.labstack.com/)
- [GORM ORM](https://gorm.io/)
- [HTMX](https://htmx.org/)
- [Tailwind CSS](https://tailwindcss.com/)

---

**Maintained by**: Development Team
**Last Updated**: 2026-01-19
**Version**: 1.0
