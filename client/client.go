package main

import (
    "context"

    "encoding/json"
    // "fmt"
    "log"
    "net/http"
    "os"
    "strings"
    "time"
)

// Strutura de retorno do Server
type Result struct {
    Erro string `json:"erro"`
    Bid  string `json:"bid"`
}

func main() {
    var Resposta Result

    // Definimos o contexto
    ctx := context.Background()
    ctx, Cancel := context.WithTimeout(ctx, 300000*time.Millisecond)
    defer Cancel()

    // Preparamos a chamada ao server
    req, err := http.NewRequestWithContext(ctx, "GET", "http://localhost:8080/cotacao", nil)

    if err != nil {
        return
    }

    req.Header.Set("Accept", "application/json")

    // Efetuamos a requisição.
    resp, err := http.DefaultClient.Do(req)

    if err != nil {
        log.Println("Erro ao realizar requisição: " + err.Error())
        return
    }
    defer resp.Body.Close()

    // Criamos o decodificador JSON
    JsonReader := json.NewDecoder(resp.Body)

    // Efetuamos a decodificação
    err = JsonReader.Decode(&Resposta)

    if err != nil {
        log.Println("Erro ao decodificar JSON: " + err.Error())
        return
    }

    // Se o campo erro tem informação enviamos para o log e saimos
    if strings.TrimSpace(Resposta.Erro) != "" {
        log.Println("Erro na requisição: " + Resposta.Erro)
        return
    }

    err = GravarCotacao(Resposta.Bid)

    if err != nil {
        log.Println("Erro ao gravar cotação no arquivo: " + err.Error())
        return
    }
}

///////////////////////////////
// GravarCotacao Grava a cotação no arquivo cotação, se este não existe ele é criado
//
// PARAMETERES
//
//     sText string
//
// RETURN
//
//     error Erro da operação
func GravarCotacao(sText string) error {
    var sNome string = "cotacao.txt"

    f, err := os.OpenFile(sNome, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)

    if err != nil {
        return err
    }
    defer f.Close()

    _, err = f.WriteString("Dólar: " + sText + "\n")

    if err != nil {
        return err
    }

    return nil
}
