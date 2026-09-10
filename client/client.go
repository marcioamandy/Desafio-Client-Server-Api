package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

const (
	serverURL     = "http://localhost:8080/cotacao"
	clientTimeout = 300 * time.Millisecond
	arquivoSaida  = "cotacao.txt"
)

// CotacaoOutput espelha o JSON devolvido pelo servidor.
type CotacaoOutput struct {
	Bid string `json:"bid"`
}

func main() {
	bid, err := buscaCotacao()
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Fatalf("timeout de %v excedido ao aguardar a resposta do servidor: %v", clientTimeout, err)
		}
		log.Fatalf("erro ao obter a cotação do servidor: %v", err)
	}

	if err := salvaArquivo(bid); err != nil {
		log.Fatalf("erro ao salvar o arquivo %s: %v", arquivoSaida, err)
	}

	log.Printf("cotação salva em %s: Dólar: %s", arquivoSaida, bid)
}

func buscaCotacao() (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), clientTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, serverURL, nil)
	if err != nil {
		return "", err
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return "", ctxErr
		}
		return "", err
	}
	defer res.Body.Close()

	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("o servidor respondeu com status %s", res.Status)
	}

	var cotacao CotacaoOutput
	if err := json.NewDecoder(res.Body).Decode(&cotacao); err != nil {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return "", ctxErr
		}
		return "", err
	}

	return cotacao.Bid, nil
}

func salvaArquivo(bid string) error {
	f, err := os.Create(arquivoSaida)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = fmt.Fprintf(f, "Dólar: %s\n", bid)
	return err
}
