/*
 * TgMusicBot - Telegram Music Bot
 *  Licensed under GNU GPL v3
 */

// Package clones lets users run their own copy of the bot from a BotFather
// token. Every clone is a separate Telegram bot client that shares the same
// handlers, queue cache, database and assistant accounts as the main bot.
package clones

import (
	"ashokshau/tgmusic/config"
	"ashokshau/tgmusic/src/core/db"
	"ashokshau/tgmusic/src/handlers"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	td "github.com/AshokShau/gotdbot"
)

// RegisterFunc connects a bot token to Telegram and returns the ready client.
// dir is the TDLib database directory reserved for this bot.
type RegisterFunc func(token, dir string) (*td.Client, error)

var (
	mu      sync.RWMutex
	running = map[int64]*td.Client{}

	mainClient *td.Client
	register   RegisterFunc

	tokenRe = regexp.MustCompile(`^\d{5,15}:[A-Za-z0-9_-]{30,}$`)
	apiHTTP = &http.Client{Timeout: 12 * time.Second}
)

// Init stores the main client and the function used to connect new bots.
func Init(main *td.Client, reg RegisterFunc) {
	mainClient = main
	register = reg
}

// BotInfo is the part of Telegram's getMe answer we care about.
type BotInfo struct {
	ID       int64
	Name     string
	Username string
}

// TokenBotID returns the numeric bot ID of a token ("123456:ABC..." -> 123456).
func TokenBotID(token string) (int64, error) {
	head, _, ok := strings.Cut(token, ":")
	if !ok {
		return 0, errors.New("invalid token")
	}
	return strconv.ParseInt(head, 10, 64)
}

// ValidToken checks the shape of a bot token.
func ValidToken(token string) bool { return tokenRe.MatchString(token) }

// Lookup asks Telegram who owns the token. It fails for revoked or wrong tokens.
func Lookup(token string) (*BotInfo, error) {
	resp, err := apiHTTP.Get("https://api.telegram.org/bot" + token + "/getMe")
	if err != nil {
		// never leak the token through the error text
		return nil, errors.New("could not reach Telegram to verify the token")
	}
	defer func() { _ = resp.Body.Close() }()

	var out struct {
		Ok     bool `json:"ok"`
		Result struct {
			ID        int64  `json:"id"`
			FirstName string `json:"first_name"`
			Username  string `json:"username"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil || !out.Ok {
		return nil, errors.New("Telegram rejected this token. Please check it or generate a new one in @BotFather")
	}
	return &BotInfo{ID: out.Result.ID, Name: out.Result.FirstName, Username: out.Result.Username}, nil
}

func clientDir(botID int64) string { return fmt.Sprintf("database_clone_%d", botID) }

// Start connects one clone and attaches the full handler set to it.
func Start(c db.Clone) error {
	if register == nil {
		return errors.New("clone manager is not initialised")
	}

	mu.Lock()
	if _, ok := running[c.ID]; ok {
		mu.Unlock()
		return nil
	}
	mu.Unlock()

	client, err := register(c.Token, clientDir(c.ID))
	if err != nil {
		return err
	}

	handlers.LoadModules(client)

	mu.Lock()
	running[c.ID] = client
	mu.Unlock()

	slog.Info("[clones] clone started", "bot_id", c.ID, "username", c.Username)
	return nil
}

// Stop disconnects a clone and removes its local TDLib data.
func Stop(botID int64) {
	mu.Lock()
	client, ok := running[botID]
	delete(running, botID)
	mu.Unlock()

	if ok && client != nil {
		client.Close()
	}
	_ = os.RemoveAll(clientDir(botID))
}

// IsRunning reports whether the clone is currently connected.
func IsRunning(botID int64) bool {
	mu.RLock()
	defer mu.RUnlock()
	_, ok := running[botID]
	return ok
}

// RunningCount is the number of connected clones.
func RunningCount() int {
	mu.RLock()
	defer mu.RUnlock()
	return len(running)
}

// StartAll connects every clone stored in the database. Bots are started one
// after another with a short pause so TDLib is not flooded at boot.
func StartAll() {
	if !config.CloneEnabled {
		return
	}
	list, err := db.Instance.GetAllClones()
	if err != nil {
		slog.Error("[clones] could not load clones", "error", err)
		return
	}

	started := 0
	for _, c := range list {
		if err := Start(c); err != nil {
			slog.Warn("[clones] could not start clone", "bot_id", c.ID, "username", c.Username, "error", err)
			continue
		}
		started++
		time.Sleep(1500 * time.Millisecond)
	}
	slog.Info("[clones] startup finished", "started", started, "total", len(list))
}

// notifyLog posts a line to the logger chat from the main bot.
func notifyLog(text string) {
	if mainClient == nil || config.LoggerId == 0 {
		return
	}
	_, _ = mainClient.SendTextMessage(config.LoggerId, text, &td.SendTextMessageOpts{
		ParseMode:             "HTML",
		DisableWebPagePreview: true,
	})
}
