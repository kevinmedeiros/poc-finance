package services

import (
	"testing"
	"time"

	"poc-finance/internal/database"
	"poc-finance/internal/models"
	"poc-finance/internal/testutil"
)

func TestHealthScoreService_CalculateUserScore_PerfectScore(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	// Need to add HealthScore to testutil migrations
	db.AutoMigrate(&models.HealthScore{})

	user := testutil.CreateTestUser(db, "perfect@example.com", "Perfect User", "hash")
	account := testutil.CreateTestAccount(db, "Personal Account", models.AccountTypeIndividual, user.ID, nil)

	// Create perfect financial situation:
	// - High income
	db.Create(&models.Income{
		AccountID:   account.ID,
		Date:        time.Now().AddDate(0, -1, 0),
		GrossAmount: 10000.00,
		TaxAmount:   1000.00,
		NetAmount:   9000.00,
	})

	// - Low fixed expenses (10% of income = excellent savings rate)
	db.Create(&models.Expense{
		AccountID: account.ID,
		Name:      "Rent",
		Amount:    900.00,
		Type:      models.ExpenseTypeFixed,
		Active:    true,
	})

	// - Active goal with good progress
	group := testutil.CreateTestGroup(db, "Test Family", user.ID)
	testutil.CreateTestGroupMember(db, group.ID, user.ID, "admin")
	goal := &models.GroupGoal{
		GroupID:       group.ID,
		Name:          "Vacation",
		TargetAmount:  5000.00,
		CurrentAmount: 4000.00, // 80% progress
		StartDate:     time.Now().AddDate(0, -2, 0),
		TargetDate:    time.Now().AddDate(0, 2, 0),
		Status:        models.GoalStatusActive,
		CreatedByID:   user.ID,
	}
	db.Create(goal)

	service := NewHealthScoreService()
	score, err := service.CalculateUserScore(user.ID, []uint{account.ID})

	if err != nil {
		t.Fatalf("CalculateUserScore() error = %v", err)
	}

	// Should be a high score (>= 75) for good financial health
	if score.Score < 75 {
		t.Errorf("Expected score >= 75 for good finances, got %.2f", score.Score)
	}

	if score.UserID == nil || *score.UserID != user.ID {
		t.Error("Score should be associated with user")
	}

	if score.GroupID != nil {
		t.Error("User score should not have GroupID")
	}
}

func TestHealthScoreService_CalculateUserScore_ZeroScore(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db
	db.AutoMigrate(&models.HealthScore{})

	user := testutil.CreateTestUser(db, "broke@example.com", "Broke User", "hash")
	account := testutil.CreateTestAccount(db, "Personal Account", models.AccountTypeIndividual, user.ID, nil)

	// Create terrible financial situation:
	// - No income
	// - High expenses
	db.Create(&models.Expense{
		AccountID: account.ID,
		Name:      "Rent",
		Amount:    2000.00,
		Type:      models.ExpenseTypeFixed,
		Active:    true,
	})

	// - No goals

	service := NewHealthScoreService()
	score, err := service.CalculateUserScore(user.ID, []uint{account.ID})

	if err != nil {
		t.Fatalf("CalculateUserScore() error = %v", err)
	}

	// Should be a very low score (<= 35) for poor financial health
	// (Budget score is neutral when no expense history exists)
	if score.Score > 35 {
		t.Errorf("Expected score <= 35 for poor finances, got %.2f", score.Score)
	}
}

func TestHealthScoreService_CalculateUserScore_NoIncome(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db
	db.AutoMigrate(&models.HealthScore{})

	user := testutil.CreateTestUser(db, "noincome@example.com", "No Income User", "hash")
	account := testutil.CreateTestAccount(db, "Personal Account", models.AccountTypeIndividual, user.ID, nil)

	service := NewHealthScoreService()
	score, err := service.CalculateUserScore(user.ID, []uint{account.ID})

	if err != nil {
		t.Fatalf("CalculateUserScore() error = %v", err)
	}

	// Should handle no income gracefully
	if score.Score < 0 || score.Score > 100 {
		t.Errorf("Score should be between 0-100, got %.2f", score.Score)
	}

	// Savings score should be 0 with no income
	if score.SavingsScore > 0 {
		t.Errorf("SavingsScore should be 0 with no income, got %.2f", score.SavingsScore)
	}
}

func TestHealthScoreService_CalculateUserScore_ComponentScores(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db
	db.AutoMigrate(&models.HealthScore{})

	user := testutil.CreateTestUser(db, "components@example.com", "Component User", "hash")
	account := testutil.CreateTestAccount(db, "Personal Account", models.AccountTypeIndividual, user.ID, nil)

	// Create income
	db.Create(&models.Income{
		AccountID:   account.ID,
		Date:        time.Now().AddDate(0, -1, 0),
		GrossAmount: 5000.00,
		TaxAmount:   500.00,
		NetAmount:   4500.00,
	})

	// Create moderate expenses (50% expense ratio)
	db.Create(&models.Expense{
		AccountID: account.ID,
		Name:      "Fixed",
		Amount:    2250.00,
		Type:      models.ExpenseTypeFixed,
		Active:    true,
	})

	service := NewHealthScoreService()
	score, err := service.CalculateUserScore(user.ID, []uint{account.ID})

	if err != nil {
		t.Fatalf("CalculateUserScore() error = %v", err)
	}

	// Verify all component scores are calculated
	if score.SavingsScore < 0 || score.SavingsScore > 100 {
		t.Errorf("SavingsScore out of range: %.2f", score.SavingsScore)
	}

	if score.DebtScore < 0 || score.DebtScore > 100 {
		t.Errorf("DebtScore out of range: %.2f", score.DebtScore)
	}

	if score.GoalScore < 0 || score.GoalScore > 100 {
		t.Errorf("GoalScore out of range: %.2f", score.GoalScore)
	}

	if score.BudgetScore < 0 || score.BudgetScore > 100 {
		t.Errorf("BudgetScore out of range: %.2f", score.BudgetScore)
	}

	// Overall score should be weighted average
	expectedScore := (score.SavingsScore * 0.30) + (score.DebtScore * 0.25) +
		(score.GoalScore * 0.25) + (score.BudgetScore * 0.20)

	if score.Score < expectedScore-0.1 || score.Score > expectedScore+0.1 {
		t.Errorf("Score = %.2f, want %.2f (weighted average)", score.Score, expectedScore)
	}
}

func TestHealthScoreService_CalculateGroupScore(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db
	db.AutoMigrate(&models.HealthScore{})

	user1 := testutil.CreateTestUser(db, "user1@example.com", "User 1", "hash")
	user2 := testutil.CreateTestUser(db, "user2@example.com", "User 2", "hash")

	group := testutil.CreateTestGroup(db, "Test Family", user1.ID)
	testutil.CreateTestGroupMember(db, group.ID, user1.ID, "admin")
	testutil.CreateTestGroupMember(db, group.ID, user2.ID, "member")

	// Create individual accounts
	account1 := testutil.CreateTestAccount(db, "User 1 Account", models.AccountTypeIndividual, user1.ID, nil)
	account2 := testutil.CreateTestAccount(db, "User 2 Account", models.AccountTypeIndividual, user2.ID, nil)

	// Create joint account
	groupID := group.ID
	jointAccount := testutil.CreateTestAccount(db, "Joint Account", models.AccountTypeJoint, user1.ID, &groupID)

	// Create income for all accounts
	db.Create(&models.Income{
		AccountID:   account1.ID,
		Date:        time.Now().AddDate(0, -1, 0),
		GrossAmount: 3000.00,
		NetAmount:   2700.00,
	})

	db.Create(&models.Income{
		AccountID:   account2.ID,
		Date:        time.Now().AddDate(0, -1, 0),
		GrossAmount: 2000.00,
		NetAmount:   1800.00,
	})

	db.Create(&models.Income{
		AccountID:   jointAccount.ID,
		Date:        time.Now().AddDate(0, -1, 0),
		GrossAmount: 1000.00,
		NetAmount:   900.00,
	})

	service := NewHealthScoreService()
	score, err := service.CalculateGroupScore(group.ID)

	if err != nil {
		t.Fatalf("CalculateGroupScore() error = %v", err)
	}

	if score.GroupID == nil || *score.GroupID != group.ID {
		t.Error("Score should be associated with group")
	}

	if score.UserID != nil {
		t.Error("Group score should not have UserID")
	}

	if score.Score < 0 || score.Score > 100 {
		t.Errorf("Score should be between 0-100, got %.2f", score.Score)
	}
}

func TestHealthScoreService_GetScoreHistory(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db
	db.AutoMigrate(&models.HealthScore{})

	user := testutil.CreateTestUser(db, "history@example.com", "History User", "hash")
	userID := user.ID

	// Create historical scores
	for i := 0; i < 6; i++ {
		db.Create(&models.HealthScore{
			UserID:       &userID,
			Score:        float64(50 + i*5),
			SavingsScore: float64(60 + i),
			DebtScore:    80.0,
			GoalScore:    70.0,
			BudgetScore:  60.0,
			CalculatedAt: time.Now().AddDate(0, -i, 0),
		})
	}

	service := NewHealthScoreService()
	scores, err := service.GetScoreHistory(&userID, nil, 6)

	if err != nil {
		t.Fatalf("GetScoreHistory() error = %v", err)
	}

	if len(scores) != 6 {
		t.Errorf("Expected 6 historical scores, got %d", len(scores))
	}

	// Scores should be ordered by date (newest first)
	for i := 1; i < len(scores); i++ {
		if scores[i].CalculatedAt.After(scores[i-1].CalculatedAt) {
			t.Error("Scores should be ordered newest first")
		}
	}
}

func TestHealthScoreService_GetScoreHistory_GroupScore(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db
	db.AutoMigrate(&models.HealthScore{})

	user := testutil.CreateTestUser(db, "grouphistory@example.com", "Group History User", "hash")
	group := testutil.CreateTestGroup(db, "Test Family", user.ID)
	groupID := group.ID

	// Create historical group scores
	for i := 0; i < 3; i++ {
		db.Create(&models.HealthScore{
			GroupID:      &groupID,
			Score:        float64(60 + i*10),
			SavingsScore: 70.0,
			DebtScore:    80.0,
			GoalScore:    75.0,
			BudgetScore:  65.0,
			CalculatedAt: time.Now().AddDate(0, -i, 0),
		})
	}

	service := NewHealthScoreService()
	scores, err := service.GetScoreHistory(nil, &groupID, 3)

	if err != nil {
		t.Fatalf("GetScoreHistory() error = %v", err)
	}

	if len(scores) != 3 {
		t.Errorf("Expected 3 historical scores, got %d", len(scores))
	}

	// All scores should be group scores
	for _, score := range scores {
		if score.GroupID == nil || *score.GroupID != groupID {
			t.Error("All scores should belong to the group")
		}
	}
}

func TestHealthScoreService_GetRecommendations(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db
	db.AutoMigrate(&models.HealthScore{})

	user := testutil.CreateTestUser(db, "recommendations@example.com", "Recommendations User", "hash")
	userID := user.ID

	// Create score with weak savings and goal components
	score := &models.HealthScore{
		UserID:       &userID,
		Score:        55.0,
		SavingsScore: 30.0, // Weak
		DebtScore:    80.0,
		GoalScore:    40.0, // Weak
		BudgetScore:  75.0,
		CalculatedAt: time.Now(),
	}
	db.Create(score)

	service := NewHealthScoreService()
	recommendations, err := service.GetRecommendations(score)

	if err != nil {
		t.Fatalf("GetRecommendations() error = %v", err)
	}

	if len(recommendations) < 2 {
		t.Errorf("Expected at least 2 recommendations, got %d", len(recommendations))
	}

	if len(recommendations) > 4 {
		t.Errorf("Expected at most 4 recommendations, got %d", len(recommendations))
	}

	// Verify recommendations are not empty
	for i, rec := range recommendations {
		if rec.Title == "" {
			t.Errorf("Recommendation %d has empty title", i)
		}
		if rec.Description == "" {
			t.Errorf("Recommendation %d has empty description", i)
		}
	}
}

func TestHealthScoreService_GetRecommendations_HighScore(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db
	db.AutoMigrate(&models.HealthScore{})

	user := testutil.CreateTestUser(db, "highscore@example.com", "High Score User", "hash")
	userID := user.ID

	// Create high score
	score := &models.HealthScore{
		UserID:       &userID,
		Score:        95.0,
		SavingsScore: 95.0,
		DebtScore:    100.0,
		GoalScore:    90.0,
		BudgetScore:  95.0,
		CalculatedAt: time.Now(),
	}
	db.Create(score)

	service := NewHealthScoreService()
	recommendations, err := service.GetRecommendations(score)

	if err != nil {
		t.Fatalf("GetRecommendations() error = %v", err)
	}

	// High score should still get some positive recommendations
	if len(recommendations) == 0 {
		t.Error("Expected at least 1 recommendation for high score")
	}
}

func TestHealthScoreService_SavesScoreToDB(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db
	db.AutoMigrate(&models.HealthScore{})

	user := testutil.CreateTestUser(db, "savedb@example.com", "Save DB User", "hash")
	account := testutil.CreateTestAccount(db, "Personal Account", models.AccountTypeIndividual, user.ID, nil)

	db.Create(&models.Income{
		AccountID:   account.ID,
		Date:        time.Now().AddDate(0, -1, 0),
		GrossAmount: 5000.00,
		NetAmount:   4500.00,
	})

	service := NewHealthScoreService()
	score, err := service.CalculateUserScore(user.ID, []uint{account.ID})

	if err != nil {
		t.Fatalf("CalculateUserScore() error = %v", err)
	}

	// Verify score was saved to database
	var dbScore models.HealthScore
	if err := db.First(&dbScore, score.ID).Error; err != nil {
		t.Fatalf("Score not saved to database: %v", err)
	}

	if dbScore.Score != score.Score {
		t.Errorf("DB Score = %.2f, want %.2f", dbScore.Score, score.Score)
	}
}

func TestHealthScoreService_NegativeBalanceHandling(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db
	db.AutoMigrate(&models.HealthScore{})

	user := testutil.CreateTestUser(db, "negative@example.com", "Negative User", "hash")
	account := testutil.CreateTestAccount(db, "Personal Account", models.AccountTypeIndividual, user.ID, nil)

	// Small income
	db.Create(&models.Income{
		AccountID:   account.ID,
		Date:        time.Now().AddDate(0, -1, 0),
		GrossAmount: 1000.00,
		NetAmount:   900.00,
	})

	// High expenses (more than income)
	db.Create(&models.Expense{
		AccountID: account.ID,
		Name:      "High Rent",
		Amount:    1500.00,
		Type:      models.ExpenseTypeFixed,
		Active:    true,
	})

	service := NewHealthScoreService()
	score, err := service.CalculateUserScore(user.ID, []uint{account.ID})

	if err != nil {
		t.Fatalf("CalculateUserScore() error = %v", err)
	}

	// Should handle negative balance gracefully
	if score.Score < 0 || score.Score > 100 {
		t.Errorf("Score should be between 0-100 even with negative balance, got %.2f", score.Score)
	}

	// Savings score should be very low (negative balance)
	if score.SavingsScore > 10 {
		t.Errorf("SavingsScore should be very low with negative balance, got %.2f", score.SavingsScore)
	}
}
