package internal

import (
	"log"
	"time"

	"github.com/nats-io/nats.go"
)

type NATSClient struct {
	Conn *nats.Conn
}

func NewNATSClient(cfg *Config) (*NATSClient, error) {
	conn, err := nats.Connect(cfg.NATSUrl,
		nats.Name(cfg.NATSClientID),
		nats.Timeout(5*time.Second),
	)
	if err != nil {
		return nil, err
	}
	return &NATSClient{Conn: conn}, nil
}

func (nc *NATSClient) Publish(subject string, data []byte) error {
	return nc.Conn.Publish(subject, data)
}

func (nc *NATSClient) StartSummonerFetchWorker(handler func(msg *nats.Msg)) (*nats.Subscription, error) {
	sub, err := nc.Conn.QueueSubscribe("tft.summoner.fetch", "summoner-workers", handler)
	if err != nil {
		return nil, err
	}
	log.Println("Worker SummonerFetch started, waiting for messages...")
	return sub, nil
}