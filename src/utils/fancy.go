/*
 * TgMusicBot - Telegram Music Bot
 *  Licensed under GNU GPL v3
 */

package utils

import "strings"

var smallCaps = map[rune]rune{
	'a': 'ᴀ', 'b': 'ʙ', 'c': 'ᴄ', 'd': 'ᴅ', 'e': 'ᴇ', 'f': 'ꜰ', 'g': 'ɢ', 'h': 'ʜ', 'i': 'ɪ',
	'j': 'ᴊ', 'k': 'ᴋ', 'l': 'ʟ', 'm': 'ᴍ', 'n': 'ɴ', 'o': 'ᴏ', 'p': 'ᴘ', 'q': 'ǫ', 'r': 'ʀ',
	's': 'ꜱ', 't': 'ᴛ', 'u': 'ᴜ', 'v': 'ᴠ', 'w': 'ᴡ', 'x': 'x', 'y': 'ʏ', 'z': 'ᴢ',
}

// SmallCaps converts latin letters to small-cap unicode letters ("play" -> "ᴘʟᴀʏ").
// Other characters are left untouched, so it is safe to use on plain labels.
// Do NOT run it on text that contains HTML tags or URLs.
func SmallCaps(s string) string {
	var sb strings.Builder
	sb.Grow(len(s) * 2)
	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			r = r - 'A' + 'a'
		}
		if v, ok := smallCaps[r]; ok {
			sb.WriteRune(v)
			continue
		}
		sb.WriteRune(r)
	}
	return sb.String()
}

// ProgressLabel renders a text progress bar such as "00:10 ●────────── 03:49".
func ProgressLabel(elapsed, total int) string {
	const slots = 9
	pos := 0
	if total > 0 {
		pos = elapsed * slots / total
		if pos > slots {
			pos = slots
		}
	}
	var sb strings.Builder
	for i := 0; i <= slots; i++ {
		if i == pos {
			sb.WriteString("●")
		} else {
			sb.WriteString("─")
		}
	}
	return clock(elapsed) + " " + sb.String() + " " + clock(total)
}

func clock(sec int) string {
	if sec < 0 {
		sec = 0
	}
	m, s := sec/60, sec%60
	return twoDigits(m) + ":" + twoDigits(s)
}

func twoDigits(n int) string {
	if n < 10 {
		return "0" + string(rune('0'+n))
	}
	// n can exceed 99 for very long tracks
	return itoa(n)
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	return string(b[i:])
}
