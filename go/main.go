package main

import (
	goSql "database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/gorilla/mux"
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
	defer sqldb.Close()
	sqlxDb := sqlx.NewDb(sqldb, "postgres")
	sqlxDb.SetMaxOpenConns(10)
	sqlxDb.SetMaxIdleConns(3)
	sqlxDb.SetConnMaxIdleTime(time.Duration(300) * time.Second)
	sqlxDb.SetConnMaxLifetime(time.Duration(1) * time.Hour)

	animalController := AnimalController{DB: sqlxDb}
	debugController := Debug{DB: sqlxDb}

	r := mux.NewRouter()
	r.HandleFunc("/v1/go-animals", animalController.GetAnimals)
	r.HandleFunc("/v1/debug", debugController.GetDBStats)

	envPort := os.Getenv("PORT")
	port := fmt.Sprintf(":%s", envPort)
	http.HandleFunc("/v1/go-animals", animalController.GetAnimals)
	if err := http.ListenAndServe(port, r); err != nil {
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

	limit := req.URL.Query().Get("limit")
	limitInt, err := strconv.Atoi(limit)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": fmt.Sprintf("failed to parse query parameters %v", err.Error()),
		})
		return
	}

	if limitInt == 0 {
		limitInt = 100
	}

	selectQuery := sq.
		Select("*").
		From("animals").
		//Where(sq.GtOrEq{"cursor_id": params.Cursor}).
		Limit(uint64(limitInt))
	sqlQuery, args, err := selectQuery.PlaceholderFormat(sq.Dollar).ToSql()
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": fmt.Sprintf("failed to build query %v", err.Error()),
		})
		return
	}

	conn, err := a.DB.Connx(ctx)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": fmt.Sprintf("failed to open connection %v", err.Error()),
		})
		return
	}
	defer conn.Close()

	var result []DbResponse
	err = conn.SelectContext(ctx, &result, sqlQuery, args...)
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

type Debug struct {
	DB *sqlx.DB
}

func (d Debug) GetDBStats(w http.ResponseWriter, req *http.Request) {
	resp := map[string]interface{}{
		"db_stats": d.DB.Stats(),
	}
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}
