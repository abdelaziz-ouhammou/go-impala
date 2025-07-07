package main

import (
	"database/sql"
	"log"

	_ "github.com/abdelaziz-ouhammou/go-impala/v3"
)

func main() {
	db, err := sql.Open("impala", "impala://pachinko-parlor.io")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	var result int8
	err = db.QueryRow("select 1").Scan(&result)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("result: %d", result)
}
