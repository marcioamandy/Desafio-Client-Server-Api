package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"time"

	_ "modernc.org/sqlite"
)

const (
	cotacaoAPIURL = "https://economia.awesomeapi.com.br/json/last/USD-BRL"
	apiTimeout    = 200 * time.Millisecond
	dbTimeout     = 10 * time.Millisecond
	dbFile        = "cotacoes.db"
	serverPort    = ":8080"
)

// USDBRL espelha o objeto retornado pela AwesomeAPI.
type USDBRL struct {
	Code       string `json:"code"`
	Codein     string `json:"codein"`
	Name       string `json:"name"`
	High       string `json:"high"`
	Low        string `json:"low"`
	VarBid     string `json:"varBid"`
	PctChange  string `json:"pctChange"`
	Bid        string `json:"bid"`
	Ask        string `json:"ask"`
	Timestamp  string `json:"timestamp"`
	CreateDate string `json:"create_date"`
}

type cotacaoResponse struct {
	USDBRL USDBRL `json:"USDBRL"`
}

// CotacaoOutput é o payload devolvido ao cliente.
type CotacaoOutput struct {
	Bid string `json:"bid"`
}

func main() {
	db, err := initDB()
	if err != nil {
		log.Fatalf("erro ao inicializar o banco de dados: %v", err)
	}
	defer db.Close()

	mux := http.NewServeMux()
	mux.HandleFunc("/cotacao", cotacaoHandler(db))

	log.Printf("servidor ouvindo em http://localhost%s/cotacao", serverPort)
	if err := http.ListenAndServe(serverPort, mux); err != nil {
		log.Fatalf("erro ao subir o servidor: %v", err)
	}
}

func initDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite", dbFile)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS cotacoes (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		code TEXT,
		codein TEXT,
		name TEXT,
		high TEXT,
		low TEXT,
		var_bid TEXT,
		pct_change TEXT,
		bid TEXT,
		ask TEXT,
		timestamp TEXT,
		create_date TEXT,
		created_at DATETIME
	)`)
	if err != nil {
		db.Close()
		return nil, err
	}

	return db, nil
}

func cotacaoHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cotacao, err := buscaCotacao(r.Context())
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				log.Printf("timeout de %v excedido ao consultar a API de cotação: %v", apiTimeout, err)
			} else {
				log.Printf("erro ao consultar a API de cotação: %v", err)
			}
			http.Error(w, "erro ao obter a cotação", http.StatusInternalServerError)
			return
		}

		// A persistência não deve impedir a resposta ao cliente: o erro é apenas logado.
		if err := salvaCotacao(r.Context(), db, cotacao); err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				log.Printf("timeout de %v excedido ao persistir a cotação no banco: %v", dbTimeout, err)
			} else {
				log.Printf("erro ao persistir a cotação no banco: %v", err)
			}
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(CotacaoOutput{Bid: cotacao.Bid}); err != nil {
			log.Printf("erro ao escrever a resposta: %v", err)
		}
	}
}

func buscaCotacao(ctx context.Context) (*USDBRL, error) {
	ctx, cancel := context.WithTimeout(ctx, apiTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, cotacaoAPIURL, nil)
	if err != nil {
		return nil, err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		// O erro do transporte encapsula o motivo do cancelamento do contexto.
		if ctxErr := ctx.Err(); ctxErr != nil {
			return nil, ctxErr
		}
		return nil, err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return nil, errors.New("a API de cotação respondeu com status " + res.Status)
	}

	var body cotacaoResponse
	if err := json.NewDecoder(res.Body).Decode(&body); err != nil {
		return nil, err
	}

	return &body.USDBRL, nil
}

func salvaCotacao(ctx context.Context, db *sql.DB, c *USDBRL) error {
	ctx, cancel := context.WithTimeout(ctx, dbTimeout)
	defer cancel()

	_, err := db.ExecContext(ctx,
		`INSERT INTO cotacoes
			(code, codein, name, high, low, var_bid, pct_change, bid, ask, timestamp, create_date, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.Code, c.Codein, c.Name, c.High, c.Low, c.VarBid, c.PctChange, c.Bid, c.Ask, c.Timestamp, c.CreateDate, time.Now(),
	)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		return err
	}

	return nil
}
