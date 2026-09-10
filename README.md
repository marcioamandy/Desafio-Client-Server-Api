# Desafio Client-Server-API

Desafio do curso **Go Expert**: dois programas em Go (`client` e `server`) que trocam a cotação do dólar respeitando limites estritos de tempo, usando `context`, SQLite e manipulação de arquivos.

## Fluxo

```
client (timeout 300ms) ──► server :8080 /cotacao ──► AwesomeAPI (timeout 200ms)
                                   │
                                   └──► SQLite (timeout 10ms)
```

## Regras implementadas

### `server/server.go`
- Servidor HTTP na porta **8080**, endpoint **`/cotacao`**.
- Consome `https://economia.awesomeapi.com.br/json/last/USD-BRL` com **timeout de 200ms** via `context.WithTimeout`.
- Registra cada cotação em um banco **SQLite** (`cotacoes.db`) com **timeout de 10ms** via `context.WithTimeout`.
- Retorna ao cliente o JSON `{"bid":"5.1268"}`.
- Loga no console sempre que qualquer um dos timeouts é estourado.
- Uma falha na persistência é apenas logada: o cliente ainda recebe a cotação.

### `client/client.go`
- Faz a requisição a `http://localhost:8080/cotacao` com **timeout de 300ms** via `context.WithTimeout`.
- Lê somente o campo `bid` da resposta.
- Salva o arquivo `cotacao.txt` no formato `Dólar: {valor}`.
- Loga no console caso o timeout de 300ms seja estourado.

## Requisitos

- Go 1.24 ou superior.
- Não é necessário CGO: o driver SQLite usado é o [`modernc.org/sqlite`](https://pkg.go.dev/modernc.org/sqlite), escrito em Go puro.

## Como rodar

Clone o repositório e baixe as dependências:

```bash
git clone https://github.com/marcioamandy/Desafio-Client-Server-Api.git
cd Desafio-Client-Server-Api
go mod download
```

### 1. Suba o servidor

Em um terminal:

```bash
go run ./server
```

Saída esperada:

```
2026/01/01 10:00:00 servidor ouvindo em http://localhost:8080/cotacao
```

### 2. Rode o cliente

Em **outro** terminal, na mesma pasta:

```bash
go run ./client
```

Saída esperada:

```
2026/01/01 10:00:05 cotação salva em cotacao.txt: Dólar: 5.1268
```

E o arquivo gerado:

```bash
$ cat cotacao.txt
Dólar: 5.1268
```

### Testando o endpoint direto

```bash
curl http://localhost:8080/cotacao
# {"bid":"5.1268"}
```

## Conferindo o que foi gravado no banco

O arquivo `cotacoes.db` é criado na pasta de onde o servidor foi executado. Com o `sqlite3` instalado:

```bash
sqlite3 cotacoes.db "select id, bid, create_date, created_at from cotacoes;"
```

## Estrutura

```
.
├── client/
│   └── client.go     # cliente HTTP, timeout de 300ms, grava cotacao.txt
├── server/
│   └── server.go     # servidor HTTP :8080, timeouts de 200ms (API) e 10ms (SQLite)
├── go.mod
├── go.sum
└── README.md
```

## Observações sobre os timeouts

Os prazos são propositalmente curtos. Em máquinas mais lentas, na primeira execução (quando o SQLite ainda está criando o arquivo do banco) ou em uma rede instável, é normal ver no console mensagens como:

```
timeout de 10ms excedido ao persistir a cotação no banco: context deadline exceeded
timeout de 200ms excedido ao consultar a API de cotação: context deadline exceeded
```

Esse é exatamente o comportamento pedido pelo desafio: o erro é registrado no log do serviço correspondente.
