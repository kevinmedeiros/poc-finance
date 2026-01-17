# PRD: Autenticação e Compartilhamento Familiar

## 1. Visão Geral

### 1.1 Objetivo
Adicionar funcionalidade de autenticação e compartilhamento familiar/casal ao sistema de gestão financeira existente, permitindo que grupos familiares (2+ pessoas) tenham uma visão holística das finanças compartilhadas enquanto mantêm privacidade sobre dados individuais.

### 1.2 Stack Tecnológica Atual
| Componente | Tecnologia |
|------------|------------|
| Backend | Go 1.25.5 + Echo v4.15.0 |
| Frontend | HTML Templates + HTMX 1.9.10 + Tailwind CSS |
| Banco de Dados | SQLite + GORM v1.31.1 |
| Autenticação Atual | Nenhuma |

### 1.3 Público-Alvo
Casais e famílias que desejam controle financeiro compartilhado com visão consolidada de entradas e saídas.

---

## 2. Funcionalidades

### 2.1 Autenticação (AUTH)

| ID | Funcionalidade | Descrição |
|----|----------------|-----------|
| AUTH-01 | Registro de usuário | Cadastro com email + senha |
| AUTH-02 | Login | Autenticação email + senha |
| AUTH-03 | Logout | Encerramento de sessão |
| AUTH-04 | Recuperação de senha | Reset via email |
| AUTH-05 | JWT Tokens | Access token + refresh token para sessões |
| AUTH-06 | Middleware de autenticação | Proteção de rotas privadas |

### 2.2 Gestão de Grupos Familiares (GROUP)

| ID | Funcionalidade | Descrição |
|----|----------------|-----------|
| GROUP-01 | Criar grupo familiar | Usuário cria grupo e se torna membro |
| GROUP-02 | Convidar membros | Gerar código/link de convite |
| GROUP-03 | Aceitar convite | Entrar em grupo via código/link |
| GROUP-04 | Listar membros | Visualizar todos os membros do grupo |
| GROUP-05 | Sair do grupo | Membro pode deixar o grupo |
| GROUP-06 | Excluir grupo | Apenas quando todos saírem |
| GROUP-07 | Membros ilimitados | Suporte a 2+ pessoas com papéis iguais |

### 2.3 Contas e Compartilhamento (ACCOUNT)

| ID | Funcionalidade | Descrição |
|----|----------------|-----------|
| ACC-01 | Conta individual | Cada membro tem dados privados por padrão |
| ACC-02 | Contas conjuntas ilimitadas | Criar múltiplas contas compartilhadas (ex: "Casa", "Viagem") |
| ACC-03 | Vincular transação à conta | Associar receita/despesa a conta individual ou conjunta |
| ACC-04 | Divisão automática | Dividir despesas por percentual customizado entre membros |
| ACC-05 | Saldo por conta | Visualizar saldo de cada conta (individual e conjuntas) |

### 2.4 Dashboard e Relatórios (DASH)

| ID | Funcionalidade | Descrição |
|----|----------------|-----------|
| DASH-01 | Dashboard individual | Visão das finanças pessoais do usuário |
| DASH-02 | Dashboard do grupo | Visão consolidada de todas as contas conjuntas |
| DASH-03 | Comparativo entre membros | Visualizar contribuições de cada membro |
| DASH-04 | Filtros por conta | Filtrar por conta individual ou conjunta específica |
| DASH-05 | Resumo holístico | Total de entradas/saídas do grupo familiar |

### 2.5 Metas Financeiras (GOAL)

| ID | Funcionalidade | Descrição |
|----|----------------|-----------|
| GOAL-01 | Criar meta do grupo | Meta compartilhada (ex: "Viagem para Europa") |
| GOAL-02 | Contribuição por membro | Rastrear quanto cada um contribuiu |
| GOAL-03 | Progresso visual | Barra de progresso da meta |
| GOAL-04 | Vincular a conta conjunta | Meta associada a uma conta específica |

### 2.6 Notificações (NOTIF)

| ID | Funcionalidade | Descrição |
|----|----------------|-----------|
| NOTIF-01 | Convite recebido | Notificar quando convidado para grupo |
| NOTIF-02 | Novo gasto do parceiro | Notificar gastos em contas conjuntas |
| NOTIF-03 | Meta atingida | Notificar quando meta for alcançada |
| NOTIF-04 | Resumo periódico | Resumo semanal/mensal do grupo |
| NOTIF-05 | Alerta de limite | Notificar quando orçamento atingir limite |

---

## 3. Modelo de Dados

### 3.1 Novas Tabelas

```
┌─────────────────────────────────────────────────────────────────┐
│                           users                                  │
├─────────────────────────────────────────────────────────────────┤
│ id (PK) │ email │ password_hash │ name │ created_at │ updated_at│
└─────────────────────────────────────────────────────────────────┘
                              │
                              │ 1:N
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                       refresh_tokens                             │
├─────────────────────────────────────────────────────────────────┤
│ id (PK) │ user_id (FK) │ token │ expires_at │ created_at        │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│                        family_groups                             │
├─────────────────────────────────────────────────────────────────┤
│ id (PK) │ name │ invite_code │ created_at │ updated_at          │
└─────────────────────────────────────────────────────────────────┘
                              │
                              │ 1:N
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                       group_members                              │
├─────────────────────────────────────────────────────────────────┤
│ id (PK) │ group_id (FK) │ user_id (FK) │ joined_at              │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│                         accounts                                 │
├─────────────────────────────────────────────────────────────────┤
│ id (PK) │ name │ type (individual/joint) │ group_id (FK, null)  │
│ user_id (FK, null) │ created_at                                 │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│                      account_splits                              │
├─────────────────────────────────────────────────────────────────┤
│ id (PK) │ account_id (FK) │ user_id (FK) │ percentage           │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│                     financial_goals                              │
├─────────────────────────────────────────────────────────────────┤
│ id (PK) │ group_id (FK) │ account_id (FK) │ name │ target_amount│
│ current_amount │ deadline │ created_at                          │
└─────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────┐
│                       notifications                              │
├─────────────────────────────────────────────────────────────────┤
│ id (PK) │ user_id (FK) │ type │ message │ read │ created_at     │
└─────────────────────────────────────────────────────────────────┘
```

### 3.2 Alterações em Tabelas Existentes

Adicionar coluna `account_id` (FK, nullable) nas tabelas:
- `incomes`
- `expenses`
- `credit_cards`
- `bills`

---

## 4. API Endpoints

### 4.1 Autenticação
| Método | Endpoint | Descrição |
|--------|----------|-----------|
| POST | `/auth/register` | Criar conta |
| POST | `/auth/login` | Fazer login |
| POST | `/auth/logout` | Fazer logout |
| POST | `/auth/refresh` | Renovar access token |
| POST | `/auth/forgot-password` | Solicitar reset |
| POST | `/auth/reset-password` | Resetar senha |

### 4.2 Grupos
| Método | Endpoint | Descrição |
|--------|----------|-----------|
| POST | `/groups` | Criar grupo |
| GET | `/groups` | Listar grupos do usuário |
| GET | `/groups/:id` | Detalhes do grupo |
| POST | `/groups/:id/invite` | Gerar código de convite |
| POST | `/groups/join` | Entrar via código |
| DELETE | `/groups/:id/leave` | Sair do grupo |

### 4.3 Contas
| Método | Endpoint | Descrição |
|--------|----------|-----------|
| POST | `/accounts` | Criar conta (individual ou conjunta) |
| GET | `/accounts` | Listar contas |
| PUT | `/accounts/:id` | Editar conta |
| DELETE | `/accounts/:id` | Excluir conta |
| PUT | `/accounts/:id/splits` | Configurar divisão percentual |

### 4.4 Metas
| Método | Endpoint | Descrição |
|--------|----------|-----------|
| POST | `/goals` | Criar meta |
| GET | `/goals` | Listar metas do grupo |
| PUT | `/goals/:id` | Atualizar meta |
| DELETE | `/goals/:id` | Excluir meta |

### 4.5 Notificações
| Método | Endpoint | Descrição |
|--------|----------|-----------|
| GET | `/notifications` | Listar notificações |
| PUT | `/notifications/:id/read` | Marcar como lida |
| PUT | `/notifications/read-all` | Marcar todas como lidas |

---

## 5. Segurança

| Aspecto | Implementação |
|---------|---------------|
| Senhas | bcrypt com cost factor 12 |
| Access Token | JWT com expiração de 15 minutos |
| Refresh Token | Token opaco, armazenado no banco, expiração 7 dias |
| Transmissão | HTTPS obrigatório |
| Cookies | HttpOnly, Secure, SameSite=Strict |
| Rate Limiting | Limite de tentativas de login |

---

## 6. Fluxos de Usuário

### 6.1 Registro e Criação de Grupo
```
Usuário → Registro → Criar Grupo → Gerar Link de Convite → Enviar ao Parceiro
```

### 6.2 Entrada no Grupo
```
Parceiro → Registro → Colar Link/Código → Confirmar Entrada → Acesso ao Grupo
```

### 6.3 Gestão de Despesas Compartilhadas
```
Membro → Nova Despesa → Selecionar Conta Conjunta → Sistema Divide Automaticamente
```

---

## 7. Wireframes Conceituais

### 7.1 Tela de Login
```
┌────────────────────────────────────────┐
│          Gestão Financeira             │
│                                        │
│   ┌──────────────────────────────┐     │
│   │ Email                        │     │
│   └──────────────────────────────┘     │
│   ┌──────────────────────────────┐     │
│   │ Senha                        │     │
│   └──────────────────────────────┘     │
│                                        │
│   [ Entrar ]                           │
│                                        │
│   Esqueceu a senha?  |  Criar conta    │
└────────────────────────────────────────┘
```

### 7.2 Seletor de Conta no Dashboard
```
┌────────────────────────────────────────┐
│  📊 Dashboard                          │
│  ┌──────────────────────────────────┐  │
│  │ Conta: [▼ Todas as Contas     ]  │  │
│  │        ○ Minha Conta Individual  │  │
│  │        ○ Casa (Conjunta)         │  │
│  │        ○ Viagem 2024 (Conjunta)  │  │
│  │        ○ Visão Consolidada       │  │
│  └──────────────────────────────────┘  │
└────────────────────────────────────────┘
```

### 7.3 Comparativo de Membros
```
┌────────────────────────────────────────┐
│  👥 Comparativo do Grupo               │
│                                        │
│  João          ████████████░░ R$ 3.200 │
│  Maria         ██████████░░░░ R$ 2.800 │
│                                        │
│  Total do Mês: R$ 6.000                │
└────────────────────────────────────────┘
```

---

## 8. Critérios de Aceite

### 8.1 Autenticação
- [ ] Usuário consegue se registrar com email e senha
- [ ] Usuário consegue fazer login e recebe tokens JWT
- [ ] Refresh token renova access token expirado
- [ ] Rotas protegidas retornam 401 sem autenticação

### 8.2 Grupos
- [ ] Usuário consegue criar grupo familiar
- [ ] Link de convite é gerado e funciona
- [ ] Múltiplos membros conseguem entrar no mesmo grupo
- [ ] Membro consegue sair do grupo

### 8.3 Contas
- [ ] Conta individual criada automaticamente no registro
- [ ] Usuário consegue criar contas conjuntas ilimitadas
- [ ] Divisão percentual é aplicada automaticamente
- [ ] Transações podem ser vinculadas a qualquer conta

### 8.4 Dashboard
- [ ] Dashboard individual mostra apenas dados do usuário
- [ ] Dashboard do grupo mostra dados consolidados
- [ ] Comparativo mostra contribuição de cada membro

### 8.5 Metas e Notificações
- [ ] Metas do grupo são visíveis para todos os membros
- [ ] Notificações são enviadas nos eventos configurados
- [ ] Notificações podem ser marcadas como lidas

---

## 9. Considerações Técnicas

### 9.1 Migração de Dados
- Dados existentes serão associados ao primeiro usuário registrado
- Usuário poderá reassociar transações a contas após configuração

### 9.2 Compatibilidade
- Manter todas as funcionalidades existentes funcionando
- Novos campos são nullable para backwards compatibility

### 9.3 Performance
- Índices em `user_id`, `group_id`, `account_id`
- Cache de sessão para reduzir queries de autenticação

---

## 10. Fora de Escopo (Futuro)

- Login social (Google, Apple)
- Autenticação 2FA
- Hierarquia de permissões (admin/membro)
- App mobile
- Criptografia end-to-end de dados financeiros