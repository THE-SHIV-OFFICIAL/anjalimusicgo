/*
 * TgMusicBot - Telegram Music Bot
 *  Licensed under GNU GPL v3
 */

package core

import (
	"sync"

	"ashokshau/tgmusic/config"
	"ashokshau/tgmusic/src/core/cache"
	"ashokshau/tgmusic/src/thumb"
	"ashokshau/tgmusic/src/utils"
	"context"
	"fmt"
	"html"
	"log/slog"
	"time"

	td "github.com/AshokShau/gotdbot"
)

// PlayerState describes how the now-playing panel should be rendered.
type PlayerState struct {
	Elapsed  int
	Duration int
	Paused   bool
	Loop     bool
	Autoplay bool
}

// PlayerKeyboard builds the now-playing control panel (progress row, transport
// row, seek/repo row and the autoplay toggle).
func PlayerKeyboard(st PlayerState) *td.ReplyMarkupInlineKeyboard {
	autoplay := "♫ ᴀᴜᴛᴏᴘʟᴀʏ: ᴅɪꜱᴀʙʟᴇᴅ"
	apStyle := rs()
	if st.Autoplay {
		autoplay = "♫ ᴀᴜᴛᴏᴘʟᴀʏ: ᴇɴᴀʙʟᴇᴅ"
	}

	pauseResume := cb("⏸", "play_pause", rs())
	if st.Paused {
		pauseResume = cb("▷", "play_resume", rs())
	}

	loopStyle := rs()
	loopText := "↻"
	if st.Loop {
		loopText = "🔁"
	}

	return &td.ReplyMarkupInlineKeyboard{
		Rows: [][]td.InlineKeyboardButton{
			{cb(utils.ProgressLabel(st.Elapsed, st.Duration), "play_noop", rs())},
			{
				pauseResume,
				cb("⏹", "play_stop", rs()),
				cb(loopText, "play_loop", loopStyle),
				cb("⏭", "play_skip", rs()),
			},
			{
				cb("-15ˢ", "play_seekb", rs()),
				url("ʀᴇᴘᴏ", config.RepoLink, rs()),
				cb("15ˢ+", "play_seekf", rs()),
			},
			{cb(autoplay, "autoplay_np", apStyle)},
		},
	}
}

// NowPlayingCaption is the caption shown under the thumbnail.
func NowPlayingCaption(t *utils.CachedTrack, status string) string {
	title := html.EscapeString(t.Name)
	if len(title) > 0 && len([]rune(t.Name)) > 38 {
		title = html.EscapeString(string([]rune(t.Name)[:38])) + "…"
	}
	if status == "" {
		status = "ᴘʟᴀʏʙᴀᴄᴋ ᴀᴄᴛɪᴠᴀᴛᴇᴅ."
	}
	return fmt.Sprintf(
		"<blockquote>🌸 <b>%s |\n"+
			"ᴇɴᴊᴏʏ ᴛʜᴇ ᴍᴜꜱɪᴄ |</b></blockquote>\n"+
			"<blockquote>▫️ <b>ᴍᴇʟᴏᴅʏ :</b> <a href='%s'>%s</a>\n"+
			"▫️ <b>ʟᴇɴɢᴛʜ :</b> %s\n"+
			"▫️ <b>ʀᴇǫᴜᴇꜱᴛᴇᴅ ʙʏ :</b> %s</blockquote>",
		status,
		html.EscapeString(t.URL), title,
		utils.SecToMin(t.Duration),
		html.EscapeString(t.User),
	)
}

var pausedChats sync.Map

// SetPaused remembers whether playback in a chat is paused (for the panel).
func SetPaused(chatID int64, paused bool) {
	if paused {
		pausedChats.Store(chatID, true)
		return
	}
	pausedChats.Delete(chatID)
}

// IsPaused reports whether playback in a chat is paused.
func IsPaused(chatID int64) bool {
	_, ok := pausedChats.Load(chatID)
	return ok
}

// NowPlayingKeyboard returns the panel for a track using the live chat state.
func NowPlayingKeyboard(chatID int64, t *utils.CachedTrack) *td.ReplyMarkupInlineKeyboard {
	return PlayerKeyboard(PlayerState{
		Duration: t.Duration,
		Paused:   IsPaused(chatID),
		Loop:     cache.ChatCache.GetLoopCount(chatID) > 0,
		Autoplay: cache.ChatCache.GetAutoplay(chatID),
	})
}

// SendNowPlaying posts the thumbnail card with the control panel. The card is
// generated here, after playback has already started, so it never delays the
// voice chat join. If the card cannot be built in time the static start image
// is used instead.
func SendNowPlaying(bot *td.Client, chatID int64, t *utils.CachedTrack) error {
	SetPaused(chatID, false)
	var file td.InputFile = td.InputFileString(config.StartImg)
	usedCard := false

	if config.ThumbEnabled {
		ctx, cancel := context.WithTimeout(context.Background(), 6*time.Second)
		path, err := thumb.Generate(ctx, t)
		cancel()
		if err != nil {
			slog.Warn("[thumb] card generation failed, using fallback image", "error", err)
		} else if path != "" {
			file = td.InputFileLocal{Path: path}
			usedCard = true
		}
	}

	opts := &td.SendPhotoOpts{
		Caption:     NowPlayingCaption(t, ""),
		ParseMode:   "HTML",
		ReplyMarkup: NowPlayingKeyboard(chatID, t),
	}

	if _, err := bot.SendPhoto(chatID, file, opts); err != nil {
		// A bad local file should never lose the message: retry with the static image.
		if usedCard {
			_, err2 := bot.SendPhoto(chatID, td.InputFileString(config.StartImg), opts)
			return err2
		}
		return err
	}
	return nil
}

// QueuedCaption is the caption shown when a track is added to the queue.
func QueuedCaption(t *utils.CachedTrack, position int) string {
	return fmt.Sprintf(
		"<blockquote>🌸 <b>ᴀᴅᴅᴇᴅ ᴛᴏ ǫᴜᴇᴜᴇ | #%d</b></blockquote>\n"+
			"<blockquote>▫️ <b>ᴍᴇʟᴏᴅʏ :</b> <a href='%s'>%s</a>\n"+
			"▫️ <b>ʟᴇɴɢᴛʜ :</b> %s\n"+
			"▫️ <b>ʀᴇǫᴜᴇꜱᴛᴇᴅ ʙʏ :</b> %s</blockquote>",
		position,
		html.EscapeString(t.URL), html.EscapeString(t.Name),
		utils.SecToMin(t.Duration),
		html.EscapeString(t.User),
	)
}
