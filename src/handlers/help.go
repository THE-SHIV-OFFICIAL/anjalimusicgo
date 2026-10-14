/*
 * TgMusicBot - Telegram Music Bot
 *  Copyright (c) 2025-2026 Ashok Shau
 *
 *  Licensed under GNU GPL v3
 *  See https://github.com/AshokShau/TgMusicBot
 */

package handlers

import (
	"ashokshau/tgmusic/config"
	"ashokshau/tgmusic/src/utils"
	"fmt"
	"html"
	"strings"

	"ashokshau/tgmusic/src/core"

	td "github.com/AshokShau/gotdbot"
)

// welcomeHTML is the start screen: banner, greeting and feature list.
func welcomeHTML(botName, userName string) string {
	bot := html.EscapeString(botName)
	return fmt.Sprintf(
		"<img src=\"%s\"/>\n"+
			"<blockquote>%s <b>%s</b>,</blockquote>\n"+
			"<blockquote>✧ <b>%s %s</b> — %s<br>"+
			"✧ %s<br>"+
			"✧ %s<br>"+
			"✧ %s<br>"+
			"•──────────────────────•<br>"+
			"✧ %s</blockquote>",
		config.StartImg,
		utils.SmallCaps("salutations"), html.EscapeString(userName),
		utils.SmallCaps("welcome to"), bot,
		utils.SmallCaps("a powerful and high-speed tg music bot"),
		utils.SmallCaps("built for smooth • stable • lag-free music streaming"),
		utils.SmallCaps("powered by an optimized youtube api for instant playback"),
		utils.SmallCaps("enjoy high quality audio with seamless control"),
		utils.SmallCaps("use help to view all commands and features"),
	)
}

// helpIntroHTML is the screen shown above the category buttons.
func helpIntroHTML(botName string) string {
	return "<img src=\"" + config.StartImg + "\"/>\n" +
		"<blockquote>🔮 <b>" + utils.SmallCaps("explore the complete command index below") + "</b></blockquote>\n" +
		"<blockquote>• " + utils.SmallCaps("access expert technical guidance & real-time support") + "</blockquote>\n" +
		"<blockquote>• " + utils.SmallCaps("execute all commands using standard prefix") + " ➜</blockquote>"
}

type helpCategory struct {
	Title   string
	Content string
}

// cmdList renders "↣ /cmd : description" lines inside one quote block.
func cmdList(lines ...[2]string) string {
	var sb strings.Builder
	sb.WriteString("<blockquote>")
	for i, l := range lines {
		fmt.Fprintf(&sb, "↣ <code>%s</code> : %s", l[0], utils.SmallCaps(l[1]))
		if i < len(lines)-1 {
			sb.WriteString("<br>")
		}
	}
	sb.WriteString("</blockquote>")
	return sb.String()
}

func getHelpCategories() map[string]helpCategory {
	return map[string]helpCategory{
		"help_user": {
			Title: "Play Commands",
			Content: "<h3>⊙ " + utils.SmallCaps("play commands") + " :</h3>" + cmdList(
				[2]string{"/play [song | url]", "play music from youtube, spotify, jiosaavn, apple music and more"},
				[2]string{"/vplay [song | url]", "play a video in the group video chat"},
				[2]string{"/fplay [song | url]", "play a track immediately and skip the queue"},
				[2]string{"/fvplay [song | url]", "play a video immediately and skip the queue"},
				[2]string{"/queue", "show the current queue"},
				[2]string{"/ping", "check response time and system info"},
				[2]string{"/privacy", "view the privacy policy"},
			),
		},
		"help_admin": {
			Title: "Admin Commands",
			Content: "<h3>⊙ " + utils.SmallCaps("admin commands") + " :</h3>" + cmdList(
				[2]string{"/skip", "skip the current track"},
				[2]string{"/pause", "pause playback"},
				[2]string{"/resume", "resume playback"},
				[2]string{"/seek [seconds]", "jump forward in the current track"},
				[2]string{"/mute", "mute the stream"},
				[2]string{"/unmute", "unmute the stream"},
				[2]string{"/remove [index]", "remove a track from the queue"},
				[2]string{"/loop [0-10]", "repeat the current track"},
				[2]string{"/end", "stop playback and clear the queue"},
				[2]string{"/reload", "refresh the admin cache"},
			),
		},
		"help_auth": {
			Title: "Auth Commands",
			Content: "<h3>⊙ " + utils.SmallCaps("auth commands") + " :</h3>" + cmdList(
				[2]string{"/auth [reply | id]", "authorize a user to use admin commands"},
				[2]string{"/rmauth [reply | id]", "remove a user's authorization"},
				[2]string{"/authlist", "show all authorized users of the chat"},
			),
		},
		"help_bcast": {
			Title: "Broadcast Commands",
			Content: "<h3>⊙ " + utils.SmallCaps("for owners & sudoers") + " :</h3>" + cmdList(
				[2]string{"/broadcast", "reply to a message to send it to everyone"},
				[2]string{"/broadcast -chat", "groups only"},
				[2]string{"/broadcast -user", "users only"},
				[2]string{"/broadcast -copy", "send as a copy instead of forwarding"},
				[2]string{"/stop_broadcast", "stop a running broadcast"},
			),
		},
		"help_devs": {
			Title: "Sudo Commands",
			Content: "<h3>⊙ " + utils.SmallCaps("for owners & sudoers") + " :</h3>" + cmdList(
				[2]string{"/stats", "bot, host and database statistics"},
				[2]string{"/av", "list all active voice chats"},
				[2]string{"/clearass", "clear assistant assignments"},
				[2]string{"/leaveall", "make the assistants leave every chat"},
				[2]string{"/logger [on|off]", "toggle play logs"},
			),
		},
		"help_owner": {
			Title: "Settings",
			Content: "<h3>⊙ " + utils.SmallCaps("chat settings") + " :</h3>" + cmdList(
				[2]string{"/settings", "play mode, admin mode, command delete and language"},
			),
		},
		"help_autoplay": {
			Title: "Autoplay",
			Content: "<h3>⊙ " + utils.SmallCaps("autoplay") + " :</h3>" + cmdList(
				[2]string{"/autoplay", "queue recommended songs automatically when the queue ends"},
			) + "<p><i>" + utils.SmallCaps("you can also toggle autoplay from the button on the player") + ".</i></p>",
		},
		"help_playlist": {
			Title: "Playlist Commands",
			Content: "<h3>⊙ " + utils.SmallCaps("playlist commands") + " :</h3>" + cmdList(
				[2]string{"/createplaylist [name]", "create a new playlist"},
				[2]string{"/deleteplaylist [id]", "delete one of your playlists"},
				[2]string{"/addtoplaylist [id]", "add the playing track to a playlist"},
				[2]string{"/removefromplaylist [id]", "remove a track from a playlist"},
				[2]string{"/playlistinfo [id]", "show a playlist"},
				[2]string{"/myplaylists", "list your playlists"},
			),
		},
		"help_logs": {
			Title: "VC Logs",
			Content: "<h3>⊙ " + utils.SmallCaps("vc logs") + " :</h3>" + cmdList(
				[2]string{"/logger on", "send a log line to the logger chat for every track played"},
				[2]string{"/logger off", "stop sending play logs"},
			),
		},
		"help_clone": {
			Title: "Clone Feature",
			Content: "<h3>⊙ " + utils.SmallCaps("clone commands") + " :</h3>" + cmdList(
				[2]string{"/clone [BOT_TOKEN]", "clone your own bot using the token from @BotFather"},
				[2]string{"/delclone (/rmbot) [BOT_ID]", "remove one of your cloned bots"},
				[2]string{"/myclone (/mybot)", "check the bots you have cloned"},
			) + "<h3>⊙ " + utils.SmallCaps("for owners & sudoers") + " :</h3>" + cmdList(
				[2]string{"/cloned", "check how many bots have been cloned"},
				[2]string{"/totalbots", "view the total number of cloned bots"},
				[2]string{"/clonedinfo", "view detailed information about all cloned bots"},
				[2]string{"/delallclone confirm", "delete all clone databases and bots"},
			),
		},
	}
}

func editRich(c *td.Client, cb *td.UpdateNewCallbackQuery, text string, markup *td.ReplyMarkupInlineKeyboard) {
	rich := &td.InputRichMessage{Source: &td.RichMessageSourceHtml{Text: text}}
	_, _ = c.EditMessageText(cb.ChatId, &td.InputMessageRichMessage{Message: rich}, cb.MessageId, &td.EditMessageTextOpts{ReplyMarkup: markup})
}

func helpCallbackHandler(c *td.Client, cb *td.UpdateNewCallbackQuery) error {
	data := cb.DataString()

	user, err := c.GetUser(cb.SenderUserId)
	if err != nil {
		user = &td.User{FirstName: "User", Id: cb.SenderUserId}
	}

	switch data {
	case "help_all":
		_ = cb.Answer(c, 0, false, "Opening help menu...", "")
		editRich(c, cb, helpIntroHTML(c.Me.FirstName), core.HelpMenuKeyboard())
		return nil

	case "help_back":
		_ = cb.Answer(c, 0, false, "Returning to main menu...", "")
		editRich(c, cb, welcomeHTML(c.Me.FirstName, user.FirstName), core.AddMeMarkup(c.Me.Usernames.EditableUsername))
		return nil
	}

	if category, ok := getHelpCategories()[data]; ok {
		_ = cb.Answer(c, 0, false, category.Title, "")
		banner := "<img src=\"" + config.StartImg + "\"/>\n"
		editRich(c, cb, banner+category.Content, core.BackHelpMenuKeyboard())
		return nil
	}

	_ = cb.Answer(c, 0, true, "Unknown help category.", "")
	return nil
}
