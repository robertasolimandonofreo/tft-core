package internal

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"
)

type RiotAPIClient struct {
	APIKey  string
	BaseURL string
	Client  *http.Client
}

func NewRiotAPIClient(cfg *Config) *RiotAPIClient {
	return &RiotAPIClient{
		APIKey:  cfg.RiotAPIKey,
		BaseURL: cfg.RiotBaseURL,
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
	path := fmt.Sprintf("/tft/summoner/v1/summoners/by-puuid/%s", puuid)
	data, err := c.doRequest(path)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *RiotAPIClient) GetMatchByID(matchId string) (map[string]interface{}, error) {
	path := fmt.Sprintf("/tft/match/v1/matches/%s", matchId)
	data, err := c.doRequest(path)
	if err != nil {
		return nil, err
	}
	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}

func (c *RiotAPIClient) GetLeagueEntries(tier, division string) ([]map[string]interface{}, error) {
	path := fmt.Sprintf("/tft/league/v1/entries/%s/%s", tier, division)
	data, err := c.doRequest(path)
	if err != nil {
		return nil, err
	}
	var result []map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}
	return result, nil
}