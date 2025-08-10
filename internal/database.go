package internal

import (
	"database/sql"
	"fmt"
	"log"
	"time"

	_ "github.com/lib/pq"
)

type DatabaseManager struct {
	DB      *sql.DB
	Enabled bool
}

type SummonerCacheEntry struct {
	PUUID       string
	GameName    string
	TagLine     string
	SummonerID  *string
	Region      string
	LastUpdated time.Time
	CreatedAt   time.Time
}

func NewDatabaseManager(cfg *Config) *DatabaseManager {
	if !cfg.DatabaseEnabled {
		log.Println("Database disabled, running without PostgreSQL")
		return &DatabaseManager{Enabled: false}
	}

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		cfg.PostgresHost,
		cfg.PostgresPort,
		cfg.PostgresUser,
		cfg.PostgresPassword,
		cfg.PostgresDb,
		cfg.PostgresSSLMode,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		log.Printf("Error connecting to database: %v", err)
		return &DatabaseManager{Enabled: false}
	}

	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(30 * time.Minute)

	if err := db.Ping(); err != nil {
		log.Printf("Error pinging database: %v", err)
		return &DatabaseManager{Enabled: false}
	}

	log.Println("Database connected successfully")
	return &DatabaseManager{
		DB:      db,
		Enabled: true,
	}
}

func (dm *DatabaseManager) GetSummonerName(puuid string) (string, error) {
	if !dm.Enabled {
		return "", fmt.Errorf("database not enabled")
	}

	var entry SummonerCacheEntry
	query := `
		SELECT puuid, game_name, tag_line, summoner_id, region, last_updated, created_at
		FROM summoner_cache 
		WHERE puuid = $1 AND last_updated > NOW() - INTERVAL '7 days'
	`

	err := dm.DB.QueryRow(query, puuid).Scan(
		&entry.PUUID,
		&entry.GameName,
		&entry.TagLine,
		&entry.SummonerID,
		&entry.Region,
		&entry.LastUpdated,
		&entry.CreatedAt,
	)

	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s#%s", entry.GameName, entry.TagLine), nil
}

func (dm *DatabaseManager) SetSummonerName(puuid, gameName, tagLine, summonerID, region string) error {
	if !dm.Enabled {
		return nil
	}

	query := `
		INSERT INTO summoner_cache (puuid, game_name, tag_line, summoner_id, region) 
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (puuid) DO UPDATE SET 
			game_name = $2, 
			tag_line = $3, 
			summoner_id = $4,
			region = $5,
			last_updated = CURRENT_TIMESTAMP
	`

	_, err := dm.DB.Exec(query, puuid, gameName, tagLine, summonerID, region)
	if err != nil {
		log.Printf("Error saving summoner cache: %v", err)
		return err
	}

	log.Printf("Summoner cached: %s#%s (PUUID: %s)", gameName, tagLine, puuid[:20]+"...")
	return nil
}

func (dm *DatabaseManager) GetCacheStats() (map[string]interface{}, error) {
	if !dm.Enabled {
		return map[string]interface{}{
			"enabled": false,
		}, nil
	}

	var total, recent int
	dm.DB.QueryRow("SELECT COUNT(*) FROM summoner_cache").Scan(&total)
	dm.DB.QueryRow("SELECT COUNT(*) FROM summoner_cache WHERE last_updated > NOW() - INTERVAL '24 hours'").Scan(&recent)

	return map[string]interface{}{
		"enabled":       true,
		"total_cached":  total,
		"recent_cached": recent,
	}, nil
}

func (dm *DatabaseManager) Close() {
	if dm.Enabled && dm.DB != nil {
		dm.DB.Close()
	}
}