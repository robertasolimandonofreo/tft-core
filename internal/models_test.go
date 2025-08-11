package internal

import "testing"

func TestLeagueEntry_GetUniqueID(t *testing.T) {
	tests := []struct {
		name     string
		entry    LeagueEntry
		expected string
	}{
		{
			name: "returns PUUID when available",
			entry: LeagueEntry{
				PUUID:      "test-puuid-123",
				SummonerID: "test-summoner-456",
			},
			expected: "test-puuid-123",
		},
		{
			name: "returns SummonerID when PUUID is empty",
			entry: LeagueEntry{
				PUUID:      "",
				SummonerID: "test-summoner-456",
			},
			expected: "test-summoner-456",
		},
		{
			name: "returns empty string when both are empty",
			entry: LeagueEntry{
				PUUID:      "",
				SummonerID: "",
			},
			expected: "",
		},
		{
			name: "prioritizes PUUID over SummonerID",
			entry: LeagueEntry{
				PUUID:      "puuid-123",
				SummonerID: "summoner-456",
			},
			expected: "puuid-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.entry.GetUniqueID()
			if result != tt.expected {
				t.Errorf("GetUniqueID() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestLeagueEntry_GetUniqueID_Benchmark(t *testing.T) {
	entry := LeagueEntry{
		PUUID:      "test-puuid-123456789",
		SummonerID: "test-summoner-456789",
	}

	for i := 0; i < 1000; i++ {
		entry.GetUniqueID()
	}
}
