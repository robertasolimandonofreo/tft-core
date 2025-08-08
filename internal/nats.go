package internal

import (
	"encoding/json"
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

func (nc *NATSClient) PublishLeagueUpdateTask(task LeagueUpdateTask) error {
	data, err := json.Marshal(task)
	if err != nil {
		return err
	}
	return nc.Publish("tft.league.update", data)
}

func (nc *NATSClient) StartSummonerFetchWorker(handler func(msg *nats.Msg)) (*nats.Subscription, error) {
	sub, err := nc.Conn.QueueSubscribe("tft.summoner.fetch", "summoner-workers", handler)
	if err != nil {
		return nil, err
	}
	log.Println("Worker SummonerFetch started, waiting for messages...")
	return sub, nil
}

func (nc *NATSClient) StartLeagueUpdateWorker(riotClient *RiotAPIClient, cacheManager *CacheManager) (*nats.Subscription, error) {
	handler := func(msg *nats.Msg) {
		var task LeagueUpdateTask
		if err := json.Unmarshal(msg.Data, &task); err != nil {
			log.Printf("Error unmarshaling league task: %v", err)
			return
		}
		
		log.Printf("Processing league update task: %+v", task)
		
		switch task.Type {
		case "challenger":
			if err := nc.updateChallengerLeague(riotClient, cacheManager, task.Region); err != nil {
				log.Printf("Error updating challenger league: %v", err)
			}
		case "grandmaster":
			if err := nc.updateGrandmasterLeague(riotClient, cacheManager, task.Region); err != nil {
				log.Printf("Error updating grandmaster league: %v", err)
			}
		case "master":
			if err := nc.updateMasterLeague(riotClient, cacheManager, task.Region); err != nil {
				log.Printf("Error updating master league: %v", err)
			}
		case "entries":
			if err := nc.updateLeagueEntries(riotClient, cacheManager, task.Region, task.Tier, task.Division, task.Page); err != nil {
				log.Printf("Error updating league entries: %v", err)
			}
		default:
			log.Printf("Unknown task type: %s", task.Type)
		}
	}
	
	sub, err := nc.Conn.QueueSubscribe("tft.league.update", "league-workers", handler)
	if err != nil {
		return nil, err
	}
	log.Println("League Update Worker started, waiting for messages...")
	return sub, nil
}

func (nc *NATSClient) updateChallengerLeague(riotClient *RiotAPIClient, cacheManager *CacheManager, region string) error {
	result, err := riotClient.GetChallengerLeague()
	if err != nil {
		return err
	}
	
	cacheKey := cacheManager.GenerateKey("challenger", region)
	return cacheManager.SetCachedData(nil, cacheKey, result, 30*time.Minute)
}

func (nc *NATSClient) updateGrandmasterLeague(riotClient *RiotAPIClient, cacheManager *CacheManager, region string) error {
	result, err := riotClient.GetGrandmasterLeague()
	if err != nil {
		return err
	}
	
	cacheKey := cacheManager.GenerateKey("grandmaster", region)
	return cacheManager.SetCachedData(nil, cacheKey, result, 30*time.Minute)
}

func (nc *NATSClient) updateMasterLeague(riotClient *RiotAPIClient, cacheManager *CacheManager, region string) error {
	result, err := riotClient.GetMasterLeague()
	if err != nil {
		return err
	}
	
	cacheKey := cacheManager.GenerateKey("master", region)
	return cacheManager.SetCachedData(nil, cacheKey, result, 30*time.Minute)
}

func (nc *NATSClient) updateLeagueEntries(riotClient *RiotAPIClient, cacheManager *CacheManager, region, tier, division string, page int) error {
	result, err := riotClient.GetLeagueEntries(tier, division, page)
	if err != nil {
		return err
	}
	
	cacheKey := cacheManager.GenerateKey("entries", region, tier, division, string(rune(page)))
	return cacheManager.SetCachedData(nil, cacheKey, result, 30*time.Minute)
}