# Planejamento - Caixinhas

Este documento descreve a funcionalidade de caixinhas planejada para o SobraAi.

A caixinha representa um objetivo financeiro do usuario. Quando o usuario guarda dinheiro em uma caixinha, esse valor tambem entra no app como uma despesa, porque o dinheiro saiu do saldo disponivel.

## Objetivo

Permitir que o usuario crie objetivos como:

- Reserva de emergencia
- Viagem
- Comprar notebook
- Entrada de carro
- Investimento mensal

Cada caixinha deve permitir acompanhar:

- valor guardado
- valor da meta
- meta mensal
- prazo
- historico de aportes, edicoes e retiradas
- previsao simples de conclusao

Nao deve calcular rendimento real. Rendimento fica com o banco. A API pode apenas simular o futuro usando a meta mensal informada pelo usuario.

## Dados da caixinha

Campos sugeridos:

```txt
id
user_id
nome
descricao
valor_meta
meta_mensal
mes_prazo
ano_prazo
status
created_at
updated_at
```

Campos calculados nas respostas:

```txt
saldo_atual
valor_restante
progresso_percentual
valor_guardado_mes_atual
diferenca_meta_mensal
status_meta_mensal
```

Status sugeridos:

```txt
ativa
pausada
concluida
```

## Categoria Caixinhas

Deve existir uma categoria especial chamada `Caixinhas`.

Regras:

- A categoria `Caixinhas` deve ser visivel na lista de categorias.
- Ela deve entrar normalmente nos relatorios, porque representa dinheiro que saiu do saldo.
- O usuario nao deve cadastrar despesa nessa categoria pela tela normal de despesas.
- Uma despesa comum nao pode ser alterada para a categoria `Caixinhas`.
- Uma despesa criada por caixinha nao pode mudar para outra categoria.
- A categoria pode ser criada automaticamente pelo back-end quando necessario.

## Guardar dinheiro

Quando o usuario guarda dinheiro em uma caixinha, a API deve criar:

1. Um registro de aporte/movimentacao da caixinha.
2. Uma despesa vinculada na categoria `Caixinhas`.

Cada aporte deve ser uma despesa diferente.

Tipos permitidos para aporte:

- `Unica`
- `Fixa`

Tipo nao permitido:

- `Parcelada`

A despesa criada pelo aporte deve usar a categoria `Caixinhas`.

Exemplo de despesa criada:

```txt
Descricao: Caixinha: Viagem - Guardei do salario
Categoria: Caixinhas
Tipo: Unica
Valor: 300
Fonte: Salario
Data: 2026-06-15
```

## Aporte fixo

Aporte fixo deve reaproveitar o mesmo comportamento atual de despesas fixas.

Ou seja:

- cria despesas futuras seguindo a regra atual do back-end
- permite editar somente o mes atual ou os futuros
- permite deletar somente o mes atual ou os futuros
- cada despesa futura continua vinculada a caixinha

Exemplo de uso:

```txt
Caixinha: Reserva de emergencia
Valor inicial mensal: 100
Tipo: Fixa
Inicio: Junho/2026
```

Se no mes seguinte o usuario colocou 150 em vez de 100, ele pode editar aquela despesa especifica. A caixinha deve recalcular o saldo usando o valor atualizado.

## Edicao de caixinha

O usuario pode editar os dados da caixinha:

- nome
- descricao
- valor da meta
- meta mensal
- prazo
- status

Editar a caixinha nao deve alterar aportes/despesas antigos.

A meta mensal e apenas uma referencia. Se o usuario diminuir a meta mensal e os aportes do mes ficarem acima dela, o front-end pode avisar:

```txt
Essa caixinha passou a meta mensal em R$ X.
```

O back-end deve retornar os dados calculados para permitir esse aviso.

## Edicao de aporte

O aporte e uma despesa. Por isso, ele pode ser editado pela tela normal de despesas.

Permitido editar:

- valor
- descricao
- data
- fonte de pagamento
- regra de edicao de despesa fixa, quando for aporte fixo

Nao permitido:

- mudar categoria
- mudar para `Parcelada`
- transformar uma despesa comum em despesa de caixinha

Se o usuario reduzir o valor de uma despesa de caixinha, isso representa uma retirada parcial.

Exemplo:

```txt
Despesa/aporte era 100
Usuario editou para 60
Caixinha passa a considerar 60
Historico registra alteracao de 100 para 60
```

## Deletar aporte

O aporte pode ser deletado em dois lugares:

- pela tela da caixinha
- pela tela normal de despesas

Deletar uma despesa vinculada a caixinha equivale a retirar aquele dinheiro da caixinha.

Se for aporte fixo, deve seguir a regra atual de despesas fixas:

- deletar somente este mes
- deletar futuros

O historico da caixinha deve registrar a retirada/remocao.

## Historico

A caixinha deve ter historico de movimentacoes.

Eventos sugeridos:

```txt
caixinha_criada
caixinha_editada
aporte_criado
aporte_editado
aporte_removido
caixinha_deletada
```

Exemplo:

```json
{
  "historico": [
    {
      "id": 1,
      "tipo": "aporte_criado",
      "valor": 300,
      "descricao": "Guardei do salario",
      "data": "2026-06-15",
      "despesa_id": 88
    },
    {
      "id": 2,
      "tipo": "aporte_editado",
      "valor_anterior": 100,
      "valor_novo": 60,
      "descricao": "Valor do aporte alterado",
      "data": "2026-06-16",
      "despesa_id": 89
    },
    {
      "id": 3,
      "tipo": "aporte_removido",
      "valor": 50,
      "descricao": "Aporte removido",
      "data": "2026-06-17",
      "despesa_id": 90
    }
  ]
}
```

## Deletar caixinha

A caixinha so pode ser deletada se o saldo atual for zero.

Regras:

- se `saldo_atual > 0`, bloquear delete
- se `saldo_atual = 0`, permitir delete
- historico antigo nao bloqueia delete
- ao deletar a caixinha, apagar o historico junto

Resposta sugerida quando bloquear:

```json
{
  "error": "Retire todo o dinheiro da caixinha antes de deletar"
}
```

Status HTTP sugerido:

```txt
409 Conflict
```

## Endpoints sugeridos

Os endpoints ficam em ingles, mantendo um padrao REST simples. O JSON fica em portugues.

```http
GET    /api/saving-boxes
POST   /api/saving-boxes
GET    /api/saving-boxes/:id
PATCH  /api/saving-boxes/:id
DELETE /api/saving-boxes/:id

POST   /api/saving-boxes/:id/deposits
GET    /api/saving-boxes/:id/history
GET    /api/saving-boxes/:id/projection
GET    /api/saving-boxes/summary
```

Edicao e delecao de aporte podem usar as rotas atuais de despesas:

```http
PATCH  /api/expenses/:id
DELETE /api/expenses/:id
```

Assim nao duplica regra de edicao/delecao de despesa.

## Criar caixinha

```http
POST /api/saving-boxes
```

Body:

```json
{
  "nome": "Viagem",
  "descricao": "Dinheiro para viajar em dezembro",
  "valor_meta": 3000,
  "meta_mensal": 300,
  "mes_prazo": 12,
  "ano_prazo": 2026,
  "status": "ativa"
}
```

Resposta:

```json
{
  "id": 1,
  "nome": "Viagem",
  "descricao": "Dinheiro para viajar em dezembro",
  "valor_meta": 3000,
  "meta_mensal": 300,
  "mes_prazo": 12,
  "ano_prazo": 2026,
  "status": "ativa",
  "saldo_atual": 0,
  "valor_restante": 3000,
  "progresso_percentual": 0
}
```

## Listar caixinhas

```http
GET /api/saving-boxes
```

Resposta:

```json
{
  "caixinhas": [
    {
      "id": 1,
      "nome": "Viagem",
      "descricao": "Dinheiro para viajar em dezembro",
      "valor_meta": 3000,
      "meta_mensal": 300,
      "mes_prazo": 12,
      "ano_prazo": 2026,
      "status": "ativa",
      "saldo_atual": 600,
      "valor_restante": 2400,
      "progresso_percentual": 20
    }
  ]
}
```

## Guardar dinheiro

```http
POST /api/saving-boxes/:id/deposits
```

Body para aporte unico:

```json
{
  "valor": 300,
  "descricao": "Guardei do salario",
  "fonte_pagamento": "Salario",
  "data": "2026-06-15",
  "tipo": "Unica"
}
```

Body para aporte fixo:

```json
{
  "valor": 100,
  "descricao": "Reserva mensal",
  "fonte_pagamento": "Salario",
  "data": "2026-06-15",
  "tipo": "Fixa"
}
```

Resposta:

```json
{
  "mensagem": "Dinheiro guardado com sucesso",
  "aporte": {
    "id": 10,
    "caixinha_id": 1,
    "despesa_id": 88,
    "valor": 300,
    "descricao": "Guardei do salario",
    "fonte_pagamento": "Salario",
    "data": "2026-06-15",
    "tipo": "Unica"
  },
  "despesa": {
    "id": 88,
    "valor": 300,
    "descricao": "Caixinha: Viagem - Guardei do salario",
    "categoria": "Caixinhas",
    "fonte_pagamento": "Salario",
    "data": "2026-06-15",
    "tipo": "Unica"
  },
  "caixinha": {
    "id": 1,
    "saldo_atual": 900,
    "valor_restante": 2100,
    "progresso_percentual": 30
  }
}
```

## Historico

```http
GET /api/saving-boxes/:id/history
```

Resposta:

```json
{
  "caixinha": {
    "id": 1,
    "nome": "Viagem",
    "saldo_atual": 900
  },
  "historico": [
    {
      "id": 1,
      "tipo": "aporte_criado",
      "valor": 300,
      "descricao": "Guardei do salario",
      "data": "2026-06-15",
      "despesa_id": 88
    }
  ]
}
```

## Projecao

```http
GET /api/saving-boxes/:id/projection?month=6&year=2026
```

Resposta:

```json
{
  "saldo_atual": 600,
  "valor_meta": 3000,
  "valor_restante": 2400,
  "meta_mensal": 300,
  "meses_estimados": 8,
  "mes_estimado_conclusao": 2,
  "ano_estimado_conclusao": 2027,
  "valor_guardado_mes_atual": 350,
  "diferenca_meta_mensal": 50,
  "status_meta_mensal": "acima_da_meta"
}
```

Status de meta mensal sugeridos:

```txt
sem_meta
abaixo_da_meta
na_meta
acima_da_meta
```

## Resumo geral

```http
GET /api/saving-boxes/summary?month=6&year=2026
```

Resposta:

```json
{
  "total_guardado": 1200,
  "total_metas": 8000,
  "total_meta_mensal": 900,
  "valor_guardado_mes_atual": 750,
  "total_caixinhas": 3,
  "caixinhas_ativas": 2
}
```

## Regras para despesas vinculadas a caixinhas

O back-end deve reconhecer quando uma despesa esta vinculada a uma caixinha.

Ao atualizar uma despesa vinculada:

- recalcular saldo da caixinha
- registrar historico quando valor mudar
- bloquear mudanca de categoria
- bloquear mudanca para `Parcelada`

Ao deletar uma despesa vinculada:

- tratar como retirada do aporte
- registrar historico
- recalcular saldo da caixinha
- respeitar regra atual de delete de despesa fixa

Ao criar despesa comum:

- bloquear categoria `Caixinhas` fora do fluxo de caixinha

## Erros

Manter padrao atual do Android:

```json
{
  "error": "mensagem"
}
```

Erros sugeridos:

```json
{
  "error": "Retire todo o dinheiro da caixinha antes de deletar"
}
```

```json
{
  "error": "Despesas da categoria Caixinhas so podem ser criadas pela tela de caixinha"
}
```

```json
{
  "error": "Despesa vinculada a caixinha nao pode mudar de categoria"
}
```

```json
{
  "error": "Aporte de caixinha nao pode ser parcelado"
}
```

## Decisoes finais

- Endpoints em ingles.
- JSON em portugues.
- Categoria `Caixinhas` visivel e contabilizada.
- Cada aporte e uma despesa diferente.
- Aporte pode ser `Unica` ou `Fixa`.
- Aporte nao pode ser `Parcelada`.
- Aporte fixo segue o padrao atual de despesas fixas.
- Editar valor da despesa atualiza saldo da caixinha.
- Reduzir valor da despesa equivale a retirada parcial.
- Deletar despesa vinculada equivale a retirada total.
- Deletar caixinha so e permitido com saldo zero.
- Ao deletar caixinha com saldo zero, apagar historico junto.
- Sem calculo de rendimento real.
