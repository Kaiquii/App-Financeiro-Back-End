# Melhoria tecnica: rate limit no login

## Contexto

Hoje a rota de login recebe tentativas de autenticacao normalmente e valida e-mail/senha usando `bcrypt`.

Isso e correto para seguranca de senha, mas o `bcrypt` tambem e uma operacao propositalmente mais pesada. Se alguem tentar muitas senhas em sequencia, a API pode gastar processamento desnecessario e ficar mais exposta a tentativa de forca bruta.

## Objetivo

Adicionar rate limit apenas na rota:

```txt
POST /api/auth/login
```

Essa melhoria deve ficar limitada ao login porque, normalmente, essa e a rota mais visada para ataques de autenticacao, principalmente tentativa de forca bruta ou muitas tentativas de senha errada.

O objetivo e reduzir tentativas repetidas de senha errada sem bloquear a conta do usuario de forma permanente e sem adicionar limitacoes desnecessarias nas demais areas do app.

## Escopo

Essa melhoria deve ser implementada somente na rota `POST /api/auth/login`.

Nenhuma outra rota deve receber rate limit nesta melhoria.

## Regra sugerida

```txt
5 falhas de login em 10 minutos por IP + e-mail
bloqueio temporario de 5 minutos
```

Ou seja: se alguem tentar 5 logins com senha errada em menos de 10 minutos, usando o mesmo IP para o mesmo e-mail, o backend deve bloquear novas tentativas desse par IP + e-mail por 5 minutos.

Depois de exceder o limite, novas tentativas devem retornar:

```txt
429 Too Many Requests
```

Resposta sugerida:

```json
{
  "error": "Muitas tentativas de login. Tente novamente em alguns minutos."
}
```

## Fluxo esperado

1. Usuario envia `POST /api/auth/login`.
2. API verifica se aquele IP + e-mail esta temporariamente bloqueado.
3. Se estiver bloqueado, retorna `429` sem validar senha.
4. Se nao estiver bloqueado, continua o login normal.
5. Se a senha estiver errada, registra uma falha.
6. Se a senha estiver correta, limpa o contador de falhas daquele IP + e-mail.

Exemplo:

```txt
1a senha errada -> erro normal
2a senha errada -> erro normal
3a senha errada -> erro normal
4a senha errada -> erro normal
5a senha errada -> erro normal
6a tentativa dentro da janela -> 429 Too Many Requests
```

## Importante

Esse bloqueio nao deve alterar `access_blocked` no banco.

Ele nao e um bloqueio administrativo da conta. E apenas uma pausa temporaria contra tentativas repetidas de login.

## Implementacao inicial recomendada

Como o projeto roda de forma simples em producao, a primeira versao pode usar controle em memoria dentro da API.

Esse modelo funciona bem enquanto existir apenas um container da API.

Se no futuro a API rodar com multiplos containers, o contador deve ir para um armazenamento compartilhado, como Redis, para que todos os containers enxerguem o mesmo limite.

## Criterio de pronto

Essa melhoria pode ser considerada concluida quando:

- `POST /api/auth/login` contar falhas por IP + e-mail;
- 5 falhas em 10 minutos bloquearem novas tentativas por 5 minutos;
- tentativas bloqueadas retornarem `429 Too Many Requests`;
- login bem-sucedido limpar o contador de falhas;
- o bloqueio nao alterar `access_blocked`;
- houver teste cobrindo falhas repetidas, bloqueio temporario e limpeza apos login correto.
