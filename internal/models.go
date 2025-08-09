package internal

type LeagueEntry struct {
	LeagueID     string      `json:"leagueId"`
	PUUID        string      `json:"puuid"`          // ✅ Campo correto da TFT API
	SummonerID   string      `json:"summonerId"`     // Fallback para outras APIs
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

func (le *LeagueEntry) GetUniqueID() string {
	// TFT API usa PUUID diretamente
	if le.PUUID != "" {
		return le.PUUID
	}
	// Fallback para outras APIs
	if le.SummonerID != "" {
		return le.SummonerID
	}
	return ""
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

type SummonerNameTask struct {
	PUUID  string `json:"puuid"`
	Region string `json:"region"`
}

type Summoner struct {
	ID            string `json:"id"`
	AccountID     string `json:"accountId"`
	PUUID         string `json:"puuid"`
	Name          string `json:"name"`
	ProfileIconID int    `json:"profileIconId"`
	RevisionDate  int64  `json:"revisionDate"`
	SummonerLevel int    `json:"summonerLevel"`
}

type AccountData struct {
	PUUID    string `json:"puuid"`
	GameName string `json:"gameName"`
	TagLine  string `json:"tagLine"`
}