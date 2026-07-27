# Melhoria de seguranca: protecao anti-bot no envio de codigos por e-mail

## Contexto atual

Os endpoints publicos abaixo enviam um codigo por e-mail:

```http
POST /api/auth/request-register-code
POST /api/auth/forgot-password
```

O primeiro confirma o e-mail antes de criar uma conta. O segundo inicia a redefinicao de senha de uma conta existente.

Hoje o backend ja aplica limites no proprio banco:

- no maximo 3 codigos para o mesmo e-mail em 30 minutos;
- no maximo 5 codigos para o mesmo IP em 30 minutos;
- bloqueio de alguns dominios de teste conhecidos;
- codigo com validade de 10 minutos.

Essas regras reduzem repeticoes simples, mas um bot pode alternar IPs e e-mails validos para continuar usando a API como disparador de e-mails. Isso tambem gera retornos de entrega no e-mail remetente quando o destinatario esta cheio ou indisponivel.

## Objetivo

Exigir uma prova anti-bot antes de o backend gerar e enviar qualquer codigo por e-mail.

Cada cliente deve usar a protecao adequada ao ambiente:

```text
Android -> Google Play Integrity API
Web     -> Cloudflare Turnstile
Backend -> valida a prova, aplica rate limit e envia o codigo
```

O backend e a unica autoridade para decidir se o e-mail sera enviado. Nenhuma validacao feita apenas no Android ou no navegador deve liberar o envio.

## Android: Google Play Integrity API

Antes de chamar um endpoint de envio de codigo, o app Android deve pedir um token de integridade ao Google Play e enviar esse token junto com o e-mail.

O token deve ser vinculado a requisicao por um hash, por exemplo:

```text
SHA-256(metodo + ":" + caminho_do_endpoint + ":" + email_normalizado)
```

O backend recalcula o hash e confere se ele corresponde ao valor devolvido pelo Google. Assim, um token obtido para um e-mail nao pode ser reutilizado para outro e-mail ou entre cadastro e redefinicao de senha.

O backend deve aceitar o pedido somente quando a verificacao confirmar, no minimo:

- app reconhecido pelo Google Play e sem alteracao;
- dispositivo com o nivel de integridade definido para producao;
- hash da requisicao correspondente;
- token valido e nao reutilizado.

O token nunca pode ser aceito apenas porque foi enviado pelo app. O backend deve decodifica-lo e validar o veredito nos servicos do Google usando uma credencial de servico privada.

### Mudancas no Android

- adicionar a biblioteca Play Integrity;
- preparar o provedor de token durante a inicializacao do app;
- gerar o token ao solicitar codigo de cadastro ou redefinicao de senha;
- incluir `play_integrity_token` no body da requisicao;
- mostrar uma mensagem orientando atualizar ou instalar pela Play Store quando a verificacao nao puder ser concluida em producao.

### Configuracao do backend

- vincular o app e o projeto Google Cloud na Play Console;
- criar uma conta de servico com permissao apenas para validar os tokens;
- fornecer a credencial por segredo de deploy ou variavel de ambiente;
- nunca versionar a credencial no repositorio, na imagem Docker ou no `docker-compose.yml`.

## Web: Cloudflare Turnstile

Nas telas web de cadastro e esqueci minha senha, o Turnstile gera um token temporario antes do envio do formulario. O modo `Managed` deve ser usado para manter a maior parte das verificacoes invisivel para usuarios reais.

O front-end envia o token no campo `turnstile_token`. O backend chama a API Siteverify da Cloudflare e so continua quando a resposta for valida.

Na validacao, o backend deve conferir:

- sucesso da verificacao;
- hostname permitido;
- acao esperada para cadastro ou redefinicao de senha, quando configurada;
- token ainda valido e nao reutilizado;
- IP remoto, quando disponivel.

O `sitekey` pode ficar no front-end. A chave secreta do Turnstile fica somente no backend, em segredo de deploy ou variavel de ambiente.

## Custos e cotas dos servicos

Para implementar somente a protecao proposta neste documento, nao ha previsao de custo recorrente direto dentro das cotas atuais dos servicos.

| Componente | Custo previsto | Cota ou limite relevante | Observacao |
| --- | --- | --- | --- |
| Cloudflare Turnstile | Gratuito no plano Free | desafios e verificacoes ilimitados; ate 20 widgets; ate 10 hostnames por widget | O limite de 20 e de configuracoes de widget, nao de usuarios, acessos simultaneos ou desafios. O projeto provavelmente precisara de poucos widgets. |
| Google Play Integrity API | A documentacao oficial nao indica cobranca por requisicao no uso padrao | ate 10.000 requisicoes por dia por aplicativo, por padrao | Se o aplicativo se aproximar da cota, e possivel solicitar aumento ao Google, sujeito aos requisitos e a aprovacao. |
| Validacao no backend | Sem novo fornecedor pago | consome apenas os recursos normais da API e da rede | Existe custo de desenvolvimento, manutencao, logs e monitoramento, mas nao uma assinatura adicional. |
| SMTP atual do Gmail | Sem nova cobranca causada pelo anti-bot | continua sujeito aos limites de envio e as regras de reputacao do Gmail | A protecao reduz disparos abusivos, mas nao transforma o Gmail em um servico transacional com controle de bounces. |
| Provedor de e-mail transacional | Opcional e potencialmente pago | depende do fornecedor e do volume enviado | Pode ser adotado depois para obter bounces, supressao automatica, logs de entrega e melhor controle de reputacao. |
| Servico externo de verificacao de e-mail | Opcional e normalmente cobrado por consulta | depende do fornecedor | Nao e necessario para a primeira entrega e nao garante com 100% de certeza que uma caixa postal existe ou recebera a mensagem. |

Na pratica, para o cenario atual do SobraAi:

- o Turnstile pode atender qualquer quantidade de usuarios e desafios no plano Free, respeitando os limites de widgets e hostnames;
- o limite de 20 widgets nao significa 20 pessoas, 20 desafios ou 20 acessos simultaneos;
- o Play Integrity deve ser chamado apenas quando o Android solicitar o envio de um codigo, preservando a cota diaria;
- a cota padrao de 10.000 requisicoes por dia deve ser monitorada, mas ultrapassar a cota nao gera automaticamente uma cobranca: novas verificacoes podem ser limitadas ate que exista cota disponivel ou um aumento seja aprovado;
- nenhum verificador pago de e-mail deve ser contratado como dependencia desta primeira fase;
- migrar do Gmail para um provedor transacional e uma melhoria separada, com custo avaliado conforme o volume real de e-mails.

Essas condicoes podem mudar pelos fornecedores. Antes da implantacao em producao e durante revisoes periodicas, devem ser consultadas as fontes oficiais:

- [Planos do Cloudflare Turnstile](https://developers.cloudflare.com/turnstile/plans/)
- [Cotas do Google Play Integrity](https://support.google.com/googleplay/android-developer/answer/11395166?hl=pt)

## Contrato dos endpoints

Os dois endpoints continuam separados, mas passam a receber a prova do cliente. Exemplo para Android:

```json
{
  "email": "usuario@exemplo.com",
  "protection_provider": "play_integrity",
  "play_integrity_token": "token-do-google"
}
```

Para o web:

```json
{
  "email": "usuario@exemplo.com",
  "protection_provider": "turnstile",
  "turnstile_token": "token-da-cloudflare"
}
```

Em producao, requisicoes sem uma prova valida devem receber erro `403` e nao devem criar `RegistrationCode` ou `PasswordResetToken`, nem chamar o SMTP.

O valor de `protection_provider` apenas indica qual verificador deve ser executado. Ele nao e confiavel por si so: a decisao depende da validacao real do token no backend.

## Regras que permanecem

Play Integrity e Turnstile nao substituem as protecoes existentes. O backend deve manter e reforcar:

- limite por e-mail;
- limite por IP;
- intervalo minimo para reenviar codigo ao mesmo e-mail, recomendado em 10 minutos;
- logs de bloqueios com IP e user agent, sem registrar codigo ou token;
- bloqueio de dominios descartaveis conhecidos;
- resposta generica quando apropriado, para reduzir enumeracao de e-mails cadastrados.

Tambem deve existir uma supressao temporaria para enderecos que retornarem falha de entrega, evitando novas tentativas imediatas para o mesmo destinatario.

O endpoint `POST /api/auth/reset-password` nao precisa de Play Integrity ou Turnstile, porque ele ja exige um codigo valido recebido pelo e-mail. A verificacao anti-bot fica apenas nas rotas que iniciam o disparo de e-mail.

## Transicao segura

Versoes antigas do Android ainda nao enviam o token. Por isso, a ativacao precisa ser gradual:

1. Configurar Play Integrity e Turnstile em ambiente de desenvolvimento com chaves de teste.
2. Publicar uma versao Android que envie o token, mantendo o backend apenas registrando tokens ausentes ou invalidos por um periodo curto.
3. Verificar os logs e a taxa de validacao do Android e do web.
4. Ativar a exigencia obrigatoria no backend de producao.
5. Definir versao minima do Android quando necessario, para impedir que versoes antigas continuem usando os endpoints sem protecao.

Nao deve existir uma liberacao permanente para pedidos sem token em producao, porque isso permitiria que bots contornassem a protecao.

## Testes necessarios

- token Play Integrity valido permite o envio para cadastro e esqueci minha senha;
- token Play Integrity ausente, invalido, reutilizado ou com hash diferente bloqueia os dois fluxos;
- token Turnstile valido permite o envio para cadastro e esqueci minha senha;
- token Turnstile ausente, invalido, expirado ou com hostname diferente bloqueia os dois fluxos;
- requisicao bloqueada nao cria `RegistrationCode` ou `PasswordResetToken` e nao chama o servico de e-mail;
- limites existentes por e-mail e IP continuam sendo aplicados apos a validacao anti-bot;
- fluxos completos de cadastro e redefinicao de senha continuam funcionando para Android e web validos.

## Criterio de pronto

Esta melhoria esta concluida quando:

- Android envia e backend valida o token do Play Integrity nos dois endpoints de envio de codigo;
- web envia e backend valida o token do Turnstile nos dois endpoints de envio de codigo;
- o backend recusa pedidos sem prova valida em producao;
- segredos do Google e da Cloudflare nao estao versionados;
- os limites de envio existentes continuam ativos;
- ha testes cobrindo aprovacao e bloqueio para os dois canais;
- ha monitoramento de bloqueios, falhas de validacao e falhas de entrega de e-mail;
- ha monitoramento do consumo da cota diaria do Play Integrity;
- os custos de qualquer provedor de e-mail ou verificador externo opcional foram aprovados antes da contratacao.
