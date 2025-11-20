package postgres

import (
	"embed"
	"errors"
	"log"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/pgx"
	"github.com/golang-migrate/migrate/v4/source/iofs"
)

//go:embed migrations/*.sql
var migrationFiles embed.FS

func (r *Repo) Migrate() error {
	log.Print("running migration")
	files, err := iofs.New(migrationFiles, "migrations")
	if err != nil {
		return err
	}
	driver, err := pgx.WithInstance(r.db.DB, &pgx.Config{})
	if err != nil {
		return err
	}
	m, err := migrate.NewWithInstance("iofs", files, "pgx", driver)
	if err != nil {
		return err
	}

	err = m.Up()

	if err != nil {
		if !errors.Is(err, migrate.ErrNoChange) {
			return err
		}
		log.Print("migration did not change anything")
	}

	log.Print("migration finished")
	return nil
}
