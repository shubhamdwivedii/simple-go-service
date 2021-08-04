package main

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql" // "_" means we are importing it but won't use directly. Just importing it will register it as driver.
)

func main() {
	fmt.Println("Drivers:", sql.Drivers()) // By default there are no Drivers // output > Drivers: []
	// to add a driver > go get github.com/go-sql-driver/mysql
	// after adding driver sql.Drivers() > Drivers: [mysql]

	// sql.Open(driverName string, dataSourceName string) (*sql.DB, error)
	db, err := sql.Open("mysql", "root:hesoyam@tcp(127.0.0.1:3306)/test") // test is the DB name
	if err != nil {
		log.Fatal("Unable to open connection to db")
	}
	defer db.Close() // To make sure db is closed properly

	// db.Query(query string, args ...interface{}) (*sql.Rows, error)
	results, err := db.Query("select * from product")
	if err != nil {
		log.Fatal("Error when fetching product table rows:", err)
	}
	defer results.Close() // Its a good practice to close results when done with them.

	// ### Reading all rows ###
	// We can use restuls.Next() to loop over results sets.
	for results.Next() {
		var (
			id    int
			name  string
			price int
		)
		err = results.Scan(&id, &name, &price) // results.Scan() takes pointers to variables and scans values into them
		fmt.Printf("ID: %d, Name: '%s', Price: %d\n", id, name, price)
	}

	// ### Reading single row ###
	var (
		id    int
		name  string
		price int
	)
	// Returns a single db.ROW so we can directly Scan()
	err = db.QueryRow("Select * from product where id = 1").Scan(&id, &name, &price)
	if err != nil {
		log.Fatal("Unable to parse row:", err)
	}
	fmt.Printf("ID: %d, Name: '%s', Price: %d\n", id, name, price)

	// ### Inserting rows ###
	products := []struct {
		name  string
		price int
	}{
		{"Light", 10},
		{"Mic", 30},
		{"Router", 90},
	}

	// Preparing a statement for insert
	stmt, err := db.Prepare("INSERT INTO product (name, price) VALUES (?, ?)")
	if err != nil {
		log.Fatal("Unable to prepare statement:", err)
	}
	for _, product := range products {
		_, err = stmt.Exec(product.name, product.price)
		if err != nil {
			log.Fatal("Unable to execute statement:", err)
		}
	}
}
