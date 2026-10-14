/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package core

import (
	"ashokshau/tgmusic/config"
	"ashokshau/tgmusic/src/utils"
	"fmt"
	"math/rand/v2"

	"github.com/AshokShau/gotdbot"
)

func cb(text, data string, style gotdbot.ButtonStyle) gotdbot.InlineKeyboardButton {
	return gotdbot.InlineKeyboardButton{
		Text: text,
		Type: &gotdbot.InlineKeyboardButtonTypeCallback{
			Data: []byte(data),
		},
		Style: style,
	}
}

func userId(text string, userId int64, style gotdbot.ButtonStyle) gotdbot.InlineKeyboardButton {
	return gotdbot.InlineKeyboardButton{
		Text:  text,
		Type:  &gotdbot.InlineKeyboardButtonTypeUser{UserId: userId},
		Style: style,
	}
}

func url(text, link string, style gotdbot.ButtonStyle) gotdbot.InlineKeyboardButton {
	return gotdbot.InlineKeyboardButton{
		Text: text,
		Type: &gotdbot.InlineKeyboardButtonTypeUrl{
			Url: link,
		},
		Style: style,
	}
}

var CloseBtn = cb("𝐂ℓσѕє", "vcplay_close", gotdbot.ButtonStyleDanger{})
var HomeBtn = cb("𝐇σмє", "help_back", gotdbot.ButtonStylePrimary{})
var HelpBtn = cb("Help", "help_all", gotdbot.ButtonStyleDefault{})
var UserBtn = cb("Users", "help_user", gotdbot.ButtonStyleDefault{})
var AdminBtn = cb("Admins", "help_admin", gotdbot.ButtonStyleDefault{})
var OwnerBtn = cb("Owner", "help_owner", gotdbot.ButtonStyleDefault{})
var DevsBtn = cb("Devs", "help_devs", gotdbot.ButtonStyleDefault{})
var PlaylistBtn = cb("Playlist", "help_playlist", gotdbot.ButtonStyleDefault{})
var AutoplayBtn = cb("Autoplay", "help_autoplay", gotdbot.ButtonStyleDefault{})

var SourceCodeBtn = url("Source Code", "https://t.me/SukkuBeatzBot", gotdbot.ButtonStylePrimary{})
var channelBtn = url("Updates", config.SupportChannel, gotdbot.ButtonStyleDefault{})
var groupBtn = url("Group", config.SupportGroup, gotdbot.ButtonStyleDefault{})

func SupportKeyboard() *gotdbot.ReplyMarkupInlineKeyboard {
	return &gotdbot.ReplyMarkupInlineKeyboard{
		Rows: [][]gotdbot.InlineKeyboardButton{
			{channelBtn, groupBtn},
			{CloseBtn},
		},
	}
}

func SupportBtn() *gotdbot.ReplyMarkupInlineKeyboard {
	return &gotdbot.ReplyMarkupInlineKeyboard{
		Rows: [][]gotdbot.InlineKeyboardButton{
			{channelBtn, groupBtn},
		},
	}
}

func SettingsKeyboard(playMode, adminMode string, cmdDelete bool, language string) *gotdbot.ReplyMarkupInlineKeyboard {
	playText := "Everyone"
	if playMode == utils.Admins {
		playText = "Admins"
	}

	deleteText := "False"
	if cmdDelete {
		deleteText = "True"
	}

	adminText := "Everyone"
	if adminMode == utils.Admins {
		adminText = "Admins"
	}

	langText := "English"
	if language != "en" && language != "" {
		langText = language
	}

	return &gotdbot.ReplyMarkupInlineKeyboard{
		Rows: [][]gotdbot.InlineKeyboardButton{
			{
				cb("Play Mode ➜", "settings_main", gotdbot.ButtonStyleDefault{}),
				cb(playText, "settings_play", gotdbot.ButtonStyleDefault{}),
			},
			{
				cb("Command Delete ➜", "settings_main", gotdbot.ButtonStyleDefault{}),
				cb(deleteText, "settings_delete", gotdbot.ButtonStyleDefault{}),
			},
			{
				cb("Admin Mode ➜", "settings_main", gotdbot.ButtonStyleDefault{}),
				cb(adminText, "settings_admin", gotdbot.ButtonStyleDefault{}),
			},
			{
				cb("Language ➜", "settings_main", gotdbot.ButtonStyleDefault{}),
				cb(langText, "settings_lang", gotdbot.ButtonStyleDefault{}),
			},
			{CloseBtn},
		},
	}
}

func HelpMenuKeyboard() *gotdbot.ReplyMarkupInlineKeyboard {
	return &gotdbot.ReplyMarkupInlineKeyboard{
		Rows: [][]gotdbot.InlineKeyboardButton{
			{
				cb("ADMIN", "help_admin", rs()),
				cb("AUTH", "help_auth", rs()),
				cb("B-CAST", "help_bcast", rs()),
			},
			{
				cb("PLAY", "help_user", rs()),
				cb("SUDO", "help_devs", rs()),
				cb("SETTINGS", "help_owner", rs()),
			},
			{
				cb("AUTOPLAY", "help_autoplay", rs()),
				cb("PLAYLIST", "help_playlist", rs()),
				cb("VC-LOGS", "help_logs", rs()),
			},
			{
				cb("CLONE FEATURE", "help_clone", rs()),
			},
			{
				cb("←", "help_back", rs()),
			},
		},
	}
}

func BackHelpMenuKeyboard() *gotdbot.ReplyMarkupInlineKeyboard {
	return &gotdbot.ReplyMarkupInlineKeyboard{
		Rows: [][]gotdbot.InlineKeyboardButton{
			{cb("←", "help_all", rs())},
		},
	}
}

func ControlButtons(mode string) *gotdbot.ReplyMarkupInlineKeyboard {
	skipBtn := cb("➔I", "play_skip", gotdbot.ButtonStyleDefault{})
	stopBtn := cb("⎕", "play_stop", gotdbot.ButtonStyleDefault{})
	pauseBtn := cb("II", "play_pause", gotdbot.ButtonStyleDefault{})
	resumeBtn := cb("▷", "play_resume", gotdbot.ButtonStyleDefault{})

	switch mode {
	case "play":
		return &gotdbot.ReplyMarkupInlineKeyboard{
			Rows: [][]gotdbot.InlineKeyboardButton{
				{resumeBtn, pauseBtn, skipBtn, stopBtn},
				{CloseBtn},
			},
		}

	case "pause":
		return &gotdbot.ReplyMarkupInlineKeyboard{
			Rows: [][]gotdbot.InlineKeyboardButton{
				{resumeBtn, stopBtn, skipBtn},
				{CloseBtn},
			},
		}

	default:
		return &gotdbot.ReplyMarkupInlineKeyboard{
			Rows: [][]gotdbot.InlineKeyboardButton{
				{CloseBtn},
			},
		}
	}
}

func AddMeMarkup(username string) *gotdbot.ReplyMarkupInlineKeyboard {
	addMeBtn := url(
		"✧₊ ᴀᴅᴅ ᴍᴇ ᴛᴏ ʏᴏᴜʀ ᴄʜᴀᴛ ₊✧",
		fmt.Sprintf("https://t.me/%s?startgroup=true", username),
		rs(),
	)

	return &gotdbot.ReplyMarkupInlineKeyboard{
		Rows: [][]gotdbot.InlineKeyboardButton{
			{addMeBtn},
			{cb("ʜᴇʟᴘ & ᴄᴏᴍᴍᴀɴᴅꜱ", "help_all", rs())},
			{
				url("ᴜᴘᴅᴀᴛᴇꜱ", config.SupportChannel, rs()),
				url("ꜱᴜᴘᴘᴏʀᴛ", config.SupportGroup, rs()),
			},
			{cb("ʜᴏᴡ ᴛᴏ ᴄʟᴏɴᴇ", "help_clone", rs())},
		},
	}
}

func PlayNowButton(trackID string) gotdbot.InlineKeyboardButton {
	return cb("Play Now", fmt.Sprintf("play_now_%s", trackID), gotdbot.ButtonStyleDanger{})
}

func QueueMarkup(trackID string) *gotdbot.ReplyMarkupInlineKeyboard {
	return &gotdbot.ReplyMarkupInlineKeyboard{
		Rows: [][]gotdbot.InlineKeyboardButton{
			{PlayNowButton(trackID), CloseBtn},
		},
	}
}

// rs returns a random button colour (blue, green or red) so panels look different every time.
func rs() gotdbot.ButtonStyle {
	switch rand.IntN(3) {
	case 0:
		return gotdbot.ButtonStylePrimary{}
	case 1:
		return gotdbot.ButtonStyleSuccess{}
	default:
		return gotdbot.ButtonStyleDanger{}
	}
}
