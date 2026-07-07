package mqttx

import (
	"fmt"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
)

type Config struct {
	Broker   string
	ClientID string
	Username string
	Password string
}

type Client struct {
	c mqtt.Client
}

func Connect(cfg Config) (*Client, error) {
	opts := mqtt.NewClientOptions().
		AddBroker(cfg.Broker).
		SetClientID(cfg.ClientID).
		SetAutoReconnect(true).
		SetConnectRetry(true).
		SetConnectRetryInterval(2 * time.Second).
		SetCleanSession(false).
		SetKeepAlive(30 * time.Second).
		SetPingTimeout(10 * time.Second)
	if cfg.Username != "" {
		opts.SetUsername(cfg.Username)
		opts.SetPassword(cfg.Password)
	}

	c := mqtt.NewClient(opts)
	tok := c.Connect()
	if !tok.WaitTimeout(10 * time.Second) {
		return nil, fmt.Errorf("mqtt connect timeout")
	}
	if err := tok.Error(); err != nil {
		return nil, fmt.Errorf("mqtt connect: %w", err)
	}
	return &Client{c: c}, nil
}

func (m *Client) Subscribe(topic string, qos byte, handler func(topic string, payload []byte)) error {
	tok := m.c.Subscribe(topic, qos, func(_ mqtt.Client, msg mqtt.Message) {
		handler(msg.Topic(), msg.Payload())
	})
	tok.Wait()
	return tok.Error()
}

func (m *Client) Disconnect() {
	m.c.Disconnect(500)
}
