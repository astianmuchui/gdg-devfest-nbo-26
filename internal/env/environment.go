package env

import (
	"os"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v2/log"
	"github.com/joho/godotenv"
)

const DEFAULT_PORT = 8080

func Load() {
	if err := godotenv.Load(); err != nil {
		log.Info("No .env file found, using environment and defaults")
	}
}

func str(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func num(key string, def int) int {
	p, err := strconv.Atoi(os.Getenv(key))
	if err != nil {
		return def
	}
	return p
}

func dur(key string, def time.Duration) time.Duration {
	d, err := time.ParseDuration(os.Getenv(key))
	if err != nil {
		return def
	}
	return d
}

func GetHttpListenPort() int { return num("IOT_PORT", DEFAULT_PORT) }

func GetMQTTBroker() string   { return str("IOT_MQTT_BROKER", "localhost") }
func GetMQTTPort() int        { return num("IOT_MQTT_PORT", 1883) }
func GetMQTTClientID() string { return str("IOT_MQTT_CLIENT_ID", "iot_ingester") }
func GetMQTTTopic() string    { return str("IOT_MQTT_TOPIC", "devices/+/readings") }
func GetMQTTUsername() string { return os.Getenv("IOT_MQTT_USERNAME") }
func GetMQTTPassword() string { return os.Getenv("IOT_MQTT_PASSWORD") }

func GetCassandraHost() string     { return str("IOT_CASSANDRA_HOST", "localhost") }
func GetCassandraPort() int        { return num("IOT_CASSANDRA_PORT", 9042) }
func GetCassandraKeyspace() string { return str("IOT_CASSANDRA_KEYSPACE", "iot") }
func GetCassandraUsername() string { return os.Getenv("IOT_CASSANDRA_USERNAME") }
func GetCassandraPassword() string { return os.Getenv("IOT_CASSANDRA_PASSWORD") }

func GetStallAfter() time.Duration { return dur("IOT_STALL_AFTER", 60*time.Second) }
func GetStallFor() time.Duration   { return dur("IOT_STALL_FOR", 0) }
func GetStallDelay() time.Duration { return dur("IOT_STALL_DELAY", 2*time.Second) }
