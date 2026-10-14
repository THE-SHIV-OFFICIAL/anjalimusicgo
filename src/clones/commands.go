/*
 * TgMusicBot - Telegram Music Bot
 *  Licensed under GNU GPL v3
 */

package clones

import (
	"ashokshau/tgmusic/config"
	"ashokshau/tgmusic/src/core/db"
	"ashokshau/tgmusic/src/handlers"
	"ashokshau/tgmusic/src/utils"
	"fmt"
	"html"
	"log/slog"
	"strconv"
	"strings"
	"time"

	td "github.com/AshokShau/gotdbot"
)

// RegisterCommands attaches the clone management commands to the main bot.
func RegisterCommands(c *td.Client) {
	if !config.CloneEnabled {
		return
	}
	c.OnCommand("clone", cloneHandler)
	c.OnCommand("delclone", delCloneHandler)
	c.OnCommand("rmbot", delCloneHandler)
	c.OnCommand("myclone", myCloneHandler)
	c.OnCommand("mybot", myCloneHandler)
	c.OnCommand("cloned", clonedHandler)
	c.OnCommand("totalbots", totalBotsHandler)
	c.OnCommand("clonedinfo", clonedInfoHandler)
	c.OnCommand("delallclone", delAllCloneHandler)
}

var htmlOpts = &td.SendTextMessageOpts{ParseMode: "HTML", DisableWebPagePreview: true}

func isDev(m *td.Message) bool {
	id := m.SenderID()
	for _, d := range config.DEVS {
		if d == id {
			return true
		}
	}
	return false
}

func reply(c *td.Client, m *td.Message, text string) {
	if _, err := m.ReplyText(c, text, htmlOpts); err != nil {
		slog.Warn("[clones] reply failed", "error", err)
	}
}

func sc(s string) string { return utils.SmallCaps(s) }

// ------------------------------------------------------------------ /clone
func cloneHandler(c *td.Client, m *td.Message) error {
	token := strings.TrimSpace(handlers.Args(m))

	// A token pasted in a chat must never stay there.
	if token != "" {
		go func() {
			_ = c.DeleteMessages(m.ChatId, []int64{m.Id}, &td.DeleteMessagesOpts{Revoke: true})
		}()
	}

	if !m.IsPrivate() {
		reply(c, m, "<b>🔒 "+sc("please use this command in my private chat")+"</b>\n\n"+
			"A bot token is a password. Open my DM and send <code>/clone [BOT_TOKEN]</code> there.")
		return td.EndGroups
	}

	if token == "" {
		reply(c, m, "<blockquote><b>"+sc("how to clone")+"</b></blockquote>\n"+
			"<blockquote>1. Open @BotFather and send <code>/newbot</code>\n"+
			"2. Copy the <b>HTTP API token</b> you receive\n"+
			"3. Send it here: <code>/clone [BOT_TOKEN]</code></blockquote>")
		return td.EndGroups
	}

	if !ValidToken(token) {
		reply(c, m, "❌ <b>Invalid token format.</b>\nIt should look like <code>123456789:ABC-DEF...</code>")
		return td.EndGroups
	}

	botID, err := TokenBotID(token)
	if err != nil {
		reply(c, m, "❌ <b>Invalid token.</b>")
		return td.EndGroups
	}

	if mainClient != nil && botID == mainClient.Me.Id {
		reply(c, m, "❌ This is the main bot's token. Please create a new bot in @BotFather.")
		return td.EndGroups
	}

	if _, exists, err := db.Instance.GetClone(botID); err == nil && exists {
		reply(c, m, "⚠️ This bot is already cloned.")
		return td.EndGroups
	}

	owner := m.SenderID()
	if !isDev(m) {
		mine, err := db.Instance.GetUserClones(owner)
		if err == nil && int64(len(mine)) >= config.ClonePerUser {
			reply(c, m, fmt.Sprintf("❌ <b>Clone limit reached.</b>\nYou can run up to <b>%d</b> cloned bot(s). Remove one with <code>/delclone [BOT_ID]</code>.", config.ClonePerUser))
			return td.EndGroups
		}
	}

	if total, err := db.Instance.CountClones(); err == nil && config.MaxClones > 0 && total >= config.MaxClones {
		reply(c, m, "❌ The clone limit of this host has been reached. Please try again later.")
		return td.EndGroups
	}

	status, err := m.ReplyText(c, "⏳ <b>"+sc("verifying your bot token")+"...</b>", htmlOpts)
	if err != nil {
		return err
	}
	edit := func(text string) {
		_, _ = status.EditText(c, text, &td.EditTextMessageOpts{ParseMode: "HTML", DisableWebPagePreview: true})
	}

	info, err := Lookup(token)
	if err != nil {
		edit("❌ <b>" + html.EscapeString(err.Error()) + "</b>")
		return td.EndGroups
	}

	edit("⚙️ <b>" + sc("starting your bot") + "...</b>")

	rec := db.Clone{
		ID:        info.ID,
		Token:     token,
		Username:  info.Username,
		Name:      info.Name,
		OwnerID:   owner,
		CreatedAt: time.Now().UTC(),
	}
	if err := db.Instance.AddClone(rec); err != nil {
		slog.Error("[clones] save failed", "error", err)
		edit("❌ <b>Could not save your bot.</b> Please try again.")
		return td.EndGroups
	}

	if err := Start(rec); err != nil {
		_ = db.Instance.DeleteClone(rec.ID)
		slog.Error("[clones] start failed", "bot_id", rec.ID, "error", err)
		edit("❌ <b>Could not start your bot:</b> <code>" + html.EscapeString(err.Error()) + "</code>")
		return td.EndGroups
	}

	edit(fmt.Sprintf(
		"<blockquote><b>✅ %s</b></blockquote>\n"+
			"<blockquote>▫️ <b>%s :</b> @%s\n▫️ <b>%s :</b> <code>%d</code></blockquote>\n"+
			"<i>Open your bot, press /start and add it to a group as admin (invite users permission).</i>",
		sc("bot cloned successfully"),
		sc("bot"), html.EscapeString(info.Username),
		sc("id"), info.ID,
	))

	notifyLog(fmt.Sprintf("<b>❖ ɴᴇᴡ ᴄʟᴏɴᴇ</b>\n\n<b>● ʙᴏᴛ ➠</b> @%s (<code>%d</code>)\n<b>● ᴏᴡɴᴇʀ ➠</b> <code>%d</code>",
		html.EscapeString(info.Username), info.ID, owner))
	return td.EndGroups
}

// ------------------------------------------------------------------ /delclone
func delCloneHandler(c *td.Client, m *td.Message) error {
	arg := strings.TrimSpace(handlers.Args(m))
	mine, _ := db.Instance.GetUserClones(m.SenderID())

	if arg == "" {
		// A single clone can be removed without typing its ID.
		if len(mine) == 1 {
			arg = strconv.FormatInt(mine[0].ID, 10)
		} else {
			reply(c, m, "<b>Usage:</b> <code>/delclone [BOT_ID]</code>\nSee your bot IDs with /myclone.")
			return td.EndGroups
		}
	}

	botID, err := strconv.ParseInt(strings.TrimPrefix(arg, "@"), 10, 64)
	if err != nil {
		// allow @username too
		botID = 0
		for _, cl := range mine {
			if strings.EqualFold(cl.Username, strings.TrimPrefix(arg, "@")) {
				botID = cl.ID
			}
		}
		if botID == 0 && isDev(m) {
			all, _ := db.Instance.GetAllClones()
			for _, cl := range all {
				if strings.EqualFold(cl.Username, strings.TrimPrefix(arg, "@")) {
					botID = cl.ID
				}
			}
		}
		if botID == 0 {
			reply(c, m, "❌ Please send a numeric BOT_ID. See /myclone.")
			return td.EndGroups
		}
	}

	rec, ok, err := db.Instance.GetClone(botID)
	if err != nil || !ok {
		reply(c, m, "❌ No cloned bot found with that ID.")
		return td.EndGroups
	}
	if rec.OwnerID != m.SenderID() && !isDev(m) {
		reply(c, m, "❌ You can only remove your own cloned bots.")
		return td.EndGroups
	}

	Stop(rec.ID)
	if err := db.Instance.DeleteClone(rec.ID); err != nil {
		reply(c, m, "❌ Could not remove the bot from the database. Please try again.")
		return td.EndGroups
	}

	reply(c, m, fmt.Sprintf("<blockquote><b>🗑 %s</b></blockquote>\n<blockquote>@%s (<code>%d</code>)</blockquote>\n"+
		"<i>Tip: revoke the token in @BotFather (/revoke) so it can no longer be used.</i>",
		sc("clone removed"), html.EscapeString(rec.Username), rec.ID))
	notifyLog(fmt.Sprintf("<b>❖ ᴄʟᴏɴᴇ ʀᴇᴍᴏᴠᴇᴅ</b>\n\n<b>● ʙᴏᴛ ➠</b> @%s (<code>%d</code>)\n<b>● ʙʏ ➠</b> <code>%d</code>",
		html.EscapeString(rec.Username), rec.ID, m.SenderID()))
	return td.EndGroups
}

// ------------------------------------------------------------------ /myclone
func myCloneHandler(c *td.Client, m *td.Message) error {
	list, err := db.Instance.GetUserClones(m.SenderID())
	if err != nil {
		reply(c, m, "❌ Could not load your bots. Please try again.")
		return td.EndGroups
	}
	if len(list) == 0 {
		reply(c, m, "<blockquote><b>"+sc("you have not cloned any bot yet")+"</b></blockquote>\nUse <code>/clone [BOT_TOKEN]</code> to create one.")
		return td.EndGroups
	}

	var sb strings.Builder
	sb.WriteString("<blockquote><b>🤖 " + sc("your cloned bots") + "</b></blockquote>\n<blockquote>")
	for i, cl := range list {
		state := "🔴"
		if IsRunning(cl.ID) {
			state = "🟢"
		}
		fmt.Fprintf(&sb, "%s <b>%d.</b> @%s — <code>%d</code>", state, i+1, html.EscapeString(cl.Username), cl.ID)
		if i < len(list)-1 {
			sb.WriteString("\n")
		}
	}
	sb.WriteString("</blockquote>\n<i>Remove one with</i> <code>/delclone [BOT_ID]</code>")
	reply(c, m, sb.String())
	return td.EndGroups
}

// ------------------------------------------------------------------ owner / sudo
func clonedHandler(c *td.Client, m *td.Message) error {
	if !isDev(m) {
		return td.EndGroups
	}
	list, err := db.Instance.GetAllClones()
	if err != nil {
		reply(c, m, "❌ Could not load clones.")
		return td.EndGroups
	}
	if len(list) == 0 {
		reply(c, m, "No bots have been cloned yet.")
		return td.EndGroups
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "<blockquote><b>🤖 %s : %d</b></blockquote>\n<blockquote>", sc("cloned bots"), len(list))
	for i, cl := range list {
		fmt.Fprintf(&sb, "<b>%d.</b> @%s", i+1, html.EscapeString(cl.Username))
		if i < len(list)-1 {
			sb.WriteString("\n")
		}
		if sb.Len() > 3500 {
			fmt.Fprintf(&sb, "\n… and %d more", len(list)-i-1)
			break
		}
	}
	sb.WriteString("</blockquote>")
	reply(c, m, sb.String())
	return td.EndGroups
}

func totalBotsHandler(c *td.Client, m *td.Message) error {
	if !isDev(m) {
		return td.EndGroups
	}
	total, err := db.Instance.CountClones()
	if err != nil {
		reply(c, m, "❌ Could not count clones.")
		return td.EndGroups
	}
	reply(c, m, fmt.Sprintf("<blockquote><b>📊 %s</b></blockquote>\n<blockquote>▫️ <b>%s :</b> <code>%d</code>\n▫️ <b>%s :</b> <code>%d</code></blockquote>",
		sc("clone statistics"), sc("total cloned"), total, sc("running now"), RunningCount()))
	return td.EndGroups
}

func clonedInfoHandler(c *td.Client, m *td.Message) error {
	if !isDev(m) {
		return td.EndGroups
	}
	list, err := db.Instance.GetAllClones()
	if err != nil {
		reply(c, m, "❌ Could not load clones.")
		return td.EndGroups
	}
	if len(list) == 0 {
		reply(c, m, "No bots have been cloned yet.")
		return td.EndGroups
	}

	// Tokens are never printed.
	var sb strings.Builder
	fmt.Fprintf(&sb, "<b>🤖 %s</b>\n\n", sc("cloned bots info"))
	flush := func() {
		if sb.Len() > 0 {
			reply(c, m, sb.String())
			sb.Reset()
		}
	}
	for i, cl := range list {
		state := "offline"
		if IsRunning(cl.ID) {
			state = "online"
		}
		fmt.Fprintf(&sb, "<blockquote><b>%d. @%s</b>\n▫️ ID: <code>%d</code>\n▫️ Name: %s\n▫️ Owner: <code>%d</code>\n▫️ Status: %s\n▫️ Created: %s</blockquote>\n",
			i+1, html.EscapeString(cl.Username), cl.ID, html.EscapeString(cl.Name), cl.OwnerID, state, cl.CreatedAt.Format("2006-01-02 15:04"))
		if sb.Len() > 3000 {
			flush()
		}
	}
	flush()
	return td.EndGroups
}

func delAllCloneHandler(c *td.Client, m *td.Message) error {
	if !isDev(m) {
		return td.EndGroups
	}
	if strings.ToLower(strings.TrimSpace(handlers.Args(m))) != "confirm" {
		reply(c, m, "⚠️ <b>This stops and deletes ALL cloned bots.</b>\nSend <code>/delallclone confirm</code> to continue.")
		return td.EndGroups
	}

	list, err := db.Instance.GetAllClones()
	if err != nil {
		reply(c, m, "❌ Could not load clones.")
		return td.EndGroups
	}
	for _, cl := range list {
		Stop(cl.ID)
	}
	removed, err := db.Instance.DeleteAllClones()
	if err != nil {
		reply(c, m, "❌ Stopped the bots but could not clear the database: "+html.EscapeString(err.Error()))
		return td.EndGroups
	}
	reply(c, m, fmt.Sprintf("🗑 <b>Removed %d cloned bot(s).</b>", removed))
	notifyLog(fmt.Sprintf("<b>❖ ᴀʟʟ ᴄʟᴏɴᴇꜱ ʀᴇᴍᴏᴠᴇᴅ</b> (%d) ʙʏ <code>%d</code>", removed, m.SenderID()))
	return td.EndGroups
}
