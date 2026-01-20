# 💰 POC Finance

**Personal Finance Management System**

A full-stack personal and family finance tracker built with Go that helps users manage income, expenses, credit cards, recurring transactions, and family group finances.

[![Go Version](https://img.shields.io/badge/Go-1.25.5-blue.svg)](https://golang.org)
[![Production Ready](https://img.shields.io/badge/Status-Production%20Ready-green.svg)](https://github.com)
[![License](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

---

## 📋 Table of Contents

- [Features](#-features)
- [Tech Stack](#-tech-stack)
- [Prerequisites](#-prerequisites)
- [Installation](#-installation)
- [Running the Application](#-running-the-application)
- [Testing](#-testing)
- [Project Structure](#-project-structure)
- [Architecture](#-architecture)
- [Contributing](#-contributing)
- [License](#-license)

---

## ✨ Features

### Core Functionality
- 💰 **Income & Expense Tracking** - Categorize and track all transactions with detailed metadata
- 💳 **Credit Card Management** - Manage multiple cards with installment tracking and payment schedules
- 🔄 **Recurring Transactions** - Automated daily/weekly/monthly/yearly recurring transactions with scheduler
- 📊 **Financial Dashboard** - Real-time overview of balances, spending trends, and upcoming payments

### Family & Group Features
- 👥 **Family Groups** - Create groups with invite codes for family finance management
- 🤝 **Joint Accounts** - Shared accounts for family members with collaborative tracking
- ✂️ **Expense Splitting** - Split expenses among family members with configurable ratios
- 📈 **Group Dashboard** - Consolidated view of family finances with weekly/monthly summaries

### Advanced Features
- 🎯 **Financial Goals** - Set goals with progress tracking and contribution history
- 🔔 **Real-time Notifications** - Stay updated on transactions, goals, and recurring payments
- 📤 **Excel Export** - Export yearly financial data for external analysis
- 💼 **Brazilian Tax Support** - Built-in considerations for Brazilian tax calculations
- 🔐 **Secure Authentication** - JWT-based auth with bcrypt password hashing and CSRF protection

---

## 🛠 Tech Stack

### Backend
- **Language**: [Go 1.25.5](https://golang.org)
- **Web Framework**: [Echo v4](https://echo.labstack.com) - High-performance HTTP router
- **ORM**: [GORM v1.31.1](https://gorm.io) - Database abstraction layer
- **Database**: SQLite (via `gorm.io/driver/sqlite`)
- **Authentication**: JWT ([golang-jwt/jwt/v5](https://github.com/golang-jwt/jwt))
- **Password Hashing**: bcrypt (`golang.org/x/crypto`)

### Frontend
- **Template Engine**: Go `html/template` (server-side rendering)
- **Interactivity**: [HTMX](https://htmx.org) - Dynamic updates without full page reloads
- **Styling**: [Tailwind CSS](https://tailwindcss.com)
- **Icons**: [Bootstrap Icons](https://icons.getbootstrap.com)

### Additional Libraries
- **Excel Export**: [excelize v2](https://github.com/xuri/excelize)
- **Security**: Echo middleware (CSRF, rate limiting, security headers)

---

## 📦 Prerequisites

Before you begin, ensure you have the following installed:

- **Go 1.25.5 or higher** - [Download here](https://golang.org/dl/)
- **Git** - [Download here](https://git-scm.com/downloads)
- **Make** (optional but recommended) - For using Makefile commands

To verify your installation:

```bash
go version  # Should output: go version go1.25.5 or higher
git --version
make --version  # Optional
```

---

## 🚀 Installation

### 1. Clone the Repository

```bash
git clone <repository-url>
cd poc-finance
```

### 2. Install Dependencies

```bash
go mod download
```

This will download all required Go modules defined in `go.mod`:
- Echo v4 web framework
- GORM ORM and SQLite driver
- JWT authentication library
- Excelize for Excel exports
- And all transitive dependencies

### 3. Verify Installation

```bash
go mod verify
```

All dependencies should be downloaded and verified successfully.

---

## 🏃 Running the Application

### Development Mode

**Option 1: Using Make (Recommended)**
```bash
make run
```

**Option 2: Using Go directly**
```bash
go run ./cmd/server
```

The application will:
1. Initialize the SQLite database (`poc-finance.db`)
2. Run automatic migrations to create tables
3. Start the recurring transaction scheduler (runs daily at midnight)
4. Launch the web server on `http://localhost:8080`

You should see:
```
Starting recurring transaction scheduler...
Servidor iniciado em http://localhost:8080
```

### Production Build

Build an optimized binary:

```bash
make build
# OR
go build -o bin/poc-finance ./cmd/server
```

Run the production binary:

```bash
./bin/poc-finance
```

### First Time Setup

1. **Open your browser** to [http://localhost:8080](http://localhost:8080)
2. **Register a new account** - Click "Register" and create your user
3. **Start tracking** - Begin adding income, expenses, and exploring features!

---

## 🧪 Testing

The project has **comprehensive test coverage** across models, handlers, services, and middleware.

### Run All Tests

```bash
make test
# OR
go test ./...
```

### Run Tests with Verbose Output

```bash
make test-verbose
# OR
go test -v ./...
```

### Run Tests with Coverage Report

```bash
make test-coverage
```

This generates:
- `coverage.out` - Coverage data file
- `coverage.html` - HTML coverage report (open in browser)

### Run Tests with Race Detection

```bash
make test-race
# OR
go test -race ./...
```

### Run Specific Test Suites

```bash
# Test only models
make test-services

# Test only handlers
make test-handlers

# Test only middleware
make test-middleware
```

### Test Coverage Summary

```bash
make coverage
```

Expected output:
```
Total coverage: XX.X%
```

### Manual Testing

For detailed manual testing instructions for specific features, see:
- **[TESTING_GUIDE.md](TESTING_GUIDE.md)** - Step-by-step testing scenarios

---

## 📁 Project Structure

```
poc-finance/
│
├── cmd/
│   └── server/
│       └── main.go                    # Application entry point
│
├── internal/
│   ├── database/
│   │   └── database.go                # Database initialization & migrations
│   │
│   ├── models/                        # Data Access Layer (GORM models)
│   │   ├── user.go                    # User authentication
│   │   ├── account.go                 # Financial accounts
│   │   ├── income.go                  # Income transactions
│   │   ├── expense.go                 # Expense transactions
│   │   ├── credit_card.go             # Credit cards
│   │   ├── installment.go             # Card installments
│   │   ├── recurring_transaction.go   # Recurring transactions
│   │   ├── group.go                   # Family groups
│   │   ├── expense_split.go           # Expense splitting
│   │   ├── notification.go            # Notifications
│   │   ├── goal.go                    # Financial goals
│   │   └── *_test.go                  # Model tests
│   │
│   ├── handlers/                      # Presentation Layer (HTTP controllers)
│   │   ├── auth.go                    # Authentication endpoints
│   │   ├── dashboard.go               # Dashboard views
│   │   ├── income.go                  # Income CRUD
│   │   ├── expense.go                 # Expense CRUD
│   │   ├── credit_card.go             # Card management
│   │   ├── recurring_transaction.go   # Recurring transaction CRUD
│   │   ├── group.go                   # Group management
│   │   ├── account.go                 # Account operations
│   │   ├── goal.go                    # Goal management
│   │   ├── notification.go            # Notification handling
│   │   ├── settings.go                # User settings
│   │   ├── export.go                  # Excel export
│   │   └── *_test.go                  # Handler tests
│   │
│   ├── services/                      # Business Logic Layer
│   │   ├── auth.go                    # JWT token management
│   │   ├── account.go                 # Balance calculations
│   │   ├── group.go                   # Group operations
│   │   ├── goal.go                    # Goal tracking
│   │   ├── notification.go            # Notification service
│   │   ├── recurring_scheduler.go     # Background scheduler
│   │   ├── summary.go                 # Financial reports
│   │   └── *_test.go                  # Service tests
│   │
│   ├── middleware/
│   │   ├── auth.go                    # JWT authentication middleware
│   │   └── auth_test.go               # Middleware tests
│   │
│   └── templates/                     # HTML templates (HTMX + Tailwind)
│       ├── base.html                  # Base layout
│       ├── dashboard.html             # Dashboard page
│       ├── login.html                 # Login page
│       ├── register.html              # Registration page
│       └── partials/                  # HTMX partial fragments
│
├── go.mod                             # Go module definition
├── go.sum                             # Dependency checksums
├── Makefile                           # Development commands
├── ARCHITECTURE.md                    # Detailed architecture docs
├── TESTING_GUIDE.md                   # Feature testing guide
└── README.md                          # This file
```

---

## 🏗 Architecture

POC Finance follows a **clean 4-layer architecture** pattern with clear separation of concerns:

```
┌─────────────────────────────────────────────┐
│          Presentation Layer                 │
│  (HTTP Handlers + HTML Templates)           │
└─────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────┐
│          Business Logic Layer               │
│  (Services)                                 │
└─────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────┐
│          Data Access Layer                  │
│  (Models + GORM)                            │
└─────────────────────────────────────────────┘
                    ↓
┌─────────────────────────────────────────────┐
│          Database Layer                     │
│  (SQLite)                                   │
└─────────────────────────────────────────────┘
```

### Key Architectural Principles

1. **Separation of Concerns** - Each layer has a single, well-defined responsibility
2. **Dependency Direction** - Dependencies flow downward (Handlers → Services → Models → Database)
3. **Testability** - Each layer can be tested independently with mocks
4. **SOLID Principles** - Following Single Responsibility, Open/Closed, and Dependency Inversion

For a detailed architecture overview, see **[ARCHITECTURE.md](ARCHITECTURE.md)**.

---

## 🤝 Contributing

Contributions are welcome! This project follows standard Go conventions and clean architecture patterns.

### Development Workflow

1. **Fork the repository** and clone your fork
2. **Create a feature branch**: `git checkout -b feature/my-new-feature`
3. **Make your changes** following the existing code patterns
4. **Write or update tests** for your changes
5. **Run tests**: `make test`
6. **Run race detection**: `make test-race`
7. **Commit your changes**: `git commit -am 'Add some feature'`
8. **Push to the branch**: `git push origin feature/my-new-feature`
9. **Submit a pull request**

### Code Style Guidelines

- **Follow Go conventions** - Use `gofmt` and `golint`
- **Write tests** - Maintain or improve test coverage
- **Document public APIs** - Add comments for exported functions
- **Use meaningful names** - Clear, descriptive variable and function names
- **Keep functions small** - Each function should do one thing well
- **Error handling** - Always handle errors explicitly, never ignore them

### Testing Requirements

All pull requests must:
- ✅ Pass all existing tests (`go test ./...`)
- ✅ Include tests for new functionality
- ✅ Pass race detection (`go test -race ./...`)
- ✅ Not decrease overall test coverage

---

## 📄 License

This project is licensed under the **MIT License** - see the [LICENSE](LICENSE) file for details.

---

## 📚 Additional Documentation

- **[ARCHITECTURE.md](ARCHITECTURE.md)** - Comprehensive architecture documentation
- **[TESTING_GUIDE.md](TESTING_GUIDE.md)** - Detailed testing scenarios and guides

---

## 🆘 Support

If you encounter any issues or have questions:

1. Check the [ARCHITECTURE.md](ARCHITECTURE.md) for architectural details
2. Review the [TESTING_GUIDE.md](TESTING_GUIDE.md) for testing examples
3. Search existing issues in the repository
4. Create a new issue with detailed information

---

**Built with ❤️ using Go, Echo, GORM, and HTMX**
