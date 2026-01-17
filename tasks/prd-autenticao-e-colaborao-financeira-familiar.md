# PRD: Autenticação e Colaboração Financeira Familiar

## 1. Visão Geral

### 1.1 Objetivo
Adicionar sistema de autenticação e funcionalidade de colaboração financeira que permita casais compartilharem e visualizarem dados financeiros de forma consolidada, mantendo também a visão individual.

### 1.2 Problema
Atualmente o sistema não suporta múltiplos usuários nem permite que casais/famílias tenham uma visão unificada de suas finanças, dificultando o planejamento financeiro conjunto.

### 1.3 Público-Alvo
Casais que desejam gerenciar finanças em conjunto, com visibilidade total dos dados de ambos.

---

## 2. Requisitos Funcionais

### 2.1 Autenticação

| ID | Requisito | Prioridade |
|----|-----------|------------|
| AUTH-01 | Registro de usuário com email e senha | Alta |
| AUTH-02 | Login com email e senha | Alta |
| AUTH-03 | Logout e invalidação de sessão | Alta |
| AUTH-04 | Recuperação de senha por email | Média |
| AUTH-05 | Hash seguro de senhas (bcrypt/argon2) | Alta |
| AUTH-06 | Tokens JWT para sessões | Alta |

### 2.2 Grupos Familiares

| ID | Requisito | Prioridade |
|----|-----------|------------|
| GRP-01 | Criar grupo familiar (usuário vira admin) | Alta |
| GRP-02 | Gerar link/código de convite | Alta |
| GRP-03 | Aceitar convite e entrar no grupo | Alta |
| GRP-04 | Limite de 2 membros por grupo | Alta |
| GRP-05 | Apenas admin pode gerar convites | Alta |
| GRP-06 | Revogar/expirar links de convite | Média |
| GRP-07 | Sair do grupo / remover membro | Média |

### 2.3 Compartilhamento de Dados

| ID | Requisito | Prioridade |
|----|-----------|------------|
| SHR-01 | Todos os dados financeiros visíveis para membros do grupo | Alta |
| SHR-02 | Identificação de quem criou cada registro | Alta |
| SHR-03 | Novos membros veem histórico completo do grupo | Alta |

### 2.4 Visualizações

| ID | Requisito | Prioridade |
|----|-----------|------------|
| VIZ-01 | Toggle para alternar entre "Meus dados" e "Dados do casal" | Alta |
| VIZ-02 | Totais combinados (receita, despesa, saldo) | Alta |
| VIZ-03 | Filtro por membro na visão consolidada | Média |
| VIZ-04 | Indicador visual de quem é o dono de cada transação | Média |

---

## 3. Requisitos Não-Funcionais

| ID | Requisito | Detalhes |
|----|-----------|----------|
| NFR-01 | Stack | Go (backend existente), SQLite |
| NFR-02 | Arquitetura | Manter Clean Architecture / Hexagonal |
| NFR-03 | Segurança | Senhas com bcrypt, JWT com expiração |
| NFR-04 | Performance | Queries otimizadas para dados consolidados |

---

## 4. Modelo de Dados (Novas Entidades)

### 4.1 Users
```
users
├── id (UUID, PK)
├── email (UNIQUE, NOT NULL)
├── password_hash (NOT NULL)
├── name (NOT NULL)
├── created_at
└── updated_at
```

### 4.2 Groups
```
groups
├── id (UUID, PK)
├── name (NOT NULL)
├── admin_user_id (FK → users.id)
├── created_at
└── updated_at
```

### 4.3 Group Members
```
group_members
├── id (UUID, PK)
├── group_id (FK → groups.id)
├── user_id (FK → users.id)
├── joined_at
└── UNIQUE(group_id, user_id)
```

### 4.4 Invites
```
invites
├── id (UUID, PK)
├── group_id (FK → groups.id)
├── code (UNIQUE, NOT NULL)
├── created_by (FK → users.id)
├── expires_at
├── used_at (nullable)
└── used_by (FK → users.id, nullable)
```

### 4.5 Alterações em Tabelas Existentes
- Adicionar `user_id` (FK) em todas as tabelas de dados financeiros (transações, categorias, etc.)
- Adicionar `group_id` (FK, nullable) para dados compartilhados em grupo

---

## 5. API Endpoints

### 5.1 Autenticação
```
POST   /api/auth/register     # Criar conta
POST   /api/auth/login        # Login
POST   /api/auth/logout       # Logout
POST   /api/auth/refresh      # Renovar token
POST   /api/auth/forgot       # Solicitar reset de senha
POST   /api/auth/reset        # Resetar senha
```

### 5.2 Grupos
```
POST   /api/groups            # Criar grupo
GET    /api/groups/:id        # Detalhes do grupo
DELETE /api/groups/:id        # Deletar grupo (admin only)
POST   /api/groups/:id/invite # Gerar convite
DELETE /api/groups/:id/invite/:code  # Revogar convite
POST   /api/groups/join       # Entrar via código
DELETE /api/groups/:id/members/:uid  # Remover membro
POST   /api/groups/:id/leave  # Sair do grupo
```

### 5.3 Dados Financeiros (alterações)
```
GET    /api/transactions?view=personal|group  # Filtrar por visão
GET    /api/dashboard?view=personal|group     # Dashboard consolidado
```

---

## 6. Fluxos de Usuário

### 6.1 Registro e Criação de Grupo
```
1. Usuário acessa /register
2. Preenche email, senha, nome
3. Sistema cria conta e faz login automático
4. Usuário cria grupo familiar (nome do grupo)
5. Sistema gera código de convite para compartilhar
```

### 6.2 Convite do Parceiro(a)
```
1. Admin copia link/código de convite
2. Compartilha com parceiro(a) (WhatsApp, etc.)
3. Parceiro(a) acessa link
4. Se não tem conta: cria conta → entra no grupo
5. Se tem conta: faz login → entra no grupo
6. Ambos agora veem dados consolidados
```

### 6.3 Uso Diário
```
1. Usuário faz login
2. Dashboard mostra por padrão "Dados do Casal"
3. Toggle permite alternar para "Meus Dados"
4. Ao adicionar transação, fica vinculada ao usuário
5. Na visão consolidada, mostra indicador de quem criou
```

---

## 7. Fases de Implementação

### Fase 1: Autenticação (Base)
- [ ] Entidade User e repositório
- [ ] Endpoints de registro e login
- [ ] Middleware de autenticação JWT
- [ ] Hash de senhas com bcrypt
- [ ] Migração: adicionar user_id às tabelas existentes

### Fase 2: Grupos e Convites
- [ ] Entidades Group, GroupMember, Invite
- [ ] CRUD de grupos
- [ ] Sistema de convites por código
- [ ] Lógica de entrada no grupo

### Fase 3: Compartilhamento de Dados
- [ ] Migração: adicionar group_id às tabelas
- [ ] Queries para dados consolidados
- [ ] Filtros personal/group nos endpoints existentes

### Fase 4: Interface de Visualização
- [ ] Toggle de visão (pessoal/casal)
- [ ] Indicadores de propriedade nas transações
- [ ] Dashboard consolidado

---

## 8. Considerações de Segurança

| Aspecto | Implementação |
|---------|---------------|
| Senhas | bcrypt com cost factor ≥ 12 |
| Tokens | JWT com expiração de 24h, refresh token de 7 dias |
| Convites | Códigos UUID, expiração de 7 dias |
| Autorização | Verificar pertencimento ao grupo em todas as queries |
| Rate Limiting | Limitar tentativas de login (5/min) |

---

## 9. Critérios de Aceite

- [ ] Usuário consegue criar conta e fazer login
- [ ] Usuário consegue criar grupo e gerar convite
- [ ] Parceiro(a) consegue entrar no grupo via código
- [ ] Ambos veem dados consolidados do casal
- [ ] Toggle funciona para alternar entre visões
- [ ] Cada transação mostra quem a criou
- [ ] Grupo limitado a 2 membros
- [ ] Apenas admin gera convites