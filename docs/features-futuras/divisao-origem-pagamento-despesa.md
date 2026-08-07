# Divisão de origem de pagamento de uma despesa

## Objetivo

Permitir que uma única despesa seja paga usando mais de uma origem de renda: `Salário`, `Adiantamento` e `Renda Extra`.

Exemplo: uma fatura de R$ 1.200,00 pode ser quitada com R$ 1.000,00 do salário e R$ 200,00 da renda extra.

## Problema atual

Atualmente a despesa possui apenas uma `payment_source` (origem do pagamento). Isso obriga o usuário a atribuir todo o valor da despesa a uma única origem, mesmo quando o pagamento real foi dividido.

## Comportamento proposto

- A despesa continua tendo um único valor total.
- Uma despesa pode ter uma ou várias divisões de pagamento.
- Cada divisão informa a origem e o valor utilizado.
- A soma das divisões deve ser exatamente igual ao valor total da despesa.
- As origens válidas são `Salário`, `Adiantamento` e `Renda Extra`.
- Não deve haver duas divisões com a mesma origem na mesma despesa; nesse caso, os valores devem ser consolidados em uma só divisão.
- Despesas atuais continuam válidas: sua origem única deve ser interpretada como uma divisão equivalente a 100% do valor da despesa.

### Exemplo

```text
Despesa: Fatura do cartão
Valor total: R$ 1.200,00

Divisões:
- Salário: R$ 1.000,00
- Renda Extra: R$ 200,00

Total distribuído: R$ 1.200,00
```

## Experiência no aplicativo

Manter o preenchimento simples para o caso mais comum, de apenas uma origem. Ao escolher dividir o pagamento, o usuário poderá adicionar linhas de origem e valor.

```text
Origem do pagamento

Salário       R$ 1.000,00
Renda Extra   R$   200,00

+ Adicionar origem

Total distribuído: R$ 1.200,00 de R$ 1.200,00
```

O botão de salvar deve permanecer desabilitado, ou apresentar uma validação clara, enquanto o total distribuído for diferente do valor da despesa.

## Modelo de dados sugerido

Criar uma tabela relacionada a `expenses`.

```sql
CREATE TABLE expense_payment_splits (
    id BIGSERIAL PRIMARY KEY,
    expense_id BIGINT NOT NULL,
    payment_source TEXT NOT NULL,
    amount NUMERIC NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (expense_id, payment_source)
);
```

Regras a serem garantidas pela API:

- `amount` deve ser maior que zero.
- A origem deve pertencer à lista permitida.
- A soma de `expense_payment_splits.amount` deve ser igual a `expenses.amount`.
- A despesa e suas divisões devem ser criadas ou atualizadas na mesma transação.

`expenses.payment_source` pode ser mantido temporariamente para compatibilidade durante a migração. Depois que todos os consumidores da API utilizarem as divisões, avaliar sua remoção em uma migration futura.

## Contrato de API sugerido

Nos endpoints de criação e edição de despesa, acrescentar `payment_splits`.

```json
{
  "amount": 1200.00,
  "description": "Fatura do cartão",
  "payment_splits": [
    { "payment_source": "Salário", "amount": 1000.00 },
    { "payment_source": "Renda Extra", "amount": 200.00 }
  ]
}
```

Para compatibilidade, uma requisição que contenha somente `payment_source` deve ser convertida internamente em uma divisão com o valor integral da despesa. A API deve rejeitar requisições que enviem `payment_source` e `payment_splits` de forma conflitante.

As respostas de despesa devem retornar as divisões. Opcionalmente, `payment_source` pode continuar sendo retornado enquanto houver versões antigas do aplicativo em uso.

## Impacto nos relatórios

- O total da despesa é contabilizado apenas uma vez nos relatórios de gastos.
- Nos relatórios por origem de pagamento, cada origem recebe somente o valor de sua divisão.
- No exemplo, a despesa acrescenta R$ 1.200,00 ao total de gastos, R$ 1.000,00 ao uso de salário e R$ 200,00 ao uso de renda extra.
- Exportações PDF e XLSX devem exibir as origens e os valores divididos de forma legível.

## Despesas parceladas e fixas

A divisão deve pertencer a cada ocorrência de despesa, não somente à série. Isso permite que uma parcela ou mês específico seja pago de forma diferente dos demais.

Ao criar uma despesa parcelada ou fixa, a primeira versão pode aplicar a mesma divisão para todas as ocorrências geradas. A edição de uma ocorrência deve alterar somente aquela ocorrência, salvo se futuramente for implementada uma ação explícita para replicar a alteração para a série.

## Plano de implementação

1. Criar migration versionada para `expense_payment_splits`.
2. Criar modelo e rotinas de persistência no domínio `internal/expenses`.
3. Atualizar criação, edição e consulta de despesas, com validação transacional das divisões.
4. Preservar compatibilidade com clientes que enviam apenas `payment_source`.
5. Atualizar consultas de relatórios e geradores PDF/XLSX.
6. Adicionar testes para validação de soma, isolamento por usuário, compatibilidade e relatórios por origem.
7. Atualizar o aplicativo mobile com a interface de divisão de pagamento.

## Critérios de aceite

- É possível criar uma despesa de R$ 1.200,00 dividida entre R$ 1.000,00 de salário e R$ 200,00 de renda extra.
- Não é possível salvar uma divisão cuja soma seja diferente do valor da despesa.
- Uma despesa não é duplicada nos totais de gastos por ter mais de uma origem.
- Os totais por origem refletem corretamente os valores divididos.
- Despesas existentes com uma única origem continuam visíveis e editáveis.
- Somente o proprietário da despesa pode visualizar ou alterar suas divisões.
