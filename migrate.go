package main
import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/lib/pq"
	_ "github.com/lib/pq"
)

func migrateUp(path, datasource string) error {
	for range time.Tick(time.Millisecond * 300) {
		conn, err := sql.Open("postgres", datasource)
		if err != nil {
			return err
		}

		_, err = conn.Exec("SELECT 1")
		if err == nil {
			conn.Close()
			break
		}

		if strings.Contains(err.Error(), "connection refused") {
			conn.Close()
			continue
		}

		pqErr, ok := err.(*pq.Error)
		if !ok {
			conn.Close()
			return errors.New("Unexpected non-postgres error : " + err.Error())
		}
		if pqErr.Code == "57P03" {
			conn.Close()
			continue
		}
		conn.Close()
	}

	mg, err := migrate.New(path, datasource)
	if err != nil {
		return err
	}

	err = mg.Up()
	if err != nil && err != migrate.ErrNoChange {
		return err
	}

	return nil
}