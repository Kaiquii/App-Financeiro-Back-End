# SobraAi - API

Back-end em Go do app SobraAi, responsavel por autenticacao, usuarios, despesas, rendas, categorias, relatorios, assistente IA e foto de perfil.

[Ver rotas da API](docs/api-routes.md)

## Resumo Do Projeto

O projeto roda uma API HTTP em Go usando Gin, GORM e PostgreSQL.

Fluxo principal:

1. A API conecta no PostgreSQL usando `DB_DSN`.
2. As tabelas sao migradas automaticamente ao iniciar.
3. Rotas publicas ficam em `/api/auth`.
4. Rotas protegidas precisam de `Authorization: Bearer TOKEN`.
5. A API pode salvar foto de perfil em storage local ou Oracle Object Storage.

Principais variaveis:

```txt
PORT=8080
DB_DSN=host=db user=postgres password=4343 dbname=app_financeiro port=5432 sslmode=disable TimeZone=America/Sao_Paulo
JWT_SECRET=sua_chave
GIN_MODE=release
```

## Como Subir Na VM Pela Primeira Vez

Na sua maquina local, gere a imagem:

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
sudo docker compose up -d
```

Conferir se subiu:

```bash
sudo docker ps
sudo docker logs -f app_financeiro_api
```

Esse primeiro `docker compose up -d` sobe a API e o banco.

## Como Atualizar Somente A API Na VM

Use quando o banco ja existe e voce quer trocar apenas o back-end.

Na maquina local, gere um novo `.tar`:

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

Esse comando recria somente o container da API e nao mexe no banco.

Conferir logs:

```bash
sudo docker logs -f app_financeiro_api
```

Importante: nao use `docker compose down -v`, porque o `-v` remove volumes e pode apagar os dados do banco.
