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

## Implementacao atual

A API nao aplica migrations durante a inicializacao. Ela executa uma validacao somente leitura e recusa a inicializacao quando a tabela `schema_migrations` estiver ausente, houver migration pendente, checksum divergente ou versao desconhecida.

Somente o binario `migrate`, por meio de `./migrate up`, cria ou altera o schema. Nao existe modo legado, variavel para reativar o fluxo antigo ou chamada de `AutoMigrate` no projeto. Em um banco vazio, a baseline cria o schema por SQL explicito e transacional.

A imagem Docker inclui dois binarios:

```txt
main
migrate
```

O segundo oferece os comandos:

```bash
./migrate status
./migrate up
./migrate baseline --confirm-schema-reviewed
```

A migration `000001_baseline_current_schema` usa uma copia congelada do schema atual. Ela nao depende dos models que serao alterados por features futuras.

## Processo seguro aplicado em producao

### 1. Fazer backup

Na VM, com o container atual do PostgreSQL em execucao:

```bash
chmod +x scripts/database/backup-production.sh
./scripts/database/backup-production.sh /home/ubuntu/migration-backup
```

O script gera:

- backup completo em formato custom do PostgreSQL;
- dump separado do schema para revisao;
- listagem do conteudo reconhecido pelo `pg_restore`;
- checksums SHA-256 quando o comando estiver disponivel.

Os arquivos contem dados financeiros e hashes de senha. Devem permanecer fora do repositorio e com acesso restrito.

### 2. Restaurar o backup

Em uma maquina com Docker, executar:

```bash
chmod +x scripts/database/verify-restore.sh
./scripts/database/verify-restore.sh /caminho/app_financeiro-data.dump
```

O script cria um PostgreSQL temporario sem volume, restaura o backup, lista as tabelas, mostra contagens principais e remove somente o container descartavel criado por ele.

O backup so pode ser considerado valido depois dessa restauracao terminar sem erro e as contagens serem comparadas com a producao.

### 3. Revisar o schema

Conferir no arquivo `app_financeiro-schema-*.sql` pelo menos:

- tabelas esperadas;
- colunas e tipos;
- chaves primarias;
- indices unicos;
- indices usados por autenticacao, codigos, despesas, assistente e versoes.

Essa revisao manual complementa a validacao automatica do comando baseline.

### 4. Conferir o status

Usar um container temporario da nova imagem, sem substituir a API em execucao:

```bash
sudo docker compose run --rm --no-deps api ./migrate status
```

A baseline deve aparecer como `pending`.

### 5. Registrar a baseline

Somente depois do backup restaurado e do schema revisado:

```bash
sudo docker compose run --rm --no-deps api ./migrate baseline --confirm-schema-reviewed
sudo docker compose run --rm --no-deps api ./migrate status
```

O comando valida tabelas, colunas e indices obrigatorios. Em seguida, cria `schema_migrations` e registra a baseline como aplicada. Ele nao recria nem apaga as tabelas da aplicacao.

### 6. Aplicar migrations e publicar a API

Para as proximas versoes, aplicar as migrations pendentes antes de recriar a API:

```bash
sudo docker compose run --rm --no-deps api ./migrate status
sudo docker compose run --rm --no-deps api ./migrate up
sudo docker compose up -d --no-deps --force-recreate api
```

O runner obtem um lock transacional no PostgreSQL e executa somente as migrations pendentes. Se qualquer migration falhar, a transacao e revertida. A API valida o estado em modo somente leitura e nao inicia se o banco estiver pendente ou incompatível. Nao usar `docker compose down -v`.

## Proxima migration

Depois da ativacao validada, a feature de tags deve comecar em uma nova versao, sem editar a baseline:

```txt
000002_create_tags
000003_create_expense_tags
```

Migrations aplicadas nunca devem ser renomeadas, reordenadas ou reescritas.

## Estado em producao

Ativacao concluida em 02/08/2026:

- backup completo criado na VM com permissao `600`;
- copia externa conferida por SHA-256;
- restauracao validada em PostgreSQL 15 temporario sem volume persistente;
- nove tabelas e contagens comparadas com a producao;
- baseline `000001_baseline_current_schema` registrada com checksum;
- API reiniciada usando exclusivamente migrations versionadas, sem `AutoMigrate`;
- rota publica e protecao JWT verificadas depois do deploy;
- imagem anterior preservada com tag de rollback;
- caminho legado e todas as chamadas de `AutoMigrate` removidos do codigo.
