package database

import (
	"log"
	"os"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"poc-finance/internal/models"
)

// isProduction returns true if ENV is set to "production"
func isProduction() bool {
	return os.Getenv("ENV") == "production"
}

var DB *gorm.DB

func Init() error {
	var err error
	// Use error-only logging in production to avoid exposing sensitive data
	// Use info logging in development for debugging
	logLevel := logger.Info
	if isProduction() {
		logLevel = logger.Error
	}

	// Get database path from environment or use default
	dbPath := os.Getenv("DATABASE_PATH")
	if dbPath == "" {
		dbPath = "finance.db"
	}

	DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logLevel),
	})
	if err != nil {
		return err
	}

	log.Println("Conectado ao banco de dados SQLite")

	// Auto migrate
	err = DB.AutoMigrate(
		&models.User{},
		&models.RefreshToken{},
		&models.PasswordResetToken{},
		&models.Account{},
		&models.Income{},
		&models.Expense{},
		&models.ExpenseSplit{},
		&models.CreditCard{},
		&models.Installment{},
		&models.Bill{},
		&models.Settings{},
		&models.ExpensePayment{},
		&models.FamilyGroup{},
		&models.GroupMember{},
		&models.GroupInvite{},
		&models.GroupGoal{},
		&models.GoalContribution{},
		&models.Notification{},
		&models.RecurringTransaction{},
		&models.HealthScore{},
		&models.Budget{},
		&models.BudgetCategory{},
	)
	if err != nil {
		return err
	}

	// Inicializa configurações padrão se não existirem
	initDefaultSettings()

	log.Println("Migrações executadas com sucesso")
	return nil
}

func initDefaultSettings() {
	defaults := map[string]string{
		models.SettingProLabore:   "0",       // Pró-labore não configurado
		models.SettingINSSCeiling: "7786.02", // Teto INSS 2024
		models.SettingINSSRate:    "11",      // 11%
	}

	for key, value := range defaults {
		var setting models.Settings
		result := DB.Where("key = ?", key).First(&setting)
		if result.Error != nil {
			DB.Create(&models.Settings{Key: key, Value: value})
		}
	}
}

func GetDB() *gorm.DB {
	return DB
}
