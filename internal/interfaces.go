package internal

import (
	"context"
)

type RiotAPI interface {
	GetSummonerByPUUID(puuid string) (map[string]interface{}, error)
	GetAccountByGameName(gameName, tagLine string) (*AccountData, error)
	GetLeagueByPUUID(puuid string) ([]LeagueEntry, error)
	GetChallengerLeague() (*ChallengerLeague, error)
	GetGrandmasterLeague() (*GrandmasterLeague, error)
	GetMasterLeague() (*MasterLeague, error)
	GetLeagueEntries(tier, division string, page int) (*LeagueEntriesResponse, error)
}

type RateLimiterInterface interface {
	Allow(ctx context.Context, key string) (bool, error)
}

type DatabaseInterface interface {
	GetSummonerName(puuid string) (string, error)
	SetSummonerName(puuid, gameName, tagLine, summonerID, region string) error
	Close()
}