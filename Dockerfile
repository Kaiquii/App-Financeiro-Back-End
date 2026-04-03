# ESTÁGIO 1: Construção (Builder)
FROM golang:1.21-alpine AS builder

# Define a pasta de trabalho dentro do container
WORKDIR /app

# Copia os arquivos de dependências e baixa tudo
COPY go.mod go.sum ./
RUN go mod download

# Copia o resto do código do projeto
COPY . .

# Compila o projeto criando um executável chamado "api-financeira"
RUN CGO_ENABLED=0 GOOS=linux go build -o api-financeira ./cmd/api/main.go


# ESTÁGIO 2: Produção (A máquina leve que vai pra nuvem)
FROM alpine:latest

WORKDIR /root/

# Copia apenas o executável do estágio anterior
COPY --from=builder /app/api-financeira .

# Expõe a porta que o Gin está usando
EXPOSE 8080

# Comando para rodar o servidor quando o container ligar
CMD ["./api-financeira"]