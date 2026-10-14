/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package handlers

import (
	"ashokshau/tgmusic/src/utils"
	"fmt"
	"html"
	"log/slog"
	"strings"

	"ashokshau/tgmusic/src/core"
	"ashokshau/tgmusic/src/core/cache"
	"ashokshau/tgmusic/src/core/db"
	"ashokshau/tgmusic/src/vc"

	td "github.com/AshokShau/gotdbot"
)

func playCallbackHandler(c *td.Client, cb *td.UpdateNewCallbackQuery) error {
	data := cb.DataString()

	// The progress label is informational only.
	if data == "play_noop" {
		chatID := cb.ChatId
		track := cache.ChatCache.GetPlayingTrack(chatID)
		if track == nil {
			_ = cb.Answer(c, 0, false, "There is no active playback.", "")
			return nil
		}
		played, err := vc.Calls.PlayedTime(chatID)
		if err != nil {
			_ = cb.Answer(c, 0, false, "Length: "+utils.SecToMin(track.Duration), "")
			return nil
		}
		_ = cb.Answer(c, 0, false, fmt.Sprintf("%s / %s", utils.SecToMin(int(played)), utils.SecToMin(track.Duration)), "")
		return nil
	}

	if !adminModeCB(c, cb) {
		return td.EndGroups
	}

	chatID := cb.ChatId
	user, err := c.GetUser(cb.SenderUserId)
	if err != nil {
		user = &td.User{FirstName: "Unknown", Id: cb.SenderUserId}
	}

	if !cache.ChatCache.IsActive(chatID) {
		text := "There is no active playback."
		_ = cb.Answer(c, 0, false, text, "")
		_, _ = cb.EditMessageText(c, text, &td.EditTextMessageOpts{ReplyMarkup: core.ControlButtons(""), ParseMode: "HTML", DisableWebPagePreview: true})
		return nil
	}

	currentTrack := cache.ChatCache.GetPlayingTrack(chatID)
	if currentTrack == nil {
		_ = cb.Answer(c, 0, false, "There is no active playback.", "")
		_, _ = cb.EditMessageText(c, "There is no active playback.", &td.EditTextMessageOpts{ReplyMarkup: core.ControlButtons(""), ParseMode: "HTML", DisableWebPagePreview: true})
		return nil
	}

	// refresh redraws the panel for the current state.
	refresh := func(status string) {
		_, _ = cb.EditMessageText(c, core.NowPlayingCaption(currentTrack, status), &td.EditTextMessageOpts{
			ReplyMarkup:           core.NowPlayingKeyboard(chatID, currentTrack),
			ParseMode:             "HTML",
			DisableWebPagePreview: true,
		})
	}

	switch {
	case strings.Contains(data, "play_skip"):
		core.SetPaused(chatID, false)
		if err := vc.Calls.PlayNext(c, chatID); err != nil {
			_ = cb.Answer(c, 0, false, "Unable to skip the current track.", "")
			_, _ = cb.EditMessageText(c, "Unable to skip the current track.", &td.EditTextMessageOpts{ReplyMarkup: core.ControlButtons(""), ParseMode: "HTML", DisableWebPagePreview: true})
			return nil
		}
		_ = cb.Answer(c, 0, false, "Track skipped.", "")
		_ = c.DeleteMessages(chatID, []int64{cb.MessageId}, &td.DeleteMessagesOpts{Revoke: true})
		return nil

	case strings.Contains(data, "play_stop"):
		if err := vc.Calls.Stop(chatID, false); err != nil {
			_ = cb.Answer(c, 0, false, "Unable to stop playback.", "")
			_, _ = cb.EditMessageText(c, "Unable to stop playback.", &td.EditTextMessageOpts{ReplyMarkup: core.ControlButtons(""), ParseMode: "HTML", DisableWebPagePreview: true})
			return nil
		}
		core.SetPaused(chatID, false)

		msg := fmt.Sprintf("<b>Playback stopped.</b>\nRequested by: %s", html.EscapeString(user.FirstName))
		_ = cb.Answer(c, 0, false, "Playback stopped.", "")
		_, err := cb.EditMessageText(c, msg, &td.EditTextMessageOpts{ReplyMarkup: core.ControlButtons(""), ParseMode: "HTML", DisableWebPagePreview: true})
		return err

	case strings.Contains(data, "play_pause"):
		if _, err = vc.Calls.Pause(chatID); err != nil {
			_ = cb.Answer(c, 0, false, "Unable to pause playback.", "")
			return nil
		}
		core.SetPaused(chatID, true)
		_ = cb.Answer(c, 0, false, "Playback paused.", "")
		refresh("ᴘᴀᴜꜱᴇᴅ ʙʏ " + utils.SmallCaps(user.FirstName) + ".")
		return nil

	case strings.Contains(data, "play_resume"):
		if _, err := vc.Calls.Resume(chatID); err != nil {
			_ = cb.Answer(c, 0, false, "Unable to resume playback.", "")
			return nil
		}
		core.SetPaused(chatID, false)
		_ = cb.Answer(c, 0, false, "Playback resumed.", "")
		refresh("")
		return nil

	case data == "play_loop":
		if cache.ChatCache.GetLoopCount(chatID) > 0 {
			cache.ChatCache.SetLoopCount(chatID, 0)
			_ = cb.Answer(c, 0, false, "Loop disabled.", "")
		} else {
			cache.ChatCache.SetLoopCount(chatID, 10)
			_ = cb.Answer(c, 0, false, "Loop enabled (10x).", "")
		}
		refresh("")
		return nil

	case data == "play_seekf" || data == "play_seekb":
		const step = 15
		played, err := vc.Calls.PlayedTime(chatID)
		if err != nil {
			_ = cb.Answer(c, 0, true, "Failed to read the current position.", "")
			return nil
		}

		target := int(played) + step
		if data == "play_seekb" {
			target = int(played) - step
		}
		if target < 0 {
			target = 0
		}
		if currentTrack.Duration > 0 && target >= currentTrack.Duration-2 {
			_ = cb.Answer(c, 0, true, "You cannot seek beyond the track duration.", "")
			return nil
		}

		if err := vc.Calls.SeekStream(c, chatID, currentTrack.FilePath, target, currentTrack.Duration, currentTrack.IsVideo); err != nil {
			_ = cb.Answer(c, 0, true, "Seek failed: "+err.Error(), "")
			return nil
		}
		_ = cb.Answer(c, 0, false, fmt.Sprintf("Moved to %s", utils.SecToMin(target)), "")
		return nil

	case strings.Contains(data, "play_mute"):
		if _, err := vc.Calls.Mute(chatID); err != nil {
			_ = cb.Answer(c, 0, false, "Unable to mute playback.", "")
			return nil
		}
		_ = cb.Answer(c, 0, false, "Playback muted.", "")
		refresh("ᴍᴜᴛᴇᴅ ʙʏ " + utils.SmallCaps(user.FirstName) + ".")
		return nil

	case strings.Contains(data, "play_unmute"):
		if _, err := vc.Calls.Unmute(chatID); err != nil {
			_ = cb.Answer(c, 0, false, "Unable to unmute playback.", "")
			return nil
		}
		_ = cb.Answer(c, 0, false, "Playback unmuted.", "")
		refresh("")
		return nil

	case strings.Contains(data, "play_add_to_list"):
		playlists, err := db.Instance.GetUserPlaylists(cb.SenderUserId)
		if err != nil {
			_ = cb.Answer(c, 0, false, "Unable to fetch playlists.", "")
			return nil
		}

		var playlistID string
		if len(playlists) == 0 {
			playlistID, err = db.Instance.CreatePlaylist("My Playlist (TgMusic)", cb.SenderUserId)
			if err != nil {
				_ = cb.Answer(c, 0, false, "Unable to create playlist.", "")
				return nil
			}
		} else {
			playlistID = playlists[0].ID
		}

		song := db.Song{
			URL:      currentTrack.URL,
			Name:     currentTrack.Name,
			TrackID:  currentTrack.TrackID,
			Duration: currentTrack.Duration,
			Platform: currentTrack.Platform,
		}

		err = db.Instance.AddSongToPlaylist(playlistID, song)
		if err != nil {
			_ = cb.Answer(c, 0, false, "Unable to add track to playlist.", "")
			return nil
		}

		playlist, err := db.Instance.GetPlaylist(playlistID)
		if err != nil {
			_ = cb.Answer(c, 0, false, "Playlist not found.", "")
			return nil
		}

		_ = cb.Answer(c, 0, false, fmt.Sprintf("Track \"%s\" added to playlist \"%s\".", song.Name, playlist.Name), "")
		return nil

	case strings.HasPrefix(data, "play_now_"):
		trackID := strings.TrimPrefix(data, "play_now_")
		if ok := cache.ChatCache.MoveTrackToFront(chatID, trackID); !ok {
			_ = cb.Answer(c, 0, false, "Track not found in queue.", "")
			return nil
		}
		if err := vc.Calls.PlayNext(c, chatID); err != nil {
			_ = cb.Answer(c, 0, false, "Unable to play the track.", "")
			return nil
		}
		_ = cb.Answer(c, 0, false, "Playing now.", "")
		_ = c.DeleteMessages(chatID, []int64{cb.MessageId}, &td.DeleteMessagesOpts{Revoke: true})
		return nil
	}

	refresh("")
	return nil
}

func vcPlayHandler(c *td.Client, cb *td.UpdateNewCallbackQuery) error {
	data := cb.DataString()

	if strings.Contains(data, "vcplay_close") {
		_ = cb.Answer(c, 0, false, "Closing panel.", "")
		_ = c.DeleteMessages(cb.ChatId, []int64{cb.MessageId}, &td.DeleteMessagesOpts{Revoke: true})
		return nil
	}

	slog.Info("Received vcplay callback", "arg1", data)
	return nil
}
