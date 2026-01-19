package services

import (
	"testing"

	"poc-finance/internal/database"
	"poc-finance/internal/models"
	"poc-finance/internal/testutil"
)

func TestAccountService_GetUserAccounts(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	// Create test users
	user1 := testutil.CreateTestUser(db, "user1@example.com", "User 1", "hash")
	user2 := testutil.CreateTestUser(db, "user2@example.com", "User 2", "hash")

	// Create individual accounts
	account1 := testutil.CreateTestAccount(db, "User 1 Personal", models.AccountTypeIndividual, user1.ID, nil)
	testutil.CreateTestAccount(db, "User 2 Personal", models.AccountTypeIndividual, user2.ID, nil)

	// Create a group and joint account
	group := testutil.CreateTestGroup(db, "Test Family", user1.ID)
	testutil.CreateTestGroupMember(db, group.ID, user1.ID, "admin")
	testutil.CreateTestGroupMember(db, group.ID, user2.ID, "member")

	groupID := group.ID
	jointAccount := testutil.CreateTestAccount(db, "Joint Account", models.AccountTypeJoint, user1.ID, &groupID)

	accountService := NewAccountService()

	// Test user1's accounts
	accounts, err := accountService.GetUserAccounts(user1.ID)
	if err != nil {
		t.Fatalf("GetUserAccounts() error = %v", err)
	}

	// User1 should have individual + joint account
	if len(accounts) != 2 {
		t.Errorf("Expected 2 accounts, got %d", len(accounts))
	}

	// Verify accounts are correct
	foundIndividual := false
	foundJoint := false
	for _, acc := range accounts {
		if acc.ID == account1.ID {
			foundIndividual = true
		}
		if acc.ID == jointAccount.ID {
			foundJoint = true
		}
	}

	if !foundIndividual {
		t.Error("Individual account not found")
	}
	if !foundJoint {
		t.Error("Joint account not found")
	}
}

func TestAccountService_GetUserAccounts_NoGroup(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	user := testutil.CreateTestUser(db, "solo@example.com", "Solo User", "hash")
	testutil.CreateTestAccount(db, "Personal Account", models.AccountTypeIndividual, user.ID, nil)

	accountService := NewAccountService()

	accounts, err := accountService.GetUserAccounts(user.ID)
	if err != nil {
		t.Fatalf("GetUserAccounts() error = %v", err)
	}

	if len(accounts) != 1 {
		t.Errorf("Expected 1 account, got %d", len(accounts))
	}

	if accounts[0].Type != models.AccountTypeIndividual {
		t.Errorf("Expected individual account, got %v", accounts[0].Type)
	}
}

func TestAccountService_GetUserAccountIDs(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	user := testutil.CreateTestUser(db, "test@example.com", "Test User", "hash")
	account := testutil.CreateTestAccount(db, "Personal Account", models.AccountTypeIndividual, user.ID, nil)

	accountService := NewAccountService()

	ids, err := accountService.GetUserAccountIDs(user.ID)
	if err != nil {
		t.Fatalf("GetUserAccountIDs() error = %v", err)
	}

	if len(ids) != 1 {
		t.Errorf("Expected 1 ID, got %d", len(ids))
	}

	if ids[0] != account.ID {
		t.Errorf("Expected ID %d, got %d", account.ID, ids[0])
	}
}

func TestAccountService_GetUserIndividualAccount(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	user := testutil.CreateTestUser(db, "individual@example.com", "Individual User", "hash")
	expected := testutil.CreateTestAccount(db, "My Personal Account", models.AccountTypeIndividual, user.ID, nil)

	accountService := NewAccountService()

	account, err := accountService.GetUserIndividualAccount(user.ID)
	if err != nil {
		t.Fatalf("GetUserIndividualAccount() error = %v", err)
	}

	if account.ID != expected.ID {
		t.Errorf("Account ID = %d, want %d", account.ID, expected.ID)
	}

	if account.Name != "My Personal Account" {
		t.Errorf("Account Name = %s, want My Personal Account", account.Name)
	}
}

func TestAccountService_GetUserIndividualAccount_NotFound(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	user := testutil.CreateTestUser(db, "noindividual@example.com", "No Individual User", "hash")
	// Don't create an individual account

	accountService := NewAccountService()

	_, err := accountService.GetUserIndividualAccount(user.ID)
	if err != ErrAccountNotFound {
		t.Errorf("Expected ErrAccountNotFound, got %v", err)
	}
}

func TestAccountService_CanUserAccessAccount_IndividualOwner(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	user := testutil.CreateTestUser(db, "owner@example.com", "Owner", "hash")
	account := testutil.CreateTestAccount(db, "Personal", models.AccountTypeIndividual, user.ID, nil)

	accountService := NewAccountService()

	if !accountService.CanUserAccessAccount(user.ID, account.ID) {
		t.Error("Owner should have access to their individual account")
	}
}

func TestAccountService_CanUserAccessAccount_NotOwner(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	owner := testutil.CreateTestUser(db, "owner@example.com", "Owner", "hash")
	other := testutil.CreateTestUser(db, "other@example.com", "Other", "hash")
	account := testutil.CreateTestAccount(db, "Personal", models.AccountTypeIndividual, owner.ID, nil)

	accountService := NewAccountService()

	if accountService.CanUserAccessAccount(other.ID, account.ID) {
		t.Error("Other user should NOT have access to someone else's individual account")
	}
}

func TestAccountService_CanUserAccessAccount_JointMember(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	user1 := testutil.CreateTestUser(db, "user1@example.com", "User 1", "hash")
	user2 := testutil.CreateTestUser(db, "user2@example.com", "User 2", "hash")

	group := testutil.CreateTestGroup(db, "Family", user1.ID)
	testutil.CreateTestGroupMember(db, group.ID, user1.ID, "admin")
	testutil.CreateTestGroupMember(db, group.ID, user2.ID, "member")

	groupID := group.ID
	jointAccount := testutil.CreateTestAccount(db, "Joint", models.AccountTypeJoint, user1.ID, &groupID)

	accountService := NewAccountService()

	// Both group members should have access
	if !accountService.CanUserAccessAccount(user1.ID, jointAccount.ID) {
		t.Error("Group member (admin) should have access to joint account")
	}

	if !accountService.CanUserAccessAccount(user2.ID, jointAccount.ID) {
		t.Error("Group member should have access to joint account")
	}
}

func TestAccountService_CanUserAccessAccount_JointNonMember(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	user1 := testutil.CreateTestUser(db, "user1@example.com", "User 1", "hash")
	outsider := testutil.CreateTestUser(db, "outsider@example.com", "Outsider", "hash")

	group := testutil.CreateTestGroup(db, "Family", user1.ID)
	testutil.CreateTestGroupMember(db, group.ID, user1.ID, "admin")

	groupID := group.ID
	jointAccount := testutil.CreateTestAccount(db, "Joint", models.AccountTypeJoint, user1.ID, &groupID)

	accountService := NewAccountService()

	if accountService.CanUserAccessAccount(outsider.ID, jointAccount.ID) {
		t.Error("Non-member should NOT have access to joint account")
	}
}

func TestAccountService_CanUserAccessAccount_NonexistentAccount(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	user := testutil.CreateTestUser(db, "user@example.com", "User", "hash")

	accountService := NewAccountService()

	if accountService.CanUserAccessAccount(user.ID, 99999) {
		t.Error("Should return false for nonexistent account")
	}
}

func TestAccountService_EnsureUserHasAccount_Existing(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	user := testutil.CreateTestUser(db, "existing@example.com", "Existing User", "hash")
	existing := testutil.CreateTestAccount(db, "Existing Account", models.AccountTypeIndividual, user.ID, nil)

	accountService := NewAccountService()

	account, err := accountService.EnsureUserHasAccount(user.ID)
	if err != nil {
		t.Fatalf("EnsureUserHasAccount() error = %v", err)
	}

	if account.ID != existing.ID {
		t.Errorf("Should return existing account, got different ID")
	}
}

func TestAccountService_EnsureUserHasAccount_Creates(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	user := testutil.CreateTestUser(db, "new@example.com", "New User", "hash")
	// Don't create an account

	accountService := NewAccountService()

	account, err := accountService.EnsureUserHasAccount(user.ID)
	if err != nil {
		t.Fatalf("EnsureUserHasAccount() error = %v", err)
	}

	if account.Name != "Conta Pessoal" {
		t.Errorf("Account Name = %s, want Conta Pessoal", account.Name)
	}

	if account.Type != models.AccountTypeIndividual {
		t.Errorf("Account Type = %v, want Individual", account.Type)
	}

	if account.UserID != user.ID {
		t.Errorf("Account UserID = %d, want %d", account.UserID, user.ID)
	}
}

func TestAccountService_CreateJointAccount(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	user := testutil.CreateTestUser(db, "creator@example.com", "Creator", "hash")
	group := testutil.CreateTestGroup(db, "Family", user.ID)
	testutil.CreateTestGroupMember(db, group.ID, user.ID, "admin")

	accountService := NewAccountService()

	account, err := accountService.CreateJointAccount("Shared Savings", group.ID, user.ID)
	if err != nil {
		t.Fatalf("CreateJointAccount() error = %v", err)
	}

	if account.Name != "Shared Savings" {
		t.Errorf("Name = %s, want Shared Savings", account.Name)
	}

	if account.Type != models.AccountTypeJoint {
		t.Errorf("Type = %v, want Joint", account.Type)
	}

	if account.GroupID == nil || *account.GroupID != group.ID {
		t.Errorf("GroupID = %v, want %d", account.GroupID, group.ID)
	}
}

func TestAccountService_CreateJointAccount_NotMember(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	owner := testutil.CreateTestUser(db, "owner@example.com", "Owner", "hash")
	outsider := testutil.CreateTestUser(db, "outsider@example.com", "Outsider", "hash")
	group := testutil.CreateTestGroup(db, "Family", owner.ID)
	testutil.CreateTestGroupMember(db, group.ID, owner.ID, "admin")

	accountService := NewAccountService()

	_, err := accountService.CreateJointAccount("Shared", group.ID, outsider.ID)
	if err != ErrUnauthorized {
		t.Errorf("Expected ErrUnauthorized, got %v", err)
	}
}

func TestAccountService_GetGroupJointAccounts(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	user := testutil.CreateTestUser(db, "user@example.com", "User", "hash")
	group := testutil.CreateTestGroup(db, "Family", user.ID)

	groupID := group.ID
	testutil.CreateTestAccount(db, "Joint 1", models.AccountTypeJoint, user.ID, &groupID)
	testutil.CreateTestAccount(db, "Joint 2", models.AccountTypeJoint, user.ID, &groupID)
	// Individual account should not be included
	testutil.CreateTestAccount(db, "Personal", models.AccountTypeIndividual, user.ID, nil)

	accountService := NewAccountService()

	accounts, err := accountService.GetGroupJointAccounts(group.ID)
	if err != nil {
		t.Fatalf("GetGroupJointAccounts() error = %v", err)
	}

	if len(accounts) != 2 {
		t.Errorf("Expected 2 joint accounts, got %d", len(accounts))
	}

	for _, acc := range accounts {
		if acc.Type != models.AccountTypeJoint {
			t.Errorf("Expected all accounts to be joint, got %v", acc.Type)
		}
	}
}

func TestAccountService_DeleteJointAccount(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	user := testutil.CreateTestUser(db, "user@example.com", "User", "hash")
	group := testutil.CreateTestGroup(db, "Family", user.ID)
	testutil.CreateTestGroupMember(db, group.ID, user.ID, "member")

	groupID := group.ID
	account := testutil.CreateTestAccount(db, "Joint", models.AccountTypeJoint, user.ID, &groupID)

	accountService := NewAccountService()

	err := accountService.DeleteJointAccount(account.ID, user.ID)
	if err != nil {
		t.Fatalf("DeleteJointAccount() error = %v", err)
	}

	// Verify account is deleted
	var deleted models.Account
	if err := db.First(&deleted, account.ID).Error; err == nil {
		t.Error("Account should be deleted")
	}
}

func TestAccountService_DeleteJointAccount_NotMember(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	owner := testutil.CreateTestUser(db, "owner@example.com", "Owner", "hash")
	outsider := testutil.CreateTestUser(db, "outsider@example.com", "Outsider", "hash")
	group := testutil.CreateTestGroup(db, "Family", owner.ID)
	testutil.CreateTestGroupMember(db, group.ID, owner.ID, "admin")

	groupID := group.ID
	account := testutil.CreateTestAccount(db, "Joint", models.AccountTypeJoint, owner.ID, &groupID)

	accountService := NewAccountService()

	err := accountService.DeleteJointAccount(account.ID, outsider.ID)
	if err != ErrUnauthorized {
		t.Errorf("Expected ErrUnauthorized, got %v", err)
	}
}

func TestAccountService_DeleteJointAccount_IndividualAccount(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	user := testutil.CreateTestUser(db, "user@example.com", "User", "hash")
	account := testutil.CreateTestAccount(db, "Personal", models.AccountTypeIndividual, user.ID, nil)

	accountService := NewAccountService()

	err := accountService.DeleteJointAccount(account.ID, user.ID)
	if err != ErrUnauthorized {
		t.Errorf("Expected ErrUnauthorized for individual account, got %v", err)
	}
}

func TestAccountService_GetAccountMembers_Individual(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	user := testutil.CreateTestUser(db, "user@example.com", "User", "hash")
	account := testutil.CreateTestAccount(db, "Personal", models.AccountTypeIndividual, user.ID, nil)

	accountService := NewAccountService()

	members, err := accountService.GetAccountMembers(account.ID)
	if err != nil {
		t.Fatalf("GetAccountMembers() error = %v", err)
	}

	if len(members) != 1 {
		t.Errorf("Expected 1 member, got %d", len(members))
	}

	if members[0].ID != user.ID {
		t.Errorf("Member ID = %d, want %d", members[0].ID, user.ID)
	}
}

func TestAccountService_GetAccountMembers_Joint(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	user1 := testutil.CreateTestUser(db, "user1@example.com", "User 1", "hash")
	user2 := testutil.CreateTestUser(db, "user2@example.com", "User 2", "hash")

	group := testutil.CreateTestGroup(db, "Family", user1.ID)
	testutil.CreateTestGroupMember(db, group.ID, user1.ID, "admin")
	testutil.CreateTestGroupMember(db, group.ID, user2.ID, "member")

	groupID := group.ID
	account := testutil.CreateTestAccount(db, "Joint", models.AccountTypeJoint, user1.ID, &groupID)

	accountService := NewAccountService()

	members, err := accountService.GetAccountMembers(account.ID)
	if err != nil {
		t.Fatalf("GetAccountMembers() error = %v", err)
	}

	if len(members) != 2 {
		t.Errorf("Expected 2 members, got %d", len(members))
	}
}

func TestAccountService_GetAccountByID(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	user := testutil.CreateTestUser(db, "user@example.com", "User", "hash")
	expected := testutil.CreateTestAccount(db, "Test Account", models.AccountTypeIndividual, user.ID, nil)

	accountService := NewAccountService()

	account, err := accountService.GetAccountByID(expected.ID)
	if err != nil {
		t.Fatalf("GetAccountByID() error = %v", err)
	}

	if account.ID != expected.ID {
		t.Errorf("ID = %d, want %d", account.ID, expected.ID)
	}

	if account.Name != "Test Account" {
		t.Errorf("Name = %s, want Test Account", account.Name)
	}
}

func TestAccountService_GetAccountByID_NotFound(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	accountService := NewAccountService()

	_, err := accountService.GetAccountByID(99999)
	if err != ErrAccountNotFound {
		t.Errorf("Expected ErrAccountNotFound, got %v", err)
	}
}

func TestAccountService_GetGroupJointAccountIDs(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	user := testutil.CreateTestUser(db, "user@example.com", "User", "hash")
	group := testutil.CreateTestGroup(db, "Family", user.ID)

	groupID := group.ID
	acc1 := testutil.CreateTestAccount(db, "Joint 1", models.AccountTypeJoint, user.ID, &groupID)
	acc2 := testutil.CreateTestAccount(db, "Joint 2", models.AccountTypeJoint, user.ID, &groupID)

	accountService := NewAccountService()

	ids, err := accountService.GetGroupJointAccountIDs(group.ID)
	if err != nil {
		t.Fatalf("GetGroupJointAccountIDs() error = %v", err)
	}

	if len(ids) != 2 {
		t.Errorf("Expected 2 IDs, got %d", len(ids))
	}

	foundAcc1 := false
	foundAcc2 := false
	for _, id := range ids {
		if id == acc1.ID {
			foundAcc1 = true
		}
		if id == acc2.ID {
			foundAcc2 = true
		}
	}

	if !foundAcc1 || !foundAcc2 {
		t.Error("Not all joint account IDs were returned")
	}
}

func TestAccountService_GetAllGroupAccountIDs(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	user1 := testutil.CreateTestUser(db, "user1@example.com", "User 1", "hash")
	user2 := testutil.CreateTestUser(db, "user2@example.com", "User 2", "hash")

	group := testutil.CreateTestGroup(db, "Family", user1.ID)
	testutil.CreateTestGroupMember(db, group.ID, user1.ID, "admin")
	testutil.CreateTestGroupMember(db, group.ID, user2.ID, "member")

	// Individual accounts for both members
	indAcc1 := testutil.CreateTestAccount(db, "User 1 Personal", models.AccountTypeIndividual, user1.ID, nil)
	indAcc2 := testutil.CreateTestAccount(db, "User 2 Personal", models.AccountTypeIndividual, user2.ID, nil)

	// Joint account for the group
	groupID := group.ID
	jointAcc := testutil.CreateTestAccount(db, "Joint", models.AccountTypeJoint, user1.ID, &groupID)

	accountService := NewAccountService()

	ids, err := accountService.GetAllGroupAccountIDs(group.ID)
	if err != nil {
		t.Fatalf("GetAllGroupAccountIDs() error = %v", err)
	}

	// Should include both individual accounts + joint account = 3
	if len(ids) != 3 {
		t.Errorf("Expected 3 IDs, got %d", len(ids))
	}

	foundInd1 := false
	foundInd2 := false
	foundJoint := false
	for _, id := range ids {
		if id == indAcc1.ID {
			foundInd1 = true
		}
		if id == indAcc2.ID {
			foundInd2 = true
		}
		if id == jointAcc.ID {
			foundJoint = true
		}
	}

	if !foundInd1 {
		t.Error("User 1's individual account not found")
	}
	if !foundInd2 {
		t.Error("User 2's individual account not found")
	}
	if !foundJoint {
		t.Error("Joint account not found")
	}
}
