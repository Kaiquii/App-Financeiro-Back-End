# Finance App - API

Back-end em Go para controle financeiro pessoal.

# Rotas da API

## Auth
1. /api/auth/register -> POST -> Criar nova Conta
2. /api/auth/login -> POST -> Autenticar usuário/Login
3. /api/auth/users -> GET -> Listar os Usuários
4. /api/auth/users -> PATCH -> Alterar senha
5. /api/auth/delete -> DELETE -> Deletar Usuário

## Despesas
1. /api/expenses/ -> POST -> Criar nova Despesa
2. /api/expenses?month=03&year=2026 -> GET -> Listar Despesas
3. /api/expenses/1 -> PATCH -> Atualizar Despesas
4. /api/expenses/1 -> DELETE -> Deletar Despesas

## Salário
1. /api/incomes/ -> POST -> Cadastrar Salário
2. /api/incomes/ -> GET -> Ver Salários
3. /api/incomes/1 -> PATCH -> Atualizar Salário
4. /api/incomes/1 -> DELETE -> Deletar Salário

## Resumo
1. /api/reports/summary?month=3&year=2026 -> GET -> Ver Resumo financeiro

## Como Rodar a API

1. go run cmd/api/main.go
