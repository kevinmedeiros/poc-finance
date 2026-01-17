package database

import (
	"log"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"poc-finance/internal/models"
)

var DB *gorm.DB

func Init() error {
	var err error
	DB, err = gorm.Open(sqlite.Open("finance.db"), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
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
		&models.CreditCard{},
		&models.Installment{},
		&models.Bill{},
		&models.Settings{},
		&models.ExpensePayment{},
		&models.FamilyGroup{},
		&models.GroupMember{},
		&models.GroupInvite{},
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
