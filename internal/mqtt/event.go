package mqtt

import (
	"context"
	"fmt"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/astianmuchui/supersimpleingestor/internal/env"
	"github.com/astianmuchui/supersimpleingestor/internal/report"
	"github.com/astianmuchui/supersimpleingestor/internal/store"
)

var onConnect pahomqtt.OnConnectHandler = func(client pahomqtt.Client) {
	fmt.Println("MQTT: connected to broker")
	subscribe(client, env.GetMQTTTopic())
}

var onConnectionLost pahomqtt.ConnectionLostHandler = func(client pahomqtt.Client, err error) {
	fmt.Printf("MQTT: connection lost: %v\n", err)
}

func newMessageHandler(st store.Store) pahomqtt.MessageHandler {
	return func(client pahomqtt.Client, msg pahomqtt.Message) {
		report.Received.Add(1)
		report.InFlight.Add(1)

		go func() {
			defer report.InFlight.Add(-1)
			if err := handleReadings(context.Background(), st, msg.Topic(), msg.Payload()); err != nil {
				report.Fail(err)
				return
			}
			report.Stored.Add(1)
		}()
	}
}
