package services

import (
	"testing"

	"poc-finance/internal/database"
	"poc-finance/internal/models"
	"poc-finance/internal/testutil"
)

func TestCreateBudget(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	user := testutil.CreateTestUser(db, "user@example.com", "User", "hash")

	budgetService := NewBudgetService()

	categories := []struct {
		Category string
		Limit    float64
	}{
		{"Food", 500.00},
		{"Transport", 200.00},
		{"Entertainment", 100.00},
	}

	budget, err := budgetService.CreateBudget(user.ID, nil, 2024, 1, "January Budget", categories)
	if err != nil {
		t.Fatalf("CreateBudget() error = %v", err)
	}

	if budget.UserID != user.ID {
		t.Errorf("UserID = %d, want %d", budget.UserID, user.ID)
	}

	if budget.Year != 2024 {
		t.Errorf("Year = %d, want 2024", budget.Year)
	}

	if budget.Month != 1 {
		t.Errorf("Month = %d, want 1", budget.Month)
	}

	if budget.Name != "January Budget" {
		t.Errorf("Name = %s, want January Budget", budget.Name)
	}

	if budget.Status != models.BudgetStatusActive {
		t.Errorf("Status = %s, want %s", budget.Status, models.BudgetStatusActive)
	}

	if len(budget.Categories) != 3 {
		t.Fatalf("Expected 3 categories, got %d", len(budget.Categories))
	}

	// Verify categories
	expectedCategories := map[string]float64{
		"Food":          500.00,
		"Transport":     200.00,
		"Entertainment": 100.00,
	}

	for _, cat := range budget.Categories {
		expectedLimit, ok := expectedCategories[cat.Category]
		if !ok {
			t.Errorf("Unexpected category: %s", cat.Category)
			continue
		}
		if cat.Limit != expectedLimit {
			t.Errorf("Category %s: Limit = %.2f, want %.2f", cat.Category, cat.Limit, expectedLimit)
		}
		if cat.Spent != 0 {
			t.Errorf("Category %s: Spent = %.2f, want 0", cat.Category, cat.Spent)
		}
	}
}

func TestCreateBudget_InvalidMonth(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	user := testutil.CreateTestUser(db, "user@example.com", "User", "hash")
	budgetService := NewBudgetService()

	_, err := budgetService.CreateBudget(user.ID, nil, 2024, 13, "Invalid Budget", []struct {
		Category string
		Limit    float64
	}{})

	if err != ErrInvalidBudgetMonth {
		t.Errorf("Expected ErrInvalidBudgetMonth, got %v", err)
	}
}

func TestCreateBudget_InvalidYear(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	user := testutil.CreateTestUser(db, "user@example.com", "User", "hash")
	budgetService := NewBudgetService()

	_, err := budgetService.CreateBudget(user.ID, nil, 1800, 1, "Invalid Budget", []struct {
		Category string
		Limit    float64
	}{})

	if err != ErrInvalidBudgetYear {
		t.Errorf("Expected ErrInvalidBudgetYear, got %v", err)
	}
}

func TestCreateBudget_GroupBudget(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	admin := testutil.CreateTestUser(db, "admin@example.com", "Admin", "hash")
	group := testutil.CreateTestGroup(db, "Family", admin.ID)
	testutil.CreateTestGroupMember(db, group.ID, admin.ID, "admin")

	budgetService := NewBudgetService()

	categories := []struct {
		Category string
		Limit    float64
	}{
		{"Food", 1000.00},
	}

	budget, err := budgetService.CreateBudget(admin.ID, &group.ID, 2024, 1, "Family Budget", categories)
	if err != nil {
		t.Fatalf("CreateBudget() error = %v", err)
	}

	if budget.GroupID == nil {
		t.Fatal("GroupID should not be nil")
	}

	if *budget.GroupID != group.ID {
		t.Errorf("GroupID = %d, want %d", *budget.GroupID, group.ID)
	}
}

func TestCreateBudget_GroupBudget_NotMember(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	admin := testutil.CreateTestUser(db, "admin@example.com", "Admin", "hash")
	outsider := testutil.CreateTestUser(db, "outsider@example.com", "Outsider", "hash")
	group := testutil.CreateTestGroup(db, "Family", admin.ID)
	testutil.CreateTestGroupMember(db, group.ID, admin.ID, "admin")

	budgetService := NewBudgetService()

	_, err := budgetService.CreateBudget(outsider.ID, &group.ID, 2024, 1, "Family Budget", []struct {
		Category string
		Limit    float64
	}{})

	if err != ErrUnauthorized {
		t.Errorf("Expected ErrUnauthorized, got %v", err)
	}
}

func TestGetUserBudgets(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	user := testutil.CreateTestUser(db, "user@example.com", "User", "hash")
	budgetService := NewBudgetService()

	// Create multiple budgets
	budgetService.CreateBudget(user.ID, nil, 2024, 1, "January Budget", []struct {
		Category string
		Limit    float64
	}{{"Food", 500.00}})

	budgetService.CreateBudget(user.ID, nil, 2024, 2, "February Budget", []struct {
		Category string
		Limit    float64
	}{{"Food", 600.00}})

	budgets, err := budgetService.GetUserBudgets(user.ID)
	if err != nil {
		t.Fatalf("GetUserBudgets() error = %v", err)
	}

	if len(budgets) != 2 {
		t.Errorf("Expected 2 budgets, got %d", len(budgets))
	}
}

func TestGetGroupBudgets(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	admin := testutil.CreateTestUser(db, "admin@example.com", "Admin", "hash")
	group := testutil.CreateTestGroup(db, "Family", admin.ID)
	testutil.CreateTestGroupMember(db, group.ID, admin.ID, "admin")

	budgetService := NewBudgetService()

	// Create group budget
	budgetService.CreateBudget(admin.ID, &group.ID, 2024, 1, "Family Budget", []struct {
		Category string
		Limit    float64
	}{{"Food", 1000.00}})

	budgets, err := budgetService.GetGroupBudgets(group.ID, admin.ID)
	if err != nil {
		t.Fatalf("GetGroupBudgets() error = %v", err)
	}

	if len(budgets) != 1 {
		t.Errorf("Expected 1 budget, got %d", len(budgets))
	}
}

func TestGetGroupBudgets_NotMember(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	admin := testutil.CreateTestUser(db, "admin@example.com", "Admin", "hash")
	outsider := testutil.CreateTestUser(db, "outsider@example.com", "Outsider", "hash")
	group := testutil.CreateTestGroup(db, "Family", admin.ID)
	testutil.CreateTestGroupMember(db, group.ID, admin.ID, "admin")

	budgetService := NewBudgetService()

	_, err := budgetService.GetGroupBudgets(group.ID, outsider.ID)
	if err != ErrUnauthorized {
		t.Errorf("Expected ErrUnauthorized, got %v", err)
	}
}

func TestGetBudgetByID(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	user := testutil.CreateTestUser(db, "user@example.com", "User", "hash")
	budgetService := NewBudgetService()

	created, _ := budgetService.CreateBudget(user.ID, nil, 2024, 1, "January Budget", []struct {
		Category string
		Limit    float64
	}{{"Food", 500.00}})

	budget, err := budgetService.GetBudgetByID(created.ID, user.ID)
	if err != nil {
		t.Fatalf("GetBudgetByID() error = %v", err)
	}

	if budget.ID != created.ID {
		t.Errorf("ID = %d, want %d", budget.ID, created.ID)
	}
}

func TestGetBudgetByID_Unauthorized(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	owner := testutil.CreateTestUser(db, "owner@example.com", "Owner", "hash")
	other := testutil.CreateTestUser(db, "other@example.com", "Other", "hash")
	budgetService := NewBudgetService()

	created, _ := budgetService.CreateBudget(owner.ID, nil, 2024, 1, "January Budget", []struct {
		Category string
		Limit    float64
	}{{"Food", 500.00}})

	_, err := budgetService.GetBudgetByID(created.ID, other.ID)
	if err != ErrUnauthorized {
		t.Errorf("Expected ErrUnauthorized, got %v", err)
	}
}

func TestGetActiveBudget(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	user := testutil.CreateTestUser(db, "user@example.com", "User", "hash")
	budgetService := NewBudgetService()

	budgetService.CreateBudget(user.ID, nil, 2024, 1, "January Budget", []struct {
		Category string
		Limit    float64
	}{{"Food", 500.00}})

	budget, err := budgetService.GetActiveBudget(user.ID, nil, 2024, 1)
	if err != nil {
		t.Fatalf("GetActiveBudget() error = %v", err)
	}

	if budget.Year != 2024 || budget.Month != 1 {
		t.Errorf("Budget Year/Month = %d/%d, want 2024/1", budget.Year, budget.Month)
	}
}

func TestUpdateBudget(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	user := testutil.CreateTestUser(db, "user@example.com", "User", "hash")
	budgetService := NewBudgetService()

	budget, _ := budgetService.CreateBudget(user.ID, nil, 2024, 1, "January Budget", []struct {
		Category string
		Limit    float64
	}{{"Food", 500.00}})

	err := budgetService.UpdateBudget(budget.ID, user.ID, "Updated Budget", 2024, 2)
	if err != nil {
		t.Fatalf("UpdateBudget() error = %v", err)
	}

	updated, _ := budgetService.GetBudgetByID(budget.ID, user.ID)
	if updated.Name != "Updated Budget" {
		t.Errorf("Name = %s, want Updated Budget", updated.Name)
	}
	if updated.Month != 2 {
		t.Errorf("Month = %d, want 2", updated.Month)
	}
}

func TestUpdateBudget_Unauthorized(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	owner := testutil.CreateTestUser(db, "owner@example.com", "Owner", "hash")
	other := testutil.CreateTestUser(db, "other@example.com", "Other", "hash")
	budgetService := NewBudgetService()

	budget, _ := budgetService.CreateBudget(owner.ID, nil, 2024, 1, "January Budget", []struct {
		Category string
		Limit    float64
	}{{"Food", 500.00}})

	err := budgetService.UpdateBudget(budget.ID, other.ID, "Hacked Budget", 2024, 2)
	if err != ErrUnauthorized {
		t.Errorf("Expected ErrUnauthorized, got %v", err)
	}
}

func TestDeleteBudget(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	user := testutil.CreateTestUser(db, "user@example.com", "User", "hash")
	budgetService := NewBudgetService()

	budget, _ := budgetService.CreateBudget(user.ID, nil, 2024, 1, "January Budget", []struct {
		Category string
		Limit    float64
	}{{"Food", 500.00}})

	err := budgetService.DeleteBudget(budget.ID, user.ID)
	if err != nil {
		t.Fatalf("DeleteBudget() error = %v", err)
	}

	// Verify budget is deleted
	_, err = budgetService.GetBudgetByID(budget.ID, user.ID)
	if err != ErrBudgetNotFound {
		t.Errorf("Expected ErrBudgetNotFound, got %v", err)
	}

	// Verify categories are deleted
	var categories []models.BudgetCategory
	db.Where("budget_id = ?", budget.ID).Find(&categories)
	if len(categories) > 0 {
		t.Error("Categories should have been deleted")
	}
}

func TestArchiveBudget(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	user := testutil.CreateTestUser(db, "user@example.com", "User", "hash")
	budgetService := NewBudgetService()

	budget, _ := budgetService.CreateBudget(user.ID, nil, 2024, 1, "January Budget", []struct {
		Category string
		Limit    float64
	}{{"Food", 500.00}})

	err := budgetService.ArchiveBudget(budget.ID, user.ID)
	if err != nil {
		t.Fatalf("ArchiveBudget() error = %v", err)
	}

	archived, _ := budgetService.GetBudgetByID(budget.ID, user.ID)
	if archived.Status != models.BudgetStatusArchived {
		t.Errorf("Status = %s, want %s", archived.Status, models.BudgetStatusArchived)
	}
}

func TestAddCategory(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	user := testutil.CreateTestUser(db, "user@example.com", "User", "hash")
	budgetService := NewBudgetService()

	budget, _ := budgetService.CreateBudget(user.ID, nil, 2024, 1, "January Budget", []struct {
		Category string
		Limit    float64
	}{{"Food", 500.00}})

	category, err := budgetService.AddCategory(budget.ID, user.ID, "Entertainment", 200.00)
	if err != nil {
		t.Fatalf("AddCategory() error = %v", err)
	}

	if category.Category != "Entertainment" {
		t.Errorf("Category = %s, want Entertainment", category.Category)
	}

	if category.Limit != 200.00 {
		t.Errorf("Limit = %.2f, want 200.00", category.Limit)
	}

	if category.Spent != 0 {
		t.Errorf("Spent = %.2f, want 0", category.Spent)
	}
}

func TestUpdateCategory(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	user := testutil.CreateTestUser(db, "user@example.com", "User", "hash")
	budgetService := NewBudgetService()

	budget, _ := budgetService.CreateBudget(user.ID, nil, 2024, 1, "January Budget", []struct {
		Category string
		Limit    float64
	}{{"Food", 500.00}})

	categoryID := budget.Categories[0].ID

	err := budgetService.UpdateCategory(categoryID, user.ID, "Groceries", 600.00)
	if err != nil {
		t.Fatalf("UpdateCategory() error = %v", err)
	}

	var category models.BudgetCategory
	db.First(&category, categoryID)

	if category.Category != "Groceries" {
		t.Errorf("Category = %s, want Groceries", category.Category)
	}

	if category.Limit != 600.00 {
		t.Errorf("Limit = %.2f, want 600.00", category.Limit)
	}
}

func TestDeleteCategory(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	user := testutil.CreateTestUser(db, "user@example.com", "User", "hash")
	budgetService := NewBudgetService()

	budget, _ := budgetService.CreateBudget(user.ID, nil, 2024, 1, "January Budget", []struct {
		Category string
		Limit    float64
	}{
		{"Food", 500.00},
		{"Transport", 200.00},
	})

	categoryID := budget.Categories[0].ID

	err := budgetService.DeleteCategory(categoryID, user.ID)
	if err != nil {
		t.Fatalf("DeleteCategory() error = %v", err)
	}

	var category models.BudgetCategory
	if err := db.First(&category, categoryID).Error; err == nil {
		t.Error("Category should have been deleted")
	}
}

func TestUpdateCategorySpending(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	user := testutil.CreateTestUser(db, "user@example.com", "User", "hash")
	budgetService := NewBudgetService()

	budget, _ := budgetService.CreateBudget(user.ID, nil, 2024, 1, "January Budget", []struct {
		Category string
		Limit    float64
	}{{"Food", 500.00}})

	err := budgetService.UpdateCategorySpending(budget.ID, "Food", 250.00)
	if err != nil {
		t.Fatalf("UpdateCategorySpending() error = %v", err)
	}

	var category models.BudgetCategory
	db.Where("budget_id = ? AND category = ?", budget.ID, "Food").First(&category)

	if category.Spent != 250.00 {
		t.Errorf("Spent = %.2f, want 250.00", category.Spent)
	}
}

func TestCopyFromPreviousMonth(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	user := testutil.CreateTestUser(db, "user@example.com", "User", "hash")
	budgetService := NewBudgetService()

	// Create December budget
	budgetService.CreateBudget(user.ID, nil, 2023, 12, "December Budget", []struct {
		Category string
		Limit    float64
	}{
		{"Food", 500.00},
		{"Transport", 200.00},
	})

	// Copy to January 2024
	budget, err := budgetService.CopyFromPreviousMonth(user.ID, nil, 2024, 1)
	if err != nil {
		t.Fatalf("CopyFromPreviousMonth() error = %v", err)
	}

	if budget.Year != 2024 || budget.Month != 1 {
		t.Errorf("Year/Month = %d/%d, want 2024/1", budget.Year, budget.Month)
	}

	if len(budget.Categories) != 2 {
		t.Errorf("Expected 2 categories, got %d", len(budget.Categories))
	}

	// Verify all categories have 0 spent
	for _, cat := range budget.Categories {
		if cat.Spent != 0 {
			t.Errorf("Category %s: Spent = %.2f, want 0", cat.Category, cat.Spent)
		}
	}
}

func TestCopyFromPreviousMonth_NoPreviousBudget(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	user := testutil.CreateTestUser(db, "user@example.com", "User", "hash")
	budgetService := NewBudgetService()

	_, err := budgetService.CopyFromPreviousMonth(user.ID, nil, 2024, 1)
	if err != ErrBudgetNotFound {
		t.Errorf("Expected ErrBudgetNotFound, got %v", err)
	}
}
