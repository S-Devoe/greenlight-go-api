package main

import (
	"context"
	"expvar"
	"log"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/s-devoe/greenlight-go/config"
	"github.com/s-devoe/greenlight-go/internal/data"
	"github.com/s-devoe/greenlight-go/internal/jsonlog"
	"github.com/s-devoe/greenlight-go/internal/mailer"
)

const version = "1.0.0"

// type config struct {
// 	port int
// 	env  string
// 	db   struct {
// 		dsn string
// 	}
// 	DB_SOURCE string
// 	limiter   struct {
// 		rps     float64
// 		burst   int
// 		enabled bool
// 	}
// 	smtp struct {
// 		host     string
// 		port     int
// 		username string
// 		password string
// 		sender   string
// 	}
// }

type application struct {
	config config.Config
	logger *jsonlog.Logger
	store  data.Store
	mailer mailer.Mailer
	wg     sync.WaitGroup
}

// these are ment to be in .env
// const (
// 	dbSource = "postgresql://greenlight_user:pa55word@localhost:5432/greenlight?sslmode=disable"
// )

func main() {
	cfg := config.InitConfig()

	expvar.NewString("version").Set(version)
	expvar.Publish("goroutines", expvar.Func(func() interface{} {
		return runtime.NumGoroutine()
	}))

	logger := jsonlog.New(os.Stdout, jsonlog.LevelInfo)

	connPool, err := pgxpool.NewWithConfig(context.Background(), PgxConfig(&cfg))
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

	log.Printf("Connected to the database %d", cfg.Port)
	logger.PrintInfo("database connection established", nil)

	app := &application{
		logger: logger,
		config: cfg,
		store:  data.NewStore(connPool),
		mailer: mailer.New(cfg.SMTPHost, cfg.SMTPPort, cfg.SMTPUsername, cfg.SMTPPassword, cfg.SMTPSender),
	}

	err = app.serve()

	logger.PrintFatal(err, nil)
}

func PgxConfig(cfg *config.Config) *pgxpool.Config {
	const defaultMaxConns = int32(4)
	const defaultMinConns = int32(0)
	const defaultMaxConnLifetime = time.Hour
	const defaultMaxConnIdletime = time.Minute * 30
	const defaultHealthCheckPeriod = time.Minute
	const defaultConnecTimeout = time.Second * 50

	dbConfig, err := pgxpool.ParseConfig(cfg.DbSource)
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
