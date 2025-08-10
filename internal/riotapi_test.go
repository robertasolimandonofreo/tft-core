package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func createTestRiotClient() *RiotAPIClient {
	cfg := &Config{
		RiotAPIKey:  "test-key",
		RiotRegion:  "BR1",
		RiotBaseURL: "http://test-api.riot.com",
	}
	
	logger := &Logger{
		level:       LogLevelError,
		service:     "test",
		environment: "test",
		logger:      log.New(bytes.NewBuffer(nil), "", 0),
	}
	
	cache := &CacheManager{enabled: false}
	metrics := NewMetricsCollector(logger)
	
	client := NewRiotAPIClient(cfg, cache, logger, metrics)
	return client
}

func TestGetAccountAPIURL(t *testing.T) {
	tests := []struct {
		region   string
		expected string
	}{
		{"BR1", "https://americas.api.riotgames.com"},
		{"NA1", "https://americas.api.riotgames.com"},
		{"EUW1", "https://europe.api.riotgames.com"},
		{"KR", "https://asia.api.riotgames.com"},
		{"UNKNOWN", "https://americas.api.riotgames.com"},
	}
	
	for _, tt := range tests {
		result := getAccountAPIURL(tt.region)
		if result != tt.expected {
			t.Errorf("getAccountAPIURL(%s): expected %s, got %s", tt.region, tt.expected, result)
		}
	}
}

func TestRiotAPIClient_DoRequest_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-Riot-Token") != "test-key" {
			t.Error("missing or incorrect riot token header")
		}
		
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"test": "data"})
	}))
	defer server.Close()
	
	client := createTestRiotClient()
	client.client = server.Client()
	
	data, err := client.doRequest(server.URL)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	
	var result map[string]string
	json.Unmarshal(data, &result)
	
	if result["test"] != "data" {
		t.Errorf("expected test data, got %v", result)
	}
}

func TestRiotAPIClient_DoRequest_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Not Found"))
	}))
	defer server.Close()
	
	client := createTestRiotClient()
	client.client = server.Client()
	
	_, err := client.doRequest(server.URL)
	if err == nil {
		t.Error("expected error, got nil")
	}
	
	if err.Error() != "riot API error: 404 Not Found - Not Found" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRiotAPIClient_GetSummonerByPUUID_Cache(t *testing.T) {
	mockCache := &mockCacheManager{
		data: map[string]interface{}{
			"tft:summoner:BR1:test123": map[string]interface{}{
				"id":   "cached-summoner",
				"puuid": "test123",
			},
		},
	}
	
	client := createTestRiotClient()
	client.cache = mockCache
	
	result, err := client.GetSummonerByPUUID("test123")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	
	if result["id"] != "cached-summoner" {
		t.Errorf("expected cached data, got %v", result)
	}
}

func TestRiotAPIClient_GetAccountByGameName_Validation(t *testing.T) {
	client := createTestRiotClient()
	
	_, err := client.GetAccountByGameName("", "BR1")
	if err == nil {
		t.Error("expected error for empty game name")
	}
	
	if err.Error() != "gameName cannot be empty" {
		t.Errorf("unexpected error message: %v", err)
	}
}

func TestRiotAPIClient_GetAccountByGameName_DefaultTag(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/riot/account/v1/accounts/by-riot-id/TestPlayer/BR1"
		if r.URL.Path != expectedPath {
			t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
		}
		
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(AccountData{
			PUUID:    "test-puuid",
			GameName: "TestPlayer",
			TagLine:  "BR1",
		})
	}))
	defer server.Close()
	
	client := createTestRiotClient()
	client.accountURL = server.URL
	client.client = server.Client()
	
	result, err := client.GetAccountByGameName("TestPlayer", "")
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	
	if result.TagLine != "BR1" {
		t.Errorf("expected default tag BR1, got %s", result.TagLine)
	}
}

func TestRiotAPIClient_EnrichEntries(t *testing.T) {
	mockCache := &mockCacheManager{
		summonerNames: map[string]string{
			"puuid1": "CachedPlayer#BR1",
		},
	}
	
	client := createTestRiotClient()
	client.cache = mockCache
	
	entries := []LeagueEntry{
		{PUUID: "puuid1", SummonerName: ""},
		{PUUID: "puuid2", SummonerName: "ExistingPlayer"},
		{PUUID: "puuid3", SummonerName: ""},
		{PUUID: "", SummonerName: ""},
	}
	
	client.enrichEntries(entries, "CHALLENGER")
	
	if entries[0].SummonerName != "CachedPlayer#BR1" {
		t.Errorf("expected cached name, got %s", entries[0].SummonerName)
	}
	
	if entries[1].SummonerName != "ExistingPlayer" {
		t.Errorf("expected existing name unchanged, got %s", entries[1].SummonerName)
	}
	
	if entries[2].SummonerName != "Loading..." {
		t.Errorf("expected Loading..., got %s", entries[2].SummonerName)
	}
	
	if entries[3].SummonerName != "Unknown" {
		t.Errorf("expected Unknown for empty PUUID, got %s", entries[3].SummonerName)
	}
	
	for _, entry := range entries {
		if entry.Tier != "CHALLENGER" {
			t.Errorf("expected tier CHALLENGER, got %s", entry.Tier)
		}
	}
}

type mockCacheManager struct {
	data          map[string]interface{}
	summonerNames map[string]string
}

func (m *mockCacheManager) Get(ctx context.Context, key string, result interface{}) error {
	if data, exists := m.data[key]; exists {
		if resultMap, ok := result.(*map[string]interface{}); ok {
			*resultMap = data.(map[string]interface{})
			return nil
		}
	}
	return redis.Nil
}

func (m *mockCacheManager) Set(ctx context.Context, key string, data interface{}, ttl time.Duration) error {
	if m.data == nil {
		m.data = make(map[string]interface{})
	}
	m.data[key] = data
	return nil
}

func (m *mockCacheManager) Key(parts ...string) string {
	key := "tft"
	for _, part := range parts {
		key = key + ":" + part
	}
	return key
}

func (m *mockCacheManager) GetSummonerName(ctx context.Context, puuid string) (string, error) {
	if m.summonerNames == nil {
		return "", redis.Nil
	}
	if name, exists := m.summonerNames[puuid]; exists {
		return name, nil
	}
	return "", redis.Nil
}

func (m *mockCacheManager) SetSummonerName(ctx context.Context, puuid, name string) error {
	if m.summonerNames == nil {
		m.summonerNames = make(map[string]string)
	}
	m.summonerNames[puuid] = name
	return nil
}

func TestRiotAPIClient_GetHighTierLeague_CacheHit(t *testing.T) {
	cachedLeague := &ChallengerLeague{
		Entries: make([]LeagueEntry, 15),
		Tier:    "CHALLENGER",
	}
	for i := range cachedLeague.Entries {
		cachedLeague.Entries[i] = LeagueEntry{
			PUUID:        "test-puuid",
			LeaguePoints: 1000 - i*10,
		}
	}
	
	mockCache := &mockCacheManager{
		data: map[string]interface{}{
			"tft:challenger:BR1": cachedLeague,
		},
	}
	
	client := createTestRiotClient()
	client.cache = mockCache
	
	result, err := client.GetChallengerLeague()
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	
	if len(result.Entries) != 10 {
		t.Errorf("expected 10 entries (truncated), got %d", len(result.Entries))
	}
}

func TestRiotAPIClient_GetLeagueEntries_Pagination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/tft/league/v1/entries/GOLD/I"
		if r.URL.Path != expectedPath {
			t.Errorf("expected path %s, got %s", expectedPath, r.URL.Path)
		}
		
		page := r.URL.Query().Get("page")
		if page != "2" {
			t.Errorf("expected page 2, got %s", page)
		}
		
		entries := make([]LeagueEntry, 200)
		for i := range entries {
			entries[i] = LeagueEntry{
				Tier:         "GOLD",
				Rank:         "I",
				LeaguePoints: 100 + i,
			}
		}
		
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(entries)
	}))
	defer server.Close()
	
	client := createTestRiotClient()
	client.baseURL = server.URL
	client.client = server.Client()
	
	result, err := client.GetLeagueEntries("GOLD", "I", 2)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	
	if result.Page != 2 {
		t.Errorf("expected page 2, got %d", result.Page)
	}
	
	if result.Tier != "GOLD" {
		t.Errorf("expected tier GOLD, got %s", result.Tier)
	}
	
	if result.Division != "I" {
		t.Errorf("expected division I, got %s", result.Division)
	}
	
	if !result.HasMore {
		t.Error("expected HasMore to be true for 200 entries")
	}
}