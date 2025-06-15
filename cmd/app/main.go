package main

import (
	"delivery/cmd"
	"fmt"
	"net/http"
	"os"

	"github.com/joho/godotenv"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
)

func main() {
	configs := getConfigs()

	app := cmd.NewCompositionRoot(
		configs,
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
