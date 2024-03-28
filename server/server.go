package main

import (
    "bytes"
    "context"
    "database/sql"
    "encoding/json"
    "io"
    _ "modernc.org/sqlite"

    //"errors"
    "log"
    "net/http"
    "os"
    "strings"
    "time"
)

type StdUsdBrl struct {
    Usdbrl struct {
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
    } `json:"USDBRL"`
}
type StdResult struct {
    Erro string `json:"erro"`
    Bid  string `json:"bid"`
}

const DbName = "logs.db"

func main() {
    if !DbExiste(DbName) {
        log.Println("Erro ao criar banco de dados. Verifique!!!!")
        os.Exit(3)
    }
    http.HandleFunc("/cotacao", Cotacao)
    println("started")
    _ = http.ListenAndServe(":8080", nil)
}

// Cotaçao Realiza a cotação do dolar
//
// PARAMETERS
//
//     w http.ResponseWriter Objeto que grava a resposta HTTP
//
//     r *http.Request Objeto que tem os dados da requisição
//
func Cotacao(w http.ResponseWriter, r *http.Request) {
    var stdUsdBrl StdUsdBrl

    db, err := DbOpen()

    if err != nil {
        log.Println("Erro ao abrir o banco de dados: " + err.Error())
        _, _ = w.Write([]byte(`{"erro":"Erro ao abrir banco de dados de logs","bid":""}`))
        return
    }
    defer db.Close()

    // Context Da requisição, com ele saberemos se foi cancelada a operação do lado do cliente
    ctxReq := r.Context()

    // Context da chamada a API. Timeout 200 milisegundos
    ctx := context.Background()
    ctx, Cancel := context.WithTimeout(ctx, 200*time.Millisecond)
    defer Cancel()

    // Preparamos a requisição
    req, err := http.NewRequestWithContext(ctx, "GET", "https://economia.awesomeapi.com.br/json/last/USD-BRL", nil)

    if err != nil {
        log.Println("Erro preparando a requisição: " + err.Error())

        _, _ = w.Write([]byte(`{"erro":"` + ErroFmt(err.Error()) + `","bid":""}`))
        return
    }

    req.Header.Set("Accept", "application/json")

    // Executamos a requisição
    resp, err := http.DefaultClient.Do(req)

    if err != nil {
        log.Println("Erro executando a requisição: " + err.Error())
        _, _ = w.Write([]byte(`{"erro":"` + ErroFmt(err.Error()) + `","bid":""}`))
        return
    }
    defer resp.Body.Close()

    // Lêmos os dados para depois gravar
    Resposta, err := io.ReadAll(resp.Body)

    if err != nil {
        log.Println("Lendo corpo da respota: " + err.Error())
        _, _ = w.Write([]byte(`{"erro":"` + ErroFmt(err.Error()) + `","bid":""}`))
        return
    }

    // Criamos o decodifocador JSON
    JsonReader := json.NewDecoder(bytes.NewReader(Resposta))

    // Decodificamos a resposta
    err = JsonReader.Decode(&stdUsdBrl)

    if err != nil {
        log.Println("JSON Reader:" + err.Error())
        _, _ = w.Write([]byte(`{"erro":"` + ErroFmt(err.Error()) + `","bid":""}`))
        return
    }

    err = GravarCotacao(db, Resposta)

    if err != nil {
        log.Println("db insert" + err.Error())
        _, _ = w.Write([]byte(`{"erro":"` + ErroFmt(err.Error()) + `","bid":""}`))
        return
    }

    // Gravamos o JSON com a cotação em casso de sucesso
    _, _ = w.Write([]byte(`{"erro":"","bid":"` + stdUsdBrl.Usdbrl.Bid + `"}`))

    select {
    case <-ctxReq.Done():
        log.Println("Request cancelado pelo cliente")
    default:

        log.Println("Request executado com sucesso")
    }
}

func ErroFmt(sErro string) string {
    return strings.ReplaceAll(sErro, `"`, ` `)
}

// Verificamos se o arquivo DB existe
func DbExiste(sNome string) bool {
    _, error := os.Stat(sNome)

    if os.IsNotExist(error) {
        db, err := sql.Open("sqlite", sNome)

        if err != nil {
            return false
        }
        defer db.Close()

        err = db.Ping()

        if err != nil {
            return false
        }

        query := `
    CREATE TABLE IF NOT EXISTS cotacoes(
        data TEXT NOT NULL,
        cotacao TEXT
    );
    `
        _, err = db.Exec(query)

        if err != nil {
            return false
        }
    }
    return true
}

func DbOpen() (*sql.DB, error) {
    db, err := sql.Open("sqlite", DbName)

    if err != nil {
        return nil, err
    }

    err = db.Ping()

    if err != nil {
        return nil, err
    }

    return db, nil
}

func GravarCotacao(db *sql.DB, Cotacao []byte) error {
    loc, _ := time.LoadLocation("America/Sao_Paulo")
    now := time.Now().In(loc)
    ctx := context.Background()
    ctx, Cancel := context.WithTimeout(ctx, 10*time.Millisecond)
    defer Cancel()

    stmt, err := db.Prepare("INSERT INTO cotacoes(data, cotacao) values($1,$2)")

    if err != nil {
        return err
    }

    _, err = stmt.ExecContext(ctx, now.Format("2006-01-02T15:04:05"), string(Cotacao))

    if err != nil {
        return err
    }

    return nil
}
