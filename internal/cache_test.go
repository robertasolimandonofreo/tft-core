package internal

import "testing"

func TestCacheManager_Key(t *testing.T) {
	cm := &CacheManager{}

	tests := []struct {
		name     string
		parts    []string
		expected string
	}{
		{
			name:     "single part",
			parts:    []string{"test"},
			expected: "tft:test",
		},
		{
			name:     "multiple parts",
			parts:    []string{"summoner", "BR1", "test-id"},
			expected: "tft:summoner:BR1:test-id",
		},
		{
			name:     "empty parts",
			parts:    []string{},
			expected: "tft",
		},
		{
			name:     "parts with empty strings",
			parts:    []string{"test", "", "value"},
			expected: "tft:test::value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cm.Key(tt.parts...)
			if result != tt.expected {
				t.Errorf("Key() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestParseName(t *testing.T) {
	tests := []struct {
		name         string
		fullName     string
		expectedGame string
		expectedTag  string
	}{
		{
			name:         "name with tag",
			fullName:     "Player#BR1",
			expectedGame: "Player",
			expectedTag:  "BR1",
		},
		{
			name:         "name without tag",
			fullName:     "Player",
			expectedGame: "Player",
			expectedTag:  "BR1",
		},
		{
			name:         "name with multiple hash",
			fullName:     "Player#Test#BR1",
			expectedGame: "Player#Test",
			expectedTag:  "BR1",
		},
		{
			name:         "empty name",
			fullName:     "",
			expectedGame: "",
			expectedTag:  "BR1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gameName, tagLine := parseName(tt.fullName)
			if gameName != tt.expectedGame {
				t.Errorf("parseName() gameName = %v, expected %v", gameName, tt.expectedGame)
			}
			if tagLine != tt.expectedTag {
				t.Errorf("parseName() tagLine = %v, expected %v", tagLine, tt.expectedTag)
			}
		})
	}
}

func TestSplitName(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "name with tag",
			input:    "Player#BR1",
			expected: []string{"Player", "BR1"},
		},
		{
			name:     "name without tag",
			input:    "Player",
			expected: []string{"Player"},
		},
		{
			name:     "name with multiple hash",
			input:    "Player#Test#BR1",
			expected: []string{"Player#Test", "BR1"},
		},
		{
			name:     "empty string",
			input:    "",
			expected: []string{""},
		},
		{
			name:     "only hash",
			input:    "#",
			expected: []string{"", ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := splitName(tt.input)
			if len(result) != len(tt.expected) {
				t.Errorf("splitName() length = %v, expected %v", len(result), len(tt.expected))
				return
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("splitName()[%d] = %v, expected %v", i, v, tt.expected[i])
				}
			}
		})
	}
}
