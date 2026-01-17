# Tasks - Autenticação e Compartilhamento Familiar

Este diretório contém as tarefas organizadas por fases para implementar o PRD de autenticação e compartilhamento familiar.

## Estrutura de Fases

| Fase | Nome | Arquivo | Tarefas |
|------|------|---------|---------|
| 1 | Autenticação | `phase1-auth.json` | 12 tarefas |
| 2 | Grupos Familiares | `phase2-groups.json` | 10 tarefas |
| 3 | Contas e Compartilhamento | `phase3-accounts.json` | 14 tarefas |
| 4 | Dashboard e Relatórios | `phase4-dashboard.json` | 10 tarefas |
| 5 | Metas Financeiras | `phase5-goals.json` | 9 tarefas |
| 6 | Notificações | `phase6-notifications.json` | 12 tarefas |

**Total: 67 tarefas**

## Ordem de Implementação

As fases devem ser implementadas na ordem numérica, pois existem dependências:

1. **Fase 1 (Auth)** - Base para todas as outras funcionalidades
2. **Fase 2 (Grupos)** - Depende de Auth para usuários
3. **Fase 3 (Contas)** - Depende de Auth e Grupos
4. **Fase 4 (Dashboard)** - Depende de Contas
5. **Fase 5 (Metas)** - Depende de Grupos e Contas
6. **Fase 6 (Notificações)** - Depende de todas as outras

## Status das Tarefas

- `pending` - Aguardando implementação
- `in_progress` - Em desenvolvimento
- `completed` - Finalizada
- `blocked` - Bloqueada por dependência

## Estrutura de uma Tarefa

```json
{
  "id": "TASK-X.Y",
  "feature_id": "FEATURE-XX",
  "name": "Nome da tarefa",
  "description": "Descrição detalhada",
  "status": "pending",
  "files": ["arquivo1.go", "arquivo2.html"],
  "dependencies": ["TASK-X.Z"]
}
```

## Como Usar

1. Escolha uma fase para trabalhar
2. Verifique as dependências das tarefas
3. Implemente na ordem de dependências
4. Atualize o status conforme progresso
