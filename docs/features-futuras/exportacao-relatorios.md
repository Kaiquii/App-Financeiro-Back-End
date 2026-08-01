# Exportação De Relatórios

Este documento registra o que já foi implementado na exportação de relatórios e planeja a evolução para PDF.

Guia prático de uso e teste: [`docs/guia-exportacao-relatorios.md`](../guia-exportacao-relatorios.md).

A ideia é permitir que o usuário baixe os dados financeiros dele em arquivo, para abrir no Excel, Google Sheets, LibreOffice ou guardar como histórico.

## Objetivo

Criar um endpoint único para exportação:

```http
GET /api/reports/export
```

O endpoint retorna um arquivo para download, não JSON.

Formatos disponíveis atualmente:

```txt
csv
xlsx
```

Formato planejado:

```txt
pdf
```

## Estado Atual Da Implementação

O backend já oferece os sete tipos de exportação em CSV e XLSX:

- despesas;
- receitas;
- categorias;
- resumo mensal;
- comparativo mensal;
- compromissos parcelados;
- relatório completo.

A implementação CSV inclui `month_comparison`, `installment_commitments` e `full_report`, além dos quatro relatórios básicos.

O XLSX foi adicionado no mesmo endpoint sem remover ou alterar a disponibilidade do CSV. O PDF continua planejado.

Para abrir corretamente em instalações brasileiras do Microsoft Excel, o CSV deve usar:

- ponto e vírgula (`;`) como separador de colunas;
- vírgula como separador decimal;
- codificação UTF-8 com BOM.

Exemplo:

```csv
Data;Descricao;Categoria;Valor
2026-06-18;Restaurante;Alimentacao;120,00
```

Os exemplos de estrutura apresentados abaixo devem ser interpretados seguindo esse padrão regional na implementação.

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

Valores aceitos atualmente:

```txt
csv
xlsx
```

Valor planejado:

```txt
pdf
```

O `format` pode ser opcional com padrão `csv`, mas a recomendação é o front enviar explicitamente.

### Opcionais

Usados em `month_comparison` e `full_report`:

```txt
compare_month
compare_year
```

Se enviados, definem o mês comparado manualmente.

Se não enviados, a API compara automaticamente com o mês anterior.

Usados em `installment_commitments` e `full_report`:

```txt
months
include_current_month_as_paid
```

`months` define a quantidade de meses da linha do tempo.

`include_current_month_as_paid=true` considera o mês base como já pago e projeta a partir do próximo mês.

No `full_report`, quando os parâmetros opcionais não forem enviados, o comparativo deve usar o mês anterior e os compromissos devem usar os mesmos valores padrão definidos pelo endpoint individual.

## Resposta Da API

A resposta atual para CSV usa os headers:

```http
Content-Type: text/csv; charset=utf-8
Content-Disposition: attachment; filename="relatorio-despesas-2026-06.csv"
```

A resposta atual para XLSX usa:

```http
Content-Type: application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
Content-Disposition: attachment; filename="relatorio-despesas-2026-06.xlsx"
```

A resposta planejada para PDF usará:

```http
Content-Type: application/pdf
Content-Disposition: attachment; filename="relatorio-despesas-2026-06.pdf"
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
Data;Descricao;Categoria;Fonte de Pagamento;Tipo;Parcela;Valor;Observacoes
2026-06-01;Mercado;Alimentacao;Salario;Unica;;120,50;
2026-06-05;Notebook;Eletronicos;Salario;Parcelada;2 de 10;300,00;Compra parcelada no cartao
2026-06-24;Barzinho;Lazer;Salario;Unica;;160,00;R$ 80 meu e R$ 80 da minha namorada.
```

Regra da coluna `Parcela`:

- Se for parcelada, exibir `parcela_atual de total_parcelas`, por exemplo `3 de 5`.
- Se não for parcelada, deixar vazio.

O formato com barra, como `3/5`, não deve ser usado no CSV porque o Excel pode interpretá-lo automaticamente como uma data.

Regra da coluna `Observacoes`:

- Deve usar o campo `notes` da despesa.
- Se a despesa não tiver observação, deixar vazio.
- Deve ser a última coluna, pois pode conter textos longos.
- Como pode ter vírgula ou quebra de linha, o CSV deve escapar o campo corretamente.

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

Exporta um CSV único com todas as seções financeiras disponíveis no escopo desta funcionalidade.

Esse tipo deve juntar:

- resumo mensal
- receitas
- despesas
- categorias
- comparativo mensal
- compromissos parcelados

O `full_report` só será considerado completo quando consolidar as seis seções acima. Comparativo mensal e compromissos parcelados não ficam para uma evolução futura desta funcionalidade.

Organização do arquivo:

```csv
RESUMO MENSAL
Campo;Valor
Receitas;3500,00
Despesas;2400,00
Saldo;1100,00

RECEITAS
Fonte;Valor;Mes;Ano
Salario;3000,00;6;2026

DESPESAS
Data;Descricao;Categoria;Fonte de Pagamento;Tipo;Parcela;Valor;Observacoes
2026-06-05;Notebook;Eletronicos;Salario;Parcelada;2 de 10;300,00;Compra parcelada no cartao

COMPARATIVO - RESUMO
Campo;Valor Atual;Valor Comparado;Diferenca;Percentual;Status
Despesas;2400,00;2160,00;240,00;11,11;subiu
```

Cada bloco deve ter um título, um cabeçalho específico e uma linha vazia antes do próximo bloco. Não devem existir nomes genéricos como `Campo1`, `Campo2` ou `Campo3`.

O relatório completo deve separar os seguintes blocos:

- resumo mensal;
- receitas;
- despesas;
- resumo por categoria;
- comparativo do resumo;
- comparativo por categoria;
- comparativo por fonte de pagamento;
- comparativo por tipo de despesa;
- insights do comparativo;
- resumo dos compromissos parcelados;
- compras parceladas;
- linha do tempo dos compromissos parcelados.

Na seção `DESPESAS`, o último campo deve conter as observações da despesa (`notes`).

As seções de comparativo e compromissos devem reutilizar as mesmas regras dos tipos `month_comparison` e `installment_commitments`, evitando divergência entre o relatório individual e o relatório completo.

Internamente, todas as linhas podem ser completadas com campos vazios até a largura da seção mais extensa. Isso mantém o CSV consistente sem mostrar títulos genéricos ao usuário.

## Validações

Erros devem seguir o padrão atual:

```json
{
  "error": "mensagem"
}
```

Validações:

- `type` é obrigatório.
- `type` deve ser um dos tipos aceitos.
- atualmente, `format` aceita `csv` e `xlsx`;
- após a próxima etapa, `format` também aceitará `pdf`;
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
  "error": "Formato invalido. Use csv ou xlsx"
}
```

Quando o PDF estiver implementado, a mensagem deverá listar os três formatos aceitos.

```json
{
  "error": "Mes e ano sao obrigatorios"
}
```

## Comportamento Sem Dados

Se não houver dados para o período, a API ainda deve retornar um arquivo válido para o formato solicitado. No CSV, o arquivo deve conter o cabeçalho.

Exemplo:

```csv
Data;Descricao;Categoria;Fonte de Pagamento;Tipo;Parcela;Valor;Observacoes
```

Isso evita erro no front e deixa claro que a exportação funcionou, mas não havia dados.

## Front-end Web

Fluxo sugerido:

1. Usuário abre Relatórios.
2. Clica em `Exportar`.
3. Escolhe tipo de relatório.
4. Escolhe mês e ano.
5. Escolhe o formato disponível.
6. O front chama `/api/reports/export`.
7. O navegador baixa o arquivo.

## Android

Fluxo sugerido:

1. Usuário abre Relatórios.
2. Clica em `Exportar`.
3. Escolhe tipo de relatório.
4. Escolhe mês e ano.
5. Escolhe o formato disponível.
6. O app chama `/api/reports/export`.
7. O app salva o arquivo ou abre a tela de compartilhamento.

## Decisões Atuais

- Um único endpoint: `/api/reports/export`.
- CSV permanece como formato padrão quando `format` não for enviado.
- XLSX é selecionado pelo parâmetro `format`; o PDF seguirá o mesmo padrão quando for implementado.
- Não usar API externa.
- Resposta será arquivo, não JSON.
- O arquivo deve ser gerado com dados do usuário autenticado.
- Os sete tipos de relatório devem estar disponíveis em cada formato implementado.
- Cada formato deve reaproveitar as mesmas consultas e regras financeiras, evitando diferenças nos valores exportados.

## O Que Já Foi Implementado Em CSV

- endpoint protegido `GET /api/reports/export`;
- sete valores aceitos em `type`;
- filtro por mês, ano e usuário autenticado;
- comparativo automático com o mês anterior ou período personalizado;
- configuração da linha do tempo de compromissos parcelados;
- nome de arquivo e headers HTTP para download;
- UTF-8 com BOM;
- ponto e vírgula como separador de colunas;
- vírgula como separador decimal;
- observações na última coluna das despesas;
- parcelas apresentadas como `3 de 5`, evitando conversão automática em data pelo Excel;
- escape de vírgulas, aspas e quebras de linha;
- neutralização de campos textuais que poderiam ser interpretados como fórmulas;
- arquivos sem dados com cabeçalho válido;
- `full_report` dividido em blocos com títulos e cabeçalhos próprios;
- testes automatizados dos sete tipos, validações, caracteres especiais, arquivos vazios e autenticação.

Itens ainda externos ao backend de exportação:

- integrar seleção e download no front-end web;
- integrar salvamento e compartilhamento no Android;
- validar a experiência final em produção com usuários autenticados.

## Implementação - XLSX

### Objetivo

Adicionar `format=xlsx` para gerar uma planilha nativa do Excel. Esse será o formato recomendado para análise detalhada e para o `full_report`.

Exemplo:

```http
GET /api/reports/export?type=full_report&month=7&year=2026&format=xlsx
```

### Organização Do Relatório Completo

O `full_report` em XLSX deve separar o conteúdo em abas reais:

- `Resumo`;
- `Receitas`;
- `Despesas`;
- `Categorias`;
- `Comparativo`;
- `Parcelamentos`;
- `Insights`.

### Formatação Implementada

- títulos e cabeçalhos destacados;
- valores numéricos armazenados como números e formatados como moeda;
- percentuais armazenados como números e formatados como percentual;
- datas armazenadas como datas;
- largura de colunas adequada ao conteúdo, com limite para textos longos;
- quebra de linha nas observações;
- cabeçalho congelado nas tabelas extensas;
- filtros nas abas de receitas e despesas;
- saldo positivo e negativo com estilos visuais distintos;
- nomes de abas e colunas em português;
- compatibilidade com Microsoft Excel, Google Sheets e LibreOffice.

### Critério De Pronto Do XLSX

- `format=xlsx` funcionar para os sete tipos de relatório;
- `full_report` possuir todas as abas planejadas;
- valores do XLSX serem iguais aos valores do CSV para os mesmos parâmetros;
- arquivo abrir sem aviso de corrupção;
- filtros, datas, moedas e percentuais funcionarem como tipos nativos;
- arquivos sem dados continuarem válidos, com aba e cabeçalho;
- testes cobrirem conteúdo, nomes das abas, formatos e isolamento por usuário;
- download funcionar no web e salvamento ou compartilhamento funcionar no Android.

Os critérios de backend estão implementados e cobertos por testes automatizados. A integração da experiência de download permanece sob responsabilidade dos clientes web e Android.

## Planejamento - PDF

### Objetivo

Adicionar `format=pdf` para gerar relatórios voltados à leitura, apresentação, impressão e compartilhamento. PDF não substitui CSV ou XLSX para análise dos dados.

Exemplo:

```http
GET /api/reports/export?type=summary&month=7&year=2026&format=pdf
```

### Organização Planejada

- título do relatório;
- nome do período exportado;
- data de geração;
- resumo financeiro em destaque;
- tabelas com cabeçalhos legíveis;
- seções separadas no relatório completo;
- quebra automática de páginas;
- repetição do cabeçalho das tabelas em novas páginas;
- orientação paisagem para tabelas largas;
- observações longas com quebra de linha;
- rodapé com número da página;
- suporte completo a acentos e caracteres em português.

### Critério De Pronto Do PDF

- `format=pdf` funcionar para os sete tipos de relatório;
- valores do PDF serem iguais aos valores do CSV e XLSX para os mesmos parâmetros;
- conteúdo não ultrapassar margens nem ficar cortado;
- tabelas extensas continuarem corretamente em novas páginas;
- arquivos sem dados apresentarem uma mensagem clara;
- relatório completo manter todas as seções identificadas;
- testes verificarem geração, páginas e conteúdo essencial;
- arquivo abrir nos leitores de PDF do navegador e Android.

## Ordem Planejada De Implementação

1. Manter e integrar a exportação CSV já implementada.
2. Implementar XLSX com abas e formatação nativa. Concluído.
3. Integrar XLSX no web e Android.
4. Implementar PDF começando pelo resumo e relatórios menores.
5. Implementar o relatório completo em PDF com paginação.
6. Integrar PDF no web e Android.
