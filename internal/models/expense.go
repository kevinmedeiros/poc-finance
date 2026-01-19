package models

import "gorm.io/gorm"

// ExpenseType representa o tipo de uma despesa
type ExpenseType string

const (
	// ExpenseTypeFixed representa despesas fixas com valores regulares
	ExpenseTypeFixed    ExpenseType = "fixed"
	// ExpenseTypeVariable representa despesas variáveis com valores flutuantes
	ExpenseTypeVariable ExpenseType = "variable"
)

// Expense representa uma despesa associada a uma conta
// Pode ser dividida entre membros do grupo através de ExpenseSplit
type Expense struct {
	gorm.Model
	AccountID uint           `json:"account_id" gorm:"not null;index"` // ID da conta à qual a despesa pertence
	Account   Account        `json:"-" gorm:"foreignKey:AccountID"`
	Name      string         `json:"name" gorm:"not null"`                // Nome descritivo da despesa
	Amount    float64        `json:"amount" gorm:"not null"`              // Valor da despesa
	Type      ExpenseType    `json:"type" gorm:"not null"`                // Tipo da despesa (fixa ou variável)
	DueDay    int            `json:"due_day" gorm:"default:1"`            // Dia do vencimento (1-31)
	Category  string         `json:"category"`                            // Categoria da despesa (alimentação, transporte, etc)
	Active    bool           `json:"active" gorm:"default:true"`          // Indica se a despesa está ativa
	IsSplit   bool           `json:"is_split" gorm:"default:false"`       // Indica se a despesa é dividida entre membros
	Splits    []ExpenseSplit `json:"splits" gorm:"foreignKey:ExpenseID"` // Divisões da despesa entre membros
}

func (e *Expense) TableName() string {
	return "expenses"
}
