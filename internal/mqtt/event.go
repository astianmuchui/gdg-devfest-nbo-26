package mqtt

import (
	"fmt"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/astianmuchui/supersimpleingestor/internal/env"
)

var onConnect pahomqtt.OnConnectHandler = func(client pahomqtt.Client) {
	fmt.Println("MQTT: connected to broker")
	subscribe(client, env.GetMQTTTopic())
}

var onConnectionLost pahomqtt.ConnectionLostHandler = func(client pahomqtt.Client, err error) {
	fmt.Printf("MQTT: connection lost: %v\n", err)
}

var messageHandler pahomqtt.MessageHandler = func(client pahomqtt.Client, msg pahomqtt.Message) {
	if err := handleReadings(msg.Topic(), msg.Payload()); err != nil {
		fmt.Printf("MQTT handler error [%s]: %v\n", msg.Topic(), err)
	}
}
