/*
 * TgMusicBot - Telegram Music Bot
 *  Licensed under GNU GPL v3
 */

// Package thumb builds the glass "now playing" card that is posted with every track.
package thumb

import (
	"ashokshau/tgmusic/src/utils"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/color"
	_ "image/gif"
	"image/jpeg"
	_ "image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	cacheDir     = "cache/thumbs"
	styleVersion = "glass1"
)

var (
	sem         = make(chan struct{}, 2) // at most two cards rendered at once
	httpClient  = &http.Client{Timeout: 8 * time.Second}
	cleanupOnce sync.Once
)

// Generate returns the path of the card for t, rendering it if it is not cached.
func Generate(ctx context.Context, t *utils.CachedTrack) (string, error) {
	if t == nil {
		return "", errors.New("nil track")
	}
	cleanupOnce.Do(func() { go cleanupLoop() })

	if err := os.MkdirAll(cacheDir, 0o755); err != nil {
		return "", err
	}

	sum := sha1.Sum([]byte(styleVersion + "|" + t.Platform + "|" + t.TrackID + "|" + t.Name + "|" + t.Thumbnail))
	out := filepath.Join(cacheDir, hex.EncodeToString(sum[:8])+".jpg")
	if st, err := os.Stat(out); err == nil && st.Size() > 0 {
		return out, nil
	}

	select {
	case sem <- struct{}{}:
		defer func() { <-sem }()
	case <-ctx.Done():
		return "", ctx.Err()
	}

	art := fetchArt(ctx, t)
	img, err := Render(art, Card{
		Title:    t.Name,
		Artist:   t.Channel,
		Duration: utils.SecToMin(t.Duration),
	})
	if err != nil {
		return "", err
	}

	tmp := out + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return "", err
	}
	if err := jpeg.Encode(f, img, &jpeg.Options{Quality: 92}); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return "", err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	if err := os.Rename(tmp, out); err != nil {
		return "", err
	}
	return out, nil
}

// fetchArt downloads the track cover. It never fails: a dark gradient is used instead.
func fetchArt(ctx context.Context, t *utils.CachedTrack) image.Image {
	url := t.Thumbnail
	if url == "" && t.Platform == utils.YouTube && t.TrackID != "" {
		url = fmt.Sprintf("https://i.ytimg.com/vi/%s/hqdefault.jpg", t.TrackID)
	}
	if url != "" {
		if img, err := download(ctx, url); err == nil {
			return img
		}
	}
	return placeholder()
}

func download(ctx context.Context, url string) (image.Image, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("thumbnail http %d", resp.StatusCode)
	}
	img, _, err := image.Decode(io.LimitReader(resp.Body, 12<<20))
	return img, err
}

func placeholder() image.Image {
	img := image.NewRGBA(image.Rect(0, 0, 480, 360))
	for y := 0; y < 360; y++ {
		t := float64(y) / 360
		c := color.RGBA{R: uint8(40 + 30*t), G: uint8(40 + 10*t), B: uint8(60 + 50*t), A: 255}
		for x := 0; x < 480; x++ {
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

// cleanupLoop removes cards older than 12 hours so the cache cannot grow forever.
func cleanupLoop() {
	for {
		entries, _ := os.ReadDir(cacheDir)
		for _, e := range entries {
			info, err := e.Info()
			if err == nil && time.Since(info.ModTime()) > 12*time.Hour {
				_ = os.Remove(filepath.Join(cacheDir, e.Name()))
			}
		}
		time.Sleep(time.Hour)
	}
}
