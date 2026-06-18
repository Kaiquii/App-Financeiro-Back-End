# Planejamento - Exportação De Relatórios

Este documento descreve a funcionalidade planejada para exportar relatórios financeiros em CSV.

A ideia é permitir que o usuário baixe os dados financeiros dele em arquivo, para abrir no Excel, Google Sheets, LibreOffice ou guardar como histórico.

## Objetivo

Criar um endpoint único para exportação:

```http
GET /api/reports/export
```

O endpoint deve retornar um arquivo CSV, não JSON.

Formato inicial:

```txt
csv
```

Não haverá PDF no primeiro momento.

## Autenticação

A rota será protegida e deverá usar o token do usuário logado:

```txt
Authorization: Bearer TOKEN
```

Cada usuário só poderá exportar os próprios dados.

## Endpoint

```http
GET /api/reports/export?type=expenses&month=6&year=2026&format=csv
```

## Parâmetros

### Obrigatórios

```txt
type
```

Tipo de relatório que será exportado.

Valores aceitos:

```txt
expenses
incomes
categories
summary
month_comparison
installment_commitments
full_report
```

```txt
month
year
```

Mês e ano base do relatório.

```txt
format
```

Formato do arquivo.

Valor aceito no MVP:

```txt
csv
```

O `format` pode ser opcional com padrão `csv`, mas a recomendação é o front enviar explicitamente.

### Opcionais

Usados apenas em `month_comparison`:

```txt
compare_month
compare_year
```

Se enviados, definem o mês comparado manualmente.

Se não enviados, a API compara automaticamente com o mês anterior.

Usados apenas em `installment_commitments`:

```txt
months
include_current_month_as_paid
```

`months` define a quantidade de meses da linha do tempo.

`include_current_month_as_paid=true` considera o mês base como já pago e projeta a partir do próximo mês.

## Resposta Da API

A API deve retornar arquivo CSV com os headers:

```http
Content-Type: text/csv; charset=utf-8
Content-Disposition: attachment; filename="relatorio-despesas-2026-06.csv"
```

O nome do arquivo deve variar conforme o tipo:

```txt
relatorio-despesas-2026-06.csv
relatorio-receitas-2026-06.csv
relatorio-categorias-2026-06.csv
relatorio-resumo-2026-06.csv
relatorio-comparativo-2026-06.csv
relatorio-compromissos-parcelados-2026-06.csv
relatorio-completo-2026-06.csv
```

## Tipos De Exportação

## 1. Despesas

```http
GET /api/reports/export?type=expenses&month=6&year=2026&format=csv
```

Exporta todas as despesas do mês.

Colunas sugeridas:

```csv
Data,Descricao,Categoria,Fonte de Pagamento,Tipo,Parcela,Valor
2026-06-01,Mercado,Alimentacao,Salario,Unica,,120.50
2026-06-05,Notebook,Eletronicos,Salario,Parcelada,2/10,300.00
2026-06-10,Internet,Casa,Adiantamento,Fixa,,99.90
```

Regra da coluna `Parcela`:

- Se for parcelada, exibir `parcela_atual/total_parcelas`.
- Se não for parcelada, deixar vazio.

## 2. Receitas

```http
GET /api/reports/export?type=incomes&month=6&year=2026&format=csv
```

Exporta todas as receitas/rendas do mês.

Colunas sugeridas:

```csv
Mes,Ano,Fonte,Valor
6,2026,Salario,3000.00
6,2026,Renda Extra,500.00
```

## 3. Categorias

```http
GET /api/reports/export?type=categories&month=6&year=2026&format=csv
```

Exporta o resumo de despesas por categoria.

Colunas sugeridas:

```csv
Categoria,Valor,Percentual
Alimentacao,620.00,25.83
Transporte,260.00,10.83
Lazer,180.00,7.50
```

## 4. Resumo

```http
GET /api/reports/export?type=summary&month=6&year=2026&format=csv
```

Exporta o resumo financeiro mensal.

Colunas sugeridas:

```csv
Campo,Valor
Receitas,3500.00
Despesas,2400.00
Saldo,1100.00
Salario,3000.00
Adiantamento,0.00
Renda Extra,500.00
Gasto Salario,1800.00
Gasto Adiantamento,400.00
Gasto Renda Extra,200.00
Restante Salario,1200.00
Restante Adiantamento,-400.00
Restante Renda Extra,300.00
```

## 5. Comparativo Mensal

```http
GET /api/reports/export?type=month_comparison&month=6&year=2026&compare_month=1&compare_year=2026&format=csv
```

Exporta a comparação entre o mês base e outro mês.

Se `compare_month` e `compare_year` não forem enviados, compara com o mês anterior.

Colunas sugeridas:

```csv
Secao,Campo,Valor Atual,Valor Comparado,Diferenca,Percentual,Status
Resumo,Receitas,3500.00,3000.00,500.00,16.67,subiu
Resumo,Despesas,2400.00,2160.00,240.00,11.11,subiu
Resumo,Saldo,1100.00,840.00,260.00,30.95,melhorou
Categoria,Alimentacao,620.00,500.00,120.00,24.00,subiu
Fonte de Pagamento,Salario,800.00,600.00,200.00,33.33,subiu
Tipo de Despesa,Unica,900.00,700.00,200.00,28.57,subiu
Insight,Mensagem,Seu gasto total aumentou R$ 240.00 em relacao ao mes anterior.,,,,
```

## 6. Compromissos Parcelados

```http
GET /api/reports/export?type=installment_commitments&month=6&year=2026&months=12&include_current_month_as_paid=true&format=csv
```

Exporta o relatório de compromissos parcelados.

Deve reaproveitar a mesma regra do endpoint:

```http
GET /api/reports/installment-commitments
```

Colunas sugeridas:

```csv
Secao,Descricao,Mes,Ano,Valor,Parcela Atual,Total Parcelas,Categoria,Fonte
Resumo,Total Original,,,3000.00,,,,
Resumo,Total Pago,,,900.00,,,,
Resumo,Total Restante,,,2100.00,,,,
Compra,Notebook,6,2026,300.00,1,10,Eletronicos,Salario
Linha do Tempo,Notebook,7,2026,300.00,2,10,Eletronicos,Salario
Linha do Tempo,Notebook,8,2026,300.00,3,10,Eletronicos,Salario
```

## 7. Relatório Completo

```http
GET /api/reports/export?type=full_report&month=6&year=2026&format=csv
```

Exporta um CSV único com várias seções no mesmo arquivo.

Esse tipo deve juntar:

- resumo mensal
- receitas
- despesas
- categorias

No futuro, pode incluir:

- comparativo mensal
- compromissos parcelados

Colunas sugeridas:

```csv
Secao,Campo1,Campo2,Campo3,Campo4,Campo5,Campo6
Resumo,Receitas,3500.00,,,,,
Resumo,Despesas,2400.00,,,,,
Resumo,Saldo,1100.00,,,,,
Receita,Salario,3000.00,6,2026,,,
Receita,Renda Extra,500.00,6,2026,,,
Despesa,Mercado,Alimentacao,Salario,Unica,120.50,2026-06-01
Despesa,Notebook,Eletronicos,Salario,Parcelada,300.00,2026-06-05
Categoria,Alimentacao,620.00,25.83,,,,
```

## Validações

Erros devem seguir o padrão atual:

```json
{
  "error": "mensagem"
}
```

Validações sugeridas:

- `type` é obrigatório.
- `type` deve ser um dos tipos aceitos.
- `format` deve ser `csv`.
- `month` deve estar entre 1 e 12.
- `year` deve ser maior ou igual a 2000.
- `compare_month` e `compare_year` devem ser enviados juntos.
- `months`, quando enviado, deve estar entre 1 e 60.

Exemplos de erro:

```json
{
  "error": "Tipo de exportacao invalido"
}
```

```json
{
  "error": "Formato invalido. Use csv"
}
```

```json
{
  "error": "Mes e ano sao obrigatorios"
}
```

## Comportamento Sem Dados

Se não houver dados para o período, a API ainda deve retornar um CSV válido com cabeçalho.

Exemplo:

```csv
Data,Descricao,Categoria,Fonte de Pagamento,Tipo,Parcela,Valor
```

Isso evita erro no front e deixa claro que a exportação funcionou, mas não havia dados.

## Front-end Web

Fluxo sugerido:

1. Usuário abre Relatórios.
2. Clica em `Exportar`.
3. Escolhe tipo de relatório.
4. Escolhe mês e ano.
5. O front chama `/api/reports/export`.
6. O navegador baixa o arquivo CSV.

## Android

Fluxo sugerido:

1. Usuário abre Relatórios.
2. Clica em `Exportar`.
3. Escolhe tipo de relatório.
4. Escolhe mês e ano.
5. O app chama `/api/reports/export`.
6. O app salva o arquivo ou abre a tela de compartilhamento.

## Decisões Finais

- Um único endpoint: `/api/reports/export`.
- Formato inicial: `csv`.
- Não usar API externa.
- Não gerar PDF no MVP.
- Resposta será arquivo, não JSON.
- O arquivo deve ser gerado com dados do usuário autenticado.
- `full_report` deve consolidar os principais dados mensais em um único CSV.
