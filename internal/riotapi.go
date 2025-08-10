package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

type RiotAPIClient struct {
	apiKey       string
	baseURL      string
	accountURL   string
	client       *http.Client
	cache        *CacheManager
	region       string
	natsClient   *NATSClient
}

func NewRiotAPIClient(cfg *Config, cache *CacheManager) *RiotAPIClient {
	return &RiotAPIClient{
		apiKey:     cfg.RiotAPIKey,
		baseURL:    cfg.RiotBaseURL,
		accountURL: getAccountAPIURL(cfg.RiotRegion),
		region:     cfg.RiotRegion,
		cache:      cache,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func getAccountAPIURL(region string) string {
	regions := map[string]string{
		"BR1": "https://americas.api.riotgames.com",
		"LA1": "https://americas.api.riotgames.com", 
		"LA2": "https://americas.api.riotgames.com",
		"NA1": "https://americas.api.riotgames.com",
		"EUW1": "https://europe.api.riotgames.com",
		"EUN1": "https://europe.api.riotgames.com",
		"TR1": "https://europe.api.riotgames.com",
		"RU": "https://europe.api.riotgames.com",
		"JP1": "https://asia.api.riotgames.com",
		"KR": "https://asia.api.riotgames.com",
		"OC1": "https://sea.api.riotgames.com",
	}
	
	if url, exists := regions[region]; exists {
		return url
	}
	return "https://americas.api.riotgames.com"
}

func (c *RiotAPIClient) SetNATSClient(natsClient *NATSClient) {
	c.natsClient = natsClient
}

func (c *RiotAPIClient) doRequest(url string) ([]byte, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Riot-Token", c.apiKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("riot API error: %s - %s", resp.Status, string(body))
	}

	return body, nil
}

func (c *RiotAPIClient) GetSummonerByPUUID(puuid string) (map[string]interface{}, error) {
	ctx := context.Background()
	cacheKey := c.cache.Key("summoner", c.region, puuid)

	var cached map[string]interface{}
	if err := c.cache.Get(ctx, cacheKey, &cached); err == nil {
		return cached, nil
	}

	url := fmt.Sprintf("%s/tft/summoner/v1/summoners/by-puuid/%s", c.baseURL, puuid)
	data, err := c.doRequest(url)
	if err != nil {
		return nil, err
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	c.cache.Set(ctx, cacheKey, result, time.Hour)
	return result, nil
}

func (c *RiotAPIClient) GetAccountByPUUID(puuid string) (*AccountData, error) {
	ctx := context.Background()
	cacheKey := c.cache.Key("account_puuid", c.region, puuid)

	var cached AccountData
	if err := c.cache.Get(ctx, cacheKey, &cached); err == nil {
		return &cached, nil
	}

	url := fmt.Sprintf("%s/riot/account/v1/accounts/by-puuid/%s", c.accountURL, puuid)
	data, err := c.doRequest(url)
	if err != nil {
		return nil, err
	}

	var result AccountData
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	c.cache.Set(ctx, cacheKey, result, 6*time.Hour)
	return &result, nil
}

func (c *RiotAPIClient) GetAccountByGameName(gameName, tagLine string) (*AccountData, error) {
	ctx := context.Background()
	
	cleanGameName := strings.TrimSpace(gameName)
	cleanTagLine := strings.TrimSpace(tagLine)
	
	if cleanGameName == "" {
		return nil, fmt.Errorf("gameName cannot be empty")
	}
	if cleanTagLine == "" {
		cleanTagLine = "BR1"
	}

	cacheKey := c.cache.Key("account_name", c.region, cleanGameName, cleanTagLine)
	
	var cached AccountData
	if err := c.cache.Get(ctx, cacheKey, &cached); err == nil {
		return &cached, nil
	}

	encodedGameName := strings.ReplaceAll(url.QueryEscape(cleanGameName), "+", "%20")
	encodedTagLine := strings.ReplaceAll(url.QueryEscape(cleanTagLine), "+", "%20")
	
	apiURL := fmt.Sprintf("%s/riot/account/v1/accounts/by-riot-id/%s/%s", 
		c.accountURL, encodedGameName, encodedTagLine)
		
	data, err := c.doRequest(apiURL)
	if err != nil {
		return nil, err
	}

	var result AccountData
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	if result.PUUID == "" {
		return nil, fmt.Errorf("invalid account data: empty PUUID")
	}

	c.cache.Set(ctx, cacheKey, result, 6*time.Hour)
	return &result, nil
}

func (c *RiotAPIClient) GetChallengerLeague() (*ChallengerLeague, error) {
	return c.getHighTierLeague("challenger", "CHALLENGER")
}

func (c *RiotAPIClient) GetGrandmasterLeague() (*GrandmasterLeague, error) {
	result, err := c.getHighTierLeague("grandmaster", "GRANDMASTER")
	if err != nil {
		return nil, err
	}
	return &GrandmasterLeague{
		LeagueID: result.LeagueID,
		Entries:  result.Entries,
		Tier:     result.Tier,
		Name:     result.Name,
		Queue:    result.Queue,
	}, nil
}

func (c *RiotAPIClient) GetMasterLeague() (*MasterLeague, error) {
	result, err := c.getHighTierLeague("master", "MASTER")
	if err != nil {
		return nil, err
	}
	return &MasterLeague{
		LeagueID: result.LeagueID,
		Entries:  result.Entries,
		Tier:     result.Tier,
		Name:     result.Name,
		Queue:    result.Queue,
	}, nil
}

func (c *RiotAPIClient) getHighTierLeague(endpoint, tier string) (*ChallengerLeague, error) {
	ctx := context.Background()
	cacheKey := c.cache.Key(endpoint, c.region)

	var cached ChallengerLeague
	if err := c.cache.Get(ctx, cacheKey, &cached); err == nil {
		if len(cached.Entries) > 10 {
			cached.Entries = cached.Entries[:10]
		}
		c.enrichEntries(cached.Entries, tier)
		return &cached, nil
	}

	url := fmt.Sprintf("%s/tft/league/v1/%s", c.baseURL, endpoint)
	data, err := c.doRequest(url)
	if err != nil {
		return nil, err
	}

	var result ChallengerLeague
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	if len(result.Entries) > 10 {
		result.Entries = result.Entries[:10]
	}

	c.enrichEntries(result.Entries, tier)
	c.cache.Set(ctx, cacheKey, result, 30*time.Minute)
	return &result, nil
}

func (c *RiotAPIClient) GetLeagueEntries(tier, division string, page int) (*LeagueEntriesResponse, error) {
	ctx := context.Background()
	cacheKey := c.cache.Key("entries", c.region, tier, division, strconv.Itoa(page))

	var cached LeagueEntriesResponse
	if err := c.cache.Get(ctx, cacheKey, &cached); err == nil {
		c.enrichEntries(cached.Entries, tier)
		return &cached, nil
	}

	url := fmt.Sprintf("%s/tft/league/v1/entries/%s/%s?page=%d", c.baseURL, tier, division, page)
	data, err := c.doRequest(url)
	if err != nil {
		return nil, err
	}

	var entries []LeagueEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}

	c.enrichEntries(entries, tier)

	result := &LeagueEntriesResponse{
		Entries:  entries,
		Page:     page,
		Tier:     tier,
		Division: division,
		HasMore:  len(entries) == 200,
	}

	c.cache.Set(ctx, cacheKey, result, 30*time.Minute)
	return result, nil
}

func (c *RiotAPIClient) GetLeagueByPUUID(puuid string) ([]LeagueEntry, error) {
	ctx := context.Background()
	cacheKey := c.cache.Key("league_by_puuid", c.region, puuid)

	var cached []LeagueEntry
	if err := c.cache.Get(ctx, cacheKey, &cached); err == nil {
		return cached, nil
	}

	url := fmt.Sprintf("%s/tft/league/v1/by-puuid/%s", c.baseURL, puuid)
	data, err := c.doRequest(url)
	if err != nil {
		return nil, err
	}

	var result []LeagueEntry
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	c.cache.Set(ctx, cacheKey, result, time.Hour)
	return result, nil
}

func (c *RiotAPIClient) enrichEntries(entries []LeagueEntry, tier string) {
	ctx := context.Background()
	
	for i := range entries {
		entries[i].Tier = tier
		
		if entries[i].SummonerName != "" && 
		   entries[i].SummonerName != "Unknown" && 
		   entries[i].SummonerName != "Loading..." &&
		   entries[i].SummonerName != "" {
			continue
		}

		if entries[i].PUUID == "" {
			entries[i].SummonerName = "Unknown"
			continue
		}

		name, err := c.cache.GetSummonerName(ctx, entries[i].PUUID)
		if err == nil && name != "" {
			entries[i].SummonerName = name
			continue
		}
			
		if c.natsClient != nil {
			task := SummonerNameTask{
				PUUID:  entries[i].PUUID,
				Region: c.region,
			}
			c.natsClient.PublishSummonerNameTask(task)
		}

		entries[i].SummonerName = "Loading..."
	}
}