package services

import (
	"testing"
	"time"

	"poc-finance/internal/database"
	"poc-finance/internal/models"
	"poc-finance/internal/testutil"
)

func TestDueDateSchedulerService_CheckUpcomingDueDates_UnpaidExpense(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	// Migrate models
	db.AutoMigrate(&models.Expense{}, &models.Notification{})

	// Create test user and account
	user := testutil.CreateTestUser(db, "user@example.com", "Test User", "hash")
	account := testutil.CreateTestAccount(db, "Personal", models.AccountTypeIndividual, user.ID, nil)

	// Calculate the day that's 3 days from now
	targetDate := time.Now().AddDate(0, 0, 3)
	targetDay := targetDate.Day()

	// Create a fixed expense due in 3 days (unpaid)
	expense := &models.Expense{
		AccountID: account.ID,
		Name:      "Internet Bill",
		Amount:    150.0,
		Type:      models.ExpenseTypeFixed,
		DueDay:    targetDay,
		Active:    true,
		Category:  "Utilities",
	}
	db.Create(expense)

	scheduler := NewDueDateSchedulerService()

	// Check upcoming due dates
	err := scheduler.CheckUpcomingDueDates()
	if err != nil {
		t.Fatalf("CheckUpcomingDueDates() error = %v", err)
	}

	// Verify notification was created
	var notifications []models.Notification
	db.Where("user_id = ?", user.ID).Find(&notifications)

	if len(notifications) != 1 {
		t.Errorf("Expected 1 notification, got %d", len(notifications))
	}

	if len(notifications) > 0 {
		if notifications[0].Type != models.NotificationTypeDueDate {
			t.Errorf("Notification type = %s, want %s", notifications[0].Type, models.NotificationTypeDueDate)
		}
		if notifications[0].Title != "Despesa próxima do vencimento" {
			t.Errorf("Notification title = %s", notifications[0].Title)
		}
	}
}

func TestDueDateSchedulerService_CheckUpcomingDueDates_AlreadyPaid(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	// Migrate models
	db.AutoMigrate(&models.Expense{}, &models.ExpensePayment{}, &models.Notification{})

	// Create test user and account
	user := testutil.CreateTestUser(db, "user@example.com", "Test User", "hash")
	account := testutil.CreateTestAccount(db, "Personal", models.AccountTypeIndividual, user.ID, nil)

	// Calculate the day that's 3 days from now
	now := time.Now()
	targetDate := now.AddDate(0, 0, 3)
	targetDay := targetDate.Day()
	currentMonth := int(now.Month())
	currentYear := now.Year()

	// Create a fixed expense due in 3 days
	expense := &models.Expense{
		AccountID: account.ID,
		Name:      "Internet Bill",
		Amount:    150.0,
		Type:      models.ExpenseTypeFixed,
		DueDay:    targetDay,
		Active:    true,
		Category:  "Utilities",
	}
	db.Create(expense)

	// Create a payment record for this expense (mark as paid)
	payment := &models.ExpensePayment{
		ExpenseID: expense.ID,
		Month:     currentMonth,
		Year:      currentYear,
		PaidAt:    now,
	}
	db.Create(payment)

	scheduler := NewDueDateSchedulerService()

	// Check upcoming due dates
	err := scheduler.CheckUpcomingDueDates()
	if err != nil {
		t.Fatalf("CheckUpcomingDueDates() error = %v", err)
	}

	// Verify NO notification was created (expense is already paid)
	var notifications []models.Notification
	db.Where("user_id = ?", user.ID).Find(&notifications)

	if len(notifications) != 0 {
		t.Errorf("Expected 0 notifications for paid expense, got %d", len(notifications))
	}
}

func TestDueDateSchedulerService_CheckUpcomingDueDates_DifferentDueDay(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	// Migrate models
	db.AutoMigrate(&models.Expense{}, &models.Notification{})

	// Create test user and account
	user := testutil.CreateTestUser(db, "user@example.com", "Test User", "hash")
	account := testutil.CreateTestAccount(db, "Personal", models.AccountTypeIndividual, user.ID, nil)

	// Calculate the day that's 3 days from now
	targetDate := time.Now().AddDate(0, 0, 3)
	targetDay := targetDate.Day()

	// Create a fixed expense due on a different day (not 3 days from now)
	differentDay := targetDay + 5
	if differentDay > 28 {
		differentDay = 1
	}

	expense := &models.Expense{
		AccountID: account.ID,
		Name:      "Internet Bill",
		Amount:    150.0,
		Type:      models.ExpenseTypeFixed,
		DueDay:    differentDay,
		Active:    true,
		Category:  "Utilities",
	}
	db.Create(expense)

	scheduler := NewDueDateSchedulerService()

	// Check upcoming due dates
	err := scheduler.CheckUpcomingDueDates()
	if err != nil {
		t.Fatalf("CheckUpcomingDueDates() error = %v", err)
	}

	// Verify NO notification was created (wrong due day)
	var notifications []models.Notification
	db.Where("user_id = ?", user.ID).Find(&notifications)

	if len(notifications) != 0 {
		t.Errorf("Expected 0 notifications for different due day, got %d", len(notifications))
	}
}

func TestDueDateSchedulerService_CheckUpcomingDueDates_InactiveExpense(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	// Migrate models
	db.AutoMigrate(&models.Expense{}, &models.Notification{})

	// Create test user and account
	user := testutil.CreateTestUser(db, "user@example.com", "Test User", "hash")
	account := testutil.CreateTestAccount(db, "Personal", models.AccountTypeIndividual, user.ID, nil)

	// Calculate the day that's 3 days from now
	targetDate := time.Now().AddDate(0, 0, 3)
	targetDay := targetDate.Day()

	// Create an inactive fixed expense due in 3 days
	expense := &models.Expense{
		AccountID: account.ID,
		Name:      "Internet Bill",
		Amount:    150.0,
		Type:      models.ExpenseTypeFixed,
		DueDay:    targetDay,
		Active:    true,
		Category:  "Utilities",
	}
	db.Create(expense)
	// Explicitly set Active to false
	db.Model(expense).Update("active", false)

	scheduler := NewDueDateSchedulerService()

	// Check upcoming due dates
	err := scheduler.CheckUpcomingDueDates()
	if err != nil {
		t.Fatalf("CheckUpcomingDueDates() error = %v", err)
	}

	// Verify NO notification was created (expense is inactive)
	var notifications []models.Notification
	db.Where("user_id = ?", user.ID).Find(&notifications)

	if len(notifications) != 0 {
		t.Errorf("Expected 0 notifications for inactive expense, got %d", len(notifications))
	}
}

func TestDueDateSchedulerService_CheckUpcomingDueDates_JointAccount(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	// Migrate models
	db.AutoMigrate(&models.Expense{}, &models.Notification{}, &models.FamilyGroup{}, &models.GroupMember{})

	// Create test users
	user1 := testutil.CreateTestUser(db, "user1@example.com", "User One", "hash1")
	user2 := testutil.CreateTestUser(db, "user2@example.com", "User Two", "hash2")

	// Create a group
	group := &models.FamilyGroup{
		Name:        "Test Group",
		CreatedByID: user1.ID,
	}
	db.Create(group)

	// Add members to the group
	db.Create(&models.GroupMember{
		GroupID: group.ID,
		UserID:  user1.ID,
		Role:    "owner",
	})
	db.Create(&models.GroupMember{
		GroupID: group.ID,
		UserID:  user2.ID,
		Role:    "member",
	})

	// Create a joint account with the group
	account := testutil.CreateTestAccount(db, "Joint Account", models.AccountTypeJoint, 0, &group.ID)

	// Calculate the day that's 3 days from now
	targetDate := time.Now().AddDate(0, 0, 3)
	targetDay := targetDate.Day()

	// Create a fixed expense due in 3 days
	expense := &models.Expense{
		AccountID: account.ID,
		Name:      "Shared Utility Bill",
		Amount:    200.0,
		Type:      models.ExpenseTypeFixed,
		DueDay:    targetDay,
		Active:    true,
		Category:  "Utilities",
	}
	db.Create(expense)

	scheduler := NewDueDateSchedulerService()

	// Check upcoming due dates
	err := scheduler.CheckUpcomingDueDates()
	if err != nil {
		t.Fatalf("CheckUpcomingDueDates() error = %v", err)
	}

	// Verify notifications were created for both users
	var notifications []models.Notification
	db.Where("group_id = ?", group.ID).Find(&notifications)

	if len(notifications) != 2 {
		t.Errorf("Expected 2 notifications (one per group member), got %d", len(notifications))
	}

	// Verify both users received a notification
	var userIDs []uint
	for _, notif := range notifications {
		userIDs = append(userIDs, notif.UserID)
		if notif.Type != models.NotificationTypeDueDate {
			t.Errorf("Notification type = %s, want %s", notif.Type, models.NotificationTypeDueDate)
		}
	}

	hasUser1 := false
	hasUser2 := false
	for _, uid := range userIDs {
		if uid == user1.ID {
			hasUser1 = true
		}
		if uid == user2.ID {
			hasUser2 = true
		}
	}

	if !hasUser1 || !hasUser2 {
		t.Error("Expected both group members to receive notifications")
	}
}

func TestDueDateSchedulerService_GetUpcomingDueDatesCount_MultipleUnpaid(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	// Migrate models
	db.AutoMigrate(&models.Expense{}, &models.ExpensePayment{})

	// Create test user and account
	user := testutil.CreateTestUser(db, "user@example.com", "Test User", "hash")
	account := testutil.CreateTestAccount(db, "Personal", models.AccountTypeIndividual, user.ID, nil)

	// Calculate the day that's 3 days from now
	targetDate := time.Now().AddDate(0, 0, 3)
	targetDay := targetDate.Day()

	// Create 3 unpaid fixed expenses due in 3 days
	for i := 0; i < 3; i++ {
		expense := &models.Expense{
			AccountID: account.ID,
			Name:      "Expense " + string(rune('A'+i)),
			Amount:    100.0 * float64(i+1),
			Type:      models.ExpenseTypeFixed,
			DueDay:    targetDay,
			Active:    true,
			Category:  "Test",
		}
		db.Create(expense)
	}

	scheduler := NewDueDateSchedulerService()

	// Get count of upcoming due dates
	count, err := scheduler.GetUpcomingDueDatesCount()
	if err != nil {
		t.Fatalf("GetUpcomingDueDatesCount() error = %v", err)
	}

	if count != 3 {
		t.Errorf("Expected 3 upcoming due dates, got %d", count)
	}
}

func TestDueDateSchedulerService_GetUpcomingDueDatesCount_ExcludesPaid(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	// Migrate models
	db.AutoMigrate(&models.Expense{}, &models.ExpensePayment{})

	// Create test user and account
	user := testutil.CreateTestUser(db, "user@example.com", "Test User", "hash")
	account := testutil.CreateTestAccount(db, "Personal", models.AccountTypeIndividual, user.ID, nil)

	// Calculate the day that's 3 days from now
	now := time.Now()
	targetDate := now.AddDate(0, 0, 3)
	targetDay := targetDate.Day()
	currentMonth := int(now.Month())
	currentYear := now.Year()

	// Create 2 unpaid expenses
	expense1 := &models.Expense{
		AccountID: account.ID,
		Name:      "Unpaid Expense 1",
		Amount:    100.0,
		Type:      models.ExpenseTypeFixed,
		DueDay:    targetDay,
		Active:    true,
		Category:  "Test",
	}
	db.Create(expense1)

	expense2 := &models.Expense{
		AccountID: account.ID,
		Name:      "Unpaid Expense 2",
		Amount:    200.0,
		Type:      models.ExpenseTypeFixed,
		DueDay:    targetDay,
		Active:    true,
		Category:  "Test",
	}
	db.Create(expense2)

	// Create 1 paid expense
	paidExpense := &models.Expense{
		AccountID: account.ID,
		Name:      "Paid Expense",
		Amount:    150.0,
		Type:      models.ExpenseTypeFixed,
		DueDay:    targetDay,
		Active:    true,
		Category:  "Test",
	}
	db.Create(paidExpense)

	// Create payment record for the paid expense
	payment := &models.ExpensePayment{
		ExpenseID: paidExpense.ID,
		Month:     currentMonth,
		Year:      currentYear,
		PaidAt:    now,
	}
	db.Create(payment)

	scheduler := NewDueDateSchedulerService()

	// Get count of upcoming due dates
	count, err := scheduler.GetUpcomingDueDatesCount()
	if err != nil {
		t.Fatalf("GetUpcomingDueDatesCount() error = %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 unpaid upcoming due dates, got %d", count)
	}
}

func TestDueDateSchedulerService_GetUpcomingDueDatesCount_DifferentDueDays(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	// Migrate models
	db.AutoMigrate(&models.Expense{})

	// Create test user and account
	user := testutil.CreateTestUser(db, "user@example.com", "Test User", "hash")
	account := testutil.CreateTestAccount(db, "Personal", models.AccountTypeIndividual, user.ID, nil)

	// Calculate the day that's 3 days from now
	targetDate := time.Now().AddDate(0, 0, 3)
	targetDay := targetDate.Day()

	// Create 2 expenses due in 3 days
	for i := 0; i < 2; i++ {
		expense := &models.Expense{
			AccountID: account.ID,
			Name:      "Due in 3 days " + string(rune('A'+i)),
			Amount:    100.0,
			Type:      models.ExpenseTypeFixed,
			DueDay:    targetDay,
			Active:    true,
			Category:  "Test",
		}
		db.Create(expense)
	}

	// Create 1 expense due on a different day
	differentDay := targetDay + 5
	if differentDay > 28 {
		differentDay = 1
	}

	expense := &models.Expense{
		AccountID: account.ID,
		Name:      "Due on different day",
		Amount:    150.0,
		Type:      models.ExpenseTypeFixed,
		DueDay:    differentDay,
		Active:    true,
		Category:  "Test",
	}
	db.Create(expense)

	scheduler := NewDueDateSchedulerService()

	// Get count of upcoming due dates
	count, err := scheduler.GetUpcomingDueDatesCount()
	if err != nil {
		t.Fatalf("GetUpcomingDueDatesCount() error = %v", err)
	}

	if count != 2 {
		t.Errorf("Expected 2 upcoming due dates (3 days from now), got %d", count)
	}
}

func TestDueDateSchedulerService_GetUpcomingDueDatesCount_InactiveExpenses(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	// Migrate models
	db.AutoMigrate(&models.Expense{})

	// Create test user and account
	user := testutil.CreateTestUser(db, "user@example.com", "Test User", "hash")
	account := testutil.CreateTestAccount(db, "Personal", models.AccountTypeIndividual, user.ID, nil)

	// Calculate the day that's 3 days from now
	targetDate := time.Now().AddDate(0, 0, 3)
	targetDay := targetDate.Day()

	// Create 1 active expense
	activeExpense := &models.Expense{
		AccountID: account.ID,
		Name:      "Active Expense",
		Amount:    100.0,
		Type:      models.ExpenseTypeFixed,
		DueDay:    targetDay,
		Active:    true,
		Category:  "Test",
	}
	db.Create(activeExpense)

	// Create 1 inactive expense
	inactiveExpense := &models.Expense{
		AccountID: account.ID,
		Name:      "Inactive Expense",
		Amount:    200.0,
		Type:      models.ExpenseTypeFixed,
		DueDay:    targetDay,
		Active:    true,
		Category:  "Test",
	}
	db.Create(inactiveExpense)
	// Explicitly set Active to false
	db.Model(inactiveExpense).Update("active", false)

	scheduler := NewDueDateSchedulerService()

	// Get count of upcoming due dates
	count, err := scheduler.GetUpcomingDueDatesCount()
	if err != nil {
		t.Fatalf("GetUpcomingDueDatesCount() error = %v", err)
	}

	if count != 1 {
		t.Errorf("Expected 1 upcoming due date (only active), got %d", count)
	}
}
