package internal

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

type mockRedisClient struct {
	data map[string]string
}

func (m *mockRedisClient) Get(ctx context.Context, key string) *redis.StringCmd {
	cmd := redis.NewStringCmd(ctx)
	if val, exists := m.data[key]; exists {
		cmd.SetVal(val)
	} else {
		cmd.SetErr(redis.Nil)
	}
	return cmd
}

func (m *mockRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	cmd := redis.NewStatusCmd(ctx)
	m.data[key] = value.(string)
	cmd.SetVal("OK")
	return cmd
}

type mockDatabase struct {
	enabled bool
	names   map[string]string
}

func (m *mockDatabase) GetSummonerName(puuid string) (string, error) {
	if !m.enabled {
		return "", redis.Nil
	}
	if name, exists := m.names[puuid]; exists {
		return name, nil
	}
	return "", redis.Nil
}

func (m *mockDatabase) SetSummonerName(puuid, gameName, tagLine, summonerID, region string) error {
	if !m.enabled {
		return nil
	}
	m.names[puuid] = gameName + "#" + tagLine
	return nil
}

func (m *mockDatabase) Close() {
	return
}

func TestCacheManager_Key(t *testing.T) {
	cm := &CacheManager{}
	
	key := cm.Key("test", "key", "parts")
	expected := "tft:test:key:parts"
	
	if key != expected {
		t.Errorf("expected key %s, got %s", expected, key)
	}
}

func TestCacheManager_GetSet_Disabled(t *testing.T) {
	cm := &CacheManager{enabled: false}
	ctx := context.Background()
	
	err := cm.Set(ctx, "test", "value", time.Hour)
	if err != nil {
		t.Errorf("set should not error when disabled: %v", err)
	}
	
	var result string
	err = cm.Get(ctx, "test", &result)
	if err != redis.Nil {
		t.Errorf("get should return redis.Nil when disabled, got %v", err)
	}
}

func TestCacheManager_GetSummonerName_Redis(t *testing.T) {
	mockRedis := &mockRedisClient{
		data: make(map[string]string),
	}
	mockRedis.data["tft:summoner_name:test123"] = `"TestPlayer#BR1"`
	
	cm := &CacheManager{
		enabled: true,
	}
	
	ctx := context.Background()
	name, err := cm.GetSummonerName(ctx, "test123")
	
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if name != "TestPlayer#BR1" {
		t.Errorf("expected 'TestPlayer#BR1', got %s", name)
	}
}

func TestCacheManager_GetSummonerName_Database_Fallback(t *testing.T) {
	mockDB := &mockDatabase{
		enabled: true,
		names:   map[string]string{"test123": "DBPlayer#BR1"},
	}
	
	cm := &CacheManager{
		enabled:  false,
		database: mockDB,
	}
	
	ctx := context.Background()
	name, err := cm.GetSummonerName(ctx, "test123")
	
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	if name != "DBPlayer#BR1" {
		t.Errorf("expected 'DBPlayer#BR1', got %s", name)
	}
}

func TestCacheManager_SetSummonerName(t *testing.T) {
	mockDB := &mockDatabase{
		enabled: true,
		names:   make(map[string]string),
	}
	
	cm := &CacheManager{
		enabled:  false,
		database: mockDB,
	}
	
	ctx := context.Background()
	err := cm.SetSummonerName(ctx, "test123", "NewPlayer#BR1")
	
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
	
	if mockDB.names["test123"] != "NewPlayer#BR1" {
		t.Errorf("expected 'NewPlayer#BR1' in database, got %s", mockDB.names["test123"])
	}
}

func TestParseName(t *testing.T) {
	tests := []struct {
		input       string
		expectedName string
		expectedTag  string
	}{
		{"Player#BR1", "Player", "BR1"},
		{"PlayerWithoutTag", "PlayerWithoutTag", "BR1"},
		{"Player#Long#Tag", "Player#Long", "Tag"},
		{"", "", "BR1"},
	}
	
	for _, tt := range tests {
		name, tag := parseName(tt.input)
		if name != tt.expectedName {
			t.Errorf("parseName(%s): expected name %s, got %s", tt.input, tt.expectedName, name)
		}
		if tag != tt.expectedTag {
			t.Errorf("parseName(%s): expected tag %s, got %s", tt.input, tt.expectedTag, tag)
		}
	}
}

func TestSplitName(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"Player#BR1", []string{"Player", "BR1"}},
		{"PlayerNoTag", []string{"PlayerNoTag"}},
		{"Player#Multi#Tags", []string{"Player#Multi", "Tags"}},
		{"", []string{""}},
	}
	
	for _, tt := range tests {
		result := splitName(tt.input)
		if len(result) != len(tt.expected) {
			t.Errorf("splitName(%s): expected %d parts, got %d", tt.input, len(tt.expected), len(result))
			continue
		}
		for i, part := range result {
			if part != tt.expected[i] {
				t.Errorf("splitName(%s): expected part %d to be %s, got %s", tt.input, i, tt.expected[i], part)
			}
		}
	}
}