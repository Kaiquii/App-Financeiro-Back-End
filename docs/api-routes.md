# Rotas da API

Base da API:

```txt
/api
```

Rotas protegidas precisam do header:

```txt
Authorization: Bearer TOKEN
```

## Auth

1. `/api/auth/register` -> POST -> Criar nova conta
2. `/api/auth/login` -> POST -> Autenticar usuario/login
3. `/api/auth/users` -> GET -> Listar usuarios
4. `/api/auth/forgot-password` -> POST -> Enviar codigo de redefinicao de senha
5. `/api/auth/reset-password` -> POST -> Redefinir senha com codigo

## Admin

1. `/api/admin/users/ID` -> DELETE -> Deletar usuario e todos os dados dele
2. `/api/admin/users/ID/revoke-access` -> PATCH -> Revogar acesso de usuario
3. `/api/admin/users/ID/restore-access` -> PATCH -> Liberar acesso de usuario bloqueado

Observacao:

O delete admin remove despesas, rendas, categorias, conversas/mensagens do assistente, codigos de redefinicao de senha e foto de perfil. Para apenas bloquear o uso do app sem apagar historico, use `revoke-access`.

## Despesas

1. `/api/expenses/` -> POST -> Criar nova despesa
2. `/api/expenses?month=6&year=2026` -> GET -> Listar despesas
3. `/api/expenses?month=6&year=2026&type=Parcelada` -> GET -> Listar despesas filtrando por tipo
4. `/api/expenses/ID` -> GET -> Ver despesa por ID
5. `/api/expenses/ID` -> PATCH -> Atualizar despesa
6. `/api/expenses/ID?delete_future=true` -> DELETE -> Deletar despesa atual e futuras

Tipos aceitos:

```txt
Unica
Parcelada
Fixa
```

No PATCH, o body pode usar `update_future: true` para atualizar despesas futuras da mesma serie.

## Rendas

1. `/api/incomes/` -> POST -> Cadastrar renda
2. `/api/incomes/?month=6&year=2026` -> GET -> Ver rendas
3. `/api/incomes/ID` -> PATCH -> Atualizar renda
4. `/api/incomes/ID?delete_future=true` -> DELETE -> Deletar renda atual e futuras

Fontes aceitas:

```txt
Salario
Adiantamento
Renda Extra
```

Tipos aceitos:

```txt
Unica
Fixa
```

No PATCH, o body pode usar `update_future: true` para atualizar rendas futuras da mesma fonte.

## Relatorios

1. `/api/reports/summary?month=6&year=2026` -> GET -> Ver resumo financeiro
2. `/api/reports/categories?month=6&year=2026` -> GET -> Ver resumo por categorias
3. `/api/reports/chart?year=2026` -> GET -> Ver dados para grafico anual
4. `/api/reports/yearly-summary?year=2026` -> GET -> Ver media mensal e economia total do ano
5. `/api/reports/installment-commitments?months=12&month=6&year=2026&include_current_month_as_paid=true` -> GET -> Ver compromissos parcelados
6. `/api/reports/month-comparison?month=6&year=2026` -> GET -> Comparar o mes selecionado com o mes anterior
7. `/api/reports/month-comparison?month=6&year=2026&compare_month=1&compare_year=2026` -> GET -> Comparar o mes selecionado com outro mes

## Categorias

1. `/api/categories/` -> POST -> Criar categoria
2. `/api/categories/` -> GET -> Listar categorias
3. `/api/categories/ID` -> PATCH -> Atualizar categoria
4. `/api/categories/ID` -> DELETE -> Deletar categoria

## Perfil

1. `/api/users/profile` -> GET -> Ver perfil do usuario logado
2. `/api/users/profile/` -> PATCH -> Atualizar perfil do usuario
3. `/api/users/profile/photo` -> PATCH -> Atualizar foto de perfil
4. `/api/users/profile/photo` -> DELETE -> Remover foto de perfil

Upload da foto:

```txt
Content-Type: multipart/form-data
Campo: photo
Formatos: JPG, JPEG, PNG ou GIF
Limite: 5 MB
```

A API salva a foto como `/uploads/users/ID/avatar.jpg` quando usa storage local e guarda apenas `avatar_url` no banco.

## Assistente IA

1. `/api/assistant/chat` -> POST -> Conversar com o assistente financeiro
2. `/api/assistant/conversations` -> GET -> Listar conversas salvas do usuario
3. `/api/assistant/conversations/ID/messages` -> GET -> Listar mensagens de uma conversa
4. `/api/assistant/conversations/ID` -> DELETE -> Apagar conversa

Body do chat:

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

Se `conversation_id` nao for enviado, a API cria uma nova conversa automaticamente e devolve o id na resposta.

