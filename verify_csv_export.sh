#!/bin/bash
# Automated CSV Export Verification Script
# This script verifies the CSV export functionality programmatically

set -e

echo "=== CSV Export Verification ==="
echo ""

# Start the server in the background
echo "Starting server..."
go build -o ./bin/server ./cmd/server/main.go
./bin/server &
SERVER_PID=$!

# Wait for server to start
echo "Waiting for server to start..."
sleep 3

# Function to cleanup
cleanup() {
    echo "Cleaning up..."
    kill $SERVER_PID 2>/dev/null || true
    rm -f test_export.csv
}
trap cleanup EXIT

# Test CSV export
echo "Testing CSV export endpoint..."
HTTP_CODE=$(curl -s -o test_export.csv -w "%{http_code}" "http://localhost:8080/export?year=2024&format=csv")

if [ "$HTTP_CODE" != "200" ]; then
    echo "❌ FAILED: HTTP status code $HTTP_CODE (expected 200)"
    exit 1
fi
echo "✅ HTTP 200 OK"

# Check if file was created
if [ ! -f test_export.csv ]; then
    echo "❌ FAILED: CSV file was not created"
    exit 1
fi
echo "✅ CSV file created"

# Verify all 4 sections are present
echo ""
echo "Verifying CSV sections..."

if grep -q "RESUMO MENSAL" test_export.csv; then
    echo "✅ Section 1: RESUMO MENSAL (Summary) - FOUND"
else
    echo "❌ Section 1: RESUMO MENSAL (Summary) - MISSING"
    exit 1
fi

if grep -q "RECEBIMENTOS" test_export.csv; then
    echo "✅ Section 2: RECEBIMENTOS (Incomes) - FOUND"
else
    echo "❌ Section 2: RECEBIMENTOS (Incomes) - MISSING"
    exit 1
fi

if grep -q "DESPESAS" test_export.csv; then
    echo "✅ Section 3: DESPESAS (Expenses) - FOUND"
else
    echo "❌ Section 3: DESPESAS (Expenses) - MISSING"
    exit 1
fi

if grep -q "PARCELAMENTOS" test_export.csv; then
    echo "✅ Section 4: PARCELAMENTOS (Installments) - FOUND"
else
    echo "❌ Section 4: PARCELAMENTOS (Installments) - MISSING"
    exit 1
fi

# Verify CSV structure (headers after section names)
echo ""
echo "Verifying CSV headers..."

# Check for Summary headers
if grep -q "Mês.*Receita Bruta.*Imposto.*Receita Líquida" test_export.csv; then
    echo "✅ Summary section has proper headers"
else
    echo "❌ Summary section headers are incorrect"
    exit 1
fi

# Check for Incomes headers
if grep -q "Data.*Valor USD.*Taxa Câmbio" test_export.csv; then
    echo "✅ Incomes section has proper headers"
else
    echo "❌ Incomes section headers are incorrect"
    exit 1
fi

# Check for Expenses headers
if grep -q "Nome.*Valor.*Tipo.*Dia Venc" test_export.csv; then
    echo "✅ Expenses section has proper headers"
else
    echo "❌ Expenses section headers are incorrect"
    exit 1
fi

# Check for Installments headers
if grep -q "Cartão.*Descrição.*Valor Total" test_export.csv; then
    echo "✅ Installments section has proper headers"
else
    echo "❌ Installments section headers are incorrect"
    exit 1
fi

# Verify CSV is valid (can be parsed)
echo ""
echo "Verifying CSV format..."
if file test_export.csv | grep -q "text"; then
    echo "✅ File is valid text/CSV format"
else
    echo "❌ File format is invalid"
    exit 1
fi

# Show sample of CSV content
echo ""
echo "CSV Content Preview (first 20 lines):"
echo "----------------------------------------"
head -20 test_export.csv
echo "----------------------------------------"

echo ""
echo "=== ALL VERIFICATION CHECKS PASSED ==="
echo ""
echo "Summary:"
echo "  ✅ HTTP 200 status code"
echo "  ✅ CSV file created successfully"
echo "  ✅ All 4 sections present (Summary, Incomes, Expenses, Installments)"
echo "  ✅ All section headers correct"
echo "  ✅ Valid CSV format"
echo ""
echo "The CSV export feature is working correctly!"
