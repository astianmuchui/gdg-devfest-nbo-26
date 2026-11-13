package mqtt

import (
	"fmt"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/astianmuchui/supersimpleingestor/internal/env"
	"github.com/astianmuchui/supersimpleingestor/internal/pipeline"
	"github.com/astianmuchui/supersimpleingestor/internal/report"
)

var onConnect pahomqtt.OnConnectHandler = func(client pahomqtt.Client) {
	fmt.Println("MQTT: connected to broker")
	subscribe(client, env.GetMQTTSubTopic())
}

var onConnectionLost pahomqtt.ConnectionLostHandler = func(client pahomqtt.Client, err error) {
	fmt.Printf("MQTT: connection lost: %v\n", err)
}

func NewMessageHandler(p *pipeline.Pipeline) pahomqtt.MessageHandler {
	return func(_ pahomqtt.Client, msg pahomqtt.Message) {
		report.Received.Add(1)
		rec, err := parseReading(msg.Topic(), msg.Payload())
		if err != nil {
			report.Fail(err)
			return
		}
		p.Submit(rec)
	}
}
