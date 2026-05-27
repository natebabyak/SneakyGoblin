package main

import (
	"database/sql"
	"log"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var db *sql.DB

type playerSuggestion struct {
	Tag      string
	Name     string
	ClanTag  string
	ClanName string
}

type clanSuggestion struct {
	Tag  string
	Name string
}

func initDb() {
	var err error
	db, err = sql.Open("sqlite3", "data.db")
	if err != nil {
		log.Fatal(err)
	}
	db.SetMaxOpenConns(1)

	stmts := []string{
		`CREATE TABLE IF NOT EXISTS user_accounts (
			discord_user_id TEXT NOT NULL,
			player_tag TEXT NOT NULL,
			is_main INTEGER NOT NULL DEFAULT 0,
			created_at TEXT NOT NULL,
			PRIMARY KEY(discord_user_id, player_tag)
		);`,
		`CREATE TABLE IF NOT EXISTS known_players (
			player_tag TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			clan_tag TEXT NOT NULL DEFAULT '',
			clan_name TEXT NOT NULL DEFAULT '',
			last_seen_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS known_clans (
			clan_tag TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			last_seen_at TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS command_usage (
			discord_user_id TEXT NOT NULL,
			entity_type TEXT NOT NULL,
			entity_tag TEXT NOT NULL,
			used_at TEXT NOT NULL
		);`,
		`CREATE INDEX IF NOT EXISTS idx_user_accounts_user ON user_accounts(discord_user_id);`,
		`CREATE INDEX IF NOT EXISTS idx_known_players_name ON known_players(name);`,
		`CREATE INDEX IF NOT EXISTS idx_known_clans_name ON known_clans(name);`,
		`CREATE INDEX IF NOT EXISTS idx_command_usage_lookup ON command_usage(discord_user_id, entity_type, used_at DESC);`,
	}

	for _, stmt := range stmts {
		if _, err := db.Exec(stmt); err != nil {
			log.Fatalf("db schema init failed: %v", err)
		}
	}
}

func upsertKnownPlayer(player Player) {
	if db == nil || player.Tag == "" || player.Name == "" {
		return
	}

	clanTag := player.Player.Tag
	clanName := player.Player.Name
	now := time.Now().UTC().Format(time.RFC3339)
	_, _ = db.Exec(
		`INSERT INTO known_players(player_tag, name, clan_tag, clan_name, last_seen_at)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(player_tag) DO UPDATE SET
		 	name=excluded.name,
		 	clan_tag=excluded.clan_tag,
		 	clan_name=excluded.clan_name,
		 	last_seen_at=excluded.last_seen_at`,
		player.Tag, player.Name, clanTag, clanName, now,
	)
}

func upsertKnownClan(clan Clan) {
	if db == nil || clan.Tag == "" || clan.Name == "" {
		return
	}

	now := time.Now().UTC().Format(time.RFC3339)
	_, _ = db.Exec(
		`INSERT INTO known_clans(clan_tag, name, last_seen_at)
		 VALUES (?, ?, ?)
		 ON CONFLICT(clan_tag) DO UPDATE SET
		 	name=excluded.name,
		 	last_seen_at=excluded.last_seen_at`,
		clan.Tag, clan.Name, now,
	)
}

func linkUserAccount(discordUserID, playerTag string) error {
	if db == nil {
		return nil
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := db.Exec(
		`INSERT OR IGNORE INTO user_accounts(discord_user_id, player_tag, is_main, created_at)
		 VALUES (?, ?, 0, ?)`,
		discordUserID, playerTag, now,
	)
	return err
}

func removeUserAccount(discordUserID, playerTag string) error {
	if db == nil {
		return nil
	}
	_, err := db.Exec(
		`DELETE FROM user_accounts WHERE discord_user_id = ? AND player_tag = ?`,
		discordUserID, playerTag,
	)
	return err
}

func setMainUserAccount(discordUserID, playerTag string) error {
	if db == nil {
		return nil
	}
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`UPDATE user_accounts SET is_main = 0 WHERE discord_user_id = ?`, discordUserID); err != nil {
		return err
	}
	if _, err := tx.Exec(`UPDATE user_accounts SET is_main = 1 WHERE discord_user_id = ? AND player_tag = ?`, discordUserID, playerTag); err != nil {
		return err
	}
	return tx.Commit()
}

func getUserMainAccount(discordUserID string) (string, bool) {
	if db == nil {
		return "", false
	}
	var tag string
	err := db.QueryRow(
		`SELECT player_tag FROM user_accounts WHERE discord_user_id = ? AND is_main = 1 LIMIT 1`,
		discordUserID,
	).Scan(&tag)
	if err != nil {
		return "", false
	}
	return tag, true
}

func listUserAccounts(discordUserID string) []string {
	if db == nil {
		return nil
	}
	rows, err := db.Query(
		`SELECT player_tag FROM user_accounts
		 WHERE discord_user_id = ?
		 ORDER BY is_main DESC, created_at ASC`,
		discordUserID,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var tag string
		if rows.Scan(&tag) == nil {
			tags = append(tags, tag)
		}
	}
	return tags
}

func recordCommandUsage(discordUserID, entityType, entityTag string) {
	if db == nil || entityTag == "" {
		return
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, _ = db.Exec(
		`INSERT INTO command_usage(discord_user_id, entity_type, entity_tag, used_at)
		 VALUES (?, ?, ?, ?)`,
		discordUserID, entityType, entityTag, now,
	)
}

func searchPlayers(discordUserID, query string, limit int) []playerSuggestion {
	if db == nil {
		return nil
	}
	q := "%" + strings.ToLower(strings.TrimSpace(query)) + "%"
	rows, err := db.Query(
		`SELECT kp.player_tag, kp.name, kp.clan_tag, kp.clan_name
		 FROM known_players kp
		 LEFT JOIN user_accounts ua
		   ON ua.player_tag = kp.player_tag
		  AND ua.discord_user_id = ?
		 LEFT JOIN (
		   SELECT entity_tag, MAX(used_at) AS last_used
		   FROM command_usage
		   WHERE discord_user_id = ? AND entity_type = 'player'
		   GROUP BY entity_tag
		 ) cu ON cu.entity_tag = kp.player_tag
		 WHERE (? = '%%' OR LOWER(kp.name) LIKE ? OR LOWER(kp.player_tag) LIKE ?)
		 ORDER BY
		   CASE WHEN ua.player_tag IS NOT NULL THEN 0 ELSE 1 END,
		   CASE WHEN ua.is_main = 1 THEN 0 ELSE 1 END,
		   CASE WHEN cu.last_used IS NULL THEN 1 ELSE 0 END,
		   cu.last_used DESC,
		   kp.last_seen_at DESC
		 LIMIT ?`,
		discordUserID, discordUserID, q, q, q, limit,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()

	out := make([]playerSuggestion, 0, limit)
	for rows.Next() {
		var item playerSuggestion
		if rows.Scan(&item.Tag, &item.Name, &item.ClanTag, &item.ClanName) == nil {
			out = append(out, item)
		}
	}
	return out
}

func searchClans(query string, limit int) []clanSuggestion {
	if db == nil {
		return nil
	}
	q := "%" + strings.ToLower(strings.TrimSpace(query)) + "%"
	rows, err := db.Query(
		`SELECT clan_tag, name
		 FROM known_clans
		 WHERE (? = '%%' OR LOWER(name) LIKE ? OR LOWER(clan_tag) LIKE ?)
		 ORDER BY last_seen_at DESC
		 LIMIT ?`,
		q, q, q, limit,
	)
	if err != nil {
		return nil
	}
	defer rows.Close()

	out := make([]clanSuggestion, 0, limit)
	for rows.Next() {
		var item clanSuggestion
		if rows.Scan(&item.Tag, &item.Name) == nil {
			out = append(out, item)
		}
	}
	return out
}

func getPlayerTagByName(name string) (string, bool) {
	if db == nil || strings.TrimSpace(name) == "" {
		return "", false
	}
	var tag string
	err := db.QueryRow(
		`SELECT player_tag FROM known_players WHERE LOWER(name) = LOWER(?) ORDER BY last_seen_at DESC LIMIT 1`,
		strings.TrimSpace(name),
	).Scan(&tag)
	if err != nil {
		return "", false
	}
	return tag, true
}

func getClanTagByName(name string) (string, bool) {
	if db == nil || strings.TrimSpace(name) == "" {
		return "", false
	}
	var tag string
	err := db.QueryRow(
		`SELECT clan_tag FROM known_clans WHERE LOWER(name) = LOWER(?) ORDER BY last_seen_at DESC LIMIT 1`,
		strings.TrimSpace(name),
	).Scan(&tag)
	if err != nil {
		return "", false
	}
	return tag, true
}
