# PRD: Autenticação e Compartilhamento de Dados para Casal

## 1. Visão Geral

### 1.1 Objetivo
Adicionar sistema de autenticação e funcionalidade de compartilhamento de dados financeiros entre casais ao aplicativo de controle financeiro existente, permitindo visão holística das finanças do casal com total transparência.

### 1.2 Problema a Resolver
Casais que desejam gerenciar finanças juntos atualmente não têm uma forma integrada de visualizar dados consolidados. Cada pessoa gerencia suas finanças isoladamente, dificultando o planejamento financeiro conjunto e a tomada de decisões como casal.

### 1.3 Público-Alvo
Casais que desejam ter controle financeiro conjunto com transparência total sobre rendas, despesas, cartões e parcelas de ambos os parceiros.

---

## 2. Requisitos Funcionais

### 2.1 Sistema de Autenticação

#### 2.1.1 Cadastro de Usuário
- **RF-001**: Usuário deve poder se cadastrar com email e senha
- **RF-002**: Email deve ser único no sistema
- **RF-003**: Senha deve ter no mínimo 8 caracteres, com pelo menos 1 letra e 1 número
- **RF-004**: Confirmação de email após cadastro (link de verificação)

#### 2.1.2 Login
- **RF-005**: Login com email + senha
- **RF-006**: Implementar JWT com refresh token para gerenciamento de sessão
- **RF-007**: Access token com expiração de 15 minutos
- **RF-008**: Refresh token com expiração de 7 dias
- **RF-009**: Possibilidade de "Lembrar de mim" (refresh token de 30 dias)

#### 2.1.3 Recuperação de Senha
- **RF-010**: Usuário solicita reset informando email
- **RF-011**: Sistema envia link de reset por email
- **RF-012**: Link válido por 1 hora
- **RF-013**: Link de uso único (invalidado após uso)
- **RF-014**: Página de definição de nova senha

#### 2.1.4 Gerenciamento de Sessão
- **RF-015**: Logout manual disponível
- **RF-016**: Renovação automática do access token via refresh token
- **RF-017**: Invalidação de todos os refresh tokens ao trocar senha

---

### 2.2 Sistema de Vinculação do Casal

#### 2.2.1 Convite
- **RF-018**: Usuário pode gerar código de convite de 6 dígitos alfanumérico
- **RF-019**: Código válido por 24 horas
- **RF-020**: Apenas um código ativo por usuário por vez
- **RF-021**: Gerar novo código invalida o anterior

#### 2.2.2 Aceite do Convite
- **RF-022**: Parceiro insere código de 6 dígitos para vincular
- **RF-023**: Sistema valida código e exibe nome de quem convidou para confirmação
- **RF-024**: Parceiro confirma vinculação
- **RF-025**: Ambos são notificados do sucesso da vinculação

#### 2.2.3 Papéis no Casal
- **RF-026**: Quem convidou é o "Admin" do casal
- **RF-027**: Admin tem permissão exclusiva para desvincular o casal
- **RF-028**: Transferência de admin não disponível na v1

#### 2.2.4 Desvinculação
- **RF-029**: Apenas Admin pode iniciar desvinculação
- **RF-030**: Confirmação obrigatória (digitar "DESVINCULAR")
- **RF-031**: Após desvinculação, cada um mantém seus dados originais
- **RF-032**: Dados do parceiro deixam de ser visíveis
- **RF-033**: Categorias "do casal" são mantidas para ambos

---

### 2.3 Compartilhamento de Dados

#### 2.3.1 Dados Compartilhados (Automático e Total)
- **RF-034**: Todas as rendas são compartilhadas
- **RF-035**: Todas as despesas são compartilhadas
- **RF-036**: Todos os cartões de crédito são compartilhados
- **RF-037**: Todas as parcelas são compartilhadas
- **RF-038**: Compartilhamento é automático ao criar/editar/excluir transações

#### 2.3.2 Categorias
- **RF-039**: Cada usuário mantém suas categorias pessoais
- **RF-040**: Usuários podem criar categorias "do casal"
- **RF-041**: Categorias do casal são visíveis e utilizáveis por ambos
- **RF-042**: Ambos podem editar/excluir categorias do casal
- **RF-043**: Categorias pessoais não são visíveis para o parceiro

---

### 2.4 Dashboards

#### 2.4.1 Dashboard Pessoal
- **RF-044**: Exibe apenas transações do próprio usuário
- **RF-045**: Métricas pessoais (receita, despesa, saldo)
- **RF-046**: Gráficos de evolução pessoal
- **RF-047**: Lista de transações pessoais

#### 2.4.2 Dashboard do Casal
- **RF-048**: Exibe transações consolidadas de ambos
- **RF-049**: Métricas consolidadas (receita total, despesa total, saldo do casal)
- **RF-050**: Identificação visual de quem fez cada transação
- **RF-051**: Gráficos comparativos (contribuição de cada um)
- **RF-052**: Filtros por pessoa (Todos / Eu / Parceiro)
- **RF-053**: Resumo de participação percentual de cada um

---

### 2.5 Notificações

#### 2.5.1 Notificações Push
- **RF-054**: Notificar parceiro quando nova transação é adicionada
- **RF-055**: Notificar parceiro quando transação é editada
- **RF-056**: Notificar parceiro quando transação é excluída
- **RF-057**: Notificar ambos quando meta conjunta atinge marco (25%, 50%, 75%, 100%)
- **RF-058**: Notificar quando parceiro aceita convite de vinculação

#### 2.5.2 Configurações de Notificação
- **RF-059**: Usuário pode ativar/desativar notificações de transações
- **RF-060**: Usuário pode ativar/desativar notificações de metas
- **RF-061**: Configuração de "Não perturbe" por horário

---

### 2.6 Metas Financeiras

#### 2.6.1 Metas Individuais
- **RF-062**: Criar meta com nome, valor alvo e prazo
- **RF-063**: Associar transações/economias à meta
- **RF-064**: Visualizar progresso da meta
- **RF-065**: Metas individuais visíveis apenas para o dono

#### 2.6.2 Metas Conjuntas
- **RF-066**: Qualquer um do casal pode criar meta conjunta
- **RF-067**: Meta conjunta visível para ambos
- **RF-068**: Ambos podem contribuir para a meta
- **RF-069**: Visualização de contribuição de cada um
- **RF-070**: Ambos podem editar/excluir meta conjunta
- **RF-071**: Histórico de contribuições com identificação de quem contribuiu

---

## 3. Requisitos Não-Funcionais

### 3.1 Arquitetura e Persistência

#### 3.1.1 Offline-First com SQLite
- **RNF-001**: Dados armazenados localmente em SQLite
- **RNF-002**: App funciona 100% offline para operações locais
- **RNF-003**: Sincronização manual com servidor quando online
- **RNF-004**: Indicador visual de status de sincronização
- **RNF-005**: Fila de operações pendentes para sync

#### 3.1.2 Sincronização
- **RNF-006**: Botão "Sincronizar" visível na interface
- **RNF-007**: Sync automático ao abrir app (se online)
- **RNF-008**: Resolução de conflitos: última modificação vence
- **RNF-009**: Log de sincronização para auditoria
- **RNF-010**: Retry automático em caso de falha de rede

### 3.2 Segurança
- **RNF-011**: Senhas armazenadas com bcrypt (cost factor 12)
- **RNF-012**: Tokens JWT assinados com RS256
- **RNF-013**: HTTPS obrigatório para todas as comunicações
- **RNF-014**: Rate limiting em endpoints de auth (5 tentativas/minuto)
- **RNF-015**: Dados sensíveis criptografados em repouso no SQLite

### 3.3 Performance
- **RNF-016**: Tempo de login < 2 segundos
- **RNF-017**: Sync inicial < 10 segundos para até 1000 transações
- **RNF-018**: Interface responsiva durante sync (não bloqueante)

### 3.4 Usabilidade
- **RNF-019**: Feedback visual claro de status de conexão
- **RNF-020**: Mensagens de erro amigáveis e acionáveis
- **RNF-021**: Onboarding explicando funcionalidade de casal

---

## 4. Estrutura de Dados

### 4.1 Novas Entidades

```
User {
  id: UUID (PK)
  email: String (unique)
  password_hash: String
  name: String
  email_verified: Boolean
  created_at: DateTime
  updated_at: DateTime
}

Couple {
  id: UUID (PK)
  admin_user_id: UUID (FK -> User)
  partner_user_id: UUID (FK -> User)
  created_at: DateTime
  status: Enum (active, dissolved)
}

CoupleInvite {
  id: UUID (PK)
  code: String (6 chars, unique)
  creator_user_id: UUID (FK -> User)
  expires_at: DateTime
  used_at: DateTime (nullable)
  created_at: DateTime
}

RefreshToken {
  id: UUID (PK)
  user_id: UUID (FK -> User)
  token_hash: String
  expires_at: DateTime
  created_at: DateTime
  revoked_at: DateTime (nullable)
}

Category {
  id: UUID (PK)
  user_id: UUID (FK -> User, nullable)
  couple_id: UUID (FK -> Couple, nullable)
  name: String
  type: Enum (income, expense)
  icon: String
  color: String
}

Goal {
  id: UUID (PK)
  user_id: UUID (FK -> User, nullable - para metas individuais)
  couple_id: UUID (FK -> Couple, nullable - para metas conjuntas)
  name: String
  target_amount: Decimal
  current_amount: Decimal
  deadline: Date
  status: Enum (active, completed, cancelled)
  created_at: DateTime
}

GoalContribution {
  id: UUID (PK)
  goal_id: UUID (FK -> Goal)
  user_id: UUID (FK -> User)
  amount: Decimal
  date: Date
  created_at: DateTime
}

SyncQueue {
  id: UUID (PK)
  user_id: UUID (FK -> User)
  operation: Enum (create, update, delete)
  entity_type: String
  entity_id: UUID
  payload: JSON
  created_at: DateTime
  synced_at: DateTime (nullable)
}
```

### 4.2 Alterações em Entidades Existentes

```
Transaction (adicionar campos) {
  + user_id: UUID (FK -> User)
  + synced_at: DateTime (nullable)
  + sync_status: Enum (pending, synced, conflict)
}

Card (adicionar campos) {
  + user_id: UUID (FK -> User)
  + synced_at: DateTime (nullable)
}
```

---

## 5. Fluxos de Usuário

### 5.1 Fluxo de Cadastro e Vinculação

```
1. Usuário A se cadastra (email + senha)
2. Usuário A verifica email
3. Usuário A faz login
4. Usuário A vai em "Configurações > Vincular Parceiro"
5. Usuário A gera código de 6 dígitos
6. Usuário A compartilha código com parceiro (WhatsApp, verbal, etc)
7. Usuário B se cadastra (se ainda não tiver conta)
8. Usuário B vai em "Configurações > Vincular Parceiro"
9. Usuário B insere código de 6 dígitos
10. Sistema exibe "Vincular com [Nome do Usuário A]?"
11. Usuário B confirma
12. Ambos recebem notificação de sucesso
13. Dashboard do Casal é liberado para ambos
```

### 5.2 Fluxo de Sincronização

```
1. Usuário cria transação offline
2. Transação salva no SQLite local com sync_status = "pending"
3. Operação adicionada à SyncQueue
4. Quando online, usuário clica "Sincronizar" (ou automático ao abrir)
5. Sistema processa SyncQueue em ordem
6. Servidor valida e persiste
7. Servidor notifica parceiro via push
8. sync_status atualizado para "synced"
9. Parceiro recebe notificação
10. Parceiro sincroniza para ver nova transação
```

---

## 6. Interface do Usuário

### 6.1 Novas Telas

| Tela | Descrição |
|------|-----------|
| Login | Email + senha, link "Esqueci senha", link "Cadastrar" |
| Cadastro | Nome, email, senha, confirmação de senha |
| Verificação de Email | Instrução para verificar email |
| Recuperar Senha | Input de email |
| Redefinir Senha | Nova senha + confirmação |
| Vincular Parceiro | Gerar código / Inserir código |
| Confirmação de Vínculo | Exibe nome do parceiro, botão confirmar |
| Dashboard Pessoal | Métricas e transações pessoais |
| Dashboard Casal | Métricas consolidadas com filtros |
| Metas | Lista de metas individuais e conjuntas |
| Nova Meta | Formulário de criação (individual ou conjunta) |
| Detalhes da Meta | Progresso, contribuições, histórico |
| Configurações de Notificação | Toggles para cada tipo |
| Status de Sincronização | Itens pendentes, botão sync, log |

### 6.2 Alterações em Telas Existentes

| Tela | Alteração |
|------|-----------|
| Menu/Navegação | Adicionar tabs "Pessoal" e "Casal" |
| Nova Transação | Adicionar seletor de categoria (pessoal/casal) |
| Lista de Transações | Indicador de quem criou cada transação |
| Configurações | Seção "Casal" com opções de vínculo |

---

## 7. API Endpoints

### 7.1 Autenticação
```
POST /api/auth/register
POST /api/auth/login
POST /api/auth/logout
POST /api/auth/refresh
POST /api/auth/forgot-password
POST /api/auth/reset-password
GET  /api/auth/verify-email/:token
```

### 7.2 Casal
```
POST /api/couple/invite/generate
POST /api/couple/invite/accept
GET  /api/couple/status
DELETE /api/couple (apenas admin)
GET  /api/couple/partner
```

### 7.3 Sincronização
```
POST /api/sync/push (enviar alterações locais)
GET  /api/sync/pull (buscar alterações do servidor)
GET  /api/sync/status
```

### 7.4 Metas
```
GET    /api/goals
POST   /api/goals
GET    /api/goals/:id
PUT    /api/goals/:id
DELETE /api/goals/:id
POST   /api/goals/:id/contribute
```

### 7.5 Categorias
```
GET    /api/categories
POST   /api/categories
PUT    /api/categories/:id
DELETE /api/categories/:id
```

---

## 8. Considerações de Implementação

### 8.1 Stack Tecnológico
- **Frontend**: Next.js (existente)
- **Banco Local**: SQLite com better-sqlite3 ou sql.js
- **Backend API**: Next.js API Routes
- **Banco Servidor**: SQLite (para sync)
- **Push Notifications**: Web Push API / Service Workers
- **JWT**: jose library

### 8.2 Estratégia Offline-First
1. Todas as operações CRUD primeiro no SQLite local
2. Fila de sincronização para operações pendentes
3. Sync manual com botão + auto-sync ao conectar
4. Timestamps de última modificação para resolução de conflitos
5. Estados visuais: "Sincronizado", "Pendente", "Conflito"

### 8.3 Ordem de Implementação (Release Única)
Apesar de ser uma release única, a implementação seguirá esta ordem lógica:

1. **Fase 1 - Fundação**
   - Setup SQLite local
   - Sistema de autenticação completo
   - Migração de dados existentes para incluir user_id

2. **Fase 2 - Casal**
   - Sistema de convite e vinculação
   - Compartilhamento de dados
   - Dashboard do casal

3. **Fase 3 - Sincronização**
   - API de sync
   - Fila de operações
   - Resolução de conflitos

4. **Fase 4 - Recursos Adicionais**
   - Sistema de metas
   - Notificações push
   - Categorias do casal

5. **Fase 5 - Polish**
   - Testes end-to-end
   - Ajustes de UX
   - Documentação

---

## 9. Métricas de Sucesso

| Métrica | Alvo |
|---------|------|
| Taxa de cadastro completo (com verificação) | > 80% |
| Taxa de vinculação de casais (dos que tentam) | > 90% |
| Uso do dashboard do casal vs pessoal | > 60% no casal |
| Transações com sync bem-sucedido | > 99% |
| Tempo médio para sync | < 5 segundos |
| Metas conjuntas criadas por casal | ≥ 1 |

---

## 10. Riscos e Mitigações

| Risco | Impacto | Mitigação |
|-------|---------|-----------|
| Conflitos de sync complexos | Alto | Estratégia "last write wins" + log de conflitos |
| Perda de dados offline | Alto | Backup automático do SQLite |
| Parceiro não aceita convite | Médio | Notificação de lembrete após 24h |
| Performance com muitos dados | Médio | Paginação e índices no SQLite |
| Notificações não entregues | Baixo | Fallback com badge no app |

---

## 11. Fora de Escopo (v1)

- Login social (Google/Apple)
- Múltiplos casais/grupos
- Permissões granulares de compartilhamento
- Transferência de admin do casal
- Exportação de dados
- Integração com bancos (Open Finance)
- App mobile nativo

---

## 12. Glossário

| Termo | Definição |
|-------|-----------|
| Admin | Usuário que criou o convite e tem permissão para desvincular o casal |
| Casal | Dois usuários vinculados para compartilhamento de dados |
| Meta Conjunta | Objetivo financeiro criado para o casal, com contribuições de ambos |
| Sync | Processo de sincronização entre banco local e servidor |
| Offline-First | Arquitetura onde o app funciona completamente offline, sincronizando quando possível |