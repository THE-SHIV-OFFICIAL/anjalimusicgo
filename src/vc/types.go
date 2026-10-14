/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package vc

import (
	"log/slog"
	"regexp"
	"sync"
	"time"

	"ashokshau/tgmusic/src/core/cache"

	td "github.com/AshokShau/gotdbot"
	tg "github.com/amarnathcjd/gogram/telegram"
)

var logger = slog.Default()
var urlRegex = regexp.MustCompile(`^https?://`)

// TelegramCalls manages the state and operations for voice calls, including userbots and the main bot client.
type TelegramCalls struct {
	mu          sync.RWMutex
	assistants  map[int]*Assistant
	clients     map[int]*tg.Client
	statusCache *cache.Cache[td.ChatMemberStatus]
	inviteCache *cache.Cache[string]

	leavingMu sync.Mutex
	leaving   map[int]bool

	// chatBots remembers which bot client (main or a clone) is serving a chat,
	// so queue messages and stream-end events go out through the right bot.
	botMu    sync.RWMutex
	chatBots map[int64]*td.Client
}

var (
	instance *TelegramCalls
	once     sync.Once
)

// getCalls returns the singleton instance of the TelegramCalls manager, ensuring that only one instance is created.
func getCalls() *TelegramCalls {
	once.Do(func() {
		instance = &TelegramCalls{
			assistants:  make(map[int]*Assistant),
			clients:     make(map[int]*tg.Client),
			statusCache: cache.NewCache[td.ChatMemberStatus](2 * time.Hour),
			inviteCache: cache.NewCache[string](2 * time.Hour),
			leaving:     make(map[int]bool),
			chatBots:    make(map[int64]*td.Client),
		}
	})
	return instance
}

// Calls is the singleton instance of TelegramCalls, initialized lazily.
var Calls = getCalls()

// rememberBot records the bot client that is serving chatID.
func (c *TelegramCalls) rememberBot(chatID int64, bot *td.Client) {
	if bot == nil {
		return
	}
	c.botMu.Lock()
	c.chatBots[chatID] = bot
	c.botMu.Unlock()
}

// botFor returns the bot serving chatID, or fallback if none is known.
func (c *TelegramCalls) botFor(chatID int64, fallback *td.Client) *td.Client {
	c.botMu.RLock()
	defer c.botMu.RUnlock()
	if b, ok := c.chatBots[chatID]; ok && b != nil {
		return b
	}
	return fallback
}

// forgetBot drops the chat -> bot mapping.
func (c *TelegramCalls) forgetBot(chatID int64) {
	c.botMu.Lock()
	delete(c.chatBots, chatID)
	c.botMu.Unlock()
}
