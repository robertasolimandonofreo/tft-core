package internal

import (
	"context"
	"encoding/json"
	"fmt"
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

func (nc *NATSClient) PublishSummonerNameTask(task SummonerNameTask) error {
	data, err := json.Marshal(task)
	if err != nil {
		return err
	}
	return nc.Publish("tft.summoner.name.fetch", data)
}

func (nc *NATSClient) StartSummonerNameWorker(riotClient *RiotAPIClient, cacheManager *CacheManager) (*nats.Subscription, error) {
	handler := func(msg *nats.Msg) {
		var task SummonerNameTask
		if err := json.Unmarshal(msg.Data, &task); err != nil {
			log.Printf("Error unmarshaling summoner name task: %v", err)
			return
		}

		log.Printf("Processing summoner name task: PUUID=%s", task.PUUID[:30]+"...")

		ctx := context.Background()

		if cachedName, err := cacheManager.GetSummonerName(ctx, task.PUUID); err == nil && cachedName != "" {
			log.Printf("Name already exists in cache for PUUID %s: %s", task.PUUID[:30]+"...", cachedName)
			return
		}

		accountData, err := riotClient.GetAccountByPUUID(task.PUUID)
		if err != nil {
			log.Printf("Error fetching account data for PUUID %s: %v", task.PUUID[:30]+"...", err)
			return
		}

		if accountData.GameName != "" {
			fullName := accountData.GameName
			if accountData.TagLine != "" {
				fullName = fmt.Sprintf("%s#%s", accountData.GameName, accountData.TagLine)
			}

			if err := cacheManager.SetSummonerName(ctx, task.PUUID, fullName); err != nil {
				log.Printf("Error caching summoner name: %v", err)
			} else {
				log.Printf("Name cached successfully: PUUID=%s, Name=%s", task.PUUID[:30]+"...", fullName)
			}
		} else {
			log.Printf("GameName not found in account data: %+v", accountData)
		}
	}

	sub, err := nc.Conn.QueueSubscribe("tft.summoner.name.fetch", "name-workers", handler)
	if err != nil {
		return nil, err
	}
	log.Println("Summoner Name Worker started, waiting for messages...")
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

	ctx := context.Background()
	cacheKey := cacheManager.Key("challenger", region)
	return cacheManager.Set(ctx, cacheKey, result, 30*time.Minute)
}

func (nc *NATSClient) updateGrandmasterLeague(riotClient *RiotAPIClient, cacheManager *CacheManager, region string) error {
	result, err := riotClient.GetGrandmasterLeague()
	if err != nil {
		return err
	}

	ctx := context.Background()
	cacheKey := cacheManager.Key("grandmaster", region)
	return cacheManager.Set(ctx, cacheKey, result, 30*time.Minute)
}

func (nc *NATSClient) updateMasterLeague(riotClient *RiotAPIClient, cacheManager *CacheManager, region string) error {
	result, err := riotClient.GetMasterLeague()
	if err != nil {
		return err
	}

	ctx := context.Background()
	cacheKey := cacheManager.Key("master", region)
	return cacheManager.Set(ctx, cacheKey, result, 30*time.Minute)
}