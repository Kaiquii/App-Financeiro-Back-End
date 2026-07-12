# Melhoria tecnica: migrations versionadas

## Contexto atual

Hoje o projeto usa GORM com PostgreSQL e cria/ajusta as tabelas automaticamente na inicializacao da API.

O ponto central esta em `cmd/api/main.go`, onde a aplicacao executa `database.DB.AutoMigrate(...)` para os models de usuarios, reset de senha, despesas, rendas, categorias e assistente.

Esse modelo e pratico para desenvolvimento e para um projeto pequeno, porque uma alteracao simples no struct Go pode ser refletida no banco quando a API sobe.

## Isso e realmente necessario?

Sim, mas como melhoria tecnica de producao, nao como emergencia funcional.

O projeto ja esta em producao e tem dados financeiros reais de usuarios. Enquanto as mudancas de banco forem pequenas, o `AutoMigrate` pode continuar funcionando. O risco aumenta quando o backend precisar fazer alteracoes mais sensiveis, como:

- renomear colunas;
- mudar tipos de colunas;
- criar ou alterar indices;
- adicionar constraints;
- remover campos antigos;
- migrar dados existentes de uma estrutura para outra;
- separar tabelas ou criar relacionamentos novos.

Nesses casos, deixar a aplicacao alterar o schema automaticamente no boot fica menos previsivel. A producao passa a depender de uma decisao implicita do GORM no momento em que o container inicia.

## Risco pratico do AutoMigrate em producao

O `AutoMigrate` tenta aproximar o banco dos models atuais, mas ele nao representa um historico claro das mudancas.

Exemplo:

1. Existe uma coluna `name`.
2. O codigo muda para usar `full_name`.
3. O `AutoMigrate` pode criar `full_name`, mas nao sabe sozinho se precisa copiar os dados de `name`, manter as duas colunas por compatibilidade, ou remover a coluna antiga em outro momento.

Para alteracoes financeiras, isso e delicado porque uma mudanca de schema mal controlada pode gerar dados vazios, duplicados, inconsistentes ou dificeis de corrigir depois.

## Como seria o modelo recomendado

Criar migrations versionadas em arquivos, por exemplo:

```txt
migrations/
  001_initial_schema.sql
  002_add_expense_notes.sql
  003_add_user_access_blocked.sql
  004_create_assistant_conversations.sql
```

Cada arquivo descreve uma mudanca especifica no banco e roda apenas uma vez, em ordem.

Com isso, a sequencia de evolucao do banco fica auditavel:

- o que mudou;
- quando mudou;
- em qual ordem mudou;
- qual comando precisa rodar antes de subir a nova versao da API.

## Caminho seguro para este projeto

Como o projeto ja esta em producao, a troca deve ser gradual.

1. Levantar o schema atual do banco de producao.
2. Criar uma migration inicial representando esse estado atual.
3. Escolher uma ferramenta simples de migrations para Go/PostgreSQL.
4. A partir desse ponto, toda mudanca nova de banco deve entrar como migration.
5. Manter `AutoMigrate` apenas em desenvolvimento durante a transicao, se ainda for util.
6. Desativar `AutoMigrate` em producao quando as migrations estiverem confiaveis.

## Prioridade sugerida

Prioridade: media.

Nao precisa parar o desenvolvimento atual so por causa disso, mas e uma melhoria importante antes de o banco crescer muito ou antes de mudancas estruturais maiores.

Itens com prioridade maior antes ou junto dessa melhoria:

- remover segredos sensiveis de arquivos versionados/de deploy;
- restringir CORS para os dominios reais;
- validar variaveis obrigatorias no boot da API;
- criar backup antes de qualquer alteracao estrutural de banco.

## Criterio de pronto

Essa melhoria pode ser considerada concluida quando:

- existir uma pasta de migrations no projeto;
- o schema atual estiver representado por uma migration inicial;
- novas mudancas de banco forem feitas via migration;
- o deploy rodar migrations antes de recriar a API;
- o `AutoMigrate` nao rodar mais em producao.
