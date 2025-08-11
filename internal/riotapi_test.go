package internal

import "testing"

func TestGetAccountAPIURL(t *testing.T) {
	tests := []struct {
		name     string
		region   string
		expected string
	}{
		{
			name:     "BR1 region",
			region:   "BR1",
			expected: AmericasAPIURL,
		},
		{
			name:     "LA1 region",
			region:   "LA1",
			expected: AmericasAPIURL,
		},
		{
			name:     "NA1 region",
			region:   "NA1",
			expected: AmericasAPIURL,
		},
		{
			name:     "EUW1 region",
			region:   "EUW1",
			expected: EuropeAPIURL,
		},
		{
			name:     "EUN1 region",
			region:   "EUN1",
			expected: EuropeAPIURL,
		},
		{
			name:     "KR region",
			region:   "KR",
			expected: AsiaAPIURL,
		},
		{
			name:     "JP1 region",
			region:   "JP1",
			expected: AsiaAPIURL,
		},
		{
			name:     "OC1 region",
			region:   "OC1",
			expected: SeaAPIURL,
		},
		{
			name:     "unknown region defaults to Americas",
			region:   "UNKNOWN",
			expected: AmericasAPIURL,
		},
		{
			name:     "empty region defaults to Americas",
			region:   "",
			expected: AmericasAPIURL,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getAccountAPIURL(tt.region)
			if result != tt.expected {
				t.Errorf("getAccountAPIURL() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestFindTFTLeague(t *testing.T) {
	tests := []struct {
		name     string
		leagues  []LeagueEntry
		expected *LeagueEntry
	}{
		{
			name:     "nil leagues",
			leagues:  nil,
			expected: nil,
		},
		{
			name:     "empty leagues",
			leagues:  []LeagueEntry{},
			expected: nil,
		},
		{
			name: "find TFT league",
			leagues: []LeagueEntry{
				{QueueType: "RANKED_SOLO_5x5", Tier: "GOLD"},
				{QueueType: "RANKED_TFT", Tier: "CHALLENGER"},
				{QueueType: "RANKED_FLEX_SR", Tier: "SILVER"},
			},
			expected: &LeagueEntry{QueueType: "RANKED_TFT", Tier: "CHALLENGER"},
		},
		{
			name: "no TFT league found",
			leagues: []LeagueEntry{
				{QueueType: "RANKED_SOLO_5x5", Tier: "GOLD"},
				{QueueType: "RANKED_FLEX_SR", Tier: "SILVER"},
			},
			expected: nil,
		},
		{
			name: "multiple entries but only one TFT",
			leagues: []LeagueEntry{
				{QueueType: "RANKED_SOLO_5x5", Tier: "GOLD"},
				{QueueType: "RANKED_TFT", Tier: "MASTER"},
			},
			expected: &LeagueEntry{QueueType: "RANKED_TFT", Tier: "MASTER"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findTFTLeague(tt.leagues)

			if tt.expected == nil && result != nil {
				t.Errorf("findTFTLeague() = %v, expected nil", result)
				return
			}

			if tt.expected != nil && result == nil {
				t.Errorf("findTFTLeague() = nil, expected %v", tt.expected)
				return
			}

			if tt.expected != nil && result != nil {
				if result.QueueType != tt.expected.QueueType || result.Tier != tt.expected.Tier {
					t.Errorf("findTFTLeague() = %v, expected %v", result, tt.expected)
				}
			}
		})
	}
}
