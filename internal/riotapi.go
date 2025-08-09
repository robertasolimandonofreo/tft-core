package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"log"
	"time"
)

type RiotAPIClient struct {
	APIKey       string
	BaseURL      string
	AccountURL   string
	Client       *http.Client
	CacheManager *CacheManager
	Region       string
	NATSClient   *NATSClient
}

func NewRiotAPIClient(cfg *Config, cacheManager *CacheManager) *RiotAPIClient {
	accountURL := getAccountURL(cfg.RiotRegion)
	
	return &RiotAPIClient{
		APIKey:       cfg.RiotAPIKey,
		BaseURL:      cfg.RiotBaseURL,
		AccountURL:   accountURL,
		Region:       cfg.RiotRegion,
		CacheManager: cacheManager,
		Client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func getAccountURL(region string) string {
	routingValues := map[string]string{
		"BR1":  "americas",
		"LA1":  "americas",
		"LA2":  "americas",
		"NA1":  "americas",
		"EUW1": "europe",
		"EUN1": "europe",
		"TR1":  "europe",
		"RU":   "europe",
		"KR":   "asia",
		"JP1":  "asia",
		"OC1":  "sea",
		"PH2":  "sea",
		"SG2":  "sea",
		"TH2":  "sea",
		"TW2":  "sea",
		"VN2":  "sea",
	}
	
	routing := routingValues[region]
	if routing == "" {
		routing = "americas"
	}
	
	return fmt.Sprintf("https://%s.api.riotgames.com", routing)
}

func (c *RiotAPIClient) SetNATSClient(natsClient *NATSClient) {
	c.NATSClient = natsClient
}

func (c *RiotAPIClient) doRequest(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Riot-Token", c.APIKey)

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := ioutil.ReadAll(resp.Body)
		return nil, fmt.Errorf("Riot API error: %s - %s", resp.Status, string(body))
	}

	return ioutil.ReadAll(resp.Body)
}

func (c *RiotAPIClient) GetSummonerByPUUID(puuid string) (map[string]interface{}, error) {
	ctx := context.Background()
	cacheKey := c.CacheManager.GenerateKey("summoner", c.Region, puuid)
	
	var cachedResult map[string]interface{}
	if err := c.CacheManager.GetCachedData(ctx, cacheKey, &cachedResult); err == nil {
		return cachedResult, nil
	}
	
	url := fmt.Sprintf("%s/tft/summoner/v1/summoners/by-puuid/%s", c.BaseURL, puuid)
	data, err := c.doRequest(url)
	if err != nil {
		return nil, err
	}
	
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	
	c.CacheManager.SetCachedData(ctx, cacheKey, result, time.Hour)
	return result, nil
}

func (c *RiotAPIClient) GetAccountByPUUID(puuid string) (*AccountData, error) {
	ctx := context.Background()
	cacheKey := c.CacheManager.GenerateKey("account_puuid", c.Region, puuid)
	
	var cachedResult AccountData
	if err := c.CacheManager.GetCachedData(ctx, cacheKey, &cachedResult); err == nil {
		return &cachedResult, nil
	}
	
	url := fmt.Sprintf("%s/riot/account/v1/accounts/by-puuid/%s", c.AccountURL, puuid)
	data, err := c.doRequest(url)
	if err != nil {
		return nil, err
	}
	
	var result AccountData
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	
	c.CacheManager.SetCachedData(ctx, cacheKey, result, 6*time.Hour)
	return &result, nil
}

func (c *RiotAPIClient) GetAccountByGameName(gameName, tagLine string) (*AccountData, error) {
	ctx := context.Background()
	cacheKey := c.CacheManager.GenerateKey("account_name", c.Region, gameName, tagLine)
	
	var cachedResult AccountData
	if err := c.CacheManager.GetCachedData(ctx, cacheKey, &cachedResult); err == nil {
		return &cachedResult, nil
	}
	
	encodedGameName := url.QueryEscape(gameName)
	encodedTagLine := url.QueryEscape(tagLine)
	
	url := fmt.Sprintf("%s/riot/account/v1/accounts/by-riot-id/%s/%s", c.AccountURL, encodedGameName, encodedTagLine)
	data, err := c.doRequest(url)
	if err != nil {
		return nil, err
	}
	
	var result AccountData
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	
	c.CacheManager.SetCachedData(ctx, cacheKey, result, 6*time.Hour)
	return &result, nil
}

func (c *RiotAPIClient) GetSummonerByID(id string) (*Summoner, error) {
	ctx := context.Background()
	cacheKey := c.CacheManager.GenerateKey("summoner_id", c.Region, id)
	var cached Summoner
	if err := c.CacheManager.GetCachedData(ctx, cacheKey, &cached); err == nil {
		return &cached, nil
	}
	url := fmt.Sprintf("%s/tft/summoner/v1/summoners/%s", c.BaseURL, id)
	data, err := c.doRequest(url)
	if err != nil {
		return nil, err
	}
	var result Summoner
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	c.CacheManager.SetCachedData(ctx, cacheKey, result, time.Hour)
	return &result, nil
}

func (c *RiotAPIClient) GetChallengerLeague() (*ChallengerLeague, error) {
	ctx := context.Background()
	cacheKey := c.CacheManager.GenerateKey("challenger", c.Region)
	
	var cachedResult ChallengerLeague
	if err := c.CacheManager.GetCachedData(ctx, cacheKey, &cachedResult); err == nil {
		cachedResult.Entries = c.enrichLeagueEntriesNames(cachedResult.Entries)
		_ = c.CacheManager.SetCachedData(ctx, cacheKey, cachedResult, 30*time.Minute)
		return &cachedResult, nil
	}
	
	url := fmt.Sprintf("%s/tft/league/v1/challenger", c.BaseURL)
	data, err := c.doRequest(url)
	if err != nil {
		return nil, err
	}
	
	var result ChallengerLeague
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	result.Entries = c.enrichLeagueEntriesNames(result.Entries)
	
	c.CacheManager.SetCachedData(ctx, cacheKey, result, 30*time.Minute)
	return &result, nil
}

func (c *RiotAPIClient) GetGrandmasterLeague() (*GrandmasterLeague, error) {
	ctx := context.Background()
	cacheKey := c.CacheManager.GenerateKey("grandmaster", c.Region)
	
	var cachedResult GrandmasterLeague
	if err := c.CacheManager.GetCachedData(ctx, cacheKey, &cachedResult); err == nil {
		cachedResult.Entries = c.enrichLeagueEntriesNames(cachedResult.Entries)
		_ = c.CacheManager.SetCachedData(ctx, cacheKey, cachedResult, 30*time.Minute)
		return &cachedResult, nil
	}
	
	url := fmt.Sprintf("%s/tft/league/v1/grandmaster", c.BaseURL)
	data, err := c.doRequest(url)
	if err != nil {
		return nil, err
	}
	
	var result GrandmasterLeague
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	result.Entries = c.enrichLeagueEntriesNames(result.Entries)
	
	c.CacheManager.SetCachedData(ctx, cacheKey, result, 30*time.Minute)
	return &result, nil
}

func (c *RiotAPIClient) GetMasterLeague() (*MasterLeague, error) {
	ctx := context.Background()
	cacheKey := c.CacheManager.GenerateKey("master", c.Region)
	
	var cachedResult MasterLeague
	if err := c.CacheManager.GetCachedData(ctx, cacheKey, &cachedResult); err == nil {
		cachedResult.Entries = c.enrichLeagueEntriesNames(cachedResult.Entries)
		_ = c.CacheManager.SetCachedData(ctx, cacheKey, cachedResult, 30*time.Minute)
		return &cachedResult, nil
	}
	
	url := fmt.Sprintf("%s/tft/league/v1/master", c.BaseURL)
	data, err := c.doRequest(url)
	if err != nil {
		return nil, err
	}
	
	var result MasterLeague
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	result.Entries = c.enrichLeagueEntriesNames(result.Entries)
	
	c.CacheManager.SetCachedData(ctx, cacheKey, result, 30*time.Minute)
	return &result, nil
}

func (c *RiotAPIClient) GetLeagueEntries(tier, division string, page int) (*LeagueEntriesResponse, error) {
	ctx := context.Background()
	cacheKey := c.CacheManager.GenerateKey("entries", c.Region, tier, division, strconv.Itoa(page))
	
	var cachedResult LeagueEntriesResponse
	if err := c.CacheManager.GetCachedData(ctx, cacheKey, &cachedResult); err == nil {
		cachedResult.Entries = c.enrichLeagueEntriesNames(cachedResult.Entries)
		_ = c.CacheManager.SetCachedData(ctx, cacheKey, cachedResult, 30*time.Minute)
		return &cachedResult, nil
	}
	
	url := fmt.Sprintf("%s/tft/league/v1/entries/%s/%s?page=%d", c.BaseURL, tier, division, page)
	data, err := c.doRequest(url)
	if err != nil {
		return nil, err
	}
	
	var entries []LeagueEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	
	entries = c.enrichLeagueEntriesNames(entries)

	result := &LeagueEntriesResponse{
		Entries:  entries,
		Page:     page,
		Tier:     tier,
		Division: division,
		HasMore:  len(entries) == 200,
	}
	
	c.CacheManager.SetCachedData(ctx, cacheKey, result, 30*time.Minute)
	return result, nil
}

func (c *RiotAPIClient) GetLeagueByPUUID(puuid string) ([]LeagueEntry, error) {
	ctx := context.Background()
	cacheKey := c.CacheManager.GenerateKey("league_by_puuid", c.Region, puuid)
	
	var cachedResult []LeagueEntry
	if err := c.CacheManager.GetCachedData(ctx, cacheKey, &cachedResult); err == nil {
		return cachedResult, nil
	}
	
	url := fmt.Sprintf("%s/tft/league/v1/by-puuid/%s", c.BaseURL, puuid)
	data, err := c.doRequest(url)
	if err != nil {
		return nil, err
	}
	
	var result []LeagueEntry
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	
	c.CacheManager.SetCachedData(ctx, cacheKey, result, time.Hour)
	return result, nil
}

func (c *RiotAPIClient) GetMatchByID(matchId string) (map[string]interface{}, error) {
	ctx := context.Background()
	cacheKey := c.CacheManager.GenerateKey("match", c.Region, matchId)
	
	var cachedResult map[string]interface{}
	if err := c.CacheManager.GetCachedData(ctx, cacheKey, &cachedResult); err == nil {
		return cachedResult, nil
	}
	
	url := fmt.Sprintf("%s/tft/match/v1/matches/%s", c.BaseURL, matchId)
	data, err := c.doRequest(url)
	if err != nil {
		return nil, err
	}
	
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	
	c.CacheManager.SetCachedData(ctx, cacheKey, result, 0)
	return result, nil
}

func (c *RiotAPIClient) enrichLeagueEntriesNames(entries []LeagueEntry) []LeagueEntry {
	ctx := context.Background()
	
	maxEntries := 10
	if len(entries) > maxEntries {
		entries = entries[:maxEntries]
	}
	
	lookups := 0
	maxLookups := len(entries)

	log.Printf("Enriching TOP %d entries with names (max %d lookups)", len(entries), maxLookups)

	for i := range entries {
		if lookups >= maxLookups {
			log.Printf("Atingido limite de lookups (%d), enviando resto para workers", maxLookups)
			
			for j := i; j < len(entries); j++ {
				entry := &entries[j]
				puuid := entry.GetUniqueID()
				if puuid != "" && c.NATSClient != nil {
					task := SummonerNameTask{
						PUUID:  puuid,
						Region: c.Region,
					}
					c.NATSClient.PublishSummonerNameTask(task)
				}
			}
			break
		}

		entry := &entries[i]
		puuid := entry.GetUniqueID()

		if puuid == "" {
			continue
		}

		if cachedName, err := c.CacheManager.GetSummonerName(ctx, puuid); err == nil && cachedName != "" {
			entry.SummonerName = cachedName
			log.Printf("Nome do cache: %s", cachedName)
			continue
		}

		if entry.SummonerName == "" {
			log.Printf("Buscando nome via Account API para PUUID: %s", puuid[:30]+"...")
			
			if lookups > 0 {
				time.Sleep(50 * time.Millisecond)
			}
			
			accountData, err := c.GetAccountByPUUID(puuid)
			if err == nil && accountData.GameName != "" {
				fullName := accountData.GameName
				if accountData.TagLine != "" {
					fullName = fmt.Sprintf("%s#%s", accountData.GameName, accountData.TagLine)
				}
				
				entry.SummonerName = fullName
				c.CacheManager.SetSummonerName(ctx, puuid, fullName)
				log.Printf("✅ Nome obtido: %s", fullName)
				lookups++
			} else {
				log.Printf("❌ Erro ao buscar account: %v", err)
				
				if c.NATSClient != nil {
					task := SummonerNameTask{
						PUUID:  puuid,
						Region: c.Region,
					}
					c.NATSClient.PublishSummonerNameTask(task)
				}
			}
		}
	}

	log.Printf("Enrichment concluído. Realizados %d lookups para TOP %d", lookups, len(entries))
	return entries
}