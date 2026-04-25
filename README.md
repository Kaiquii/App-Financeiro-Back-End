# Finance App - API

Back-end em Go para controle financeiro pessoal.

# Rotas da API

## Auth
1. /api/auth/register -> POST -> Criar nova Conta
2. /api/auth/login -> POST -> Autenticar usuário/Login
3. /api/auth/users -> GET -> Listar os Usuários
4. /api/auth/forgot-password -> POST -> Enviar codigo de redefinicao de senha
5. /api/auth/reset-password -> POST -> Redefinir senha com codigo

## Admin
1. /api/admin/users/ID -> Delete -> Deletar Usuário, se usuário for Admin

## Despesas
1. /api/expenses/ -> POST -> Criar nova Despesa
2. /api/expenses?month=03&year=2026 -> GET -> Listar Despesas
3. /api/expenses/ID -> PATCH -> Atualizar Despesas
4. /api/expenses/ID -> DELETE -> Deletar Despesas

## Salário
1. /api/incomes/ -> POST -> Cadastrar Salário
2. /api/incomes/ -> GET -> Ver Salários
3. /api/incomes/ID -> PATCH -> Atualizar Salário
4. /api/incomes/ID -> DELETE -> Deletar Salário

## Resumo
1. /api/reports/summary?month=3&year=2026 -> GET -> Ver Resumo financeiro
2. /api/reports/categories?month=3&year=2026 -> GET -> Ver Resumo de Categorias
3. /api/reports/chart?year=2026 -> GET -> Ver Dados para o Gráfico de Barras
4. /api/reports/yearly-summary?year=2026 -> GET -> Ver Média Mensal

## Categorias
1. /api/categories/ -> POST -> Criar categoria
2. /api/categories/ -> GET -> Listar categorias
3. /api/categories/{{Category_ID}} -> PATCH -> Atualizar categorias
4. /api/categories/{{Category_ID}} -> DELETE -> Deletar categorias

## Perfil
1. /api/users/profile/ -> PATCH -> Atualizar perfil de Usuario

## Como Rodar a API

1. go run cmd/api/main.go
