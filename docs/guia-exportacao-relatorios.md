# Guia De Uso E Teste Da Exportação De Relatórios

Este guia mostra como autenticar, gerar, baixar e validar os relatórios exportados pelo backend.

## Estado Atual

O backend aceita atualmente:

```txt
format=csv
```

Os formatos `xlsx` e `pdf` estão planejados, mas ainda não estão implementados.

Endpoint único:

```http
GET /api/reports/export
```

A rota é protegida. Todas as requisições precisam enviar um JWT válido:

```http
Authorization: Bearer TOKEN
```

## 1. Preparar O Postman

### Criar Um Environment

No Postman:

1. Abra `Environments`.
2. Crie um environment chamado `SobraAi Local` ou `SobraAi Produção`.
3. Adicione as variáveis abaixo.

| Variável | Exemplo local | Observação |
| --- | --- | --- |
| `base_url` | `http://localhost:8080` | Não coloque `/` no final. |
| `email` | `usuario@exemplo.com` | Usuário já cadastrado. |
| `password` | `sua-senha` | Use apenas no seu environment privado. |
| `token` | vazio | Será preenchido após o login. |

Selecione esse environment no canto superior direito do Postman.

Em produção, altere somente `base_url` para o endereço da API publicada.

## 2. Fazer Login E Salvar O Token Automaticamente

Crie uma requisição chamada `Login`.

### Método E URL

```http
POST {{base_url}}/api/auth/login
```

### Body

Escolha `Body`, `raw` e `JSON`:

```json
{
  "email": "{{email}}",
  "password": "{{password}}"
}
```

### Script Para Salvar O Token

Na aba `Scripts`, adicione em `Post-response`:

```javascript
const response = pm.response.json();

if (response.token) {
  pm.environment.set("token", response.token);
}
```

Clique em `Send`.

Resposta esperada:

```json
{
  "message": "Login realizado com sucesso!",
  "token": "JWT",
  "user": {}
}
```

Confira no environment se a variável `token` foi preenchida.

## 3. Criar A Requisição De Exportação

Crie uma requisição chamada `Exportar Relatório`.

### Método E URL

```http
GET {{base_url}}/api/reports/export
```

### Autorização

Na aba `Authorization`:

1. Selecione `Bearer Token`.
2. Informe:

```txt
{{token}}
```

Não escreva a palavra `Bearer` no campo. O Postman adiciona automaticamente.

### Parâmetros Básicos

Na aba `Params`, adicione:

| Key | Value | Obrigatório |
| --- | --- | --- |
| `type` | `expenses` | Sim |
| `month` | `7` | Sim |
| `year` | `2026` | Sim |
| `format` | `csv` | Recomendado |

A URL montada será:

```http
{{base_url}}/api/reports/export?type=expenses&month=7&year=2026&format=csv
```

## 4. Baixar O Arquivo No Postman

O endpoint retorna um arquivo, não JSON.

Forma recomendada:

1. Clique na seta ao lado de `Send`.
2. Escolha `Send and Download`.
3. Selecione a pasta e salve o arquivo com extensão `.csv`.

Alternativa:

1. Clique em `Send` normalmente.
2. Na resposta, abra o menu de opções.
3. Escolha `Save Response` e depois `Save to a file`.

Não copie o conteúdo exibido em `Raw` para criar o arquivo manualmente. Use o download da resposta para preservar codificação e BOM.

## 5. Tipos De Relatório

### Despesas

```http
GET {{base_url}}/api/reports/export?type=expenses&month=7&year=2026&format=csv
```

### Receitas

```http
GET {{base_url}}/api/reports/export?type=incomes&month=7&year=2026&format=csv
```

### Resumo Por Categoria

```http
GET {{base_url}}/api/reports/export?type=categories&month=7&year=2026&format=csv
```

### Resumo Mensal

```http
GET {{base_url}}/api/reports/export?type=summary&month=7&year=2026&format=csv
```

### Comparativo Com O Mês Anterior

```http
GET {{base_url}}/api/reports/export?type=month_comparison&month=7&year=2026&format=csv
```

Se `compare_month` e `compare_year` não forem enviados, o backend usa automaticamente o mês anterior.

### Comparativo Personalizado

```http
GET {{base_url}}/api/reports/export?type=month_comparison&month=7&year=2026&compare_month=1&compare_year=2026&format=csv
```

`compare_month` e `compare_year` devem sempre ser enviados juntos.

### Compromissos Parcelados

```http
GET {{base_url}}/api/reports/export?type=installment_commitments&month=7&year=2026&months=12&include_current_month_as_paid=false&format=csv
```

Parâmetros adicionais:

| Parâmetro | Descrição | Padrão |
| --- | --- | --- |
| `months` | Quantidade de meses projetados, entre 1 e 60. | `12` |
| `include_current_month_as_paid` | Define se o mês base já deve contar como pago. | `false` |

### Relatório Completo

```http
GET {{base_url}}/api/reports/export?type=full_report&month=7&year=2026&format=csv
```

Com todas as opções:

```http
GET {{base_url}}/api/reports/export?type=full_report&month=7&year=2026&compare_month=6&compare_year=2026&months=12&include_current_month_as_paid=false&format=csv
```

O CSV completo contém blocos separados para:

- resumo mensal;
- receitas;
- despesas;
- resumo por categoria;
- comparativo do resumo;
- comparativo por categoria;
- comparativo por fonte de pagamento;
- comparativo por tipo de despesa;
- insights;
- resumo dos compromissos parcelados;
- compras parceladas;
- linha do tempo dos parcelamentos.

## 6. Conferir A Resposta

Status esperado:

```http
200 OK
```

Headers esperados:

```http
Content-Type: text/csv; charset=utf-8
Content-Disposition: attachment; filename="relatorio-despesas-2026-07.csv"
X-Content-Type-Options: nosniff
```

O nome muda conforme o tipo solicitado.

## 7. Validar O Arquivo No Excel

Abra o arquivo baixado diretamente no Excel e confira:

- cada campo aparece em uma coluna diferente;
- a acentuação está correta;
- valores aparecem com vírgula decimal;
- despesas parceladas aparecem como `3 de 5`, não como data;
- `Observacoes` é a última coluna das despesas;
- observações com textos longos continuam na mesma linha lógica;
- o relatório completo possui títulos e cabeçalhos próprios em cada bloco;
- não aparecem colunas chamadas `Campo1`, `Campo2` ou semelhantes.

O CSV usa:

```txt
separador de colunas: ;
separador decimal: ,
codificação: UTF-8 com BOM
```

## 8. Teste Rápido Pelo PowerShell

Essa opção é útil para repetir testes sem abrir o Postman.

### Fazer Login

```powershell
$loginBody = @{
    email    = "usuario@exemplo.com"
    password = "sua-senha"
} | ConvertTo-Json

$login = Invoke-RestMethod `
    -Method Post `
    -Uri "http://localhost:8080/api/auth/login" `
    -ContentType "application/json" `
    -Body $loginBody

$token = $login.token
```

### Baixar O Relatório Completo

```powershell
Invoke-WebRequest `
    -Uri "http://localhost:8080/api/reports/export?type=full_report&month=7&year=2026&format=csv" `
    -Headers @{ Authorization = "Bearer $token" } `
    -OutFile ".\relatorio-completo-2026-07.csv"
```

### Abrir No Excel

```powershell
Invoke-Item ".\relatorio-completo-2026-07.csv"
```

## 9. Erros Comuns

### `401 Unauthorized`

Possíveis causas:

- token não enviado;
- variável `token` vazia;
- token expirado;
- tipo de autorização diferente de `Bearer Token`.

Faça login novamente e confira o environment selecionado.

### `400 Tipo de exportacao invalido`

Use um dos valores:

```txt
expenses
incomes
categories
summary
month_comparison
installment_commitments
full_report
```

### `400 Formato invalido. Use csv`

Atualmente apenas `format=csv` está implementado. XLSX e PDF ainda estão planejados.

### `400 Mes e ano sao obrigatorios e devem ser validos`

Confira:

- `month` entre 1 e 12;
- `year` maior ou igual a 2000;
- parâmetros marcados como ativos na aba `Params`.

### Arquivo Abre Em Uma Única Coluna

Confirme que a API publicada contém a versão que usa `;` como separador. Baixe um arquivo novo depois de atualizar o backend.

### Parcela Aparece Como Data

Confirme que a API publicada contém a versão que exporta parcelas como `3 de 5`. Arquivos antigos com `3/5` podem ser convertidos automaticamente pelo Excel.

### Postman Mostra Texto Em Vez De Baixar

Isso é esperado ao usar apenas `Send`. Use `Send and Download` ou `Save Response` para salvar o arquivo.

## 10. Checklist Antes De Publicar

- executar `go test ./...`;
- reconstruir a imagem Docker;
- recriar somente o serviço da API;
- fazer login com um usuário de teste;
- baixar `expenses`;
- baixar `summary`;
- baixar `month_comparison`;
- baixar `installment_commitments`;
- baixar `full_report`;
- abrir os arquivos no Excel;
- testar um mês sem dados;
- confirmar que dados de outro usuário não aparecem;
- validar nome, extensão e headers do arquivo.

