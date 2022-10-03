package main

import (
	"context"
	goSql "database/sql"
	"encoding/csv"
	"log"
	"os"
	"strconv"
	"strings"
	"time"

	sq "github.com/Masterminds/squirrel"
	"github.com/jmoiron/sqlx"
	jsoninter "github.com/json-iterator/go"
	_ "github.com/lib/pq"
)

var localDB = "postgres://pguser:pgpass@localhost:9001/shelters?sslmode=disable"

func main() {

	dbURL := os.Getenv("db_url")
	if dbURL == "" {
		dbURL = localDB
	}

	log.Println("url set", dbURL)

	sqldb, err := goSql.Open("postgres", dbURL)
	if err != nil {
		log.Fatalln("error opening sql", err.Error())
	}
	sqlx.NewDb(sqldb, "postgres")

	// open file
	f, err := os.Open("rawdata/sonoma_shelter_renamed.csv")
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	// read csv values using csv.Reader
	csvReader := csv.NewReader(f)
	data, err := csvReader.ReadAll()
	if err != nil {
		log.Fatal(err)
	}

	for idx, line := range data {
		if idx == 0 {
			continue
		}

		// build and insert the animal structure
		animal := struct {
			ID          string     `db:"id"`
			AnimalName  string     `db:"animal_name"`
			AnimalType  string     `db:"animal_type"`
			Breed       string     `db:"breed"`
			Color       string     `db:"color"`
			Sex         string     `db:"sex"`
			AnimalSize  string     `db:"animal_size"`
			DateOfBirth *time.Time `db:"date_of_birth"`
		}{
			ID:          line[dataMap.AnimalID],
			AnimalName:  line[dataMap.AnimalName],
			AnimalType:  line[dataMap.AnimalType],
			Breed:       line[dataMap.Breed],
			Color:       line[dataMap.Color],
			Sex:         line[dataMap.Sex],
			AnimalSize:  line[dataMap.AnimalSize],
			DateOfBirth: parseDate(line[dataMap.DateOfBirth]),
		}

		tagMap := map[string]interface{}{}
		var rjson = jsoninter.Config{TagKey: "db"}.Froze()
		data, err := rjson.Marshal(animal)
		if err != nil {
			log.Fatal(err)
		}
		err = rjson.Unmarshal(data, &tagMap)
		if err != nil {
			log.Fatal(err)
		}

		query := sq.Insert("animals").SetMap(tagMap)
		sql, args, err := query.PlaceholderFormat(sq.Dollar).ToSql()
		if err != nil {
			log.Fatal(err)
		}
		_, err = sqldb.ExecContext(context.Background(), sql, args...)
		if err != nil {
			if strings.Contains(err.Error(), "duplicate") {
				log.Println("duplicate insert, moving on", args, err.Error())
			} else {
				log.Fatalln("error inserting animal")
			}
		}

		// build and insert animal intake
		dis := strings.Split(line[dataMap.DaysInShelter], ",")
		daysInShelter, err := strconv.Atoi(strings.Join(dis, ""))
		if err != nil {
			log.Fatalln("unable to parse days in shelter", err)
		}
		animalCount, err := strconv.Atoi(line[dataMap.AnimalCount])
		if err != nil {
			log.Fatalln("unable to parse animal count", err)
		}

		zc := strings.Split(line[dataMap.Zipcode], ".")
		var zipCode int
		if len(zc) > 1 {
			zipCode, err = strconv.Atoi(zc[0])
			if err != nil {
				log.Fatalln("unable to parse zipcode", err, len(zc))
			}
		}

		intake := struct {
			ImpoundNumber       string     `db:"impound_number"`
			KennelNumber        string     `db:"Kennel_number"`
			AnimalID            string     `db:"animal_id"`
			IntakeDate          *time.Time `db:"intake_date"`
			OutcomeDate         *time.Time `db:"outcome_date"`
			DaysInShelter       int        `db:"days_in_shelter"`
			IntakeType          string     `db:"intake_type"`
			IntakeSubtype       string     `db:"intake_subtype"`
			OutcomeType         string     `db:"outcome_type"`
			OutcomeSubtype      string     `db:"outcome_subtype"`
			IntakeCondition     string     `db:"intake_condition"`
			OutcomeCondition    string     `db:"outcome_condition"`
			IntakeJurisdiction  string     `db:"intake_jurisdiction"`
			OutcomeJurisidction string     `db:"outcome_jurisdiction"`
			Location            string     `db:"location"`
			AnimalCount         int        `db:"animal_count"`
			ZipCode             int        `db:"zip_code"`
		}{
			ImpoundNumber:       line[dataMap.ImpoundNumber],
			KennelNumber:        line[dataMap.KennelNumber],
			AnimalID:            line[dataMap.AnimalID],
			IntakeDate:          parseDate(line[dataMap.IntakeDate]),
			OutcomeDate:         parseDate(line[dataMap.OutcomeDate]),
			DaysInShelter:       daysInShelter,
			IntakeType:          line[dataMap.IntakeType],
			IntakeSubtype:       line[dataMap.IntakeSubType],
			OutcomeType:         line[dataMap.OutcomeType],
			OutcomeSubtype:      line[dataMap.OutcomeSubType],
			IntakeCondition:     line[dataMap.IntakeCondition],
			OutcomeCondition:    line[dataMap.OutcomeCondition],
			IntakeJurisdiction:  line[dataMap.IntakeJurisdiction],
			OutcomeJurisidction: line[dataMap.OutcomeJurisdiction],
			Location:            line[dataMap.Location],
			AnimalCount:         animalCount,
			ZipCode:             zipCode,
		}

		tagMap = map[string]interface{}{}
		data, err = rjson.Marshal(intake)
		if err != nil {
			log.Fatal(err)
		}
		err = rjson.Unmarshal(data, &tagMap)
		if err != nil {
			log.Fatal(err)
		}

		query = sq.Insert("animal_intake").SetMap(tagMap)
		sql, args, err = query.PlaceholderFormat(sq.Dollar).ToSql()
		if err != nil {
			log.Fatal(err)
		}
		_, err = sqldb.ExecContext(context.Background(), sql, args...)
		if err != nil {
			if strings.Contains(err.Error(), "duplicate") {
				log.Println("duplicate insert, moving on", args, err.Error())
			} else {
				log.Fatalln("error inserting intake", err.Error())
			}

		}
	}
}

func parseDate(date string) *time.Time {

	splitDate := strings.Split(date, "/")
	if len(splitDate) < 3 {
		return nil
	}
	year, _ := strconv.Atoi(splitDate[2])
	month, _ := strconv.Atoi(splitDate[1])
	day, _ := strconv.Atoi(splitDate[0])
	monthObj := time.Month(month)

	finalDate := time.Date(year, monthObj, day, 0, 0, 0, 0, time.Local)
	return &finalDate
}

var dataMap = struct {

	// animal
	AnimalID    int
	AnimalName  int
	AnimalType  int
	Breed       int
	Color       int
	Sex         int
	AnimalSize  int
	DateOfBirth int

	// shelter
	ImpoundNumber       int
	KennelNumber        int
	IntakeDate          int
	OutcomeDate         int
	DaysInShelter       int
	IntakeType          int
	IntakeSubType       int
	OutcomeType         int
	OutcomeSubType      int
	IntakeCondition     int
	OutcomeCondition    int
	IntakeJurisdiction  int
	OutcomeJurisdiction int
	Zipcode             int
	Location            int
	AnimalCount         int
}{
	// animal
	AnimalID:    10,
	AnimalName:  1,
	AnimalType:  2,
	Breed:       3,
	Color:       4,
	Sex:         5,
	AnimalSize:  6,
	DateOfBirth: 7,

	// shelter
	ImpoundNumber: 8,
	KennelNumber:  9,
	// animal_id 10
	IntakeDate:          11,
	OutcomeDate:         12,
	DaysInShelter:       13,
	IntakeType:          14,
	IntakeSubType:       15,
	OutcomeType:         16,
	OutcomeSubType:      17,
	IntakeCondition:     18,
	OutcomeCondition:    19,
	IntakeJurisdiction:  20,
	OutcomeJurisdiction: 21,
	Zipcode:             22,
	Location:            23,
	AnimalCount:         24,
}
