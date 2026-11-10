package main

import (
	"archive/zip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

type Response struct {
	Code int    `json:"code"`
	Data []Post `json:"data"`
}
type Post struct {
	Content string `json:"content"`
	Hot     *struct {
		Content string `json:"content"`
	} `json:"hot_tucao"`
}
type Entry struct {
	ID      string `json:"id"`
	Time    string `json:"time"`
	File    string `json:"file"`
	Release string `json:"release"`
}
type Snapshot struct {
	Time   string            `json:"time"`
	Source string            `json:"source"`
	Data   json.RawMessage   `json:"data"`
	Media  map[string]string `json:"media"`
}

var client = &http.Client{Timeout: 60 * time.Second}
var sources = regexp.MustCompile(`(?i)<(?:img|video|source)\b[^>]*?\s(?:src|data-src)\s*=\s*["']([^"']+)["']`)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
func get(address string) ([]byte, error) {
	var last error
	for attempt := 0; attempt < 3; attempt++ {
		req, err := http.NewRequest("GET", address, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 JandanTopSnapshot/2.0 BotID("+os.Getenv("BOT_ID")+")")
		req.Header.Set("Referer", "https://jandan.net/")
		resp, err := client.Do(req)
		if err == nil {
			body, e := io.ReadAll(io.LimitReader(resp.Body, 256<<20))
			resp.Body.Close()
			if e == nil && resp.StatusCode == 200 && len(body) > 0 {
				return body, nil
			}
			err = fmt.Errorf("GET %s: status %d: %v", address, resp.StatusCode, e)
		}
		last = err
		if attempt < 2 {
			time.Sleep(time.Second * time.Duration(attempt+1))
		}
	}
	return nil, last
}
func mediaURL(raw string) string {
	raw = html.UnescapeString(raw)
	if strings.HasPrefix(raw, "//") {
		raw = "https:" + raw
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "https" && u.Scheme != "http") {
		return ""
	}
	ext := strings.ToLower(path.Ext(u.Path))
	if ext == ".gif" {
		u.Path = "/large/" + strings.TrimSuffix(path.Base(u.Path), path.Ext(u.Path)) + ".mp4"
	} else if ext != ".mp4" && ext != ".webm" && ext != ".webp" {
		u.Path += ".webp"
	}
	return u.String()
}
func writeJSON(name string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	if err = os.MkdirAll(filepath.Dir(name), 0755); err != nil {
		return err
	}
	if err = os.WriteFile(name+".tmp", append(b, '\n'), 0644); err != nil {
		return err
	}
	return os.Rename(name+".tmp", name)
}
func run() error {
	endpoint := os.Getenv("API_URL")
	if endpoint == "" {
		endpoint = "https://jandan.net/api/top/post/26402?cdn_line=default"
	}
	body, err := get(endpoint)
	if err != nil {
		return err
	}
	var response Response
	err = json.Unmarshal(body, &response)
	if err == nil && response.Code == 0 && len(response.Data) == 0 && strings.Contains(endpoint, "?cdn_line=default") {
		endpoint = strings.Replace(endpoint, "?cdn_line=default", "", 1)
		body, err = get(endpoint)
		if err == nil {
			err = json.Unmarshal(body, &response)
		}
	}
	if err != nil {
		return err
	}
	if response.Code != 0 || len(response.Data) == 0 {
		return fmt.Errorf("API returned code=%d or empty data; no snapshot saved", response.Code)
	}
	var envelope struct {
		Data json.RawMessage `json:"data"`
	}
	if err = json.Unmarshal(body, &envelope); err != nil {
		return err
	}
	zone, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return err
	}
	now := time.Now().In(zone)
	id := now.Format("20060102-150405")
	media := map[string]string{}
	for _, p := range response.Data {
		contents := []string{p.Content}
		if p.Hot != nil {
			contents = append(contents, p.Hot.Content)
		}
		for _, content := range contents {
			for _, m := range sources.FindAllStringSubmatch(content, -1) {
				u := mediaURL(m[1])
				if u != "" {
					sum := sha256.Sum256([]byte(u))
					media[u] = hex.EncodeToString(sum[:]) + path.Ext(strings.Split(u, "?")[0])
				}
			}
		}
	}
	if err = os.MkdirAll("dist", 0755); err != nil {
		return err
	}
	archive := "dist/media-" + id + ".zip"
	f, err := os.Create(archive + ".tmp")
	if err != nil {
		return err
	}
	defer os.Remove(archive + ".tmp")
	zw := zip.NewWriter(f)
	keys := make([]string, 0, len(media))
	for u := range media {
		keys = append(keys, u)
	}
	sort.Strings(keys)
	for _, u := range keys {
		fmt.Println("Downloading", u)
		data, e := get(u)
		if e != nil {
			zw.Close()
			f.Close()
			return e
		}
		w, e := zw.CreateHeader(&zip.FileHeader{Name: media[u], Method: zip.Store})
		if e == nil {
			_, e = w.Write(data)
		}
		if e != nil {
			zw.Close()
			f.Close()
			return e
		}
	}
	if err = zw.Close(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err = os.Rename(archive+".tmp", archive); err != nil {
		return err
	}
	file := "snapshots/" + now.Format("200601") + "/" + id + ".json"
	if err = writeJSON("docs/"+file, Snapshot{now.Format(time.RFC3339), endpoint, envelope.Data, media}); err != nil {
		return err
	}
	entries := []Entry{}
	old, err := os.ReadFile("docs/snapshots.json")
	if err == nil {
		if err = json.Unmarshal(old, &entries); err != nil {
			return err
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	repo := os.Getenv("GITHUB_REPOSITORY")
	if repo == "" {
		repo = "imdong/jandan-top-snapshot"
	}
	entries = append(entries, Entry{id, now.Format("2006-01-02 15:04:05"), file, "https://github.com/" + repo + "/releases/download/snapshot-" + id + "/media-" + id + ".zip"})
	sort.Slice(entries, func(i, j int) bool { return entries[i].ID > entries[j].ID })
	if err = writeJSON("docs/snapshots.json", entries); err != nil {
		return err
	}
	if output := os.Getenv("GITHUB_OUTPUT"); output != "" {
		f, e := os.OpenFile(output, os.O_APPEND|os.O_WRONLY, 0644)
		if e != nil {
			return e
		}
		defer f.Close()
		_, err = fmt.Fprintf(f, "tag=snapshot-%s\narchive=%s\n", id, archive)
	}
	fmt.Printf("Saved %s (%d posts, %d media)\n", file, len(response.Data), len(media))
	return err
}
