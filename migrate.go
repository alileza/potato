package main

import (
	"database/sql"
	"errors"
	"io/ioutil"
	"strings"
	"time"

	"github.com/lib/pq"
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

	files, err := ioutil.ReadDir(path)
	if err != nil {
		return err
	}

	conn, err := sql.Open("postgres", datasource)
	if err != nil {
		return err
	}

	for _, file := range files {
		if !strings.Contains(file.Name(), ".up.") {
			continue
		}
		if !strings.Contains(file.Name(), ".sql") {
			continue
		}

		content, err := ioutil.ReadFile(path + "/" + file.Name())
		if err != nil {
			return err
		}
		if _, err := conn.Exec(string(content)); err != nil && !strings.Contains(err.Error(), "already exist") {
			return err
		}
	}

	return nil
}
