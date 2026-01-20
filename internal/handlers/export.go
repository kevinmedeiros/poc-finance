package handlers

import (
	"encoding/csv"
	"fmt"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/xuri/excelize/v2"

	"poc-finance/internal/database"
	"poc-finance/internal/i18n"
	"poc-finance/internal/models"
	"poc-finance/internal/services"
)

type ExportHandler struct{}

func NewExportHandler() *ExportHandler {
	return &ExportHandler{}
}

func (h *ExportHandler) ExportYear(c echo.Context) error {
	yearStr := c.QueryParam("year")
	year, err := strconv.Atoi(yearStr)
	if err != nil {
		year = time.Now().Year()
	}

	format := c.QueryParam("format")
	if format == "" {
		format = "xlsx"
	}

	if format == "csv" {
		return h.exportYearCSV(c, year)
	}

	f := excelize.NewFile()
	defer f.Close()

	// Cria sheet de resumo mensal
	h.createSummarySheet(f, year)

	// Cria sheet de recebimentos
	h.createIncomesSheet(f, year)

	// Cria sheet de despesas
	h.createExpensesSheet(f)

	// Cria sheet de cartões
	h.createCardsSheet(f)

	// Remove sheet padrão
	f.DeleteSheet("Sheet1")

	// Define headers para download
	c.Response().Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=financeiro_%d.xlsx", year))

	return f.Write(c.Response().Writer)
}

func (h *ExportHandler) createSummarySheet(f *excelize.File, year int) {
	sheet := "Resumo Mensal"
	f.NewSheet(sheet)

	// Headers
	headers := []string{"Mês", "Receita Bruta", "Imposto", "Receita Líquida", "Despesas Fixas", "Despesas Variáveis", "Cartões", "Contas", "Total Despesas", "Saldo"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
	}

	// Estilo do header
	style, _ := f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true},
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#4472C4"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center"},
	})
	f.SetCellStyle(sheet, "A1", "J1", style)

	// Dados
	summaries := services.GetYearlySummaries(database.DB, year)

	for i, s := range summaries {
		row := i + 2
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), i18n.MonthNamesSlice[i])
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), s.TotalIncomeGross)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), s.TotalTax)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), s.TotalIncomeNet)
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), s.TotalFixed)
		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), s.TotalVariable)
		f.SetCellValue(sheet, fmt.Sprintf("G%d", row), s.TotalCards)
		f.SetCellValue(sheet, fmt.Sprintf("H%d", row), s.TotalBills)
		f.SetCellValue(sheet, fmt.Sprintf("I%d", row), s.TotalExpenses)
		f.SetCellValue(sheet, fmt.Sprintf("J%d", row), s.Balance)
	}

	// Ajusta largura das colunas
	for i := 1; i <= 10; i++ {
		col, _ := excelize.ColumnNumberToName(i)
		f.SetColWidth(sheet, col, col, 15)
	}
}

func (h *ExportHandler) createIncomesSheet(f *excelize.File, year int) {
	sheet := "Recebimentos"
	f.NewSheet(sheet)

	headers := []string{"Data", "Valor USD", "Taxa Câmbio", "Valor BRL", "Imposto", "Líquido", "Descrição"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
	}

	style, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#70AD47"}, Pattern: 1},
	})
	f.SetCellStyle(sheet, "A1", "G1", style)

	var incomes []models.Income
	startDate := time.Date(year, 1, 1, 0, 0, 0, 0, time.Local)
	endDate := time.Date(year, 12, 31, 23, 59, 59, 0, time.Local)
	database.DB.Where("date BETWEEN ? AND ?", startDate, endDate).Order("date").Find(&incomes)

	for i, inc := range incomes {
		row := i + 2
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), inc.Date.Format("02/01/2006"))
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), inc.AmountUSD)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), inc.ExchangeRate)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), inc.AmountBRL)
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), inc.TaxAmount)
		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), inc.NetAmount)
		f.SetCellValue(sheet, fmt.Sprintf("G%d", row), inc.Description)
	}

	for i := 1; i <= 7; i++ {
		col, _ := excelize.ColumnNumberToName(i)
		f.SetColWidth(sheet, col, col, 15)
	}
}

func (h *ExportHandler) createExpensesSheet(f *excelize.File) {
	sheet := "Despesas"
	f.NewSheet(sheet)

	headers := []string{"Nome", "Valor", "Tipo", "Dia Venc.", "Categoria", "Ativa"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
	}

	style, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#ED7D31"}, Pattern: 1},
	})
	f.SetCellStyle(sheet, "A1", "F1", style)

	var expenses []models.Expense
	database.DB.Order("type, name").Find(&expenses)

	for i, e := range expenses {
		row := i + 2
		tipo := "Fixa"
		if e.Type == models.ExpenseTypeVariable {
			tipo = "Variável"
		}
		ativa := "Sim"
		if !e.Active {
			ativa = "Não"
		}
		f.SetCellValue(sheet, fmt.Sprintf("A%d", row), e.Name)
		f.SetCellValue(sheet, fmt.Sprintf("B%d", row), e.Amount)
		f.SetCellValue(sheet, fmt.Sprintf("C%d", row), tipo)
		f.SetCellValue(sheet, fmt.Sprintf("D%d", row), e.DueDay)
		f.SetCellValue(sheet, fmt.Sprintf("E%d", row), e.Category)
		f.SetCellValue(sheet, fmt.Sprintf("F%d", row), ativa)
	}

	for i := 1; i <= 6; i++ {
		col, _ := excelize.ColumnNumberToName(i)
		f.SetColWidth(sheet, col, col, 15)
	}
}

func (h *ExportHandler) createCardsSheet(f *excelize.File) {
	sheet := "Parcelamentos"
	f.NewSheet(sheet)

	headers := []string{"Cartão", "Descrição", "Valor Total", "Parcela", "Total Parcelas", "Parcela Atual", "Início", "Categoria"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheet, cell, h)
	}

	style, _ := f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true},
		Fill: excelize.Fill{Type: "pattern", Color: []string{"#5B9BD5"}, Pattern: 1},
	})
	f.SetCellStyle(sheet, "A1", "H1", style)

	var installments []models.Installment
	database.DB.Preload("CreditCard").Find(&installments)

	now := time.Now()
	row := 2
	for _, inst := range installments {
		monthsPassed := monthsBetween(inst.StartDate, now)
		if monthsPassed < inst.TotalInstallments {
			f.SetCellValue(sheet, fmt.Sprintf("A%d", row), inst.CreditCard.Name)
			f.SetCellValue(sheet, fmt.Sprintf("B%d", row), inst.Description)
			f.SetCellValue(sheet, fmt.Sprintf("C%d", row), inst.TotalAmount)
			f.SetCellValue(sheet, fmt.Sprintf("D%d", row), inst.InstallmentAmount)
			f.SetCellValue(sheet, fmt.Sprintf("E%d", row), inst.TotalInstallments)
			f.SetCellValue(sheet, fmt.Sprintf("F%d", row), monthsPassed+1)
			f.SetCellValue(sheet, fmt.Sprintf("G%d", row), inst.StartDate.Format("02/01/2006"))
			f.SetCellValue(sheet, fmt.Sprintf("H%d", row), inst.Category)
			row++
		}
	}

	for i := 1; i <= 8; i++ {
		col, _ := excelize.ColumnNumberToName(i)
		f.SetColWidth(sheet, col, col, 15)
	}
}

func (h *ExportHandler) exportYearCSV(c echo.Context, year int) error {
	c.Response().Header().Set("Content-Type", "text/csv")
	c.Response().Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=financeiro_%d.csv", year))

	writer := csv.NewWriter(c.Response().Writer)
	defer writer.Flush()

	// Escreve seção de resumo mensal
	if err := h.writeSummaryCSV(writer, year); err != nil {
		return err
	}

	// Linha em branco entre seções
	writer.Write([]string{})

	// Escreve seção de recebimentos
	if err := h.writeIncomesCSV(writer, year); err != nil {
		return err
	}

	// Linha em branco entre seções
	writer.Write([]string{})

	// Escreve seção de despesas
	if err := h.writeExpensesCSV(writer); err != nil {
		return err
	}

	// Linha em branco entre seções
	writer.Write([]string{})

	// Escreve seção de parcelamentos
	if err := h.writeInstallmentsCSV(writer); err != nil {
		return err
	}

	return nil
}

func (h *ExportHandler) writeSummaryCSV(writer *csv.Writer, year int) error {
	// Cabeçalho da seção
	writer.Write([]string{"RESUMO MENSAL"})

	// Headers das colunas
	headers := []string{"Mês", "Receita Bruta", "Imposto", "Receita Líquida", "Despesas Fixas", "Despesas Variáveis", "Cartões", "Contas", "Total Despesas", "Saldo"}
	if err := writer.Write(headers); err != nil {
		return err
	}

	// Dados
	summaries := services.GetYearlySummaries(database.DB, year)

	for i, s := range summaries {
		row := []string{
			i18n.MonthNamesSlice[i],
			fmt.Sprintf("%.2f", s.TotalIncomeGross),
			fmt.Sprintf("%.2f", s.TotalTax),
			fmt.Sprintf("%.2f", s.TotalIncomeNet),
			fmt.Sprintf("%.2f", s.TotalFixed),
			fmt.Sprintf("%.2f", s.TotalVariable),
			fmt.Sprintf("%.2f", s.TotalCards),
			fmt.Sprintf("%.2f", s.TotalBills),
			fmt.Sprintf("%.2f", s.TotalExpenses),
			fmt.Sprintf("%.2f", s.Balance),
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}

func (h *ExportHandler) writeIncomesCSV(writer *csv.Writer, year int) error {
	// Cabeçalho da seção
	writer.Write([]string{"RECEBIMENTOS"})

	// Headers das colunas
	headers := []string{"Data", "Valor USD", "Taxa Câmbio", "Valor BRL", "Imposto", "Líquido", "Descrição"}
	if err := writer.Write(headers); err != nil {
		return err
	}

	// Dados
	var incomes []models.Income
	startDate := time.Date(year, 1, 1, 0, 0, 0, 0, time.Local)
	endDate := time.Date(year, 12, 31, 23, 59, 59, 0, time.Local)
	database.DB.Where("date BETWEEN ? AND ?", startDate, endDate).Order("date").Find(&incomes)

	for _, inc := range incomes {
		row := []string{
			inc.Date.Format("02/01/2006"),
			fmt.Sprintf("%.2f", inc.AmountUSD),
			fmt.Sprintf("%.4f", inc.ExchangeRate),
			fmt.Sprintf("%.2f", inc.AmountBRL),
			fmt.Sprintf("%.2f", inc.TaxAmount),
			fmt.Sprintf("%.2f", inc.NetAmount),
			inc.Description,
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}

func (h *ExportHandler) writeExpensesCSV(writer *csv.Writer) error {
	// Cabeçalho da seção
	writer.Write([]string{"DESPESAS"})

	// Headers das colunas
	headers := []string{"Nome", "Valor", "Tipo", "Dia Venc.", "Categoria", "Ativa"}
	if err := writer.Write(headers); err != nil {
		return err
	}

	// Dados
	var expenses []models.Expense
	database.DB.Order("type, name").Find(&expenses)

	for _, e := range expenses {
		tipo := "Fixa"
		if e.Type == models.ExpenseTypeVariable {
			tipo = "Variável"
		}
		ativa := "Sim"
		if !e.Active {
			ativa = "Não"
		}
		row := []string{
			e.Name,
			fmt.Sprintf("%.2f", e.Amount),
			tipo,
			fmt.Sprintf("%d", e.DueDay),
			e.Category,
			ativa,
		}
		if err := writer.Write(row); err != nil {
			return err
		}
	}

	return nil
}

func (h *ExportHandler) writeInstallmentsCSV(writer *csv.Writer) error {
	// Cabeçalho da seção
	writer.Write([]string{"PARCELAMENTOS"})

	// Headers das colunas
	headers := []string{"Cartão", "Descrição", "Valor Total", "Parcela", "Total Parcelas", "Parcela Atual", "Início", "Categoria"}
	if err := writer.Write(headers); err != nil {
		return err
	}

	// Dados
	var installments []models.Installment
	database.DB.Preload("CreditCard").Find(&installments)

	now := time.Now()
	for _, inst := range installments {
		monthsPassed := monthsBetween(inst.StartDate, now)
		if monthsPassed < inst.TotalInstallments {
			row := []string{
				inst.CreditCard.Name,
				inst.Description,
				fmt.Sprintf("%.2f", inst.TotalAmount),
				fmt.Sprintf("%.2f", inst.InstallmentAmount),
				fmt.Sprintf("%d", inst.TotalInstallments),
				fmt.Sprintf("%d", monthsPassed+1),
				inst.StartDate.Format("02/01/2006"),
				inst.Category,
			}
			if err := writer.Write(row); err != nil {
				return err
			}
		}
	}

	return nil
}
