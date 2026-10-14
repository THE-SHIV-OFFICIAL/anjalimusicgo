/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package handlers

import (
	"fmt"
	"runtime"
	"time"

	"ashokshau/tgmusic/src/core"
	"ashokshau/tgmusic/src/core/db"

	td "github.com/AshokShau/gotdbot"
)

// pingHandler handles the /ping command.
func pingHandler(c *td.Client, m *td.Message) error {

	start := time.Now()

	msg, err := m.ReplyText(c, "Pinging… please wait…", nil)
	if err != nil {
		return err
	}

	latency := time.Since(start).Milliseconds()
	uptime := getFormattedDuration(time.Since(startTime))

	response := fmt.Sprintf(
		"<b>📊 System Performance Metrics</b>\n\n"+
			"<b>Bot Latency:</b> <code>%d ms</code>\n"+
			"<b>Uptime:</b> <code>%s</code>\n"+
			"<b>Go Routines:</b> <code>%d</code>\n",
		latency, uptime, runtime.NumGoroutine(),
	)

	_, err = msg.EditText(c, response, &td.EditTextMessageOpts{ParseMode: "HTML"})
	return err
}

// startHandler handles the /start command.
func startHandler(c *td.Client, m *td.Message) error {
	chatID := m.ChatId

	if m.IsPrivate() {
		go func(chatID int64) {
			_ = db.Instance.AddUser(chatID)
		}(chatID)

		response := welcomeHTML(c.Me.FirstName, firstName(c, m))

		richMessage := &td.InputRichMessage{
			Source: &td.RichMessageSourceHtml{
				Text: response,
			},
		}

		_, err := m.ReplyRichMessage(c, richMessage, &td.SendTextMessageOpts{
			ReplyMarkup: core.AddMeMarkup(c.Me.Usernames.EditableUsername),
		})

		return err
	}

	go func(chatID int64) {
		_ = db.Instance.AddChat(chatID)
	}(chatID)

	uptime := getFormattedDuration(time.Since(startTime))
	htmlText := fmt.Sprintf(
		"<h3>%s is ready!</h3>\n"+
			"<p><b>Uptime:</b> <code>%s</code></p>\n"+
			"<p><i>A feature-rich music bot for your group video chats. Play your favorite tracks seamlessly.</i></p>",
		c.Me.FirstName,
		uptime,
	)

	richMessage := &td.InputRichMessage{
		Source: &td.RichMessageSourceHtml{
			Text: htmlText,
		},
	}

	_, err := m.ReplyRichMessage(c, richMessage, &td.SendTextMessageOpts{
		ReplyMarkup: core.SupportBtn(),
	})

	return err
}
