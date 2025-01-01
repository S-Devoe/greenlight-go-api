package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/s-devoe/greenlight-go/internal/data"
)

const version = "1.0.0"

type config struct {
	port int
	env  string
	db   struct {
		dsn string
	}
}

type application struct {
	config config
	logger *log.Logger
	store  data.Store
}

// these are ment to be in .env
const (
	dbDriver      = "postgres"
	dbSource      = "postgresql://greenlight_user:pa55word@localhost:5432/greenlight?sslmode=disable"
	serverAddress = "0.0.0.0:4000"
)

func main() {
	var cfg config

	flag.IntVar(&cfg.port, "port", 4000, "API server port")
	flag.StringVar(&cfg.env, "env", "development", "Environment (development|staging|production)")
	flag.StringVar(&cfg.db.dsn, "db-dsn", dbSource, "POSTGRESQL DSN")

	flag.Parse()

	logger := log.New(os.Stdout, "", log.Ldate|log.Ltime)

	connPool, err := pgxpool.NewWithConfig(context.Background(), PgxConfig())
	if err != nil {
		log.Fatal("error while connecting to the database ", err)
	}

	connection, err := connPool.Acquire(context.Background())
	if err != nil {
		log.Fatal("error while aquiring connection to the database ", err)
	}

	err = connection.Ping(context.Background())
	if err != nil {
		log.Fatal("Could not ping database")
	}

	fmt.Println("database connection established")

	defer connPool.Close()

	app := &application{
		logger: logger,
		config: cfg,
		store:  data.NewStore(connPool),
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.port),
		Handler:      app.routes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	logger.Printf("Starting %s server on %s", cfg.env, srv.Addr)
	err = srv.ListenAndServe()

	logger.Fatal(err)
}

func PgxConfig() *pgxpool.Config {
	const defaultMaxConns = int32(4)
	const defaultMinConns = int32(0)
	const defaultMaxConnLifetime = time.Hour
	const defaultMaxConnIdletime = time.Minute * 30
	const defaultHealthCheckPeriod = time.Minute
	const defaultConnecTimeout = time.Second * 50

	dbConfig, err := pgxpool.ParseConfig(dbSource)
	if err != nil {
		log.Fatal("failed to create a config, error:", err)
	}
	dbConfig.MaxConns = defaultMaxConns
	dbConfig.MinConns = defaultMinConns
	dbConfig.MaxConnLifetime = defaultMaxConnLifetime
	dbConfig.MaxConnIdleTime = defaultMaxConnIdletime
	dbConfig.HealthCheckPeriod = defaultHealthCheckPeriod
	dbConfig.ConnConfig.ConnectTimeout = defaultConnecTimeout

	dbConfig.BeforeAcquire = func(ctx context.Context, c *pgx.Conn) bool {
		log.Println("Before acquiring the connection pool to the database!!")
		return true
	}

	dbConfig.AfterRelease = func(c *pgx.Conn) bool {
		log.Println("After releasing the connection pool to the database!!")
		return true
	}

	dbConfig.BeforeClose = func(c *pgx.Conn) {
		log.Println("Before closing the connection pool to the database!!")

	}

	return dbConfig

}
