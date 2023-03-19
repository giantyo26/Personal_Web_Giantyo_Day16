package connection

import (
	"context"
	"fmt"
	"os"

	"github.com/jackc/pgx/v4"
)

var Conn *pgx.Conn

func ConnectDatabase() {
	databaseUrl := "postgres://postgres:kronos777@localhost:5432/personal-web"

	var err error
	Conn, err = pgx.Connect(context.Background(), databaseUrl)

	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to connect database: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Successfully connected to database")
}
