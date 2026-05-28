# SneakyGoblin

SneakyGoblin is a Discord bot for [Clash of Clans](https://supercell.com/en/games/clashofclans/) clans and players. It pulls live data from the official Clash of Clans API and presents it as rich embeds in Discord. Users can link their in-game accounts with token verification, set a default profile, and query clan or player stats without leaving the server.

## Features

- **Clan summaries** — Members, level, points, war record, capital points, requirements, labels, and more
- **Player profiles** — Town Hall, trophies, leagues, donations, builder base, legend stats, and clan membership
- **Progression views** — Heroes, troops, spells, hero equipment, achievements, and combined upgrade progress
- **Account linking** — Verify ownership via the in-game API token (Clash `verifytoken` endpoint)
- **Smart defaults** — Optional `player` and `clan` arguments fall back to your linked main account; `/clan stats` uses your verified account’s clan when you omit a tag
- **Autocomplete** — Player and clan options suggest tags and names the bot has seen before
- **Local cache** — SQLite stores known players/clans and linked Discord accounts for faster autocomplete and lookups

## Requirements

- Go 1.25 or newer
- CGO enabled (used by `github.com/mattn/go-sqlite3`)
- A [Clash of Clans API key](https://developer.clashofclans.com/)
- A [Discord application](https://discord.com/developers/applications) with a bot token

## Quick start

### 1. Clone and configure

```bash
git clone https://github.com/<your-user>/SneakyGoblin.git
cd SneakyGoblin
cp .env.example .env
```

Edit `.env` with your tokens (see [Configuration](#configuration)).

### 2. Install dependencies and run

```bash
go mod download
go run .
```

On startup the bot opens a SQLite database (`data.db`), connects to Discord, clears global slash commands, and registers commands on the guild specified by `DISCORD_GUILD_ID`. Restart the bot after changing command definitions so Discord receives the updated slash commands.

## Configuration

| Variable | Required | Description |
|----------|----------|-------------|
| `COC_TOKEN` | Yes | Clash of Clans API bearer token |
| `DISCORD_TOKEN` | Yes | Discord bot token |
| `DISCORD_APPLICATION_ID` | Yes | Discord application ID (used for command registration) |
| `DISCORD_GUILD_ID` | Yes | Guild (server) ID where slash commands are registered |
| `DISCORD_PUBLIC_KEY` | No | Loaded from `.env` but not used by the current codebase |

`DISCORD_GUILD_ID` is required because commands are registered as **guild commands** for faster iteration during development. To serve multiple servers you would need to adjust command registration in `main.go` (for example, global registration or per-guild loops).

## Discord setup

1. Create an application in the [Discord Developer Portal](https://discord.com/developers/applications).
2. Add a **Bot** user and copy the bot token into `DISCORD_TOKEN`.
3. Copy the **Application ID** into `DISCORD_APPLICATION_ID`.
4. Enable **Message Content Intent** only if you plan to extend the bot beyond slash commands (not required for current features).
5. Under **OAuth2 → URL Generator**, select `bot` and `applications.commands`, then invite the bot to your server.
6. Enable **Developer Mode** in Discord, right-click your server, and copy the server ID into `DISCORD_GUILD_ID`.

## Clash of Clans API setup

1. Log in at [developer.clashofclans.com](https://developer.clashofclans.com/).
2. Create an API key tied to your deployment IP (or update the key when your IP changes).
3. Set `COC_TOKEN` to that key.

Player verification uses `POST /v1/players/{tag}/verifytoken`. Users generate a one-time token in-game under **Settings → More Settings → API Token**.

## Commands

All commands are slash commands. Most player and clan commands accept an optional tag or name; if omitted, the bot uses your **main** linked account where applicable.

### Help

| Command | Description |
|---------|-------------|
| `/help` | List all commands |
| `/help command:<name>` | Usage for a specific command (autocomplete) |

### Clan

| Command | Description |
|---------|-------------|
| `/clan stats` | Clan overview for your verified account’s clan |
| `/clan stats clan:<tag or name>` | Clan overview for a specific clan |

### Player

| Command | Description |
|---------|-------------|
| `/player profile` | Player summary |
| `/player heroes` | Hero levels |
| `/player troops` | Troop levels |
| `/player spells` | Spell levels |
| `/player equipment` | Hero equipment levels |
| `/player achievements` | Achievement progress |
| `/player upgrade-progress` | Troops, heroes, and spells in one view |

Append `player:<tag or name>` to any player subcommand to target a specific account. Without it, the bot uses your main linked player.

### Verify (account linking)

| Command | Description |
|---------|-------------|
| `/verify verify player:<tag>` | Open a modal to enter your in-game API token and link the account |
| `/verify list` | List accounts linked to your Discord user |
| `/verify remove player:<tag>` | Unlink an account |
| `/verify set-main player:<tag>` | Set your default player for commands that omit a tag |

The first successfully linked account becomes your main account if none is set. Use `/verify set-main` to change it.

## How verification works

1. You run `/verify verify` with your player tag (for example `#2ABC123`).
2. The bot shows a modal asking for the one-time API token from the game.
3. The bot calls the Clash API verify endpoint to confirm you own that account.
4. On success, the tag is stored in SQLite and associated with your Discord user ID.

Linked accounts are per Discord user and per guild context (stored locally on the machine running the bot).

## Project layout

```
.
├── main.go       # Bot entrypoint, env loading, Discord session, command sync
├── commands.go   # Slash command definitions, handlers, embed builders
├── api.go        # Clash of Clans HTTP client
├── models.go     # API response types (Clash data model)
├── db.go         # SQLite schema and persistence
├── go.mod
├── .env.example
└── data.db       # Created at runtime (gitignored)
```

## Data storage

SQLite file `data.db` (created on first run) holds:

- **user_accounts** — Discord user ID, player tag, main-account flag
- **known_players** — Cached player names, clan tags, last seen
- **known_clans** — Cached clan names and tags
- **command_usage** — Recent player lookups for autocomplete ranking

The database is local to the bot process. Back it up if you migrate hosts or risk losing linked accounts and cache data.

## Building for production

```bash
go build -o sneakygoblin .
./sneakygoblin
```

Run the binary under a process manager (systemd, Docker, etc.) and ensure the host IP matches your Clash API key restrictions.

## Troubleshooting

| Issue | What to check |
|-------|----------------|
| Slash commands missing | Bot restarted after code changes; `DISCORD_GUILD_ID` matches the server; bot has `applications.commands` scope |
| `COC token is not configured` | `COC_TOKEN` set in `.env` and loaded (`.env` present in working directory) |
| Clan command asks to verify | Link an account with `/verify verify`; account must be in a clan; run `/player profile` once to refresh cached clan data |
| Verification fails | Token is fresh (one-time use); tag includes `#`; API key is valid |
| SQLite errors on Linux | CGO and `libsqlite3` dev packages installed |

## Acknowledgements

- Game data via the [Clash of Clans API](https://developer.clashofclans.com/) (Supercell)
- Discord integration via [discordgo](https://github.com/bwmarrin/discordgo)

This project is not affiliated with or endorsed by Supercell. Clash of Clans is a trademark of Supercell Oy.
