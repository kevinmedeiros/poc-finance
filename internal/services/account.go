package services

import (
	"errors"

	"poc-finance/internal/database"
	"poc-finance/internal/models"
)

var (
	ErrAccountNotFound = errors.New("conta não encontrada")
	ErrUnauthorized    = errors.New("sem permissão para acessar esta conta")
)

type AccountService struct{}

func NewAccountService() *AccountService {
	return &AccountService{}
}

// GetUserAccounts returns all accounts accessible by a user (individual + joint accounts from groups)
func (s *AccountService) GetUserAccounts(userID uint) ([]models.Account, error) {
	var accounts []models.Account

	// Get user's individual account
	if err := database.DB.Where("user_id = ? AND type = ?", userID, models.AccountTypeIndividual).Find(&accounts).Error; err != nil {
		return nil, err
	}

	// TODO: In future iterations, also get joint accounts from user's groups

	return accounts, nil
}

// GetUserAccountIDs returns all account IDs accessible by a user
func (s *AccountService) GetUserAccountIDs(userID uint) ([]uint, error) {
	accounts, err := s.GetUserAccounts(userID)
	if err != nil {
		return nil, err
	}

	ids := make([]uint, len(accounts))
	for i, acc := range accounts {
		ids[i] = acc.ID
	}

	return ids, nil
}

// GetUserIndividualAccount returns the user's individual (personal) account
func (s *AccountService) GetUserIndividualAccount(userID uint) (*models.Account, error) {
	var account models.Account
	if err := database.DB.Where("user_id = ? AND type = ?", userID, models.AccountTypeIndividual).First(&account).Error; err != nil {
		return nil, ErrAccountNotFound
	}
	return &account, nil
}

// CanUserAccessAccount checks if a user has access to a specific account
func (s *AccountService) CanUserAccessAccount(userID, accountID uint) bool {
	var account models.Account
	if err := database.DB.First(&account, accountID).Error; err != nil {
		return false
	}

	// User can access their own individual account
	if account.Type == models.AccountTypeIndividual && account.UserID == userID {
		return true
	}

	// TODO: In future iterations, check if user is a member of the group for joint accounts

	return false
}

// EnsureUserHasAccount creates an individual account for a user if they don't have one
// (useful for migrating existing users)
func (s *AccountService) EnsureUserHasAccount(userID uint) (*models.Account, error) {
	// Check if user already has an individual account
	account, err := s.GetUserIndividualAccount(userID)
	if err == nil {
		return account, nil
	}

	// Create individual account
	account = &models.Account{
		Name:   "Conta Pessoal",
		Type:   models.AccountTypeIndividual,
		UserID: userID,
	}

	if err := database.DB.Create(account).Error; err != nil {
		return nil, err
	}

	return account, nil
}
