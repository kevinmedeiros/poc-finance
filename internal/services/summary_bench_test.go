package services

import (
	"testing"
	"time"

	"gorm.io/gorm"

	"poc-finance/internal/models"
	"poc-finance/internal/testutil"
)

// setupBenchmarkData creates realistic test data with 100+ transactions for benchmarking
func setupBenchmarkData(b *testing.B, db *gorm.DB) ([]uint, int, int) {
	b.Helper()

	// Create test users
	user1 := testutil.CreateTestUser(db, "bench1@example.com", "Benchmark User 1", "hash")
	user2 := testutil.CreateTestUser(db, "bench2@example.com", "Benchmark User 2", "hash")

	// Create test accounts
	account1 := testutil.CreateTestAccount(db, "Benchmark Account 1", models.AccountTypeIndividual, user1.ID, nil)
	account2 := testutil.CreateTestAccount(db, "Benchmark Account 2", models.AccountTypeIndividual, user2.ID, nil)
	accountIDs := []uint{account1.ID, account2.ID}

	// Define 6-month range for realistic dashboard scenario
	startYear, startMonth := 2024, 1
	endMonth := 6

	// Create 20+ incomes per account across 6 months (40+ total)
	for month := startMonth; month <= endMonth; month++ {
		// 3-4 incomes per account per month
		for i := 0; i < 3; i++ {
			db.Create(&models.Income{
				AccountID:   account1.ID,
				Date:        time.Date(2024, time.Month(month), 5+(i*7), 0, 0, 0, 0, time.Local),
				GrossAmount: 5000.00 + float64(i*100),
				TaxAmount:   500.00 + float64(i*10),
				NetAmount:   4500.00 + float64(i*90),
				Description: "Benchmark Income",
			})

			db.Create(&models.Income{
				AccountID:   account2.ID,
				Date:        time.Date(2024, time.Month(month), 10+(i*7), 0, 0, 0, 0, time.Local),
				GrossAmount: 3000.00 + float64(i*100),
				TaxAmount:   300.00 + float64(i*10),
				NetAmount:   2700.00 + float64(i*90),
				Description: "Benchmark Income",
			})
		}
	}

	// Create 5 fixed expenses per account (10 total, apply to all months)
	for i := 0; i < 5; i++ {
		db.Create(&models.Expense{
			AccountID: account1.ID,
			Name:      "Fixed Expense " + string(rune('A'+i)),
			Amount:    200.00 + float64(i*50),
			Type:      models.ExpenseTypeFixed,
			Active:    true,
		})

		db.Create(&models.Expense{
			AccountID: account2.ID,
			Name:      "Fixed Expense " + string(rune('A'+i)),
			Amount:    150.00 + float64(i*30),
			Type:      models.ExpenseTypeFixed,
			Active:    true,
		})
	}

	// Create 30+ variable expenses across 6 months (5+ per month)
	for month := startMonth; month <= endMonth; month++ {
		for i := 0; i < 5; i++ {
			expense1 := &models.Expense{
				AccountID: account1.ID,
				Name:      "Variable Expense",
				Amount:    100.00 + float64(i*20),
				Type:      models.ExpenseTypeVariable,
				Active:    true,
			}
			db.Create(expense1)
			db.Model(expense1).Update("created_at", time.Date(2024, time.Month(month), 15+(i*3), 0, 0, 0, 0, time.Local))

			expense2 := &models.Expense{
				AccountID: account2.ID,
				Name:      "Variable Expense",
				Amount:    80.00 + float64(i*15),
				Type:      models.ExpenseTypeVariable,
				Active:    true,
			}
			db.Create(expense2)
			db.Model(expense2).Update("created_at", time.Date(2024, time.Month(month), 18+(i*3), 0, 0, 0, 0, time.Local))
		}
	}

	// Create credit cards with multiple installments (20+ installment entries)
	creditCard1 := &models.CreditCard{
		AccountID:  account1.ID,
		Name:       "Benchmark Card 1",
		ClosingDay: 15,
		DueDay:     25,
	}
	db.Create(creditCard1)

	creditCard2 := &models.CreditCard{
		AccountID:  account2.ID,
		Name:       "Benchmark Card 2",
		ClosingDay: 10,
		DueDay:     20,
	}
	db.Create(creditCard2)

	// Multiple installment plans
	installmentPlans := []struct {
		cardID            uint
		startMonth        int
		totalInstallments int
		amount            float64
	}{
		{creditCard1.ID, 1, 6, 300.00},
		{creditCard1.ID, 2, 4, 200.00},
		{creditCard1.ID, 3, 3, 150.00},
		{creditCard2.ID, 1, 5, 250.00},
		{creditCard2.ID, 3, 6, 180.00},
	}

	for _, plan := range installmentPlans {
		db.Create(&models.Installment{
			CreditCardID:      plan.cardID,
			Description:       "Benchmark Purchase",
			TotalAmount:       plan.amount * float64(plan.totalInstallments),
			InstallmentAmount: plan.amount,
			TotalInstallments: plan.totalInstallments,
			StartDate:         time.Date(2024, time.Month(plan.startMonth), 1, 0, 0, 0, 0, time.Local),
		})
	}

	// Create 30+ bills across 6 months (5+ per month)
	for month := startMonth; month <= endMonth; month++ {
		for i := 0; i < 5; i++ {
			db.Create(&models.Bill{
				AccountID: account1.ID,
				Name:      "Benchmark Bill",
				Amount:    100.00 + float64(i*25),
				DueDate:   time.Date(2024, time.Month(month), 5+(i*5), 0, 0, 0, 0, time.Local),
			})

			db.Create(&models.Bill{
				AccountID: account2.ID,
				Name:      "Benchmark Bill",
				Amount:    80.00 + float64(i*20),
				DueDate:   time.Date(2024, time.Month(month), 7+(i*5), 0, 0, 0, 0, time.Local),
			})
		}
	}

	return accountIDs, startYear, startMonth
}

// BenchmarkGetMonthlySummaryLoop benchmarks the old loop-based implementation
// This simulates the original dashboard behavior: calling GetMonthlySummaryForAccounts 6 times
func BenchmarkGetMonthlySummaryLoop(b *testing.B) {
	db := testutil.SetupTestDB()
	accountIDs, startYear, startMonth := setupBenchmarkData(b, db)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Simulate dashboard: call GetMonthlySummaryForAccounts 6 times (once per month)
		results := make([]MonthlySummary, 0, 6)
		for month := 0; month < 6; month++ {
			currentMonth := startMonth + month
			summary := GetMonthlySummaryForAccounts(db, startYear, currentMonth, accountIDs)
			results = append(results, summary)
		}

		// Prevent compiler optimizations from eliminating the benchmark code
		if len(results) != 6 {
			b.Fatalf("Expected 6 results, got %d", len(results))
		}
	}
}

// BenchmarkGetBatchMonthlySummaries benchmarks the new batch implementation
// This simulates the optimized dashboard behavior: single call to GetBatchMonthlySummariesForAccounts
func BenchmarkGetBatchMonthlySummaries(b *testing.B) {
	db := testutil.SetupTestDB()
	accountIDs, startYear, startMonth := setupBenchmarkData(b, db)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Optimized: single batch call for all 6 months
		results := GetBatchMonthlySummariesForAccounts(db, startYear, startMonth, startYear, startMonth+5, accountIDs)

		// Prevent compiler optimizations from eliminating the benchmark code
		if len(results) != 6 {
			b.Fatalf("Expected 6 results, got %d", len(results))
		}
	}
}

// BenchmarkGetMonthlySummaryLoop_3Months benchmarks loop implementation with 3 months
func BenchmarkGetMonthlySummaryLoop_3Months(b *testing.B) {
	db := testutil.SetupTestDB()
	accountIDs, startYear, startMonth := setupBenchmarkData(b, db)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		results := make([]MonthlySummary, 0, 3)
		for month := 0; month < 3; month++ {
			currentMonth := startMonth + month
			summary := GetMonthlySummaryForAccounts(db, startYear, currentMonth, accountIDs)
			results = append(results, summary)
		}

		if len(results) != 3 {
			b.Fatalf("Expected 3 results, got %d", len(results))
		}
	}
}

// BenchmarkGetBatchMonthlySummaries_3Months benchmarks batch implementation with 3 months
func BenchmarkGetBatchMonthlySummaries_3Months(b *testing.B) {
	db := testutil.SetupTestDB()
	accountIDs, startYear, startMonth := setupBenchmarkData(b, db)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		results := GetBatchMonthlySummariesForAccounts(db, startYear, startMonth, startYear, startMonth+2, accountIDs)

		if len(results) != 3 {
			b.Fatalf("Expected 3 results, got %d", len(results))
		}
	}
}

// BenchmarkGetMonthlySummaryLoop_12Months benchmarks loop implementation with 12 months
func BenchmarkGetMonthlySummaryLoop_12Months(b *testing.B) {
	db := testutil.SetupTestDB()
	accountIDs, startYear, startMonth := setupBenchmarkData(b, db)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		results := make([]MonthlySummary, 0, 12)
		for month := 0; month < 12; month++ {
			currentMonth := startMonth + month
			currentYear := startYear
			if currentMonth > 12 {
				currentMonth -= 12
				currentYear++
			}
			summary := GetMonthlySummaryForAccounts(db, currentYear, currentMonth, accountIDs)
			results = append(results, summary)
		}

		if len(results) != 12 {
			b.Fatalf("Expected 12 results, got %d", len(results))
		}
	}
}

// BenchmarkGetBatchMonthlySummaries_12Months benchmarks batch implementation with 12 months
func BenchmarkGetBatchMonthlySummaries_12Months(b *testing.B) {
	db := testutil.SetupTestDB()
	accountIDs, startYear, startMonth := setupBenchmarkData(b, db)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		results := GetBatchMonthlySummariesForAccounts(db, startYear, startMonth, startYear+1, startMonth-1, accountIDs)

		if len(results) != 12 {
			b.Fatalf("Expected 12 results, got %d", len(results))
		}
	}
}
