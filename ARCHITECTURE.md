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

### 3. HTMX Partial Rendering

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

### 4. Background Scheduler

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
