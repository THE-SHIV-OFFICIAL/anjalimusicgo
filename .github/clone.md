# Clone system

Users can run their own copy of the bot from a BotFather token.

| Command | Who | What it does |
| --- | --- | --- |
| `/clone [BOT_TOKEN]` | everyone (private chat) | create a cloned bot |
| `/delclone` (`/rmbot`) `[BOT_ID]` | owner of the clone / devs | remove a cloned bot |
| `/myclone` (`/mybot`) | everyone | list your cloned bots |
| `/cloned` | owner / devs | list all cloned bots |
| `/totalbots` | owner / devs | total + currently running clones |
| `/clonedinfo` | owner / devs | detailed info (tokens are never shown) |
| `/delallclone confirm` | owner / devs | stop and delete every clone |

## How it works

* A clone is an extra bot client created with the same API ID/hash. It gets the
  exact same handler set (`handlers.LoadModules`) as the main bot, so every
  command, button and the new player panel work on clones too.
* Queue, settings, playlists, auth lists, users and the assistant (userbot)
  accounts are shared with the main bot.
* Clone records live in the `clones` MongoDB collection (`_id` = bot id) and
  every clone is started again automatically on boot.
* The `/clone` message is deleted right after it is read, and the command only
  works in a private chat, so the token never stays in a group.
* Playback events (queue messages, "next song") are always sent through the bot
  that started the playback (`vc.botFor`).

## Settings (`.env`)

```
CLONE_ENABLED=true   # turn the feature on/off
CLONE_PER_USER=3     # clones one user may create (devs are unlimited)
MAX_CLONES=30        # hard cap for this host (each clone is a TDLib instance)
THUMB_ENABLED=true   # glass now-playing card
```

Every running clone uses its own TDLib database folder (`database_clone_<bot id>`)
and roughly 60-100 MB of RAM, so size `MAX_CLONES` to your server.

## Building

`golang.org/x/image` was added for the thumbnail text. Run `go mod tidy` once
(the Dockerfile already does) to refresh `go.sum`.
