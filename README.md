# SobraAí - API

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
2. /api/users/profile -> GET -> Ver perfil do usuario logado
3. /api/users/profile/photo -> PATCH -> Atualizar foto de perfil
4. /api/users/profile/photo -> DELETE -> Remover foto de perfil

Upload da foto:
```txt
Content-Type: multipart/form-data
Campo: photo
Formatos: JPG, JPEG, PNG ou GIF
Limite: 5 MB
```

A API salva a foto como `/uploads/users/ID/avatar.jpg` e guarda apenas `avatar_url` no banco.

Storage da foto de perfil:
1. AVATAR_STORAGE_DRIVER -> `local` ou `oci`. Padrao: local
2. UPLOADS_DIR -> Pasta local onde fotos de perfil sao salvas. Padrao: uploads
3. OCI_NAMESPACE -> Namespace do Object Storage
4. OCI_BUCKET -> Nome do bucket do Object Storage
5. OCI_REGION -> Regiao do bucket. Padrao: sa-saopaulo-1
6. OCI_ACCESS_KEY -> Access Key da Customer Secret Key
7. OCI_SECRET_KEY -> Secret Key da Customer Secret Key
8. OCI_PUBLIC_BASE_URL -> URL publica opcional para CDN/bucket customizado

## Assistente IA
1. /api/assistant/chat -> POST -> Conversar com o assistente financeiro
2. /api/assistant/conversations -> GET -> Listar conversas salvas do usuario
3. /api/assistant/conversations/ID/messages -> GET -> Listar mensagens de uma conversa
4. /api/assistant/conversations/ID -> DELETE -> Apagar conversa

Body:
```json
{
  "message": "Quanto gastei do salario em maio?",
  "conversation_id": 1,
  "history": [
    {
      "role": "assistant",
      "content": "Entendi a despesa Pao, R$ 4,00, paga com salario em maio. Posso cadastrar?"
    }
  ]
}
```

Se conversation_id nao for enviado, a API cria uma nova conversa automaticamente e devolve o id na resposta.

Variaveis de ambiente:
1. GEMINI_API_KEY -> Chave da API do Gemini
2. GEMINI_MODEL -> Modelo opcional. Padrao: gemini-2.5-flash
3. GROQ_API_KEY -> Chave da API da Groq para fallback quando o Gemini atingir limite
4. GROQ_MODEL -> Modelo opcional. Padrao: llama-3.1-8b-instant
5. UPLOADS_DIR -> Pasta onde fotos de perfil sao salvas quando AVATAR_STORAGE_DRIVER=local. Padrao: uploads

## Como Rodar a API

1. go run cmd/api/main.go

## Criar arquivo .tar
1. docker build -t app-financeiro-api .
2. docker save -o app-financeiro-backend.tar app-financeiro-api
3. Passar o .tar e o docker-compose.yml para a pasta

## Subir na VPS
1. sudo docker load -i app-financeiro-backend.tar
2. sudo mkdir -p /var/app-financeiro/uploads
3. sudo docker compose up -d api

# Pare apenas o contêiner da API atual
- docker compose stop api

# Remova o contêiner antigo (os dados do banco estão a salvo em volumes)
- docker compose rm -f api

# Carregue a nova imagem Docker que você transferiu
- docker load -i app-financeiro-backend.tar

# Inicie novamente a API utilizando a nova imagem
- docker compose up -d api
