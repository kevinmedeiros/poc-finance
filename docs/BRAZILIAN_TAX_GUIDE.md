# Guia de Tributação Brasileira - Simples Nacional Anexo III

**Sistema**: poc-finance - Sistema de Gestão Financeira Pessoal
**Status**: ✅ Documentação Completa
**Versão**: 1.0
**Criado**: 2026-01-19

---

## 📋 Visão Geral

Este guia explica como funciona a **tributação pelo Simples Nacional - Anexo III** no Brasil, regime utilizado para empresas de serviços de **consultoria, tecnologia e desenvolvimento de software**.

O sistema poc-finance implementa o cálculo automático de impostos baseado nas regras oficiais da Receita Federal do Brasil, incluindo:

- ✅ Cálculo de alíquota efetiva baseado no faturamento dos últimos 12 meses
- ✅ Aplicação das 6 faixas de tributação do Anexo III
- ✅ Cálculo de INSS sobre pró-labore
- ✅ Cálculo do valor líquido após impostos

---

## 💰 Faixas de Tributação - Anexo III

O Simples Nacional Anexo III possui **6 faixas de tributação** baseadas na Receita Bruta Total dos últimos 12 meses (RBT12):

### Tabela Completa de Faixas

| Faixa | Receita Bruta Total (12 meses) | Alíquota Nominal | Valor a Deduzir |
|-------|--------------------------------|------------------|-----------------|
| **1** | Até R$ 180.000,00 | 6,00% | R$ 0,00 |
| **2** | De R$ 180.000,01 a R$ 360.000,00 | 11,20% | R$ 9.360,00 |
| **3** | De R$ 360.000,01 a R$ 720.000,00 | 13,50% | R$ 17.640,00 |
| **4** | De R$ 720.000,01 a R$ 1.800.000,00 | 16,00% | R$ 35.640,00 |
| **5** | De R$ 1.800.000,01 a R$ 3.600.000,00 | 21,00% | R$ 125.640,00 |
| **6** | De R$ 3.600.000,01 a R$ 4.800.000,00 | 33,00% | R$ 648.000,00 |

### Detalhamento por Faixa

#### 📊 Faixa 1: Até R$ 180.000,00/ano

**Características:**
- **Alíquota nominal**: 6,00%
- **Dedução**: R$ 0,00
- **Alíquota efetiva**: 6,00% (fixa)
- **Receita mensal média**: Até R$ 15.000,00

**Ideal para**: Microempreendedores iniciantes, freelancers, pequenos prestadores de serviços.

**Exemplo:**
```
Faturamento 12 meses: R$ 120.000,00
Recebimento atual: R$ 10.000,00
Alíquota efetiva: 6,00%
Imposto a pagar: R$ 600,00
Valor líquido: R$ 9.400,00
```

---

#### 📊 Faixa 2: R$ 180.000,01 a R$ 360.000,00/ano

**Características:**
- **Alíquota nominal**: 11,20%
- **Dedução**: R$ 9.360,00
- **Alíquota efetiva**: Varia de 6,01% a 8,60%
- **Receita mensal média**: R$ 15.000,01 a R$ 30.000,00

**Ideal para**: Pequenas empresas de consultoria e desenvolvimento em crescimento.

**Exemplo:**
```
Faturamento 12 meses: R$ 270.000,00
Recebimento atual: R$ 10.000,00

Cálculo da alíquota efetiva:
= (RBT12 × Alíquota - Dedução) / RBT12
= (270.000 × 0,112 - 9.360) / 270.000
= (30.240 - 9.360) / 270.000
= 20.880 / 270.000
= 0,07733 = 7,73%

Imposto a pagar: R$ 10.000 × 7,73% = R$ 773,30
Valor líquido: R$ 9.226,70
```

---

#### 📊 Faixa 3: R$ 360.000,01 a R$ 720.000,00/ano

**Características:**
- **Alíquota nominal**: 13,50%
- **Dedução**: R$ 17.640,00
- **Alíquota efetiva**: Varia de 8,61% a 11,05%
- **Receita mensal média**: R$ 30.000,01 a R$ 60.000,00

**Ideal para**: Empresas de médio porte com faturamento consolidado.

**Exemplo:**
```
Faturamento 12 meses: R$ 500.000,00
Recebimento atual: R$ 25.000,00

Cálculo da alíquota efetiva:
= (500.000 × 0,135 - 17.640) / 500.000
= (67.500 - 17.640) / 500.000
= 49.860 / 500.000
= 0,09972 = 9,97%

Imposto a pagar: R$ 25.000 × 9,97% = R$ 2.492,50
Valor líquido: R$ 22.507,50
```

---

#### 📊 Faixa 4: R$ 720.000,01 a R$ 1.800.000,00/ano

**Características:**
- **Alíquota nominal**: 16,00%
- **Dedução**: R$ 35.640,00
- **Alíquota efetiva**: Varia de 11,05% a 14,02%
- **Receita mensal média**: R$ 60.000,01 a R$ 150.000,00

**Ideal para**: Empresas estabelecidas com múltiplos projetos/clientes.

**Exemplo:**
```
Faturamento 12 meses: R$ 1.000.000,00
Recebimento atual: R$ 50.000,00

Cálculo da alíquota efetiva:
= (1.000.000 × 0,16 - 35.640) / 1.000.000
= (160.000 - 35.640) / 1.000.000
= 124.360 / 1.000.000
= 0,12436 = 12,44%

Imposto a pagar: R$ 50.000 × 12,44% = R$ 6.220,00
Valor líquido: R$ 43.780,00
```

---

#### 📊 Faixa 5: R$ 1.800.000,01 a R$ 3.600.000,00/ano

**Características:**
- **Alíquota nominal**: 21,00%
- **Dedução**: R$ 125.640,00
- **Alíquota efetiva**: Varia de 14,02% a 17,51%
- **Receita mensal média**: R$ 150.000,01 a R$ 300.000,00

**Ideal para**: Empresas de tecnologia de grande porte.

**Exemplo:**
```
Faturamento 12 meses: R$ 2.500.000,00
Recebimento atual: R$ 100.000,00

Cálculo da alíquota efetiva:
= (2.500.000 × 0,21 - 125.640) / 2.500.000
= (525.000 - 125.640) / 2.500.000
= 399.360 / 2.500.000
= 0,15974 = 15,97%

Imposto a pagar: R$ 100.000 × 15,97% = R$ 15.970,00
Valor líquido: R$ 84.030,00
```

---

#### 📊 Faixa 6: R$ 3.600.000,01 a R$ 4.800.000,00/ano

**Características:**
- **Alíquota nominal**: 33,00%
- **Dedução**: R$ 648.000,00
- **Alíquota efetiva**: Varia de 17,51% a 19,50%
- **Receita mensal média**: R$ 300.000,01 a R$ 400.000,00
- **Limite máximo do Simples Nacional**

**Ideal para**: Grandes empresas próximas ao limite do Simples.

**Exemplo:**
```
Faturamento 12 meses: R$ 4.000.000,00
Recebimento atual: R$ 150.000,00

Cálculo da alíquota efetiva:
= (4.000.000 × 0,33 - 648.000) / 4.000.000
= (1.320.000 - 648.000) / 4.000.000
= 672.000 / 4.000.000
= 0,168 = 16,80%

Imposto a pagar: R$ 150.000 × 16,80% = R$ 25.200,00
Valor líquido: R$ 124.800,00
```

⚠️ **IMPORTANTE**: Empresas que ultrapassarem R$ 4.800.000,00 de faturamento anual devem migrar para o Lucro Presumido ou Lucro Real.

---

## 📐 Fórmula da Alíquota Efetiva

A alíquota efetiva é calculada usando a seguinte fórmula:

```
Alíquota Efetiva = (RBT12 × Alíquota Nominal - Dedução) / RBT12

Onde:
- RBT12 = Receita Bruta Total dos últimos 12 meses
- Alíquota Nominal = Taxa da faixa atual (da tabela acima)
- Dedução = Valor a deduzir da faixa atual (da tabela acima)
```

### Exemplo Prático Completo

**Cenário**: Empresa de desenvolvimento de software

**Dados:**
- Faturamento nos últimos 12 meses: R$ 270.000,00
- Recebimento atual de cliente: R$ 15.000,00

**Passo 1**: Identificar a faixa
```
R$ 270.000,00 está entre R$ 180.000,01 e R$ 360.000,00
→ Faixa 2
→ Alíquota nominal: 11,20%
→ Dedução: R$ 9.360,00
```

**Passo 2**: Calcular a alíquota efetiva
```
Alíquota Efetiva = (270.000 × 0,112 - 9.360) / 270.000
                 = (30.240 - 9.360) / 270.000
                 = 20.880 / 270.000
                 = 0,07733
                 = 7,73%
```

**Passo 3**: Calcular o imposto
```
Imposto = R$ 15.000,00 × 7,73%
        = R$ 1.159,50
```

**Passo 4**: Calcular o valor líquido
```
Valor Líquido = R$ 15.000,00 - R$ 1.159,50
              = R$ 13.840,50
```

**Resumo:**
- 💵 Valor bruto recebido: R$ 15.000,00
- 📊 Alíquota efetiva: 7,73%
- 💰 Imposto Simples Nacional: R$ 1.159,50
- ✅ Valor líquido: R$ 13.840,50

---

## 🏥 Cálculo do INSS sobre Pró-Labore

Além do Simples Nacional, empresários e sócios que retiram pró-labore devem pagar INSS.

### Regras do INSS

**Alíquota**: 11% sobre o pró-labore
**Teto**: R$ 7.786,02 (valor de 2026)
**Base de cálculo**: O menor valor entre o pró-labore e o teto

### Fórmula do INSS

```
Base de Cálculo = MIN(Pró-Labore, Teto do INSS)
INSS = Base de Cálculo × 11%
```

### Exemplos de Cálculo de INSS

#### Exemplo 1: Pró-labore abaixo do teto

```
Pró-labore: R$ 5.000,00
Teto: R$ 7.786,02

Base de cálculo: R$ 5.000,00 (menor valor)
INSS = R$ 5.000,00 × 11%
INSS = R$ 550,00
```

#### Exemplo 2: Pró-labore igual ao teto

```
Pró-labore: R$ 7.786,02
Teto: R$ 7.786,02

Base de cálculo: R$ 7.786,02
INSS = R$ 7.786,02 × 11%
INSS = R$ 856,46
```

#### Exemplo 3: Pró-labore acima do teto

```
Pró-labore: R$ 15.000,00
Teto: R$ 7.786,02

Base de cálculo: R$ 7.786,02 (limitado ao teto)
INSS = R$ 7.786,02 × 11%
INSS = R$ 856,46 (valor máximo)
```

⚠️ **IMPORTANTE**: O INSS é limitado ao teto, mesmo que o pró-labore seja maior.

---

## 💡 Cenários Práticos Completos

### Cenário 1: Freelancer Iniciante

**Perfil:**
- Desenvolvedor autônomo iniciando atividades
- Primeiro recebimento no ano
- Sem histórico de faturamento

**Dados:**
```
Faturamento 12 meses: R$ 0,00 (empresa nova)
Recebimento atual: R$ 8.000,00
Pró-labore: R$ 3.000,00
```

**Cálculos:**

1. **Simples Nacional** (usa Faixa 1):
   ```
   Alíquota efetiva: 6,00%
   Imposto: R$ 8.000 × 6% = R$ 480,00
   ```

2. **INSS**:
   ```
   Base: R$ 3.000,00
   INSS: R$ 3.000 × 11% = R$ 330,00
   ```

3. **Total:**
   ```
   Valor bruto: R$ 8.000,00
   Simples Nacional: R$ 480,00
   INSS: R$ 330,00
   Total de impostos: R$ 810,00
   Valor líquido: R$ 7.190,00
   Carga tributária efetiva: 10,13%
   ```

---

### Cenário 2: Empresa de Consultoria Consolidada

**Perfil:**
- Empresa com 3 anos de operação
- Faturamento estável
- 2 sócios com pró-labore

**Dados:**
```
Faturamento 12 meses: R$ 850.000,00
Recebimento atual: R$ 45.000,00
Pró-labore total: R$ 12.000,00 (R$ 6.000 por sócio)
```

**Cálculos:**

1. **Simples Nacional** (Faixa 4):
   ```
   Alíquota efetiva = (850.000 × 0,16 - 35.640) / 850.000
                    = (136.000 - 35.640) / 850.000
                    = 0,11807 = 11,81%

   Imposto: R$ 45.000 × 11,81% = R$ 5.314,50
   ```

2. **INSS** (2 sócios):
   ```
   Sócio 1: R$ 6.000 × 11% = R$ 660,00
   Sócio 2: R$ 6.000 × 11% = R$ 660,00
   Total INSS: R$ 1.320,00
   ```

3. **Total:**
   ```
   Valor bruto: R$ 45.000,00
   Simples Nacional: R$ 5.314,50
   INSS: R$ 1.320,00
   Total de impostos: R$ 6.634,50
   Valor líquido: R$ 38.365,50
   Carga tributária efetiva: 14,74%
   ```

---

### Cenário 3: Empresa de Grande Porte

**Perfil:**
- Empresa de tecnologia estabelecida
- Múltiplos projetos e clientes
- Equipe de 20+ colaboradores

**Dados:**
```
Faturamento 12 meses: R$ 3.200.000,00
Recebimento atual: R$ 180.000,00
Pró-labore total: R$ 25.000,00
```

**Cálculos:**

1. **Simples Nacional** (Faixa 5):
   ```
   Alíquota efetiva = (3.200.000 × 0,21 - 125.640) / 3.200.000
                    = (672.000 - 125.640) / 3.200.000
                    = 0,17074 = 17,07%

   Imposto: R$ 180.000 × 17,07% = R$ 30.726,00
   ```

2. **INSS** (limitado ao teto):
   ```
   Pró-labore: R$ 25.000,00
   Base (limitada): R$ 7.786,02
   INSS: R$ 7.786,02 × 11% = R$ 856,46
   ```

3. **Total:**
   ```
   Valor bruto: R$ 180.000,00
   Simples Nacional: R$ 30.726,00
   INSS: R$ 856,46
   Total de impostos: R$ 31.582,46
   Carga tributária efetiva: 17,55%
   ```

---

## 📊 Comparativo de Carga Tributária por Faixa

Tabela comparativa mostrando a carga tributária efetiva em cada faixa:

| Faixa | Faturamento Anual | Alíquota Mínima | Alíquota Máxima | Média |
|-------|------------------|-----------------|-----------------|-------|
| 1 | Até R$ 180k | 6,00% | 6,00% | 6,00% |
| 2 | R$ 180k - R$ 360k | 6,01% | 8,60% | 7,31% |
| 3 | R$ 360k - R$ 720k | 8,61% | 11,05% | 9,83% |
| 4 | R$ 720k - R$ 1,8M | 11,05% | 14,02% | 12,54% |
| 5 | R$ 1,8M - R$ 3,6M | 14,02% | 17,51% | 15,77% |
| 6 | R$ 3,6M - R$ 4,8M | 17,51% | 19,50% | 18,51% |

**Observações:**
- ✅ Carga tributária progressiva conforme o faturamento aumenta
- ✅ Alíquota efetiva sempre menor que a alíquota nominal
- ✅ Sistema de dedução torna a transição entre faixas mais suave

---

## ⚠️ Pontos de Atenção

### 1. Apuração dos Últimos 12 Meses

A receita bruta deve ser calculada considerando os **últimos 12 meses**, não o ano-calendário:

```
Exemplo: Cálculo em junho/2026
- Período considerado: julho/2025 a junho/2026
- Soma de todas as notas fiscais emitidas neste período
```

### 2. Primeira Nota Fiscal

Para empresas novas sem histórico:
- Usa-se a alíquota da primeira faixa (6%)
- Nos meses seguintes, acumula-se o faturamento

### 3. Mudança de Faixa

Ao mudar de faixa de tributação:
- A nova alíquota é aplicada imediatamente
- Não há retroatividade
- Planejamento é essencial para otimizar a carga tributária

### 4. Limite do Simples Nacional

⚠️ **ATENÇÃO**: Empresas que ultrapassarem R$ 4.800.000,00:
- Devem migrar para Lucro Presumido ou Lucro Real
- Terão carga tributária maior (aproximadamente 13,33% a 16,33%)
- Planejamento tributário é crucial antes de ultrapassar o limite

### 5. Fator R (não implementado)

O sistema atual **não** implementa o Fator R, que é:
```
Fator R = (Folha de Pagamento dos últimos 12 meses) / (Receita Bruta dos últimos 12 meses)
```

Se Fator R ≥ 28%, a empresa pode optar pelo Anexo III (serviços).
Se Fator R < 28%, deve usar Anexo V (alíquotas mais altas).

---

## 🔗 Links Oficiais e Recursos

### Receita Federal do Brasil

1. **Portal do Simples Nacional**
   - URL: https://www8.receita.fazenda.gov.br/simplesnacional/
   - Conteúdo: Legislação, tabelas, calculadoras oficiais

2. **Lei Complementar 123/2006**
   - URL: http://www.planalto.gov.br/ccivil_03/leis/lcp/lcp123.htm
   - Conteúdo: Lei do Simples Nacional completa

3. **Resolução CGSN 140/2018**
   - URL: http://normas.receita.fazenda.gov.br/sijut2consulta/
   - Conteúdo: Regulamentação do Simples Nacional

4. **Tabelas do Anexo III**
   - URL: https://www8.receita.fazenda.gov.br/simplesnacional/documentos/pagina.aspx?id=3
   - Conteúdo: Tabelas oficiais atualizadas

### Previdência Social (INSS)

5. **Tabela de Contribuição INSS**
   - URL: https://www.gov.br/inss/pt-br/direitos-e-deveres/inscricao-e-contribuicao/tabela-de-contribuicao-mensal
   - Conteúdo: Tetos e alíquotas atualizadas

6. **Pró-Labore e Contribuição Patronal**
   - URL: https://www.gov.br/empresas-e-negocios/pt-br
   - Conteúdo: Obrigações previdenciárias empresariais

### Ferramentas Úteis

7. **Calculadora do Simples Nacional**
   - URL: https://www8.receita.fazenda.gov.br/simplesnacional/aplicacoes.aspx?id=21
   - Ferramenta oficial para simular impostos

8. **Portal do Empreendedor**
   - URL: https://www.gov.br/empresas-e-negocios/pt-br/empreendedor
   - Conteúdo: Guias e orientações para empresários

---

## 💻 Implementação no poc-finance

O sistema poc-finance implementa automaticamente todos estes cálculos através do arquivo `internal/services/tax.go`.

### Funcionalidades Implementadas

✅ **Cálculo automático da alíquota efetiva**
```go
// Baseado no faturamento dos últimos 12 meses
result := services.CalculateTax(revenue12M, grossAmount)
```

✅ **Identificação da faixa de tributação**
```go
// Retorna faixa atual, alíquota e próxima faixa
bracket, rate, nextBracketAt := services.GetBracketInfo(revenue12M)
```

✅ **Cálculo de INSS sobre pró-labore**
```go
// Com teto e alíquota configuráveis
inss := services.CalculateINSS(INSSConfig{
    ProLabore: 5000,
    Ceiling:   7786.02,
    Rate:      0.11,
})
```

### Estrutura de Dados

```go
type TaxCalculation struct {
    GrossAmount    float64  // Valor bruto
    Revenue12M     float64  // Faturamento 12 meses
    EffectiveRate  float64  // Alíquota efetiva
    TaxAmount      float64  // Imposto calculado
    INSSAmount     float64  // INSS
    TotalTax       float64  // Total de impostos
    NetAmount      float64  // Valor líquido
    BracketApplied int      // Faixa aplicada (1-6)
}
```

### Testes Automatizados

O sistema possui **39 testes automatizados** cobrindo:
- ✅ Todas as 6 faixas de tributação
- ✅ Fórmula da alíquota efetiva
- ✅ Cálculo de INSS com diferentes cenários
- ✅ Limites e casos extremos
- ✅ Continuidade entre faixas

Para executar os testes:
```bash
go test ./internal/services/tax_test.go -v
```

---

## 📚 Glossário

**Alíquota Nominal**: Taxa percentual indicada na tabela do Simples Nacional para cada faixa.

**Alíquota Efetiva**: Taxa real de tributação após aplicar a dedução prevista na lei.

**Anexo III**: Tabela de tributação do Simples Nacional para empresas prestadoras de serviços (consultoria, tecnologia, etc.).

**Dedução**: Valor fixo subtraído do cálculo para determinar a alíquota efetiva.

**Fator R**: Proporção entre folha de pagamento e receita bruta (não implementado no sistema atual).

**INSS**: Instituto Nacional do Seguro Social - contribuição previdenciária obrigatória.

**Pró-Labore**: Remuneração mensal dos sócios ou administradores da empresa.

**RBT12**: Receita Bruta Total dos últimos 12 meses - base para determinação da faixa de tributação.

**Simples Nacional**: Regime tributário simplificado para micro e pequenas empresas no Brasil.

**Teto do INSS**: Valor máximo sobre o qual incide a contribuição previdenciária.

---

## 📞 Suporte

Para dúvidas sobre a implementação técnica, consulte:
- Código fonte: `internal/services/tax.go`
- Testes: `internal/services/tax_test.go`
- Arquitetura: `ARCHITECTURE.md`

Para dúvidas sobre legislação tributária, consulte um contador profissional ou acesse os links oficiais da Receita Federal listados acima.

---

**Última atualização**: 2026-01-19
**Versão do documento**: 1.0
**Legislação base**: Lei Complementar 123/2006 e Resolução CGSN 140/2018
