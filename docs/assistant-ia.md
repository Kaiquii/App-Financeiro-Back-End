# Assistente IA

## Regra principal

O assistente nao deve depender de padroes fixos de texto para entender perguntas do usuario.

Exemplos do que evitar:

- Se a mensagem contem "quanto gastei", fazer resumo mensal direto.
- Se a mensagem contem "categoria", tentar cortar a frase manualmente.
- Criar uma lista crescente de frases esperadas para cada pergunta.

Esse caminho deixa o assistente engessado e contradiz a proposta do produto.

## Como deve funcionar

O Gemini deve ser usado para interpretar a mensagem livre do usuario.

O fluxo correto e:

1. O usuario envia uma mensagem natural.
2. O back-end envia essa mensagem para o Gemini interpretar.
3. O Gemini devolve uma intencao estruturada, por exemplo:
   - resumo mensal
   - gastos por categoria
   - cadastro de despesa
   - duvida sobre o app
   - conversa comum
4. O Go executa a acao com seguranca usando o usuario autenticado pelo JWT.
5. O Go monta uma resposta curta e clara usando o resultado consultado.

Para economizar cota, o fluxo financeiro deve usar apenas uma chamada ao Gemini sempre que possivel: a chamada de interpretacao da intencao. Depois disso, o Go executa a acao e responde com base nos dados retornados.

Quando o Gemini retornar limite de cota, o back-end deve respeitar o tempo de retry informado pela API e evitar novas chamadas ate esse periodo acabar. Durante esse cooldown, a API deve responder com `error_code=gemini_quota_exceeded` e `retry_after_seconds`.

O back-end pode usar um provedor secundario, como Groq, quando o Gemini atingir limite ou falhar. O fallback deve preservar o mesmo contrato: interpretar a mensagem em JSON estruturado e deixar o Go executar a acao com seguranca.

## Responsabilidades

### Gemini

- Entender a intencao da mensagem.
- Extrair entidades como mes, ano, categoria, valor, fonte de pagamento e descricao.
- Gerar respostas finais apenas quando for conversa comum, ajuda sobre o app ou quando uma resposta deterministica do Go nao for suficiente.

### Go

- Autenticar o usuario.
- Buscar dados no banco.
- Criar, atualizar ou deletar dados somente com confirmacao quando necessario.
- Garantir que o usuario so acesse os proprios dados.
- Salvar conversas e mensagens.

## Regra para novas features

Antes de adicionar qualquer condicao baseada em frase, perguntar:

"Isso pode ser interpretado pelo Gemini em vez de virar mais um padrao no Go?"

Se a resposta for sim, usar o Gemini para interpretar e manter o Go apenas como executor seguro.
