# SneakyGoblin

SneakyGoblin is a Discord bot for [Clash of Clans](https://supercell.com/en/games/clashofclans/) clans and players. It pulls live data from the official Clash of Clans API and presents it as rich embeds in Discord. Users can link their in-game accounts with token verification, set a default profile, and query clan or player stats without leaving the server.

## Features

- **Clan panel** — Tabbed view (Overview, Members, Wars, Clan Capital) with interactive buttons
- **Member roster** — Paginated inline table (15 per page) with sortable columns: league & trophies, trophies, town hall, role, donations, builder trophies, and more
- **War log** — Paginated war history (15 per page), current-war summary, win/loss/tie record, and sortable columns
- **Player profiles** — Town Hall, trophies, leagues, donations, builder base, legend stats, and clan membership
- **Progression views** — Tabbed upgrade progress (troops, heroes, spells, equipment), achievements (paginated), and player profile
- **Achievements browser** — Total star completion, compact number formatting (K/M/B), and sort by default order or progress
- **Account linking** — Verify ownership via the in-game API token (Clash `verifytoken` endpoint)
- **Smart defaults** — Optional `player` and `clan` arguments fall back to your linked main account; `/clan` uses your verified account’s clan when you omit a tag
- **Autocomplete** — Player and clan options suggest tags and names the bot has seen before
- **Remote cache** — [Turso](https://turso.tech/) stores known players/clans and linked Discord accounts for faster autocomplete and lookups

## Requirements

- Go 1.25 or newer
- A [Turso](https://turso.tech/) database (see [docs/turso.md](docs/turso.md))
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

On startup the bot connects to your Turso database, connects to Discord, clears global slash commands, and registers commands on the guild specified by `DISCORD_GUILD_ID`. Restart the bot after changing command definitions or interactive components so Discord receives the updates.

## Configuration

| Variable | Required | Description |
|----------|----------|-------------|
| `COC_TOKEN` | Yes | Clash of Clans API bearer token |
| `DISCORD_TOKEN` | Yes | Discord bot token |
| `DISCORD_APPLICATION_ID` | Yes | Discord application ID (used for command registration) |
| `DISCORD_GUILD_ID` | Yes | Guild (server) ID where slash commands are registered |
| `DISCORD_PUBLIC_KEY` | No | Loaded from `.env` but not used by the current codebase |
| `TURSO_DATABASE_URL` | Yes | Turso database URL (from `turso db show --url`) |
| `TURSO_DATABASE_TOKEN` | Yes | Turso database auth token |

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
| `/clan overview [clan]` | Open the clan panel on the overview tab |
| `/clan members [clan]` | Open the clan panel on the members tab |
| `/clan wars [clan]` | Open the clan panel on the wars tab |
| `/clan capital [clan]` | Open the clan panel on the clan capital tab |

Omit `clan` to use your verified account’s clan where applicable.

The clan panel uses tabs:

| Tab | What it shows |
|-----|----------------|
| **Overview** | Level, points, war record, requirements, labels, and clan details |
| **Members** | Paginated roster table with sort menu and prev/next controls |
| **Wars** | Current war (if any), war-log record (W/L/T), and paginated war history |
| **Clan Capital** | Capital points, leagues, and district info |

**Member sort options:** League & Trophies (default), Trophies, Town Hall, Role, Troops Donated, Troops Received, XP Level, Builder Trophies.

**War log sort options:** Date (default), Result, Opponent, Stars, Destruction, War Size.

### Player

| Command | Description |
|---------|-------------|
| `/player overview [player]` | Player profile summary |
| `/player troops [player]` | Troop levels |
| `/player heroes [player]` | Hero levels |
| `/player spells [player]` | Spell levels |
| `/player equipment [player]` | Hero equipment levels |
| `/player achievements [player]` | Achievement progress (paginated, sortable) |

Each subcommand opens the same tabbed panel on the matching page. Append `player:<tag or name>` to target a specific account. Without it, the bot uses your main linked player.

**Achievements view** includes total star progress (`earned / possible`), 15 achievements per page, prev/next buttons, and a sort menu:

- Default Order
- Progress (Low to High)
- Progress (High to Low)

Values of 1,000 or greater use compact notation with three significant figures (for example `1.00B`, `1.23M`, `1.50K`).

### Verify (account linking)

| Command | Description |
|---------|-------------|
| `/verify add player:<tag>` | Open a modal to enter your in-game API token and link the account |
| `/verify list` | List accounts linked to your Discord user |
| `/verify remove player:<tag>` | Unlink an account |
| `/verify main player:<tag>` | Set your default player for commands that omit a tag |

The first successfully linked account becomes your main account if none is set. Use `/verify main` to change it.

## How verification works

1. You run `/verify add` with your player tag (for example `#2ABC123`).
2. The bot shows a modal asking for the one-time API token from the game.
3. The bot calls the Clash API verify endpoint to confirm you own that account.
4. On success, the tag is stored in Turso and associated with your Discord user ID.

Linked accounts are per Discord user and stored in your Turso database.

## Embeds and UI

- Embeds use brand color `#00c950` and a **SneakyGoblin** footer with a timestamp.
- Player tags appear as subtext under the title for easy scanning.
- Clan and player panels use Discord message components (tabs, select menus, and pagination buttons). These update the same message in place when you interact with them.

## Project layout

```
.
├── main.go       # Bot entrypoint, env loading, Discord session, command sync
├── commands.go   # Slash command definitions, handlers, embed builders, UI components
├── api.go        # Clash of Clans HTTP client
├── models.go     # API response types (Clash data model)
├── db.go         # Turso schema and persistence
├── go.mod
└── .env.example
```

## Data storage

The bot connects to a remote [Turso](https://turso.tech/) database (schema is created on first run) and stores:

- **user_accounts** — Discord user ID, player tag, main-account flag
- **known_players** — Cached player names, clan tags, last seen
- **known_clans** — Cached clan names and tags
- **command_usage** — Recent player lookups for autocomplete ranking

See [docs/turso.md](docs/turso.md) for setup. Turso handles persistence across deployments; no local database file is required.

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
| Buttons or menus do nothing | Bot restarted after component ID changes; you are clicking a message from the current bot session |
| `COC token is not configured` | `COC_TOKEN` set in `.env` and loaded (`.env` present in working directory) |
| Clan command asks to verify | Link an account with `/verify add`; account must be in a clan; run `/player overview` once to refresh cached clan data |
| War log empty or private | Clan war log must be public in-game; API returns limited data when private |
| Verification fails | Token is fresh (one-time use); tag includes `#`; API key is valid |
| Turso connection errors | `TURSO_DATABASE_URL` and `TURSO_DATABASE_TOKEN` set in `.env`; database exists and token is valid |

## Acknowledgements

- Game data via the [Clash of Clans API](https://developer.clashofclans.com/) (Supercell)
- Discord integration via [discordgo](https://github.com/bwmarrin/discordgo)

This project is not affiliated with or endorsed by Supercell. Clash of Clans is a trademark of Supercell Oy. This material is unofficial and is not endorsed by Supercell. For more information see Supercell's Fan Content Policy: www.supercell.com/fan-content-policy.
