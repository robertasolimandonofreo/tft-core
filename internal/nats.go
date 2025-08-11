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
		processSummonerNameTask(msg, riotClient, cacheManager)
	}

	sub, err := nc.Conn.QueueSubscribe("tft.summoner.name.fetch", "name-workers", handler)
	if err != nil {
		return nil, err
	}
	log.Println("Summoner Name Worker started, waiting for messages...")
	return sub, nil
}

func processSummonerNameTask(msg *nats.Msg, riotClient *RiotAPIClient, cacheManager *CacheManager) {
	var task SummonerNameTask
	if err := json.Unmarshal(msg.Data, &task); err != nil {
		log.Printf("Error unmarshaling summoner name task: %v", err)
		return
	}

	log.Printf("Processing summoner name task: PUUID=%s", task.PUUID[:30]+"...")

	ctx := context.Background()

	if shouldSkipTask(task.PUUID, cacheManager, ctx) {
		return
	}

	accountData, err := riotClient.GetAccountByPUUID(task.PUUID)
	if err != nil {
		log.Printf("Error fetching account data for PUUID %s: %v", task.PUUID[:30]+"...", err)
		return
	}

	cacheSummonerName(accountData, task.PUUID, cacheManager, ctx)
}

func shouldSkipTask(puuid string, cacheManager *CacheManager, ctx context.Context) bool {
	if cachedName, err := cacheManager.GetSummonerName(ctx, puuid); err == nil && cachedName != "" {
		log.Printf("Name already exists in cache for PUUID %s: %s", puuid[:30]+"...", cachedName)
		return true
	}
	return false
}

func cacheSummonerName(accountData *AccountData, puuid string, cacheManager *CacheManager, ctx context.Context) {
	if accountData.GameName != "" {
		fullName := buildFullName(accountData)

		if err := cacheManager.SetSummonerName(ctx, puuid, fullName); err != nil {
			log.Printf("Error caching summoner name: %v", err)
		} else {
			log.Printf("Name cached successfully: PUUID=%s, Name=%s", puuid[:30]+"...", fullName)
		}
	} else {
		log.Printf("GameName not found in account data: %+v", accountData)
	}
}

func buildFullName(accountData *AccountData) string {
	fullName := accountData.GameName
	if accountData.TagLine != "" {
		fullName = fmt.Sprintf("%s#%s", accountData.GameName, accountData.TagLine)
	}
	return fullName
}

func (nc *NATSClient) StartLeagueUpdateWorker(riotClient *RiotAPIClient, cacheManager *CacheManager) (*nats.Subscription, error) {
	handler := func(msg *nats.Msg) {
		processLeagueUpdateTask(msg, riotClient, cacheManager, nc)
	}

	sub, err := nc.Conn.QueueSubscribe("tft.league.update", "league-workers", handler)
	if err != nil {
		return nil, err
	}
	log.Println("League Update Worker started, waiting for messages...")
	return sub, nil
}

func processLeagueUpdateTask(msg *nats.Msg, riotClient *RiotAPIClient, cacheManager *CacheManager, nc *NATSClient) {
	var task LeagueUpdateTask
	if err := json.Unmarshal(msg.Data, &task); err != nil {
		log.Printf("Error unmarshaling league task: %v", err)
		return
	}

	log.Printf("Processing league update task: %+v", task)

	updateFuncs := map[string]func() error{
		"challenger":  func() error { return nc.updateChallengerLeague(riotClient, cacheManager, task.Region) },
		"grandmaster": func() error { return nc.updateGrandmasterLeague(riotClient, cacheManager, task.Region) },
		"master":      func() error { return nc.updateMasterLeague(riotClient, cacheManager, task.Region) },
	}

	if updateFunc, exists := updateFuncs[task.Type]; exists {
		if err := updateFunc(); err != nil {
			log.Printf("Error updating %s league: %v", task.Type, err)
		}
	} else {
		log.Printf("Unknown task type: %s", task.Type)
	}
}

func (nc *NATSClient) updateChallengerLeague(riotClient *RiotAPIClient, cacheManager *CacheManager, region string) error {
	result, err := riotClient.GetChallengerLeague()
	if err != nil {
		return err
	}

	return cacheLeagueResult(cacheManager, "challenger", region, result)
}

func (nc *NATSClient) updateGrandmasterLeague(riotClient *RiotAPIClient, cacheManager *CacheManager, region string) error {
	result, err := riotClient.GetGrandmasterLeague()
	if err != nil {
		return err
	}

	return cacheLeagueResult(cacheManager, "grandmaster", region, result)
}

func (nc *NATSClient) updateMasterLeague(riotClient *RiotAPIClient, cacheManager *CacheManager, region string) error {
	result, err := riotClient.GetMasterLeague()
	if err != nil {
		return err
	}

	return cacheLeagueResult(cacheManager, "master", region, result)
}

func cacheLeagueResult(cacheManager *CacheManager, leagueType, region string, result interface{}) error {
	ctx := context.Background()
	cacheKey := cacheManager.Key(leagueType, region)
	return cacheManager.Set(ctx, cacheKey, result, 30*time.Minute)
}
