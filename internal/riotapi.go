package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"time"
)

type RiotAPIClient struct {
	APIKey       string
	BaseURL      string
	Client       *http.Client
	CacheManager *CacheManager
	Region       string
}

func NewRiotAPIClient(cfg *Config, cacheManager *CacheManager) *RiotAPIClient {
	return &RiotAPIClient{
		APIKey:       cfg.RiotAPIKey,
		BaseURL:      cfg.RiotBaseURL,
		Region:       cfg.RiotRegion,
		CacheManager: cacheManager,
		Client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

func (c *RiotAPIClient) doRequest(path string) ([]byte, error) {
	url := fmt.Sprintf("%s%s", c.BaseURL, path)
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
	
	path := fmt.Sprintf("/tft/summoner/v1/summoners/by-puuid/%s", puuid)
	data, err := c.doRequest(path)
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

func (c *RiotAPIClient) GetChallengerLeague() (*ChallengerLeague, error) {
	ctx := context.Background()
	cacheKey := c.CacheManager.GenerateKey("challenger", c.Region)
	
	var cachedResult ChallengerLeague
	if err := c.CacheManager.GetCachedData(ctx, cacheKey, &cachedResult); err == nil {
		return &cachedResult, nil
	}
	
	path := "/tft/league/v1/challenger"
	data, err := c.doRequest(path)
	if err != nil {
		return nil, err
	}
	
	var result ChallengerLeague
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	
	c.CacheManager.SetCachedData(ctx, cacheKey, result, 30*time.Minute)
	return &result, nil
}

func (c *RiotAPIClient) GetGrandmasterLeague() (*GrandmasterLeague, error) {
	ctx := context.Background()
	cacheKey := c.CacheManager.GenerateKey("grandmaster", c.Region)
	
	var cachedResult GrandmasterLeague
	if err := c.CacheManager.GetCachedData(ctx, cacheKey, &cachedResult); err == nil {
		return &cachedResult, nil
	}
	
	path := "/tft/league/v1/grandmaster"
	data, err := c.doRequest(path)
	if err != nil {
		return nil, err
	}
	
	var result GrandmasterLeague
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	
	c.CacheManager.SetCachedData(ctx, cacheKey, result, 30*time.Minute)
	return &result, nil
}

func (c *RiotAPIClient) GetMasterLeague() (*MasterLeague, error) {
	ctx := context.Background()
	cacheKey := c.CacheManager.GenerateKey("master", c.Region)
	
	var cachedResult MasterLeague
	if err := c.CacheManager.GetCachedData(ctx, cacheKey, &cachedResult); err == nil {
		return &cachedResult, nil
	}
	
	path := "/tft/league/v1/master"
	data, err := c.doRequest(path)
	if err != nil {
		return nil, err
	}
	
	var result MasterLeague
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	
	c.CacheManager.SetCachedData(ctx, cacheKey, result, 30*time.Minute)
	return &result, nil
}

func (c *RiotAPIClient) GetLeagueEntries(tier, division string, page int) (*LeagueEntriesResponse, error) {
	ctx := context.Background()
	cacheKey := c.CacheManager.GenerateKey("entries", c.Region, tier, division, strconv.Itoa(page))
	
	var cachedResult LeagueEntriesResponse
	if err := c.CacheManager.GetCachedData(ctx, cacheKey, &cachedResult); err == nil {
		return &cachedResult, nil
	}
	
	path := fmt.Sprintf("/tft/league/v1/entries/%s/%s?page=%d", tier, division, page)
	data, err := c.doRequest(path)
	if err != nil {
		return nil, err
	}
	
	var entries []LeagueEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, err
	}
	
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
	
	path := fmt.Sprintf("/tft/league/v1/by-puuid/%s", puuid)
	data, err := c.doRequest(path)
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

func (c *RiotAPIClient) GetRatedLadderTop(queue string) (*RatedLadder, error) {
	ctx := context.Background()
	cacheKey := c.CacheManager.GenerateKey("rated_ladder", c.Region, queue)
	
	var cachedResult RatedLadder
	if err := c.CacheManager.GetCachedData(ctx, cacheKey, &cachedResult); err == nil {
		return &cachedResult, nil
	}
	
	path := fmt.Sprintf("/tft/league/v1/rated-ladders/%s/top", queue)
	data, err := c.doRequest(path)
	if err != nil {
		return nil, err
	}
	
	var result RatedLadder
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	
	c.CacheManager.SetCachedData(ctx, cacheKey, result, time.Hour)
	return &result, nil
}

func (c *RiotAPIClient) GetMatchByID(matchId string) (map[string]interface{}, error) {
	ctx := context.Background()
	cacheKey := c.CacheManager.GenerateKey("match", c.Region, matchId)
	
	var cachedResult map[string]interface{}
	if err := c.CacheManager.GetCachedData(ctx, cacheKey, &cachedResult); err == nil {
		return cachedResult, nil
	}
	
	path := fmt.Sprintf("/tft/match/v1/matches/%s", matchId)
	data, err := c.doRequest(path)
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