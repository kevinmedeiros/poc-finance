package services

import (
	"database/sql"
	"testing"
	"time"

	"gorm.io/gorm"

	"poc-finance/internal/models"
	"poc-finance/internal/testutil"
)

// QueryCounter wraps a GORM DB and counts executed queries
type QueryCounter struct {
	db          *gorm.DB
	queryCount  int
	queryCounts map[string]int // Track counts per query type
}

// NewQueryCounter creates a query counter wrapper
func NewQueryCounter(db *gorm.DB) *QueryCounter {
	return &QueryCounter{
		db:          db,
		queryCount:  0,
		queryCounts: make(map[string]int),
	}
}

// Execute wraps GORM callbacks to count queries
func (qc *QueryCounter) WrapDB() *gorm.DB {
	db := qc.db.Session(&gorm.Session{})

	// Register callback to count queries
	db.Callback().Query().Before("gorm:query").Register("query_counter:before", func(db *gorm.DB) {
		qc.queryCount++
	})

	db.Callback().Create().Before("gorm:create").Register("query_counter:create", func(db *gorm.DB) {
		qc.queryCount++
	})

	db.Callback().Update().Before("gorm:update").Register("query_counter:update", func(db *gorm.DB) {
		qc.queryCount++
	})

	db.Callback().Delete().Before("gorm:delete").Register("query_counter:delete", func(db *gorm.DB) {
		qc.queryCount++
	})

	return db
}

// TestAnalyticsPerformance_BatchOptimization tests that analytics functions use batch queries
func TestAnalyticsPerformance_BatchOptimization(t *testing.T) {
	db := testutil.SetupTestDB()

	// Setup test data
	user := testutil.CreateTestUser(db, "test@example.com", "Test User", "hash")
	account1 := testutil.CreateTestAccount(db, "Account 1", models.AccountTypeIndividual, user.ID, nil)
	account2 := testutil.CreateTestAccount(db, "Account 2", models.AccountTypeIndividual, user.ID, nil)
	accountIDs := []uint{account1.ID, account2.ID}

	// Create realistic data across 6 months
	for month := 1; month <= 6; month++ {
		// Create incomes
		db.Create(&models.Income{
			AccountID:   account1.ID,
			Date:        time.Date(2024, time.Month(month), 15, 0, 0, 0, 0, time.Local),
			GrossAmount: 5000.00,
			TaxAmount:   500.00,
			NetAmount:   4500.00,
			Description: "Salary",
		})

		db.Create(&models.Income{
			AccountID:   account2.ID,
			Date:        time.Date(2024, time.Month(month), 15, 0, 0, 0, 0, time.Local),
			GrossAmount: 3000.00,
			TaxAmount:   300.00,
			NetAmount:   2700.00,
			Description: "Salary",
		})

		// Create variable expenses with categories
		categories := []string{"Food", "Transportation", "Entertainment"}
		for _, category := range categories {
			expense := &models.Expense{
				AccountID: account1.ID,
				Name:      category + " Expense",
				Category:  category,
				Amount:    200.00,
				Type:      models.ExpenseTypeVariable,
				Active:    true,
			}
			db.Create(expense)
			db.Model(expense).Update("created_at", time.Date(2024, time.Month(month), 20, 0, 0, 0, 0, time.Local))
		}
	}

	// Create fixed expenses
	db.Create(&models.Expense{
		AccountID: account1.ID,
		Name:      "Rent",
		Amount:    1000.00,
		Type:      models.ExpenseTypeFixed,
		Active:    true,
	})

	db.Create(&models.Expense{
		AccountID: account2.ID,
		Name:      "Utilities",
		Amount:    500.00,
		Type:      models.ExpenseTypeFixed,
		Active:    true,
	})

	// Create credit card with installments
	creditCard := &models.CreditCard{
		AccountID:  account1.ID,
		Name:       "Test Card",
		ClosingDay: 15,
		DueDay:     25,
	}
	db.Create(creditCard)

	db.Create(&models.Installment{
		CreditCardID:      creditCard.ID,
		Description:       "Purchase",
		TotalAmount:       600.00,
		InstallmentAmount: 100.00,
		TotalInstallments: 6,
		StartDate:         time.Date(2024, 1, 1, 0, 0, 0, 0, time.Local),
	})

	// Create bills
	for month := 1; month <= 6; month++ {
		db.Create(&models.Bill{
			AccountID: account1.ID,
			Name:      "Electric",
			Amount:    150.00,
			DueDate:   time.Date(2024, time.Month(month), 10, 0, 0, 0, 0, time.Local),
		})
	}

	t.Run("GetMonthOverMonthComparison uses batch queries", func(t *testing.T) {
		// Create a query counter
		qc := NewQueryCounter(db)
		wrappedDB := qc.WrapDB()

		// Execute the analytics function
		_ = GetMonthOverMonthComparison(wrappedDB, 2024, 2, accountIDs)

		// Verify query count
		// Expected: 5 batch queries (incomes, fixed, variable, cards, bills)
		// vs 10 queries without batching (5 per month * 2 months)
		t.Logf("GetMonthOverMonthComparison executed %d queries", qc.queryCount)

		// Should be significantly less than 10
		if qc.queryCount > 8 {
			t.Errorf("GetMonthOverMonthComparison used %d queries, expected ≤8 with batch optimization", qc.queryCount)
		}

		if qc.queryCount < 3 {
			t.Errorf("GetMonthOverMonthComparison used %d queries, expected ≥3 (seems suspiciously low)", qc.queryCount)
		}
	})

	t.Run("GetCategoryBreakdownWithPercentages uses single query", func(t *testing.T) {
		qc := NewQueryCounter(db)
		wrappedDB := qc.WrapDB()

		_ = GetCategoryBreakdownWithPercentages(wrappedDB, 2024, 2, accountIDs)

		t.Logf("GetCategoryBreakdownWithPercentages executed %d queries", qc.queryCount)

		// Should be 1 query (single aggregated query with GROUP BY)
		if qc.queryCount > 2 {
			t.Errorf("GetCategoryBreakdownWithPercentages used %d queries, expected ≤2", qc.queryCount)
		}
	})

	t.Run("GetIncomeVsExpenseTrend uses batch queries for 6 months", func(t *testing.T) {
		qc := NewQueryCounter(db)
		wrappedDB := qc.WrapDB()

		_ = GetIncomeVsExpenseTrend(wrappedDB, 6, accountIDs)

		t.Logf("GetIncomeVsExpenseTrend (6 months) executed %d queries", qc.queryCount)

		// Expected: 5 batch queries (incomes, fixed, variable, cards, bills)
		// vs 30 queries without batching (5 per month * 6 months)
		if qc.queryCount > 8 {
			t.Errorf("GetIncomeVsExpenseTrend used %d queries for 6 months, expected ≤8 with batch optimization", qc.queryCount)
		}

		if qc.queryCount < 3 {
			t.Errorf("GetIncomeVsExpenseTrend used %d queries, expected ≥3 (seems suspiciously low)", qc.queryCount)
		}
	})

	t.Run("Dashboard analytics scenario (all 3 functions)", func(t *testing.T) {
		qc := NewQueryCounter(db)
		wrappedDB := qc.WrapDB()

		// Simulate dashboard: call all 3 analytics functions
		_ = GetMonthOverMonthComparison(wrappedDB, 2024, 2, accountIDs)
		_ = GetCategoryBreakdownWithPercentages(wrappedDB, 2024, 2, accountIDs)
		_ = GetIncomeVsExpenseTrend(wrappedDB, 6, accountIDs)

		t.Logf("Dashboard analytics (all 3 functions) executed %d total queries", qc.queryCount)

		// Expected total: ~8-12 queries
		// Without batch optimization, this would be: 10 + 1 + 30 = 41 queries
		if qc.queryCount > 15 {
			t.Errorf("Dashboard analytics used %d queries, expected ≤15 with batch optimization", qc.queryCount)
		}

		if qc.queryCount < 5 {
			t.Errorf("Dashboard analytics used %d queries, expected ≥5 (seems suspiciously low)", qc.queryCount)
		}

		// Calculate theoretical improvement
		theoreticalUnoptimized := 41
		improvement := float64(theoreticalUnoptimized-qc.queryCount) / float64(theoreticalUnoptimized) * 100
		t.Logf("Query reduction: %d queries saved (%.1f%% improvement)", theoreticalUnoptimized-qc.queryCount, improvement)

		// Should save at least 60% of queries
		if improvement < 60 {
			t.Errorf("Batch optimization saved only %.1f%% of queries, expected ≥60%%", improvement)
		}
	})
}

// BenchmarkAnalyticsFullDashboard benchmarks the full dashboard analytics scenario
func BenchmarkAnalyticsFullDashboard(b *testing.B) {
	db := testutil.SetupTestDB()

	// Setup test data
	user := testutil.CreateTestUser(db, "bench@example.com", "Benchmark User", "hash")
	account1 := testutil.CreateTestAccount(db, "Account 1", models.AccountTypeIndividual, user.ID, nil)
	account2 := testutil.CreateTestAccount(db, "Account 2", models.AccountTypeIndividual, user.ID, nil)
	accountIDs := []uint{account1.ID, account2.ID}

	// Create 6 months of data
	for month := 1; month <= 6; month++ {
		db.Create(&models.Income{
			AccountID:   account1.ID,
			Date:        time.Date(2024, time.Month(month), 15, 0, 0, 0, 0, time.Local),
			GrossAmount: 5000.00,
			TaxAmount:   500.00,
			NetAmount:   4500.00,
		})

		db.Create(&models.Income{
			AccountID:   account2.ID,
			Date:        time.Date(2024, time.Month(month), 15, 0, 0, 0, 0, time.Local),
			GrossAmount: 3000.00,
			TaxAmount:   300.00,
			NetAmount:   2700.00,
		})

		// Variable expenses with categories
		for _, category := range []string{"Food", "Transportation"} {
			expense := &models.Expense{
				AccountID: account1.ID,
				Name:      category,
				Category:  category,
				Amount:    200.00,
				Type:      models.ExpenseTypeVariable,
				Active:    true,
			}
			db.Create(expense)
			db.Model(expense).Update("created_at", time.Date(2024, time.Month(month), 20, 0, 0, 0, 0, time.Local))
		}
	}

	// Fixed expenses
	db.Create(&models.Expense{
		AccountID: account1.ID,
		Name:      "Rent",
		Amount:    1000.00,
		Type:      models.ExpenseTypeFixed,
		Active:    true,
	})

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Simulate dashboard: call all 3 analytics functions
		comparison := GetMonthOverMonthComparison(db, 2024, 2, accountIDs)
		breakdown := GetCategoryBreakdownWithPercentages(db, 2024, 2, accountIDs)
		trend := GetIncomeVsExpenseTrend(db, 6, accountIDs)

		// Prevent compiler optimizations
		if comparison.CurrentMonth.TotalIncomeGross == 0 {
			b.Fatal("Invalid result")
		}
		if len(breakdown) == 0 {
			b.Fatal("Invalid result")
		}
		if len(trend) == 0 {
			b.Fatal("Invalid result")
		}
	}
}

// TestAnalyticsPerformance_CompareWithNaive compares batch vs naive implementation
func TestAnalyticsPerformance_CompareWithNaive(t *testing.T) {
	db := testutil.SetupTestDB()

	// Setup test data
	user := testutil.CreateTestUser(db, "test@example.com", "Test User", "hash")
	account := testutil.CreateTestAccount(db, "Test Account", models.AccountTypeIndividual, user.ID, nil)
	accountIDs := []uint{account.ID}

	// Create 6 months of data
	for month := 1; month <= 6; month++ {
		db.Create(&models.Income{
			AccountID:   account.ID,
			Date:        time.Date(2024, time.Month(month), 15, 0, 0, 0, 0, time.Local),
			GrossAmount: 5000.00,
			TaxAmount:   500.00,
			NetAmount:   4500.00,
		})

		expense := &models.Expense{
			AccountID: account.ID,
			Name:      "Groceries",
			Category:  "Food",
			Amount:    300.00,
			Type:      models.ExpenseTypeVariable,
			Active:    true,
		}
		db.Create(expense)
		db.Model(expense).Update("created_at", time.Date(2024, time.Month(month), 20, 0, 0, 0, 0, time.Local))
	}

	db.Create(&models.Expense{
		AccountID: account.ID,
		Name:      "Rent",
		Amount:    1000.00,
		Type:      models.ExpenseTypeFixed,
		Active:    true,
	})

	// Test batch implementation (GetIncomeVsExpenseTrend uses GetBatchMonthlySummariesForAccounts)
	qcBatch := NewQueryCounter(db)
	dbBatch := qcBatch.WrapDB()

	trendBatch := GetIncomeVsExpenseTrend(dbBatch, 6, accountIDs)

	// Test naive implementation (loop calling GetMonthlySummaryForAccounts)
	qcNaive := NewQueryCounter(db)
	dbNaive := qcNaive.WrapDB()

	trendNaive := make([]MonthlySummary, 0, 6)
	for month := 1; month <= 6; month++ {
		summary := GetMonthlySummaryForAccounts(dbNaive, 2024, month, accountIDs)
		trendNaive = append(trendNaive, summary)
	}

	// Verify results match
	if len(trendBatch) != len(trendNaive) {
		t.Errorf("Result mismatch: batch=%d vs naive=%d months", len(trendBatch), len(trendNaive))
	}

	// Compare query counts
	t.Logf("Batch implementation: %d queries", qcBatch.queryCount)
	t.Logf("Naive implementation: %d queries", qcNaive.queryCount)
	t.Logf("Improvement: %.1fx faster (query count)", float64(qcNaive.queryCount)/float64(qcBatch.queryCount))

	// Batch should use significantly fewer queries
	if qcBatch.queryCount >= qcNaive.queryCount {
		t.Errorf("Batch implementation (%d queries) did not reduce query count vs naive (%d queries)", qcBatch.queryCount, qcNaive.queryCount)
	}

	// Should be at least 2x improvement
	if float64(qcNaive.queryCount)/float64(qcBatch.queryCount) < 2.0 {
		t.Errorf("Batch optimization improvement (%.1fx) is less than expected (≥2x)", float64(qcNaive.queryCount)/float64(qcBatch.queryCount))
	}
}

// Helper function to execute a raw query and count results
func countQueryRows(db *gorm.DB, query string, args ...interface{}) int {
	var count int64
	db.Raw(query, args...).Count(&count)
	return int(count)
}

// TestGetBatchMonthlySummariesForAccounts_QueryCount verifies the batch function query count
func TestGetBatchMonthlySummariesForAccounts_QueryCount(t *testing.T) {
	db := testutil.SetupTestDB()

	user := testutil.CreateTestUser(db, "test@example.com", "Test User", "hash")
	account := testutil.CreateTestAccount(db, "Test Account", models.AccountTypeIndividual, user.ID, nil)
	accountIDs := []uint{account.ID}

	// Create data
	for month := 1; month <= 6; month++ {
		db.Create(&models.Income{
			AccountID:   account.ID,
			Date:        time.Date(2024, time.Month(month), 15, 0, 0, 0, 0, time.Local),
			GrossAmount: 5000.00,
			TaxAmount:   500.00,
			NetAmount:   4500.00,
		})
	}

	db.Create(&models.Expense{
		AccountID: account.ID,
		Name:      "Rent",
		Amount:    1000.00,
		Type:      models.ExpenseTypeFixed,
		Active:    true,
	})

	// Count queries
	qc := NewQueryCounter(db)
	wrappedDB := qc.WrapDB()

	results := GetBatchMonthlySummariesForAccounts(wrappedDB, 2024, 1, 2024, 6, accountIDs)

	t.Logf("GetBatchMonthlySummariesForAccounts (6 months) executed %d queries", qc.queryCount)

	// Should be exactly 5 queries (one per transaction type: income, fixed, variable, cards, bills)
	// regardless of the number of months
	if qc.queryCount > 8 {
		t.Errorf("GetBatchMonthlySummariesForAccounts used %d queries, expected ≤8 (should be constant regardless of month count)", qc.queryCount)
	}

	if len(results) != 6 {
		t.Errorf("GetBatchMonthlySummariesForAccounts returned %d months, expected 6", len(results))
	}
}
