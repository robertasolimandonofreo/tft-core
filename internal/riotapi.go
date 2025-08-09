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

func (c *RiotAPIClient) GetChallengerLeague() (*ChallengerLeague, error) {
	ctx := context.Background()
	cacheKey := c.CacheManager.GenerateKey("challenger", c.Region)
	
	var cachedResult ChallengerLeague
	if err := c.CacheManager.GetCachedData(ctx, cacheKey, &cachedResult); err == nil {
		log.Printf("💾 Cache hit Challenger: %d entries", len(cachedResult.Entries))
		if len(cachedResult.Entries) > 10 {
			cachedResult.Entries = cachedResult.Entries[:10]
		}
		// ✅ GARANTIR QUE CADA ENTRY TENHA O TIER CORRETO
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
	
	log.Printf("🔍 Raw Challenger API Response (primeiros 500 chars): %s", string(data)[:min(500, len(data))])
	
	var result ChallengerLeague
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	
	log.Printf("📊 Challenger API response: %d entries", len(result.Entries))
	
	if len(result.Entries) > 10 {
		log.Printf("🔪 Cortando Challenger de %d para TOP 10", len(result.Entries))
		result.Entries = result.Entries[:10]
	}
	
	// ✅ GARANTIR QUE CADA ENTRY TENHA O TIER CORRETO
	for i := range result.Entries {
		result.Entries[i].Tier = "CHALLENGER"
		log.Printf("🏆 Challenger Entry %d tier: %s, PUUID: %s", i, result.Entries[i].Tier, result.Entries[i].PUUID[:30]+"...")
	}
	
	if len(result.Entries) > 0 {
		firstEntry := result.Entries[0]
		log.Printf("🔍 Primeiro Challenger entry - Tier: %s, PUUID: %s, SummonerID: %s", firstEntry.Tier, firstEntry.PUUID, firstEntry.SummonerID)
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
		log.Printf("💾 Cache hit Grandmaster: %d entries", len(cachedResult.Entries))
		if len(cachedResult.Entries) > 10 {
			cachedResult.Entries = cachedResult.Entries[:10]
		}
		// ✅ GARANTIR QUE CADA ENTRY TENHA O TIER CORRETO
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
	
	log.Printf("📊 Grandmaster API response: %d entries", len(result.Entries))
	
	if len(result.Entries) > 10 {
		log.Printf("🔪 Cortando Grandmaster de %d para TOP 10", len(result.Entries))
		result.Entries = result.Entries[:10]
	}
	
	// ✅ GARANTIR QUE CADA ENTRY TENHA O TIER CORRETO
	for i := range result.Entries {
		result.Entries[i].Tier = "GRANDMASTER"
		log.Printf("🏆 Grandmaster Entry %d tier: %s", i, result.Entries[i].Tier)
	}
	
	if len(result.Entries) > 0 {
		log.Printf("🔍 GM primeiro entry - Tier: %s, PUUID: %s", result.Entries[0].Tier, result.Entries[0].PUUID)
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
		log.Printf("💾 Cache hit Master: %d entries", len(cachedResult.Entries))
		if len(cachedResult.Entries) > 10 {
			cachedResult.Entries = cachedResult.Entries[:10]
		}
		// ✅ GARANTIR QUE CADA ENTRY TENHA O TIER CORRETO
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
	
	log.Printf("📊 Master API response: %d entries", len(result.Entries))
	
	if len(result.Entries) > 10 {
		log.Printf("🔪 Cortando Master de %d para TOP 10", len(result.Entries))
		result.Entries = result.Entries[:10]
	}
	
	// ✅ GARANTIR QUE CADA ENTRY TENHA O TIER CORRETO
	for i := range result.Entries {
		result.Entries[i].Tier = "MASTER"
		log.Printf("🏆 Master Entry %d tier: %s", i, result.Entries[i].Tier)
	}
	
	if len(result.Entries) > 0 {
		log.Printf("🔍 Master primeiro entry - Tier: %s, PUUID: %s", result.Entries[0].Tier, result.Entries[0].PUUID)
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
	
	// 🏆 LIMITAÇÃO FIXA PARA TOP 10
	totalEntries := len(entries)
	maxEntries := 10
	if totalEntries > maxEntries {
		log.Printf("🏆 Limitando para TOP %d entries (de %d total)", maxEntries, totalEntries)
		entries = entries[:maxEntries]
	}
	
	lookups := 0
	maxLookups := 10 // Exato para TOP 10
	cacheHits := 0
	errors := 0

	log.Printf("🚀 Iniciando enrichment TOP %d entries", len(entries))

	for i := range entries {
		entry := &entries[i]

		log.Printf("Entry %d: 🔍 Tier atual: %s, PUUID: %s", i, entry.Tier, entry.PUUID[:30]+"...")

		if entry.SummonerName != "" && entry.SummonerName != "Unknown" {
			continue
		}

		puuid := entry.PUUID
		if puuid == "" {
			log.Printf("Entry %d: ❌ PUUID vazio", i)
			continue
		}

		// Verificar cache primeiro
		if cachedName, err := c.CacheManager.GetSummonerName(ctx, puuid); err == nil && cachedName != "" {
			entry.SummonerName = cachedName
			cacheHits++
			log.Printf("Entry %d: 💾 Cache hit: %s (Tier: %s)", i, cachedName, entry.Tier)
			continue
		}

		if lookups >= maxLookups {
			log.Printf("Entry %d: ⏸️ Limite atingido, usando worker async", i)
			
			if c.NATSClient != nil {
				go func(puuidCopy string) {
					c.fetchNameAsyncViaPUUID(puuidCopy)
				}(puuid)
			}
			continue
		}

		if lookups > 0 {
			time.Sleep(150 * time.Millisecond) // Rate limiting otimizado
		}
		
		log.Printf("Entry %d: 🌐 Buscando via Account API...", i)
		name := c.fetchNameDirectlyViaPUUID(puuid)
		
		if name != "" {
			entry.SummonerName = name
			c.CacheManager.SetSummonerName(ctx, puuid, name)
			log.Printf("Entry %d: ✅ Nome obtido: %s (Tier: %s)", i, name, entry.Tier)
		} else {
			log.Printf("Entry %d: ❌ Erro ao obter nome", i)
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

	log.Printf("🏁 TOP %d Enrichment concluído - Nomes: %d/%d, Cache: %d, Lookups: %d, Erros: %d", 
		len(entries), successfulNames, len(entries), cacheHits, lookups, errors)
	
	return entries
}

func (c *RiotAPIClient) fetchNameDirectlyViaPUUID(puuid string) string {
	log.Printf("🔍 Buscando Account via PUUID: %s", puuid[:30]+"...")
	
	accountData, err := c.GetAccountByPUUID(puuid)
	if err != nil {
		log.Printf("❌ Erro Account API: %v", err)
		return ""
	}

	if accountData.GameName == "" {
		log.Printf("❌ GameName vazio na response")
		return ""
	}

	fullName := accountData.GameName
	if accountData.TagLine != "" {
		fullName = fmt.Sprintf("%s#%s", accountData.GameName, accountData.TagLine)
	}
	
	log.Printf("✅ Nome obtido: %s", fullName)
	return fullName
}

func (c *RiotAPIClient) fetchNameAsyncViaPUUID(puuid string) {
	ctx := context.Background()
	
	// Verificar cache primeiro
	if cachedName, err := c.CacheManager.GetSummonerName(ctx, puuid); err == nil && cachedName != "" {
		log.Printf("💾 Nome já em cache: %s", cachedName)
		return
	}
	
	name := c.fetchNameDirectlyViaPUUID(puuid)
	
	if name != "" {
		c.CacheManager.SetSummonerName(ctx, puuid, name)
		log.Printf("💾 Nome cacheado async: %s", name)
	}
}