package services

import (
	"testing"
	"time"

	"poc-finance/internal/database"
	"poc-finance/internal/models"
	"poc-finance/internal/testutil"
)

func TestGroupService_GenerateInviteCode(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	admin := testutil.CreateTestUser(db, "admin@example.com", "Admin", "hash")
	group := testutil.CreateTestGroup(db, "Family", admin.ID)
	testutil.CreateTestGroupMember(db, group.ID, admin.ID, "admin")

	groupService := NewGroupService()

	invite, err := groupService.GenerateInviteCode(group.ID, admin.ID)
	if err != nil {
		t.Fatalf("GenerateInviteCode() error = %v", err)
	}

	if invite.Code == "" {
		t.Error("Invite code should not be empty")
	}

	if len(invite.Code) != 32 { // 16 bytes = 32 hex chars
		t.Errorf("Invite code length = %d, want 32", len(invite.Code))
	}

	if invite.GroupID != group.ID {
		t.Errorf("GroupID = %d, want %d", invite.GroupID, group.ID)
	}

	if invite.CreatedByID != admin.ID {
		t.Errorf("CreatedByID = %d, want %d", invite.CreatedByID, admin.ID)
	}

	if invite.Revoked {
		t.Error("Invite should not be revoked on creation")
	}

	if invite.UsedCount != 0 {
		t.Errorf("UsedCount = %d, want 0", invite.UsedCount)
	}

	// Check expiration is about 7 days from now
	expectedExpiry := time.Now().Add(InviteCodeDuration)
	if invite.ExpiresAt.Before(expectedExpiry.Add(-time.Minute)) || invite.ExpiresAt.After(expectedExpiry.Add(time.Minute)) {
		t.Errorf("ExpiresAt = %v, want around %v", invite.ExpiresAt, expectedExpiry)
	}
}

func TestGroupService_GenerateInviteCode_NotAdmin(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	admin := testutil.CreateTestUser(db, "admin@example.com", "Admin", "hash")
	member := testutil.CreateTestUser(db, "member@example.com", "Member", "hash")
	group := testutil.CreateTestGroup(db, "Family", admin.ID)
	testutil.CreateTestGroupMember(db, group.ID, admin.ID, "admin")
	testutil.CreateTestGroupMember(db, group.ID, member.ID, "member")

	groupService := NewGroupService()

	_, err := groupService.GenerateInviteCode(group.ID, member.ID)
	if err != ErrNotGroupAdmin {
		t.Errorf("Expected ErrNotGroupAdmin, got %v", err)
	}
}

func TestGroupService_GenerateInviteCode_NotMember(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	admin := testutil.CreateTestUser(db, "admin@example.com", "Admin", "hash")
	outsider := testutil.CreateTestUser(db, "outsider@example.com", "Outsider", "hash")
	group := testutil.CreateTestGroup(db, "Family", admin.ID)
	testutil.CreateTestGroupMember(db, group.ID, admin.ID, "admin")

	groupService := NewGroupService()

	_, err := groupService.GenerateInviteCode(group.ID, outsider.ID)
	if err != ErrNotGroupAdmin {
		t.Errorf("Expected ErrNotGroupAdmin, got %v", err)
	}
}

func TestGroupService_GetInviteByCode(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	admin := testutil.CreateTestUser(db, "admin@example.com", "Admin", "hash")
	group := testutil.CreateTestGroup(db, "Family", admin.ID)
	testutil.CreateTestGroupMember(db, group.ID, admin.ID, "admin")

	groupService := NewGroupService()

	created, _ := groupService.GenerateInviteCode(group.ID, admin.ID)

	invite, err := groupService.GetInviteByCode(created.Code)
	if err != nil {
		t.Fatalf("GetInviteByCode() error = %v", err)
	}

	if invite.ID != created.ID {
		t.Errorf("Invite ID = %d, want %d", invite.ID, created.ID)
	}
}

func TestGroupService_GetInviteByCode_NotFound(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	groupService := NewGroupService()

	_, err := groupService.GetInviteByCode("nonexistent-code")
	if err != ErrInviteNotFound {
		t.Errorf("Expected ErrInviteNotFound, got %v", err)
	}
}

func TestGroupService_ValidateInvite_Valid(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	admin := testutil.CreateTestUser(db, "admin@example.com", "Admin", "hash")
	group := testutil.CreateTestGroup(db, "Family", admin.ID)
	testutil.CreateTestGroupMember(db, group.ID, admin.ID, "admin")

	groupService := NewGroupService()

	created, _ := groupService.GenerateInviteCode(group.ID, admin.ID)

	invite, err := groupService.ValidateInvite(created.Code)
	if err != nil {
		t.Fatalf("ValidateInvite() error = %v", err)
	}

	if invite.ID != created.ID {
		t.Errorf("Invite ID = %d, want %d", invite.ID, created.ID)
	}
}

func TestGroupService_ValidateInvite_Revoked(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	admin := testutil.CreateTestUser(db, "admin@example.com", "Admin", "hash")
	group := testutil.CreateTestGroup(db, "Family", admin.ID)
	testutil.CreateTestGroupMember(db, group.ID, admin.ID, "admin")

	groupService := NewGroupService()

	created, _ := groupService.GenerateInviteCode(group.ID, admin.ID)

	// Revoke the invite
	db.Model(&models.GroupInvite{}).Where("id = ?", created.ID).Update("revoked", true)

	_, err := groupService.ValidateInvite(created.Code)
	if err != ErrInviteInvalid {
		t.Errorf("Expected ErrInviteInvalid, got %v", err)
	}
}

func TestGroupService_ValidateInvite_Expired(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	admin := testutil.CreateTestUser(db, "admin@example.com", "Admin", "hash")
	group := testutil.CreateTestGroup(db, "Family", admin.ID)
	testutil.CreateTestGroupMember(db, group.ID, admin.ID, "admin")

	groupService := NewGroupService()

	created, _ := groupService.GenerateInviteCode(group.ID, admin.ID)

	// Set invite to expired
	db.Model(&models.GroupInvite{}).Where("id = ?", created.ID).Update("expires_at", time.Now().Add(-time.Hour))

	_, err := groupService.ValidateInvite(created.Code)
	if err != ErrInviteExpired {
		t.Errorf("Expected ErrInviteExpired, got %v", err)
	}
}

func TestGroupService_ValidateInvite_MaxUsesReached(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	admin := testutil.CreateTestUser(db, "admin@example.com", "Admin", "hash")
	group := testutil.CreateTestGroup(db, "Family", admin.ID)
	testutil.CreateTestGroupMember(db, group.ID, admin.ID, "admin")

	groupService := NewGroupService()

	created, _ := groupService.GenerateInviteCode(group.ID, admin.ID)

	// Set max uses to 1 and used count to 1
	db.Model(&models.GroupInvite{}).Where("id = ?", created.ID).Updates(map[string]interface{}{
		"max_uses":   1,
		"used_count": 1,
	})

	_, err := groupService.ValidateInvite(created.Code)
	if err != ErrInviteMaxUsed {
		t.Errorf("Expected ErrInviteMaxUsed, got %v", err)
	}
}

func TestGroupService_AcceptInvite(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	admin := testutil.CreateTestUser(db, "admin@example.com", "Admin", "hash")
	newMember := testutil.CreateTestUser(db, "newmember@example.com", "New Member", "hash")
	group := testutil.CreateTestGroup(db, "Family", admin.ID)
	testutil.CreateTestGroupMember(db, group.ID, admin.ID, "admin")

	groupService := NewGroupService()

	invite, _ := groupService.GenerateInviteCode(group.ID, admin.ID)

	resultGroup, err := groupService.AcceptInvite(invite.Code, newMember.ID)
	if err != nil {
		t.Fatalf("AcceptInvite() error = %v", err)
	}

	if resultGroup.ID != group.ID {
		t.Errorf("Group ID = %d, want %d", resultGroup.ID, group.ID)
	}

	// Verify new member was added
	var member models.GroupMember
	if err := db.Where("group_id = ? AND user_id = ?", group.ID, newMember.ID).First(&member).Error; err != nil {
		t.Error("New member not found in group")
	}

	if member.Role != "member" {
		t.Errorf("Role = %s, want member", member.Role)
	}

	// Verify used count was incremented
	var updatedInvite models.GroupInvite
	db.First(&updatedInvite, invite.ID)
	if updatedInvite.UsedCount != 1 {
		t.Errorf("UsedCount = %d, want 1", updatedInvite.UsedCount)
	}
}

func TestGroupService_AcceptInvite_AlreadyMember(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	admin := testutil.CreateTestUser(db, "admin@example.com", "Admin", "hash")
	group := testutil.CreateTestGroup(db, "Family", admin.ID)
	testutil.CreateTestGroupMember(db, group.ID, admin.ID, "admin")

	groupService := NewGroupService()

	invite, _ := groupService.GenerateInviteCode(group.ID, admin.ID)

	// Admin is already a member
	_, err := groupService.AcceptInvite(invite.Code, admin.ID)
	if err != ErrAlreadyMember {
		t.Errorf("Expected ErrAlreadyMember, got %v", err)
	}
}

func TestGroupService_GetGroupInvites(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	admin := testutil.CreateTestUser(db, "admin@example.com", "Admin", "hash")
	group := testutil.CreateTestGroup(db, "Family", admin.ID)
	testutil.CreateTestGroupMember(db, group.ID, admin.ID, "admin")

	groupService := NewGroupService()

	// Create multiple invites
	groupService.GenerateInviteCode(group.ID, admin.ID)
	groupService.GenerateInviteCode(group.ID, admin.ID)

	invites, err := groupService.GetGroupInvites(group.ID, admin.ID)
	if err != nil {
		t.Fatalf("GetGroupInvites() error = %v", err)
	}

	if len(invites) != 2 {
		t.Errorf("Expected 2 invites, got %d", len(invites))
	}
}

func TestGroupService_GetGroupInvites_NotAdmin(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	admin := testutil.CreateTestUser(db, "admin@example.com", "Admin", "hash")
	member := testutil.CreateTestUser(db, "member@example.com", "Member", "hash")
	group := testutil.CreateTestGroup(db, "Family", admin.ID)
	testutil.CreateTestGroupMember(db, group.ID, admin.ID, "admin")
	testutil.CreateTestGroupMember(db, group.ID, member.ID, "member")

	groupService := NewGroupService()

	_, err := groupService.GetGroupInvites(group.ID, member.ID)
	if err != ErrNotGroupAdmin {
		t.Errorf("Expected ErrNotGroupAdmin, got %v", err)
	}
}

func TestGroupService_RevokeInvite(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	admin := testutil.CreateTestUser(db, "admin@example.com", "Admin", "hash")
	group := testutil.CreateTestGroup(db, "Family", admin.ID)
	testutil.CreateTestGroupMember(db, group.ID, admin.ID, "admin")

	groupService := NewGroupService()

	invite, _ := groupService.GenerateInviteCode(group.ID, admin.ID)

	err := groupService.RevokeInvite(invite.ID, admin.ID)
	if err != nil {
		t.Fatalf("RevokeInvite() error = %v", err)
	}

	// Verify invite is revoked
	var revokedInvite models.GroupInvite
	db.First(&revokedInvite, invite.ID)
	if !revokedInvite.Revoked {
		t.Error("Invite should be revoked")
	}
}

func TestGroupService_RevokeInvite_NotAdmin(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	admin := testutil.CreateTestUser(db, "admin@example.com", "Admin", "hash")
	member := testutil.CreateTestUser(db, "member@example.com", "Member", "hash")
	group := testutil.CreateTestGroup(db, "Family", admin.ID)
	testutil.CreateTestGroupMember(db, group.ID, admin.ID, "admin")
	testutil.CreateTestGroupMember(db, group.ID, member.ID, "member")

	groupService := NewGroupService()

	invite, _ := groupService.GenerateInviteCode(group.ID, admin.ID)

	err := groupService.RevokeInvite(invite.ID, member.ID)
	if err != ErrNotGroupAdmin {
		t.Errorf("Expected ErrNotGroupAdmin, got %v", err)
	}
}

func TestGroupService_GetGroupByID(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	admin := testutil.CreateTestUser(db, "admin@example.com", "Admin", "hash")
	group := testutil.CreateTestGroup(db, "Family", admin.ID)
	testutil.CreateTestGroupMember(db, group.ID, admin.ID, "admin")

	groupService := NewGroupService()

	result, err := groupService.GetGroupByID(group.ID)
	if err != nil {
		t.Fatalf("GetGroupByID() error = %v", err)
	}

	if result.ID != group.ID {
		t.Errorf("Group ID = %d, want %d", result.ID, group.ID)
	}

	if result.Name != "Family" {
		t.Errorf("Group Name = %s, want Family", result.Name)
	}
}

func TestGroupService_GetGroupByID_NotFound(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	groupService := NewGroupService()

	_, err := groupService.GetGroupByID(99999)
	if err != ErrGroupNotFound {
		t.Errorf("Expected ErrGroupNotFound, got %v", err)
	}
}

func TestGroupService_IsGroupAdmin(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	admin := testutil.CreateTestUser(db, "admin@example.com", "Admin", "hash")
	member := testutil.CreateTestUser(db, "member@example.com", "Member", "hash")
	group := testutil.CreateTestGroup(db, "Family", admin.ID)
	testutil.CreateTestGroupMember(db, group.ID, admin.ID, "admin")
	testutil.CreateTestGroupMember(db, group.ID, member.ID, "member")

	groupService := NewGroupService()

	if !groupService.IsGroupAdmin(group.ID, admin.ID) {
		t.Error("Admin should be recognized as admin")
	}

	if groupService.IsGroupAdmin(group.ID, member.ID) {
		t.Error("Member should not be recognized as admin")
	}
}

func TestGroupService_IsGroupMember(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	admin := testutil.CreateTestUser(db, "admin@example.com", "Admin", "hash")
	outsider := testutil.CreateTestUser(db, "outsider@example.com", "Outsider", "hash")
	group := testutil.CreateTestGroup(db, "Family", admin.ID)
	testutil.CreateTestGroupMember(db, group.ID, admin.ID, "admin")

	groupService := NewGroupService()

	if !groupService.IsGroupMember(group.ID, admin.ID) {
		t.Error("Admin should be recognized as member")
	}

	if groupService.IsGroupMember(group.ID, outsider.ID) {
		t.Error("Outsider should not be recognized as member")
	}
}

func TestGroupService_GetGroupMembers(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	user1 := testutil.CreateTestUser(db, "user1@example.com", "User 1", "hash")
	user2 := testutil.CreateTestUser(db, "user2@example.com", "User 2", "hash")
	group := testutil.CreateTestGroup(db, "Family", user1.ID)
	testutil.CreateTestGroupMember(db, group.ID, user1.ID, "admin")
	testutil.CreateTestGroupMember(db, group.ID, user2.ID, "member")

	groupService := NewGroupService()

	members, err := groupService.GetGroupMembers(group.ID)
	if err != nil {
		t.Fatalf("GetGroupMembers() error = %v", err)
	}

	if len(members) != 2 {
		t.Errorf("Expected 2 members, got %d", len(members))
	}
}

func TestGroupService_LeaveGroup(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	admin := testutil.CreateTestUser(db, "admin@example.com", "Admin", "hash")
	member := testutil.CreateTestUser(db, "member@example.com", "Member", "hash")
	group := testutil.CreateTestGroup(db, "Family", admin.ID)
	testutil.CreateTestGroupMember(db, group.ID, admin.ID, "admin")
	testutil.CreateTestGroupMember(db, group.ID, member.ID, "member")

	groupService := NewGroupService()

	err := groupService.LeaveGroup(group.ID, member.ID)
	if err != nil {
		t.Fatalf("LeaveGroup() error = %v", err)
	}

	// Verify member was removed
	var membership models.GroupMember
	if err := db.Where("group_id = ? AND user_id = ?", group.ID, member.ID).First(&membership).Error; err == nil {
		t.Error("Member should have been removed from group")
	}

	// Group should still exist
	var existingGroup models.FamilyGroup
	if err := db.First(&existingGroup, group.ID).Error; err != nil {
		t.Error("Group should still exist")
	}
}

func TestGroupService_LeaveGroup_LastMemberDeletesGroup(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	admin := testutil.CreateTestUser(db, "admin@example.com", "Admin", "hash")
	group := testutil.CreateTestGroup(db, "Family", admin.ID)
	testutil.CreateTestGroupMember(db, group.ID, admin.ID, "admin")

	groupService := NewGroupService()

	err := groupService.LeaveGroup(group.ID, admin.ID)
	if err != nil {
		t.Fatalf("LeaveGroup() error = %v", err)
	}

	// Group should be deleted
	var deletedGroup models.FamilyGroup
	if err := db.First(&deletedGroup, group.ID).Error; err == nil {
		t.Error("Group should have been deleted")
	}
}

func TestGroupService_LeaveGroup_LastAdminCannotLeave(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	admin := testutil.CreateTestUser(db, "admin@example.com", "Admin", "hash")
	member := testutil.CreateTestUser(db, "member@example.com", "Member", "hash")
	group := testutil.CreateTestGroup(db, "Family", admin.ID)
	testutil.CreateTestGroupMember(db, group.ID, admin.ID, "admin")
	testutil.CreateTestGroupMember(db, group.ID, member.ID, "member")

	groupService := NewGroupService()

	err := groupService.LeaveGroup(group.ID, admin.ID)
	if err != ErrLastAdminCannotLeave {
		t.Errorf("Expected ErrLastAdminCannotLeave, got %v", err)
	}
}

func TestGroupService_LeaveGroup_NotMember(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	admin := testutil.CreateTestUser(db, "admin@example.com", "Admin", "hash")
	outsider := testutil.CreateTestUser(db, "outsider@example.com", "Outsider", "hash")
	group := testutil.CreateTestGroup(db, "Family", admin.ID)
	testutil.CreateTestGroupMember(db, group.ID, admin.ID, "admin")

	groupService := NewGroupService()

	err := groupService.LeaveGroup(group.ID, outsider.ID)
	if err != ErrNotGroupMember {
		t.Errorf("Expected ErrNotGroupMember, got %v", err)
	}
}

func TestGroupService_DeleteGroup(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	admin := testutil.CreateTestUser(db, "admin@example.com", "Admin", "hash")
	group := testutil.CreateTestGroup(db, "Family", admin.ID)
	testutil.CreateTestGroupMember(db, group.ID, admin.ID, "admin")

	groupService := NewGroupService()

	// Create an invite too
	groupService.GenerateInviteCode(group.ID, admin.ID)

	err := groupService.DeleteGroup(group.ID, admin.ID)
	if err != nil {
		t.Fatalf("DeleteGroup() error = %v", err)
	}

	// Verify group is deleted
	var deletedGroup models.FamilyGroup
	if err := db.First(&deletedGroup, group.ID).Error; err == nil {
		t.Error("Group should have been deleted")
	}

	// Verify memberships are deleted
	var members []models.GroupMember
	db.Where("group_id = ?", group.ID).Find(&members)
	if len(members) > 0 {
		t.Error("All memberships should have been deleted")
	}

	// Verify invites are deleted
	var invites []models.GroupInvite
	db.Where("group_id = ?", group.ID).Find(&invites)
	if len(invites) > 0 {
		t.Error("All invites should have been deleted")
	}
}

func TestGroupService_DeleteGroup_NotAdmin(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	admin := testutil.CreateTestUser(db, "admin@example.com", "Admin", "hash")
	member := testutil.CreateTestUser(db, "member@example.com", "Member", "hash")
	group := testutil.CreateTestGroup(db, "Family", admin.ID)
	testutil.CreateTestGroupMember(db, group.ID, admin.ID, "admin")
	testutil.CreateTestGroupMember(db, group.ID, member.ID, "member")

	groupService := NewGroupService()

	err := groupService.DeleteGroup(group.ID, member.ID)
	if err != ErrNotGroupAdmin {
		t.Errorf("Expected ErrNotGroupAdmin, got %v", err)
	}
}

func TestGroupService_RemoveMember(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	admin := testutil.CreateTestUser(db, "admin@example.com", "Admin", "hash")
	member := testutil.CreateTestUser(db, "member@example.com", "Member", "hash")
	group := testutil.CreateTestGroup(db, "Family", admin.ID)
	testutil.CreateTestGroupMember(db, group.ID, admin.ID, "admin")
	testutil.CreateTestGroupMember(db, group.ID, member.ID, "member")

	groupService := NewGroupService()

	err := groupService.RemoveMember(group.ID, member.ID, admin.ID)
	if err != nil {
		t.Fatalf("RemoveMember() error = %v", err)
	}

	// Verify member was removed
	var membership models.GroupMember
	if err := db.Where("group_id = ? AND user_id = ?", group.ID, member.ID).First(&membership).Error; err == nil {
		t.Error("Member should have been removed")
	}
}

func TestGroupService_RemoveMember_CannotRemoveSelf(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	admin := testutil.CreateTestUser(db, "admin@example.com", "Admin", "hash")
	group := testutil.CreateTestGroup(db, "Family", admin.ID)
	testutil.CreateTestGroupMember(db, group.ID, admin.ID, "admin")

	groupService := NewGroupService()

	err := groupService.RemoveMember(group.ID, admin.ID, admin.ID)
	if err != ErrNotGroupMember {
		t.Errorf("Expected ErrNotGroupMember (cannot remove self), got %v", err)
	}
}

func TestGroupService_RemoveMember_NotAdmin(t *testing.T) {
	db := testutil.SetupTestDB()
	database.DB = db

	admin := testutil.CreateTestUser(db, "admin@example.com", "Admin", "hash")
	member1 := testutil.CreateTestUser(db, "member1@example.com", "Member 1", "hash")
	member2 := testutil.CreateTestUser(db, "member2@example.com", "Member 2", "hash")
	group := testutil.CreateTestGroup(db, "Family", admin.ID)
	testutil.CreateTestGroupMember(db, group.ID, admin.ID, "admin")
	testutil.CreateTestGroupMember(db, group.ID, member1.ID, "member")
	testutil.CreateTestGroupMember(db, group.ID, member2.ID, "member")

	groupService := NewGroupService()

	// Member1 tries to remove Member2
	err := groupService.RemoveMember(group.ID, member2.ID, member1.ID)
	if err != ErrNotGroupAdmin {
		t.Errorf("Expected ErrNotGroupAdmin, got %v", err)
	}
}
