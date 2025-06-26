package main

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"os"

	"delivery/cmd"
	"delivery/internal/adapters/out/postgres/courierrepo"
	"delivery/internal/adapters/out/postgres/orderrepo"
	"delivery/internal/pkg/errs"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	configs := getConfigs()

	connectionString, err := makeConnectionString(
		configs.DBHost,
		configs.DBPort,
		configs.DBUser,
		configs.DBPassword,
		configs.DBName,
		configs.DBSslMode)
	if err != nil {
		log.Fatal(err.Error())
	}

	createDBIfNotExists(configs.DBHost,
		configs.DBPort,
		configs.DBUser,
		configs.DBPassword,
		configs.DBName,
		configs.DBSslMode)
	gormDB := mustGormOpen(connectionString)
	mustAutoMigrate(gormDB)

	app := cmd.NewCompositionRoot(
		configs,
		gormDB,
	)
	startWebServer(app, configs.HTTPPort)
}

func getConfigs() cmd.Config {
	config := cmd.Config{
		HTTPPort:                  goDotEnvVariable("HTTP_PORT"),
		DBHost:                    goDotEnvVariable("DB_HOST"),
		DBPort:                    goDotEnvVariable("DB_PORT"),
		DBUser:                    goDotEnvVariable("DB_USER"),
		DBPassword:                goDotEnvVariable("DB_PASSWORD"),
		DBName:                    goDotEnvVariable("DB_NAME"),
		DBSslMode:                 goDotEnvVariable("DB_SSLMODE"),
		GeoServiceGrpcHost:        goDotEnvVariable("GEO_SERVICE_GRPC_HOST"),
		KafkaHost:                 goDotEnvVariable("KAFKA_HOST"),
		KafkaConsumerGroup:        goDotEnvVariable("KAFKA_CONSUMER_GROUP"),
		KafkaBasketConfirmedTopic: goDotEnvVariable("KAFKA_BASKET_CONFIRMED_TOPIC"),
		KafkaOrderChangedTopic:    goDotEnvVariable("KAFKA_ORDER_CHANGED_TOPIC"),
	}
	return config
}

func goDotEnvVariable(key string) string {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatalf("Error loading .env file")
	}
	return os.Getenv(key)
}

func startWebServer(_ cmd.CompositionRoot, port string) {
	e := echo.New()
	e.GET("/health", func(c echo.Context) error {
		return c.String(http.StatusOK, "Healthy")
	})

	e.Logger.Fatal(e.Start(fmt.Sprintf("0.0.0.0:%s", port)))
}

func makeConnectionString(
	host string,
	port string,
	user string,
	password string,
	dbName string,
	sslMode string,
) (string, error) {
	if err := errors.Join(
		func() error {
			if host == "" {
				return errs.NewValueIsRequiredError(host)
			}
			return nil
		}(),
		func() error {
			if port == "" {
				return errs.NewValueIsRequiredError(port)
			}
			return nil
		}(),
		func() error {
			if user == "" {
				return errs.NewValueIsRequiredError(user)
			}
			return nil
		}(),
		func() error {
			if password == "" {
				return errs.NewValueIsRequiredError(password)
			}
			return nil
		}(),
		func() error {
			if dbName == "" {
				return errs.NewValueIsRequiredError(dbName)
			}
			return nil
		}(),
		func() error {
			if sslMode == "" {
				return errs.NewValueIsRequiredError(sslMode)
			}
			return nil
		}(),
	); err != nil {
		return "", err
	}

	return fmt.Sprintf("host=%v port=%v user=%v password=%v dbname=%v sslmode=%v",
		host, port, user, password, dbName, sslMode,
	), nil
}

func createDBIfNotExists(
	host string,
	port string,
	user string,
	password string,
	dbName string,
	sslMode string,
) {
	dsn, err := makeConnectionString(host, port, user, password, "postgres", sslMode)
	if err != nil {
		log.Fatalf("Ошибка подключения к PostgreSQL: %v", err)
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Fatalf("Ошибка подключения к PostgreSQL: %v", err)
	}

	defer func() {
		if err = db.Close(); err != nil {
			log.Printf("ошибка при закрытии db: %v", err)
		}
	}()

	// Создаём базу данных, если её нет
	_, err = db.Exec(fmt.Sprintf("CREATE DATABASE %s", dbName))
	if err != nil {
		log.Printf("Ошибка создания БД (возможно, уже существует): %v", err)
	}
}

func mustGormOpen(connectionString string) *gorm.DB {
	pgGorm, err := gorm.Open(postgres.New(
		postgres.Config{
			DSN:                  connectionString,
			PreferSimpleProtocol: true,
		},
	), &gorm.Config{})
	if err != nil {
		log.Fatalf("connection to postgres through gorm\n: %s", err)
	}
	return pgGorm
}

func mustAutoMigrate(db *gorm.DB) {
	err := db.AutoMigrate(&courierrepo.CourierDTO{})
	if err != nil {
		log.Fatalf("Ошибка миграции: %v", err)
	}

	err = db.AutoMigrate(&courierrepo.StoragePlaceDTO{})
	if err != nil {
		log.Fatalf("Ошибка миграции: %v", err)
	}

	err = db.AutoMigrate(&orderrepo.OrderDTO{})
	if err != nil {
		log.Fatalf("Ошибка миграции: %v", err)
	}
}
