package main

import (
	goSql "database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/gorilla/schema"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

var localDB = "postgres://pguser:pgpass@localhost:9001/shelters?sslmode=disable"

func main() {
	// init db
	dbURL := localDB
	if os.Getenv("ENV") == "server" {
		dbURL = os.Getenv("DATABASE_URL")
	}

	sqldb, err := goSql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalln("error opening sql", err.Error())
	}
	sqlxDb := sqlx.NewDb(sqldb, "postgres")

	animalController := AnimalController{DB: sqlxDb}
	http.HandleFunc("/go-animals", animalController.GetAnimals)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatalln("server crashed", err)
	}
}

type GetAnimalsParams struct {
	Limit  int `json:"limit"`
	Cursor int `json:"cursor"`
}

type DbResponse struct {
	ID          string     `db:"id"`
	AnimalName  string     `db:"animal_name"`
	AnimalType  string     `db:"animal_type"`
	Breed       string     `db:"breed"`
	Color       string     `db:"color"`
	Sex         string     `db:"sex"`
	AnimalSize  string     `db:"animal_size"`
	DateOfBirth *time.Time `db:"date_of_birth"`
}

var (
	decoder = schema.NewDecoder()
)

type AnimalController struct {
	DB *sqlx.DB
}

func (a AnimalController) GetAnimals(w http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	params := GetAnimalsParams{}

	err := req.ParseForm()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": fmt.Sprintf("failed to parse request %v", err.Error()),
		})
		return
	}

	err = decoder.Decode(&params, req.Form)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": fmt.Sprintf("failed to decode request %v", err.Error()),
		})
		return
	}

	limit := params.Limit
	if limit == 0 {
		limit = 100
	}

	selectQuery := sq.Select("*").
		From("animals").
		//Where(sq.GtOrEq{"cursor_id": params.Cursor}).
		Limit(uint64(limit))
	sqlQuery, args, err := selectQuery.PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": fmt.Sprintf("failed to build query %v", err.Error()),
		})
		return
	}

	var result []DbResponse
	err = a.DB.SelectContext(ctx, &result, sqlQuery, args...)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": fmt.Sprintf("failed to get data %v", err.Error()),
		})
		return
	}

	resp := map[string]interface{}{
		"animals": result,
	}

	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}
