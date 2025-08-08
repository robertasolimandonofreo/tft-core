package internal

type LeagueEntry struct {
	LeagueID     string      `json:"leagueId"`
	SummonerID   string      `json:"summonerId"`
	SummonerName string      `json:"summonerName"`
	QueueType    string      `json:"queueType"`
	Tier         string      `json:"tier"`
	Rank         string      `json:"rank"`
	LeaguePoints int         `json:"leaguePoints"`
	Wins         int         `json:"wins"`
	Losses       int         `json:"losses"`
	HotStreak    bool        `json:"hotStreak"`
	Veteran      bool        `json:"veteran"`
	FreshBlood   bool        `json:"freshBlood"`
	Inactive     bool        `json:"inactive"`
	MiniSeries   *MiniSeries `json:"miniSeries,omitempty"`
}

type MiniSeries struct {
	Target   int    `json:"target"`
	Wins     int    `json:"wins"`
	Losses   int    `json:"losses"`
	Progress string `json:"progress"`
}

type ChallengerLeague struct {
	LeagueID string        `json:"leagueId"`
	Entries  []LeagueEntry `json:"entries"`
	Tier     string        `json:"tier"`
	Name     string        `json:"name"`
	Queue    string        `json:"queue"`
}

type GrandmasterLeague struct {
	LeagueID string        `json:"leagueId"`
	Entries  []LeagueEntry `json:"entries"`
	Tier     string        `json:"tier"`
	Name     string        `json:"name"`
	Queue    string        `json:"queue"`
}

type MasterLeague struct {
	LeagueID string        `json:"leagueId"`
	Entries  []LeagueEntry `json:"entries"`
	Tier     string        `json:"tier"`
	Name     string        `json:"name"`
	Queue    string        `json:"queue"`
}

type RatedLadder struct {
	Queue   string             `json:"queue"`
	Tier    string             `json:"tier"`
	Entries []RatedLadderEntry `json:"entries"`
}

type RatedLadderEntry struct {
	SummonerID     string `json:"summonerId"`
	SummonerName   string `json:"summonerName"`
	RatedTier      string `json:"ratedTier"`
	RatedRating    int    `json:"ratedRating"`
	LeaguePoints   int    `json:"leaguePoints"`
	Wins           int    `json:"wins"`
	PreviousUpdate int64  `json:"previousUpdate"`
}

type LeagueEntriesResponse struct {
	Entries  []LeagueEntry `json:"entries"`
	Page     int           `json:"page"`
	Tier     string        `json:"tier"`
	Division string        `json:"division"`
	HasMore  bool          `json:"hasMore"`
}

type LeagueUpdateTask struct {
	Type     string `json:"type"`
	Tier     string `json:"tier,omitempty"`
	Division string `json:"division,omitempty"`
	Region   string `json:"region"`
	Page     int    `json:"page,omitempty"`
}