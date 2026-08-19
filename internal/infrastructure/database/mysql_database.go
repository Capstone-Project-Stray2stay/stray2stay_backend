package database

import (
	"database/sql"
	"fmt"
	"log"

	_ "github.com/go-sql-driver/mysql"

	"github.com/S-nudhana/stray2stay/internal/infrastructure/config"
)

func NewMySQLDatabase(cfg config.MySQLConfig) (*sql.DB, error) {
	// clientFoundRows: without it, RowsAffected() reports rows *changed* by an
	// UPDATE rather than rows *matched* — so re-saving identical values (e.g.
	// retrying after a partial failure) reports 0 and gets misread as "not
	// found" by the RowsAffected()==0 checks in the adapters.
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?parseTime=true&clientFoundRows=true",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name,
	)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, err
	}
	log.Println("MySQL Database connected successfully")
	return db, nil
}