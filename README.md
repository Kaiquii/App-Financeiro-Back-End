## Finance App - API

Back-end em Go para controle financeiro pessoal.

## Rotas da API

# Auth
1. /api/auth/register -> POST -> Criar nova Conta
2. /api/auth/login -> POST -> Autenticar usuário/Login
3. /api/auth/users -> GET -> Listar os Usuários
4. /api/auth/users -> PATCH -> Alterar senha
5. /api/auth/delete -> DELETE -> Deletar Usuário

## Como Rodar a API

1. go run cmd/api/main.go
