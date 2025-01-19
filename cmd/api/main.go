package main

import (
	"context"
	"expvar"
	"flag"
	"log"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/s-devoe/greenlight-go/internal/data"
	"github.com/s-devoe/greenlight-go/internal/jsonlog"
	"github.com/s-devoe/greenlight-go/internal/mailer"
)

const version = "1.0.0"

type config struct {
	port int
	env  string
	db   struct {
		dsn string
	}
	limiter struct {
		rps     float64
		burst   int
		enabled bool
	}
	smtp struct {
		host     string
		port     int
		username string
		password string
		sender   string
	}
}

type application struct {
	config config
	logger *jsonlog.Logger
	store  data.Store
	mailer mailer.Mailer
	wg     sync.WaitGroup
}

// these are ment to be in .env
const (
	dbDriver      = "postgres"
	dbSource      = "postgresql://greenlight_user:pa55word@localhost:5432/greenlight?sslmode=disable"
	serverAddress = "0.0.0.0:4000"
)

func main() {
	var cfg config

	expvar.NewString("version").Set(version)
	expvar.Publish("goroutines", expvar.Func(func() interface{} {
		return runtime.NumGoroutine()
	}))

	flag.IntVar(&cfg.port, "port", 4000, "API server port")
	flag.StringVar(&cfg.env, "env", "development", "Environment (development|staging|production)")
	flag.StringVar(&cfg.db.dsn, "db-dsn", dbSource, "POSTGRESQL DSN")
	flag.Float64Var(&cfg.limiter.rps, "limiter-rps", 2, "Rate limiter maximum request per seconds")
	flag.IntVar(&cfg.limiter.burst, "limiter-burst", 4, "Rate limiter maximum burst")
	flag.BoolVar(&cfg.limiter.enabled, "limiter-enabled", true, "Enable rate limiter")
	flag.StringVar(&cfg.smtp.host, "smtp-host", "sandbox.smtp.mailtrap.io", "SMTP host")
	flag.IntVar(&cfg.smtp.port, "smtp-port", 2525, "SMTP port")
	flag.StringVar(&cfg.smtp.username, "smtp-username", "d8e6e86f30e9f0", "SMTP username")
	flag.StringVar(&cfg.smtp.password, "smtp-password", "d16b2255b71d35", "SMTP password")
	flag.StringVar(&cfg.smtp.sender, "smtp-sender", "Greenlight <no-reply@greenlight.alexedwards.net>", "SMTP sender")

	flag.Parse()

	logger := jsonlog.New(os.Stdout, jsonlog.LevelInfo)

	connPool, err := pgxpool.NewWithConfig(context.Background(), PgxConfig())
	if err != nil {
		log.Fatal("error while connecting to the database ", err)
		logger.PrintFatal(err, nil)
	}

	expvar.Publish("database", expvar.Func(func() interface{} {
		stats := connPool.Stat()

		// Convert the stats to a map for easier printing
		return map[string]interface{}{
			"TotalConns":        stats.TotalConns(),
			"IdleConns":         stats.IdleConns(),
			"AcquiredConns":     stats.AcquiredConns(),
			"MaxConns":          stats.MaxConns(),
			"NewConnsCount":     stats.NewConnsCount(),
			"AcquireCount":      stats.AcquireCount(),
			"AcquireDuration":   stats.AcquireDuration(),
			"EmptyAcquireCount": stats.EmptyAcquireCount(),
			"ConstructingConns": stats.ConstructingConns(),
		}
	}))
	expvar.Publish("timestamp", expvar.Func(func() interface{} {
		return time.Now().Unix()
	}))

	connection, err := connPool.Acquire(context.Background())
	if err != nil {
		log.Fatal("error while aquiring connection to the database ", err)
		logger.PrintFatal(err, nil)
	}

	err = connection.Ping(context.Background())
	if err != nil {
		log.Fatal("Could not ping database")
		logger.PrintFatal(err, nil)
	}

	defer connPool.Close()

	logger.PrintInfo("database connection established", nil)

	app := &application{
		logger: logger,
		config: cfg,
		store:  data.NewStore(connPool),
		mailer: mailer.New(cfg.smtp.host, cfg.smtp.port, cfg.smtp.username, cfg.smtp.password, cfg.smtp.sender),
	}

	err = app.serve()

	logger.PrintFatal(err, nil)
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
