package cmd

type Config struct {
	HTTPPort                  string
	DBHost                    string
	DBPort                    string
	DBUser                    string
	DBPassword                string
	DBName                    string
	DBSslMode                 string
	GeoServiceGrpcHost        string
	KafkaHost                 string
	KafkaConsumerGroup        string
	KafkaBasketConfirmedTopic string
	KafkaOrderChangedTopic    string
}
