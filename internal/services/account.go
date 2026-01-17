package services

import (
	"errors"
	"time"

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

	// Get joint accounts from user's groups
	var groupIDs []uint
	database.DB.Model(&models.GroupMember{}).Where("user_id = ?", userID).Pluck("group_id", &groupIDs)

	if len(groupIDs) > 0 {
		var jointAccounts []models.Account
		if err := database.DB.Where("group_id IN ? AND type = ?", groupIDs, models.AccountTypeJoint).Find(&jointAccounts).Error; err != nil {
			return nil, err
		}
		accounts = append(accounts, jointAccounts...)
	}

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

	// For joint accounts, check if user is a member of the group
	if account.Type == models.AccountTypeJoint && account.GroupID != nil {
		var member models.GroupMember
		if err := database.DB.Where("group_id = ? AND user_id = ?", *account.GroupID, userID).First(&member).Error; err == nil {
			return true
		}
	}

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

// CreateJointAccount creates a new joint account linked to a group
func (s *AccountService) CreateJointAccount(name string, groupID, userID uint) (*models.Account, error) {
	// Verify user is a member of the group
	var member models.GroupMember
	if err := database.DB.Where("group_id = ? AND user_id = ?", groupID, userID).First(&member).Error; err != nil {
		return nil, ErrUnauthorized
	}

	account := &models.Account{
		Name:    name,
		Type:    models.AccountTypeJoint,
		UserID:  userID, // Creator
		GroupID: &groupID,
	}

	if err := database.DB.Create(account).Error; err != nil {
		return nil, err
	}

	return account, nil
}

// GetGroupJointAccounts returns all joint accounts for a group
func (s *AccountService) GetGroupJointAccounts(groupID uint) ([]models.Account, error) {
	var accounts []models.Account
	if err := database.DB.Where("group_id = ? AND type = ?", groupID, models.AccountTypeJoint).Find(&accounts).Error; err != nil {
		return nil, err
	}
	return accounts, nil
}

// DeleteJointAccount deletes a joint account (only group members can delete)
func (s *AccountService) DeleteJointAccount(accountID, userID uint) error {
	var account models.Account
	if err := database.DB.First(&account, accountID).Error; err != nil {
		return ErrAccountNotFound
	}

	if account.Type != models.AccountTypeJoint || account.GroupID == nil {
		return ErrUnauthorized
	}

	// Verify user is a member of the group
	var member models.GroupMember
	if err := database.DB.Where("group_id = ? AND user_id = ?", *account.GroupID, userID).First(&member).Error; err != nil {
		return ErrUnauthorized
	}

	return database.DB.Delete(&account).Error
}

// GetAccountMembers returns all members for an account (for joint accounts returns group members)
func (s *AccountService) GetAccountMembers(accountID uint) ([]models.User, error) {
	var account models.Account
	if err := database.DB.First(&account, accountID).Error; err != nil {
		return nil, ErrAccountNotFound
	}

	// For individual accounts, return only the owner
	if account.Type == models.AccountTypeIndividual {
		var user models.User
		if err := database.DB.First(&user, account.UserID).Error; err != nil {
			return nil, err
		}
		return []models.User{user}, nil
	}

	// For joint accounts, return all group members
	if account.GroupID == nil {
		return nil, ErrAccountNotFound
	}

	var members []models.GroupMember
	if err := database.DB.Preload("User").Where("group_id = ?", *account.GroupID).Find(&members).Error; err != nil {
		return nil, err
	}

	users := make([]models.User, len(members))
	for i, m := range members {
		users[i] = m.User
	}

	return users, nil
}

// GetAccountByID returns an account by its ID
func (s *AccountService) GetAccountByID(accountID uint) (*models.Account, error) {
	var account models.Account
	if err := database.DB.First(&account, accountID).Error; err != nil {
		return nil, ErrAccountNotFound
	}
	return &account, nil
}

// AccountBalance holds balance information for an account
type AccountBalance struct {
	Account       models.Account
	TotalIncome   float64
	TotalExpenses float64
	Balance       float64
}

// GetAccountBalance calculates the current balance for a specific account
func (s *AccountService) GetAccountBalance(accountID uint) (*AccountBalance, error) {
	var account models.Account
	if err := database.DB.First(&account, accountID).Error; err != nil {
		return nil, ErrAccountNotFound
	}

	// Calculate total income (net)
	var totalIncome float64
	database.DB.Model(&models.Income{}).
		Where("account_id = ?", accountID).
		Select("COALESCE(SUM(net_amount), 0)").
		Scan(&totalIncome)

	// Calculate total fixed expenses (active)
	var totalFixed float64
	database.DB.Model(&models.Expense{}).
		Where("account_id = ? AND type = ? AND active = ?", accountID, models.ExpenseTypeFixed, true).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&totalFixed)

	// Calculate total variable expenses (active)
	var totalVariable float64
	database.DB.Model(&models.Expense{}).
		Where("account_id = ? AND type = ? AND active = ?", accountID, models.ExpenseTypeVariable, true).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&totalVariable)

	// Calculate total bills
	var totalBills float64
	database.DB.Model(&models.Bill{}).
		Where("account_id = ?", accountID).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&totalBills)

	// Calculate total credit card installments for current month
	now := time.Now()
	year, month := now.Year(), int(now.Month())
	var totalCards float64
	var creditCards []models.CreditCard
	database.DB.Where("account_id = ?", accountID).Preload("Installments").Find(&creditCards)
	for _, card := range creditCards {
		for _, inst := range card.Installments {
			installmentMonth := inst.StartDate
			for i := 1; i <= inst.TotalInstallments; i++ {
				if installmentMonth.Year() == year && int(installmentMonth.Month()) == month {
					totalCards += inst.InstallmentAmount
					break
				}
				installmentMonth = installmentMonth.AddDate(0, 1, 0)
			}
		}
	}

	totalExpenses := totalFixed + totalVariable + totalBills + totalCards

	return &AccountBalance{
		Account:       account,
		TotalIncome:   totalIncome,
		TotalExpenses: totalExpenses,
		Balance:       totalIncome - totalExpenses,
	}, nil
}

// GetUserAccountsWithBalances returns all user accounts with their balances
func (s *AccountService) GetUserAccountsWithBalances(userID uint) ([]AccountBalance, error) {
	accounts, err := s.GetUserAccounts(userID)
	if err != nil {
		return nil, err
	}

	balances := make([]AccountBalance, 0, len(accounts))
	for _, acc := range accounts {
		balance, err := s.GetAccountBalance(acc.ID)
		if err != nil {
			continue
		}
		balances = append(balances, *balance)
	}

	return balances, nil
}

// GetGroupJointAccountIDs returns all joint account IDs for a group
func (s *AccountService) GetGroupJointAccountIDs(groupID uint) ([]uint, error) {
	var accountIDs []uint
	if err := database.DB.Model(&models.Account{}).
		Where("group_id = ? AND type = ?", groupID, models.AccountTypeJoint).
		Pluck("id", &accountIDs).Error; err != nil {
		return nil, err
	}
	return accountIDs, nil
}

// GetGroupJointAccountsWithBalances returns all joint accounts for a group with their balances
func (s *AccountService) GetGroupJointAccountsWithBalances(groupID uint) ([]AccountBalance, error) {
	accounts, err := s.GetGroupJointAccounts(groupID)
	if err != nil {
		return nil, err
	}

	balances := make([]AccountBalance, 0, len(accounts))
	for _, acc := range accounts {
		balance, err := s.GetAccountBalance(acc.ID)
		if err != nil {
			continue
		}
		balances = append(balances, *balance)
	}

	return balances, nil
}
