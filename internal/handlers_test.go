package internal

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"testing"
)

type mockRiotClient struct {
	shouldError bool
	errorType   string
}

func (m *mockRiotClient) GetSummonerByPUUID(puuid string) (map[string]interface{}, error) {
	if m.shouldError {
		if m.errorType == "404" {
			return nil, errors.New("404 not found")
		}
		return nil, errors.New("api error")
	}
	return map[string]interface{}{
		"id":   "summoner123",
		"name": "TestPlayer",
		"puuid": puuid,
	}, nil
}

func (m *mockRiotClient) GetAccountByGameName(gameName, tagLine string) (*AccountData, error) {
	if m.shouldError {
		if m.errorType == "404" {
			return nil, errors.New("404 not found")
		}
		return nil, errors.New("api error")
	}
	return &AccountData{
		PUUID:    "test-puuid-123",
		GameName: gameName,
		TagLine:  tagLine,
	}, nil
}

func (m *mockRiotClient) GetLeagueByPUUID(puuid string) ([]LeagueEntry, error) {
	if m.shouldError {
		return nil, errors.New("api error")
	}
	return []LeagueEntry{
		{
			QueueType:    "RANKED_TFT",
			Tier:         "CHALLENGER",
			Rank:         "I",
			LeaguePoints: 1000,
			Wins:         50,
			Losses:       10,
		},
	}, nil
}

func (m *mockRiotClient) GetChallengerLeague() (*ChallengerLeague, error) {
	if m.shouldError {
		return nil, errors.New("api error")
	}
	return &ChallengerLeague{
		Entries: []LeagueEntry{
			{Tier: "CHALLENGER", LeaguePoints: 1000},
		},
	}, nil
}

func (m *mockRiotClient) GetGrandmasterLeague() (*GrandmasterLeague, error) {
	if m.shouldError {
		return nil, errors.New("api error")
	}
	return &GrandmasterLeague{
		Entries: []LeagueEntry{
			{Tier: "GRANDMASTER", LeaguePoints: 800},
		},
	}, nil
}

func (m *mockRiotClient) GetMasterLeague() (*MasterLeague, error) {
	if m.shouldError {
		return nil, errors.New("api error")
	}
	return &MasterLeague{
		Entries: []LeagueEntry{
			{Tier: "MASTER", LeaguePoints: 600},
		},
	}, nil
}

func (m *mockRiotClient) GetLeagueEntries(tier, division string, page int) (*LeagueEntriesResponse, error) {
	if m.shouldError {
		return nil, errors.New("api error")
	}
	return &LeagueEntriesResponse{
		Entries: []LeagueEntry{
			{Tier: tier, Rank: division, LeaguePoints: 100},
		},
		Page:     page,
		Tier:     tier,
		Division: division,
		HasMore:  false,
	}, nil
}

type mockRateLimiter struct {
	shouldBlock bool
	shouldError bool
}

func (m *mockRateLimiter) Allow(ctx context.Context, key string) (bool, error) {
	if m.shouldError {
		return false, errors.New("rate limiter error")
	}
	return !m.shouldBlock, nil
}

func createTestLogger() *Logger {
	return &Logger{
		level:       LogLevelError,
		service:     "test",
		environment: "test",
		logger:      log.New(bytes.NewBuffer(nil), "", 0),
	}
}

func TestHealthHandler(t *testing.T) {
	logger := createTestLogger()
	handler := HealthHandler(logger)
	
	req := httptest.NewRequest("GET", "/healthz", nil)
	w := httptest.NewRecorder()
	
	handler(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	
	if response["status"] != "ok" {
		t.Errorf("expected status ok, got %v", response["status"])
	}
}

func TestSummonerHandler_Success(t *testing.T) {
	logger := createTestLogger()
	riotClient := &mockRiotClient{}
	rateLimiter := &mockRateLimiter{}
	
	handler := SummonerHandler(riotClient, rateLimiter, logger)
	
	req := httptest.NewRequest("GET", "/summoner?puuid=test123", nil)
	req = req.WithContext(context.WithValue(req.Context(), RequestIDKey, "test-request-id"))
	w := httptest.NewRecorder()
	
	handler(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	
	if response["puuid"] != "test123" {
		t.Errorf("expected puuid test123, got %v", response["puuid"])
	}
}

func TestSummonerHandler_MissingPUUID(t *testing.T) {
	logger := createTestLogger()
	riotClient := &mockRiotClient{}
	rateLimiter := &mockRateLimiter{}
	
	handler := SummonerHandler(riotClient, rateLimiter, logger)
	
	req := httptest.NewRequest("GET", "/summoner", nil)
	req = req.WithContext(context.WithValue(req.Context(), RequestIDKey, "test-request-id"))
	w := httptest.NewRecorder()
	
	handler(w, req)
	
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestSummonerHandler_NotFound(t *testing.T) {
	logger := createTestLogger()
	riotClient := &mockRiotClient{shouldError: true, errorType: "404"}
	rateLimiter := &mockRateLimiter{}
	
	handler := SummonerHandler(riotClient, rateLimiter, logger)
	
	req := httptest.NewRequest("GET", "/summoner?puuid=notfound", nil)
	req = req.WithContext(context.WithValue(req.Context(), RequestIDKey, "test-request-id"))
	w := httptest.NewRecorder()
	
	handler(w, req)
	
	if w.Code != http.StatusNotFound {
		t.Errorf("expected status 404, got %d", w.Code)
	}
}

func TestSummonerHandler_RateLimit(t *testing.T) {
	logger := createTestLogger()
	riotClient := &mockRiotClient{}
	rateLimiter := &mockRateLimiter{shouldBlock: true}
	
	handler := SummonerHandler(riotClient, rateLimiter, logger)
	
	req := httptest.NewRequest("GET", "/summoner?puuid=test123", nil)
	req = req.WithContext(context.WithValue(req.Context(), RequestIDKey, "test-request-id"))
	w := httptest.NewRecorder()
	
	handler(w, req)
	
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("expected status 429, got %d", w.Code)
	}
}

func TestSearchPlayerHandler_Success(t *testing.T) {
	logger := createTestLogger()
	riotClient := &mockRiotClient{}
	rateLimiter := &mockRateLimiter{}
	
	handler := SearchPlayerHandler(riotClient, rateLimiter, logger)
	
	req := httptest.NewRequest("GET", "/search/player?gameName=TestPlayer&tagLine=BR1", nil)
	req = req.WithContext(context.WithValue(req.Context(), RequestIDKey, "test-request-id"))
	w := httptest.NewRecorder()
	
	handler(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	
	if response["gameName"] != "TestPlayer" {
		t.Errorf("expected gameName TestPlayer, got %v", response["gameName"])
	}
	if response["tagLine"] != "BR1" {
		t.Errorf("expected tagLine BR1, got %v", response["tagLine"])
	}
}

func TestSearchPlayerHandler_MissingGameName(t *testing.T) {
	logger := createTestLogger()
	riotClient := &mockRiotClient{}
	rateLimiter := &mockRateLimiter{}
	
	handler := SearchPlayerHandler(riotClient, rateLimiter, logger)
	
	req := httptest.NewRequest("GET", "/search/player", nil)
	req = req.WithContext(context.WithValue(req.Context(), RequestIDKey, "test-request-id"))
	w := httptest.NewRecorder()
	
	handler(w, req)
	
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestSearchPlayerHandler_DefaultTagLine(t *testing.T) {
	logger := createTestLogger()
	riotClient := &mockRiotClient{}
	rateLimiter := &mockRateLimiter{}
	
	handler := SearchPlayerHandler(riotClient, rateLimiter, logger)
	
	req := httptest.NewRequest("GET", "/search/player?gameName=TestPlayer", nil)
	req = req.WithContext(context.WithValue(req.Context(), RequestIDKey, "test-request-id"))
	w := httptest.NewRecorder()
	
	handler(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	
	if response["tagLine"] != "BR1" {
		t.Errorf("expected default tagLine BR1, got %v", response["tagLine"])
	}
}

func TestChallengerHandler_Success(t *testing.T) {
	logger := createTestLogger()
	riotClient := &mockRiotClient{}
	rateLimiter := &mockRateLimiter{}
	
	handler := ChallengerHandler(riotClient, rateLimiter, logger)
	
	req := httptest.NewRequest("GET", "/league/challenger", nil)
	req = req.WithContext(context.WithValue(req.Context(), RequestIDKey, "test-request-id"))
	w := httptest.NewRecorder()
	
	handler(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	
	var response ChallengerLeague
	json.Unmarshal(w.Body.Bytes(), &response)
	
	if len(response.Entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(response.Entries))
	}
}

func TestEntriesHandler_Success(t *testing.T) {
	logger := createTestLogger()
	riotClient := &mockRiotClient{}
	rateLimiter := &mockRateLimiter{}
	
	handler := EntriesHandler(riotClient, rateLimiter, logger)
	
	req := httptest.NewRequest("GET", "/league/entries?tier=GOLD&division=I&page=1", nil)
	req = req.WithContext(context.WithValue(req.Context(), RequestIDKey, "test-request-id"))
	w := httptest.NewRecorder()
	
	handler(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", w.Code)
	}
	
	var response LeagueEntriesResponse
	json.Unmarshal(w.Body.Bytes(), &response)
	
	if response.Tier != "GOLD" {
		t.Errorf("expected tier GOLD, got %s", response.Tier)
	}
	if response.Division != "I" {
		t.Errorf("expected division I, got %s", response.Division)
	}
	if response.Page != 1 {
		t.Errorf("expected page 1, got %d", response.Page)
	}
}

func TestEntriesHandler_MissingParams(t *testing.T) {
	logger := createTestLogger()
	riotClient := &mockRiotClient{}
	rateLimiter := &mockRateLimiter{}
	
	handler := EntriesHandler(riotClient, rateLimiter, logger)
	
	req := httptest.NewRequest("GET", "/league/entries", nil)
	req = req.WithContext(context.WithValue(req.Context(), RequestIDKey, "test-request-id"))
	w := httptest.NewRecorder()
	
	handler(w, req)
	
	if w.Code != http.StatusBadRequest {
		t.Errorf("expected status 400, got %d", w.Code)
	}
}

func TestAPIError(t *testing.T) {
	err := NewAPIError("test error", 400)
	
	if err.Message != "test error" {
		t.Errorf("expected message 'test error', got %s", err.Message)
	}
	if err.Status != 400 {
		t.Errorf("expected status 400, got %d", err.Status)
	}
	if err.Error() != "test error" {
		t.Errorf("expected error string 'test error', got %s", err.Error())
	}
}