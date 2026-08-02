# SobraAí - API

Back-end em Go do app SobraAí, responsável por autenticação, usuários, despesas, rendas, categorias, relatórios, assistente IA e foto de perfil.

## Resumo Do Projeto

O projeto roda uma API HTTP em Go usando Gin, GORM e PostgreSQL.

Fluxo principal:

1. A API conecta no PostgreSQL usando `DB_DSN`.
2. A API valida se todas as migrations já foram aplicadas, sem alterar o banco durante a inicialização.
3. Rotas públicas ficam em `/api/auth`.
4. Rotas protegidas precisam de `Authorization: Bearer TOKEN`.
5. A API pode salvar foto de perfil em storage local ou Oracle Object Storage.

Principais variáveis:

```txt
PORT=8080
DB_DSN=host=db user=postgres password=4343 dbname=app_financeiro port=5432 sslmode=disable TimeZone=America/Sao_Paulo
JWT_SECRET=sua_chave
GIN_MODE=release
```

## Como Subir Na VM Pela Primeira Vez

Na sua máquina local, gere a imagem:

```bash
docker build -t app-financeiro-api .
docker save -o app-financeiro-backend.tar app-financeiro-api
```

Envie para a VM:

```txt
app-financeiro-backend.tar
docker-compose.yml
```

Na VM:

```bash
cd /home/ubuntu/app-financeiro
sudo docker load -i app-financeiro-backend.tar
sudo mkdir -p /var/app-financeiro/uploads
sudo docker compose up -d db
sudo docker compose run --rm --no-deps api ./migrate up
sudo docker compose up -d api
```

Conferir se subiu:

```bash
sudo docker ps
sudo docker logs -f app_financeiro_api
```

Na primeira instalação, o banco precisa estar em execução antes de `migrate up`. A API só deve ser iniciada depois que o comando terminar com sucesso.

## Como Atualizar Somente A API Na VM

Use quando o banco já existe e você quer trocar apenas o back-end.

Na máquina local, gere um novo `.tar`:

```bash
docker build -t app-financeiro-api .
docker save -o app-financeiro-backend.tar app-financeiro-api
```

Envie o novo `app-financeiro-backend.tar` para a VM.

Na VM:

```bash
cd /home/ubuntu/app-financeiro
sudo docker load -i app-financeiro-backend.tar
sudo docker compose up -d --no-deps --force-recreate api
```

Esse comando recria somente o container da API. A inicialização apenas consulta o estado das migrations e não executa DDL nem altera o schema. Se a nova imagem exigir uma migration ainda pendente, a API recusará a inicialização e informará que `./migrate up` precisa ser executado.

Conferir logs:

```bash
sudo docker logs -f app_financeiro_api
```

Importante: não use `docker compose down -v`, porque o `-v` remove volumes e pode apagar os dados do banco.

## Como Atualizar A API Com Migration Na VM

Use este fluxo quando a nova versão do back-end incluir uma migration de banco.

Antes de aplicar a migration, faça e valide um backup do banco de produção. Depois de enviar a nova imagem para a VM, execute:

```bash
cd /home/ubuntu/app-financeiro
sudo docker load -i app-financeiro-backend.tar
sudo docker compose run --rm --no-deps api ./migrate status
sudo docker compose run --rm --no-deps api ./migrate up
sudo docker compose up -d --no-deps --force-recreate api
```

O `docker compose run` usa a imagem e a configuração atuais do serviço em um container temporário, sem substituir a API que já está rodando. A API só é recriada depois que `migrate up` conclui com sucesso.

Confira o estado e os logs:

```bash
sudo docker compose run --rm --no-deps api ./migrate status
sudo docker logs -f app_financeiro_api
```

O comando `./migrate up` é o único fluxo normal autorizado a criar ou alterar o schema. Não edite migrations que já tenham sido aplicadas; crie sempre uma nova versão.
