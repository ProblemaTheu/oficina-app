package database

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	_ "github.com/lib/pq"
)

var migrationsFS embed.FS

func RunMigrations(databaseURL string) error {
	db, err := sql.Open("postgres", databaseURL)
	if err != nil {
		return fmt.Errorf("erro ao abrir conexão com banco de dados: %w", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		return fmt.Errorf("erro ao conectar ao banco de dados: %w", err)
	}

	source, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("erro ao criar fonte de migrations: %w", err)
	}

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	if err != nil {
		return fmt.Errorf("erro ao criar driver de migrations: %w", err)
	}

	m, err := migrate.NewWithInstance("iofs", source, "postgres", driver)
	if err != nil {
		return fmt.Errorf("erro ao inicializar migrator: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("erro ao executar migrations: %w", err)
	}

	return nil
}
