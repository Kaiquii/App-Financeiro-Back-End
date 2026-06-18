# Planejamento - Tags Nas Despesas

Este documento descreve a funcionalidade planejada de tags nas despesas.

A ideia é permitir que o usuário organize despesas por contexto, sem substituir as categorias.

## Objetivo

Hoje a categoria responde:

```txt
Que tipo de gasto foi?
```

Exemplo:

```txt
Alimentação
Transporte
Saúde
Lazer
```

A tag responde:

```txt
Em qual contexto esse gasto aconteceu?
```

Exemplo:

```txt
Viagem
Família
Trabalho
Casa
Faculdade
Férias
```

Exemplo de uso:

```txt
Despesa: Restaurante
Categoria: Alimentação
Tags: Viagem, Família
```

Outro exemplo:

```txt
Despesa: Uber
Categoria: Transporte
Tags: Trabalho
```

## Regra Principal

Tags são opcionais.

Se o usuário não quiser usar tags, o cadastro de despesa continua funcionando normalmente.

A categoria continua sendo o campo principal da despesa. Tags são apenas marcadores extras.

## Limite Por Despesa

Cada despesa pode ter no máximo 3 tags.

Regras:

- 0 tags: permitido
- 1 tag: permitido
- 2 tags: permitido
- 3 tags: permitido
- 4 ou mais tags: bloquear

Erro sugerido:

```json
{
  "error": "Uma despesa pode ter no máximo 3 tags"
}
```

Esse limite deve existir no front-end e no back-end.

## Experiência No Front-end

Na tela de criar ou editar despesa, haverá uma área opcional de tags.

Sugestão visual:

```txt
[Selecionar tags ▼]  [+ Criar tag]
```

### Selecionar Tags

O botão/dropdown mostra as tags existentes do usuário:

```txt
Viagem
Família
Trabalho
Casa
Faculdade
```

O usuário pode selecionar uma ou mais tags, até o limite de 3.

As tags selecionadas aparecem como chips:

```txt
Viagem  x
Família x
```

### Criar Tag

Ao clicar em `+ Criar tag`, o front abre um modal ou campo simples:

```txt
Nome da tag: ______
Salvar
```

Depois de salvar:

1. Front chama a API para criar a tag.
2. API cria a tag para o usuário logado.
3. A nova tag aparece na lista.
4. A nova tag pode ficar selecionada automaticamente na despesa.

## Modelo De Dados Sugerido

Como uma despesa pode ter várias tags, o modelo ideal usa duas tabelas novas.

### tags

```txt
id
user_id
name
created_at
updated_at
```

### expense_tags

```txt
expense_id
tag_id
```

Essa tabela faz o vínculo entre despesas e tags.

## Regras De Segurança

- Tag pertence ao usuário logado.
- Usuário não pode ver tags de outro usuário.
- Usuário não pode usar tag de outro usuário em uma despesa.
- Usuário não pode editar tag de outro usuário.
- Usuário não pode deletar tag de outro usuário.

## Regras De Nome

Regras sugeridas:

- Nome da tag é obrigatório.
- Nome deve ter no máximo 30 caracteres.
- Não permitir tag vazia.
- Não permitir tag duplicada para o mesmo usuário.
- Comparação de duplicidade deve ignorar diferença entre maiúsculas e minúsculas.

Exemplo:

```txt
Viagem
viagem
VIAGEM
```

Devem ser consideradas a mesma tag para o mesmo usuário.

## Endpoints Planejados

### Listar tags

```http
GET /api/tags
```

Retorna as tags do usuário logado.

Resposta sugerida:

```json
{
  "tags": [
    {
      "id": 1,
      "name": "Viagem"
    },
    {
      "id": 2,
      "name": "Família"
    }
  ]
}
```

### Criar tag

```http
POST /api/tags
```

Body:

```json
{
  "name": "Trabalho"
}
```

Resposta sugerida:

```json
{
  "message": "Tag criada com sucesso",
  "tag": {
    "id": 3,
    "name": "Trabalho"
  }
}
```

### Atualizar tag

```http
PATCH /api/tags/:id
```

Body:

```json
{
  "name": "Viagem 2026"
}
```

### Deletar tag

```http
DELETE /api/tags/:id
```

Regra:

Deletar uma tag não deleta despesas.

Ao deletar uma tag, o back-end remove apenas os vínculos dela com as despesas.

## Alterações Nas Despesas

### Criar despesa com tags

O endpoint atual de criar despesa passará a aceitar `tag_ids`.

```http
POST /api/expenses/
```

Body sugerido:

```json
{
  "amount": 120,
  "description": "Restaurante",
  "category_id": 1,
  "payment_source": "Salário",
  "date": "2026-06-18",
  "type": "Única",
  "tag_ids": [1, 2]
}
```

O back-end deve:

1. Criar a despesa normalmente.
2. Validar se as tags pertencem ao usuário.
3. Validar se existem no máximo 3 tags.
4. Criar os vínculos entre despesa e tags.

### Editar tags da despesa

O endpoint atual de atualizar despesa passará a aceitar `tag_ids`.

```http
PATCH /api/expenses/:id
```

Body sugerido:

```json
{
  "tag_ids": [1, 4]
}
```

Regra:

Quando `tag_ids` for enviado, a API substitui as tags antigas da despesa pela nova lista.

### Listar despesas com tags

As respostas de despesas devem trazer as tags vinculadas.

Exemplo:

```json
{
  "id": 10,
  "amount": 120,
  "description": "Restaurante",
  "category_id": 1,
  "payment_source": "Salário",
  "date": "2026-06-18",
  "type": "Única",
  "tags": [
    {
      "id": 1,
      "name": "Viagem"
    },
    {
      "id": 2,
      "name": "Família"
    }
  ]
}
```

### Filtrar despesas por tag

O endpoint de listagem de despesas pode aceitar `tag_id`.

```http
GET /api/expenses?month=6&year=2026&tag_id=1
```

Retorna despesas do mês que possuem a tag informada.

## Relatório Por Tags

Novo endpoint planejado:

```http
GET /api/reports/tags?month=6&year=2026
```

Resposta sugerida:

```json
{
  "tags": [
    {
      "id": 1,
      "name": "Viagem",
      "total": 1450,
      "quantidade": 6
    },
    {
      "id": 2,
      "name": "Família",
      "total": 620,
      "quantidade": 3
    }
  ]
}
```

Regra importante:

Se uma despesa tiver duas tags, ela entra no total das duas tags.

Exemplo:

```txt
Despesa: Restaurante
Valor: R$ 120
Tags: Viagem, Família
```

Conta:

```txt
Viagem +120
Família +120
```

Isso é esperado, porque tag é marcador de contexto, não divisão contábil.

## Exportação CSV

Quando a funcionalidade de exportação de relatórios existir, as tags podem aparecer no CSV de despesas.

Exemplo:

```csv
Data,Descricao,Categoria,Fonte de Pagamento,Tipo,Tags,Valor
2026-06-18,Restaurante,Alimentação,Salário,Única,"Viagem, Família",120.00
```

Também pode existir exportação do relatório por tags:

```csv
Tag,Total,Quantidade
Viagem,1450.00,6
Família,620.00,3
```

## Comportamento Ao Deletar

### Deletar despesa

Quando uma despesa for deletada, os vínculos dela com tags também devem ser removidos.

As tags continuam existindo.

### Deletar tag

Quando uma tag for deletada:

- remove a tag
- remove os vínculos com despesas
- não remove nenhuma despesa

## Decisões Finais

- Tags são opcionais.
- Categoria continua sendo o campo principal.
- Uma despesa pode ter até 3 tags.
- Tags pertencem ao usuário logado.
- Não permitir tags duplicadas para o mesmo usuário.
- Front terá dropdown de tags existentes.
- Front terá botão `+ Criar tag`.
- Deletar tag não deleta despesa.
- Deletar despesa remove apenas os vínculos daquela despesa com tags.
- Relatório por tags deve somar despesas por marcador de contexto.
