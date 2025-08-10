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
	accountURL := getAccountAPIURL(cfg.RiotRegion)
	
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

func getAccountAPIURL(region string) string {
	switch region {
	case "BR1", "LA1", "LA2", "NA1":
		return "https://americas.api.riotgames.com"
	case "EUW1", "EUN1", "TR1", "RU":
		return "https://europe.api.riotgames.com"
	case "JP1", "KR":
		return "https://asia.api.riotgames.com"
	case "OC1":
		return "https://sea.api.riotgames.com"
	default:
		return "https://americas.api.riotgames.com"
	}
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
	
	cleanGameName := strings.TrimSpace(gameName)
	cleanTagLine := strings.TrimSpace(tagLine)
	
	if cleanGameName == "" {
		return nil, fmt.Errorf("gameName cannot be empty")
	}
	
	if cleanTagLine == "" {
		cleanTagLine = "BR1"
	}
	
	cacheKey := c.CacheManager.GenerateKey("account_name", c.Region, cleanGameName, cleanTagLine)
	
	var cachedResult AccountData
	if err := c.CacheManager.GetCachedData(ctx, cacheKey, &cachedResult); err == nil {
		log.Printf("Cache hit for account: %s#%s", cleanGameName, cleanTagLine)
		return &cachedResult, nil
	}
	
	encodedGameName := strings.ReplaceAll(url.QueryEscape(cleanGameName), "+", "%20")
	encodedTagLine := strings.ReplaceAll(url.QueryEscape(cleanTagLine), "+", "%20")
	
	apiURL := fmt.Sprintf("%s/riot/account/v1/accounts/by-riot-id/%s/%s", c.AccountURL, encodedGameName, encodedTagLine)
	
	log.Printf("Searching account: '%s#%s'", cleanGameName, cleanTagLine)
	log.Printf("API URL: %s", apiURL)
	log.Printf("Encoded: gameName='%s', tagLine='%s'", encodedGameName, encodedTagLine)
	
	data, err := c.doRequest(apiURL)
	if err != nil && strings.Contains(err.Error(), "404") {
		log.Printf("First attempt failed, trying case variations...")
		
		lowerGameName := strings.ToLower(cleanGameName)
		lowerTagLine := strings.ToLower(cleanTagLine)
		
		if lowerGameName != cleanGameName || lowerTagLine != cleanTagLine {
			encodedLowerGameName := strings.ReplaceAll(url.QueryEscape(lowerGameName), "+", "%20")
			encodedLowerTagLine := strings.ReplaceAll(url.QueryEscape(lowerTagLine), "+", "%20")
			
			lowerURL := fmt.Sprintf("%s/riot/account/v1/accounts/by-riot-id/%s/%s", c.AccountURL, encodedLowerGameName, encodedLowerTagLine)
			
			log.Printf("Trying lowercase: %s#%s", lowerGameName, lowerTagLine)
			log.Printf("Lower URL: %s", lowerURL)
			
			data, err = c.doRequest(lowerURL)
			if err == nil {
				log.Printf("Success with lowercase!")
			}
		}
		
		if err != nil && strings.Contains(err.Error(), "404") {
			variations := [][]string{
				{strings.Title(cleanGameName), strings.ToUpper(cleanTagLine)}, // Title Case + UPPER
				{strings.ToUpper(cleanGameName), strings.ToUpper(cleanTagLine)}, // ALL UPPER
			}
			
			for _, variant := range variations {
				varGameName, varTagLine := variant[0], variant[1]
				if varGameName == cleanGameName && varTagLine == cleanTagLine {
					continue
				}
				
				encodedVarGameName := strings.ReplaceAll(url.QueryEscape(varGameName), "+", "%20")
				encodedVarTagLine := strings.ReplaceAll(url.QueryEscape(varTagLine), "+", "%20")
				
				varURL := fmt.Sprintf("%s/riot/account/v1/accounts/by-riot-id/%s/%s", c.AccountURL, encodedVarGameName, encodedVarTagLine)
				
				log.Printf("Trying variation: %s#%s", varGameName, varTagLine)
				log.Printf("Variant URL: %s", varURL)
				
				data, err = c.doRequest(varURL)
				if err == nil {
					log.Printf("Success with variation: %s#%s", varGameName, varTagLine)
					cleanGameName, cleanTagLine = varGameName, varTagLine
					break
				}
			}
		}
	}
	
	if err != nil {
		log.Printf("Account API error for %s#%s: %v", cleanGameName, cleanTagLine, err)
		return nil, err
	}
	
	var result AccountData
	if err := json.Unmarshal(data, &result); err != nil {
		log.Printf("JSON unmarshal error for %s#%s: %v", cleanGameName, cleanTagLine, err)
		log.Printf("Raw response: %s", string(data))
		return nil, err
	}
	
	if result.PUUID == "" {
		log.Printf("Empty PUUID in response for %s#%s", cleanGameName, cleanTagLine)
		log.Printf("Response data: %+v", result)
		return nil, fmt.Errorf("invalid account data: empty PUUID")
	}
	
	log.Printf("Account found: %s#%s -> PUUID: %s", result.GameName, result.TagLine, result.PUUID)
	
	cacheKey = c.CacheManager.GenerateKey("account_name", c.Region, cleanGameName, cleanTagLine)
	c.CacheManager.SetCachedData(ctx, cacheKey, result, 6*time.Hour)
	return &result, nil
}

func (c *RiotAPIClient) GetChallengerLeague() (*ChallengerLeague, error) {
	ctx := context.Background()
	cacheKey := c.CacheManager.GenerateKey("challenger", c.Region)
	
	var cachedResult ChallengerLeague
	if err := c.CacheManager.GetCachedData(ctx, cacheKey, &cachedResult); err == nil {
		log.Printf("Cache hit Challenger: %d entries", len(cachedResult.Entries))
		if len(cachedResult.Entries) > 10 {
			cachedResult.Entries = cachedResult.Entries[:10]
		}
		for i := range cachedResult.Entries {
			cachedResult.Entries[i].Tier = "CHALLENGER"
		}
		cachedResult.Entries = c.enrichLeagueEntriesNames(cachedResult.Entries)
		return &cachedResult, nil
	}
	
	url := fmt.Sprintf("%s/tft/league/v1/challenger", c.BaseURL)
	data, err := c.doRequest(url)
	if err != nil {
		return nil, err
	}
	
	log.Printf("Raw Challenger API Response (first 500 chars): %s", string(data)[:min(500, len(data))])
	
	var result ChallengerLeague
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	
	log.Printf("Challenger API response: %d entries", len(result.Entries))
	
	if len(result.Entries) > 10 {
		log.Printf("Cutting Challenger from %d to top 10", len(result.Entries))
		result.Entries = result.Entries[:10]
	}
	
	for i := range result.Entries {
		result.Entries[i].Tier = "CHALLENGER"
		log.Printf("Challenger Entry %d tier: %s, PUUID: %s", i, result.Entries[i].Tier, result.Entries[i].PUUID[:30]+"...")
	}
	
	if len(result.Entries) > 0 {
		firstEntry := result.Entries[0]
		log.Printf("First Challenger entry - Tier: %s, PUUID: %s, SummonerID: %s", firstEntry.Tier, firstEntry.PUUID, firstEntry.SummonerID)
	}
	
	result.Entries = c.enrichLeagueEntriesNames(result.Entries)
	c.CacheManager.SetCachedData(ctx, cacheKey, result, 30*time.Minute)
	return &result, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (c *RiotAPIClient) GetGrandmasterLeague() (*GrandmasterLeague, error) {
	ctx := context.Background()
	cacheKey := c.CacheManager.GenerateKey("grandmaster", c.Region)
	
	var cachedResult GrandmasterLeague
	if err := c.CacheManager.GetCachedData(ctx, cacheKey, &cachedResult); err == nil {
		log.Printf("Cache hit Grandmaster: %d entries", len(cachedResult.Entries))
		if len(cachedResult.Entries) > 10 {
			cachedResult.Entries = cachedResult.Entries[:10]
		}
		for i := range cachedResult.Entries {
			cachedResult.Entries[i].Tier = "GRANDMASTER"
		}
		cachedResult.Entries = c.enrichLeagueEntriesNames(cachedResult.Entries)
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
	
	log.Printf("Grandmaster API response: %d entries", len(result.Entries))
	
	if len(result.Entries) > 10 {
		log.Printf("Cutting Grandmaster from %d to top 10", len(result.Entries))
		result.Entries = result.Entries[:10]
	}
	
	for i := range result.Entries {
		result.Entries[i].Tier = "GRANDMASTER"
		log.Printf("Grandmaster Entry %d tier: %s", i, result.Entries[i].Tier)
	}
	
	if len(result.Entries) > 0 {
		log.Printf("GM first entry - Tier: %s, PUUID: %s", result.Entries[0].Tier, result.Entries[0].PUUID)
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
		log.Printf("Cache hit Master: %d entries", len(cachedResult.Entries))
		if len(cachedResult.Entries) > 10 {
			cachedResult.Entries = cachedResult.Entries[:10]
		}
		for i := range cachedResult.Entries {
			cachedResult.Entries[i].Tier = "MASTER"
		}
		cachedResult.Entries = c.enrichLeagueEntriesNames(cachedResult.Entries)
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
	
	log.Printf("Master API response: %d entries", len(result.Entries))
	
	if len(result.Entries) > 10 {
		log.Printf("Cutting Master from %d to top 10", len(result.Entries))
		result.Entries = result.Entries[:10]
	}
	
	for i := range result.Entries {
		result.Entries[i].Tier = "MASTER"
		log.Printf("Master Entry %d tier: %s", i, result.Entries[i].Tier)
	}
	
	if len(result.Entries) > 0 {
		log.Printf("Master first entry - Tier: %s, PUUID: %s", result.Entries[0].Tier, result.Entries[0].PUUID)
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
	
	totalEntries := len(entries)
	maxEntries := 10
	if totalEntries > maxEntries {
		log.Printf("Limiting to TOP %d entries (of %d total)", maxEntries, totalEntries)
		entries = entries[:maxEntries]
	}
	
	lookups := 0
	maxLookups := 10
	cacheHits := 0
	errors := 0

	log.Printf("Starting enrichment TOP %d entries", len(entries))

	for i := range entries {
		entry := &entries[i]

		log.Printf("Entry %d: Tier: %s, PUUID: %s", i, entry.Tier, entry.PUUID[:30]+"...")

		if entry.SummonerName != "" && entry.SummonerName != "Unknown" {
			continue
		}

		puuid := entry.PUUID
		if puuid == "" {
			log.Printf("Entry %d: Empty PUUID", i)
			continue
		}

		if cachedName, err := c.CacheManager.GetSummonerName(ctx, puuid); err == nil && cachedName != "" {
			entry.SummonerName = cachedName
			cacheHits++
			log.Printf("Entry %d: Cache hit: %s (Tier: %s)", i, cachedName, entry.Tier)
			continue
		}

		if lookups >= maxLookups {
			log.Printf("Entry %d: Limit reached, using async worker", i)
			
			if c.NATSClient != nil {
				go func(puuidCopy string) {
					c.fetchNameAsyncViaPUUID(puuidCopy)
				}(puuid)
			}
			continue
		}

		if lookups > 0 {
			time.Sleep(150 * time.Millisecond)
		}
		
		log.Printf("Entry %d: Searching via Account API...", i)
		name := c.fetchNameDirectlyViaPUUID(puuid)
		
		if name != "" {
			entry.SummonerName = name
			c.CacheManager.SetSummonerName(ctx, puuid, name)
			log.Printf("Entry %d: Name obtained: %s (Tier: %s)", i, name, entry.Tier)
		} else {
			log.Printf("Entry %d: Error obtaining name", i)
			errors++
			
			if c.NATSClient != nil {
				go func(puuidCopy string) {
					c.fetchNameAsyncViaPUUID(puuidCopy)
				}(puuid)
			}
		}
		lookups++
	}

	successfulNames := 0
	for _, entry := range entries {
		if entry.SummonerName != "" && entry.SummonerName != "Unknown" {
			successfulNames++
		}
	}

	log.Printf("TOP %d Enrichment completed - Names: %d/%d, Cache: %d, Lookups: %d, Errors: %d", 
		len(entries), successfulNames, len(entries), cacheHits, lookups, errors)
	
	return entries
}

func (c *RiotAPIClient) fetchNameDirectlyViaPUUID(puuid string) string {
	log.Printf("Searching Account via PUUID: %s", puuid[:30]+"...")
	
	accountData, err := c.GetAccountByPUUID(puuid)
	if err != nil {
		log.Printf("Error Account API: %v", err)
		return ""
	}

	if accountData.GameName == "" {
		log.Printf("GameName empty in response")
		return ""
	}

	fullName := accountData.GameName
	if accountData.TagLine != "" {
		fullName = fmt.Sprintf("%s#%s", accountData.GameName, accountData.TagLine)
	}
	
	log.Printf("Name obtained: %s", fullName)
	return fullName
}

func (c *RiotAPIClient) fetchNameAsyncViaPUUID(puuid string) {
	ctx := context.Background()
	
	if cachedName, err := c.CacheManager.GetSummonerName(ctx, puuid); err == nil && cachedName != "" {
		log.Printf("Name already in cache: %s", cachedName)
		return
	}
	
	name := c.fetchNameDirectlyViaPUUID(puuid)
	
	if name != "" {
		c.CacheManager.SetSummonerName(ctx, puuid, name)
		log.Printf("Name cached async: %s", name)
	}
}