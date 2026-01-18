package services

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"poc-finance/internal/database"
	"poc-finance/internal/models"
)

var (
	ErrGroupNotFound        = errors.New("grupo não encontrado")
	ErrInviteNotFound       = errors.New("convite não encontrado")
	ErrInviteExpired        = errors.New("convite expirado")
	ErrInviteInvalid        = errors.New("convite inválido")
	ErrNotGroupAdmin        = errors.New("você não é administrador deste grupo")
	ErrAlreadyMember        = errors.New("você já é membro deste grupo")
	ErrInviteMaxUsed        = errors.New("convite atingiu o limite de usos")
	ErrNotGroupMember       = errors.New("você não é membro deste grupo")
	ErrLastAdminCannotLeave = errors.New("você é o único administrador e não pode sair do grupo")
)

const (
	InviteCodeDuration = 7 * 24 * time.Hour // 7 days
)

type GroupService struct{}

func NewGroupService() *GroupService {
	return &GroupService{}
}

// GenerateInviteCode creates a new invite code for a group
func (s *GroupService) GenerateInviteCode(groupID, userID uint) (*models.GroupInvite, error) {
	// Verify user is admin of the group
	var member models.GroupMember
	if err := database.DB.Where("group_id = ? AND user_id = ? AND role = ?", groupID, userID, "admin").First(&member).Error; err != nil {
		return nil, ErrNotGroupAdmin
	}

	// Generate random code
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return nil, err
	}
	code := hex.EncodeToString(bytes)

	// Create invite
	invite := &models.GroupInvite{
		Code:        code,
		GroupID:     groupID,
		CreatedByID: userID,
		ExpiresAt:   time.Now().Add(InviteCodeDuration),
		MaxUses:     0, // unlimited by default
		UsedCount:   0,
		Revoked:     false,
	}

	if err := database.DB.Create(invite).Error; err != nil {
		return nil, err
	}

	// Preload group info
	database.DB.Preload("Group").Preload("CreatedBy").First(invite, invite.ID)

	return invite, nil
}

// GetInviteByCode retrieves an invite by its code
func (s *GroupService) GetInviteByCode(code string) (*models.GroupInvite, error) {
	var invite models.GroupInvite
	if err := database.DB.Where("code = ?", code).Preload("Group").Preload("CreatedBy").First(&invite).Error; err != nil {
		return nil, ErrInviteNotFound
	}

	return &invite, nil
}

// ValidateInvite checks if an invite is valid
func (s *GroupService) ValidateInvite(code string) (*models.GroupInvite, error) {
	invite, err := s.GetInviteByCode(code)
	if err != nil {
		return nil, err
	}

	if invite.Revoked {
		return nil, ErrInviteInvalid
	}

	if time.Now().After(invite.ExpiresAt) {
		return nil, ErrInviteExpired
	}

	if invite.MaxUses > 0 && invite.UsedCount >= invite.MaxUses {
		return nil, ErrInviteMaxUsed
	}

	return invite, nil
}

// AcceptInvite adds a user to a group using an invite code
func (s *GroupService) AcceptInvite(code string, userID uint) (*models.FamilyGroup, error) {
	invite, err := s.ValidateInvite(code)
	if err != nil {
		return nil, err
	}

	// Check if user is already a member
	var existingMember models.GroupMember
	if err := database.DB.Where("group_id = ? AND user_id = ?", invite.GroupID, userID).First(&existingMember).Error; err == nil {
		return nil, ErrAlreadyMember
	}

	// Add user as member
	member := &models.GroupMember{
		GroupID: invite.GroupID,
		UserID:  userID,
		Role:    "member",
	}

	if err := database.DB.Create(member).Error; err != nil {
		return nil, err
	}

	// Increment used count
	database.DB.Model(invite).Update("used_count", invite.UsedCount+1)

	// Return the group
	var group models.FamilyGroup
	database.DB.Preload("Members.User").First(&group, invite.GroupID)

	return &group, nil
}

// GetGroupInvites returns all active invites for a group
func (s *GroupService) GetGroupInvites(groupID, userID uint) ([]models.GroupInvite, error) {
	// Verify user is admin of the group
	var member models.GroupMember
	if err := database.DB.Where("group_id = ? AND user_id = ? AND role = ?", groupID, userID, "admin").First(&member).Error; err != nil {
		return nil, ErrNotGroupAdmin
	}

	var invites []models.GroupInvite
	database.DB.Where("group_id = ? AND revoked = ? AND expires_at > ?", groupID, false, time.Now()).
		Preload("CreatedBy").
		Order("created_at DESC").
		Find(&invites)

	return invites, nil
}

// RevokeInvite revokes an invite
func (s *GroupService) RevokeInvite(inviteID, userID uint) error {
	var invite models.GroupInvite
	if err := database.DB.First(&invite, inviteID).Error; err != nil {
		return ErrInviteNotFound
	}

	// Verify user is admin of the group
	var member models.GroupMember
	if err := database.DB.Where("group_id = ? AND user_id = ? AND role = ?", invite.GroupID, userID, "admin").First(&member).Error; err != nil {
		return ErrNotGroupAdmin
	}

	return database.DB.Model(&invite).Update("revoked", true).Error
}

// GetGroupByID retrieves a group by ID
func (s *GroupService) GetGroupByID(groupID uint) (*models.FamilyGroup, error) {
	var group models.FamilyGroup
	if err := database.DB.Preload("Members.User").First(&group, groupID).Error; err != nil {
		return nil, ErrGroupNotFound
	}
	return &group, nil
}

// IsGroupAdmin checks if a user is an admin of a group
func (s *GroupService) IsGroupAdmin(groupID, userID uint) bool {
	var member models.GroupMember
	err := database.DB.Where("group_id = ? AND user_id = ? AND role = ?", groupID, userID, "admin").First(&member).Error
	return err == nil
}

// IsGroupMember checks if a user is a member of a group
func (s *GroupService) IsGroupMember(groupID, userID uint) bool {
	var member models.GroupMember
	err := database.DB.Where("group_id = ? AND user_id = ?", groupID, userID).First(&member).Error
	return err == nil
}

// GetGroupMembers returns all users that are members of a group
func (s *GroupService) GetGroupMembers(groupID uint) ([]models.User, error) {
	var members []models.GroupMember
	if err := database.DB.Where("group_id = ?", groupID).
		Preload("User").
		Find(&members).Error; err != nil {
		return nil, err
	}

	users := make([]models.User, len(members))
	for i, m := range members {
		users[i] = m.User
	}
	return users, nil
}

// LeaveGroup removes a user from a group
// If the last member leaves, the group is automatically deleted
func (s *GroupService) LeaveGroup(groupID, userID uint) error {
	// Check if user is a member
	var member models.GroupMember
	if err := database.DB.Where("group_id = ? AND user_id = ?", groupID, userID).First(&member).Error; err != nil {
		return ErrNotGroupMember
	}

	// Count total members in the group
	var memberCount int64
	database.DB.Model(&models.GroupMember{}).
		Where("group_id = ?", groupID).
		Count(&memberCount)

	// If this is the last member, delete the group entirely
	if memberCount <= 1 {
		// Delete the membership first
		if err := database.DB.Delete(&member).Error; err != nil {
			return err
		}
		// Delete associated invites
		database.DB.Where("group_id = ?", groupID).Delete(&models.GroupInvite{})
		// Delete the group
		return database.DB.Delete(&models.FamilyGroup{}, groupID).Error
	}

	// If user is admin, check if they're the last admin
	if member.Role == "admin" {
		var adminCount int64
		database.DB.Model(&models.GroupMember{}).
			Where("group_id = ? AND role = ?", groupID, "admin").
			Count(&adminCount)

		if adminCount <= 1 {
			return ErrLastAdminCannotLeave
		}
	}

	// Soft delete the membership
	return database.DB.Delete(&member).Error
}

// DeleteGroup deletes a group and all its related data (only admin can delete)
func (s *GroupService) DeleteGroup(groupID, userID uint) error {
	// Verify user is admin of the group
	var member models.GroupMember
	if err := database.DB.Where("group_id = ? AND user_id = ? AND role = ?", groupID, userID, "admin").First(&member).Error; err != nil {
		return ErrNotGroupAdmin
	}

	// Delete all memberships
	database.DB.Where("group_id = ?", groupID).Delete(&models.GroupMember{})

	// Delete all invites
	database.DB.Where("group_id = ?", groupID).Delete(&models.GroupInvite{})

	// Delete the group
	return database.DB.Delete(&models.FamilyGroup{}, groupID).Error
}

// RemoveMember removes a member from a group (only admin can remove)
func (s *GroupService) RemoveMember(groupID, memberUserID, adminUserID uint) error {
	// Cannot remove yourself (use LeaveGroup instead)
	if memberUserID == adminUserID {
		return ErrNotGroupMember
	}

	// Verify requester is admin of the group
	var adminMember models.GroupMember
	if err := database.DB.Where("group_id = ? AND user_id = ? AND role = ?", groupID, adminUserID, "admin").First(&adminMember).Error; err != nil {
		return ErrNotGroupAdmin
	}

	// Find the member to remove
	var member models.GroupMember
	if err := database.DB.Where("group_id = ? AND user_id = ?", groupID, memberUserID).First(&member).Error; err != nil {
		return ErrNotGroupMember
	}

	// Delete the membership
	return database.DB.Delete(&member).Error
}
