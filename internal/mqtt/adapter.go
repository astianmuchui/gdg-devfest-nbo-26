package mqtt

import (
	"fmt"
	"os"

	pahomqtt "github.com/eclipse/paho.mqtt.golang"

	"github.com/astianmuchui/supersimpleingestor/internal/env"
)

func ConnectAll(n int, h pahomqtt.MessageHandler) {
	for i := 0; i < n; i++ {
		go connectOne(fmt.Sprintf("%s-%d", env.GetMQTTClientID(), i), h)
	}
}

func connectOne(id string, h pahomqtt.MessageHandler) pahomqtt.Client {
	opts := pahomqtt.NewClientOptions()
	opts.AddBroker(fmt.Sprintf("mqtt://%s:%d", env.GetMQTTBroker(), env.GetMQTTPort()))
	opts.SetClientID(id)

	if u := env.GetMQTTUsername(); u != "" {
		opts.SetUsername(u)
		opts.SetPassword(env.GetMQTTPassword())
	}

	opts.SetAutoReconnect(true)
	opts.SetConnectRetry(true)
	opts.SetCleanSession(false)
	opts.WillQos = 1

	opts.OnConnect = onConnect
	opts.OnConnectionLost = onConnectionLost
	opts.SetDefaultPublishHandler(h)

	client := pahomqtt.NewClient(opts)
	if token := client.Connect(); token.Wait() && token.Error() != nil {
		fmt.Fprintf(os.Stderr, "MQTT connect error [%s]: %v\n", id, token.Error())
	}
	return client
}

func subscribe(c pahomqtt.Client, topic string) {
	token := c.Subscribe(topic, 1, nil)
	token.Wait()
	if token.Error() != nil {
		fmt.Fprintf(os.Stderr, "MQTT subscribe error [%s]: %v\n", topic, token.Error())
	}
}
