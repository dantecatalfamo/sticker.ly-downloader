package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

const UserAgent = "androidapp.stickerly/1.13.3 (G011A; U; Android 22; pt-BR; br;)"
const Host = "api.sticker.ly"
const IndexUrl = "http://api.sticker.ly/v3.1/stickerPack/%s"

type Sticker struct {
	Liked      bool     `json:"liked"`
	Tags       []string `json:"tags"`
	FileName   string   `json:"fileName"`
	SID        string   `json:"sid"`
	Animated   bool     `json:"animated"`
	ViewCount  int      `json:"viewCount"`
	IsAnimated bool     `json:"isAnimated"`
}

type StickerIndex struct {
	Stickers          []Sticker `json:"stickers"`
	PackID            string    `json:"packId"`
	Animated          bool      `json:"animated"`
	TrayIndex         int       `json:"trayIndex"`
	ViewCount         int       `json:"viewCount"`
	AuthorName        string    `json:"authorName"`
	ExportCount       int       `json:"exportCount"`
	IsAnimated        bool      `json:"isAnimated"`
	ShareURL          string    `json:"shareUrl"`
	ResourceURLPrefix string    `json:"resourceUrlPrefix"`
	ResourceVersion   int       `json:"resourceVersion"`
	ResourceZip       string    `json:"resourceZip"`
	Updated           uint64    `json:"updated"`
	Owner             string    `json:"owner"`
	Name              string    `json:"name"`
}

type APIError struct {
	ErrorCode       string `json:"errorCode"`
	ErrorMessage    string `json:"errorMessage"`
	InternalTraceID string `json:"internalTraceId"`
	Timestamp       uint64 `json:"timestamp"`
}

type StickerIndexResult struct {
	Result *StickerIndex `json:"result"`
	Error  APIError      `json:"error"`
}

func main() {
	packID := flag.String("pack", "", "Sticker pack ID (required)")
	flag.Parse()

	if *packID == "" {
		flag.Usage()
		os.Exit(1)
	}

	log.Printf("downloading index %s", *packID)
	index, err := GetStickerIndex(*packID)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Pack: %q, %d stickers located", index.Name, len(index.Stickers))

	packDir := fmt.Sprintf("%s - %s", *packID, index.Name)

	if err := os.Mkdir(packDir, os.ModePerm); err != nil {
		if !os.IsExist(err) {
			log.Fatal(err)
		}
	}

	indexPath := filepath.Join(packDir, "index.json")
	indexFile, err := os.Create(indexPath)
	if err != nil {
		log.Fatalf("creating index file: %v", err)
	}

	json.NewEncoder(nil)
	indexEncoder := json.NewEncoder(indexFile)
	indexEncoder.SetIndent("", "  ")
	if err := indexEncoder.Encode(index); err != nil {
		log.Fatalf("writing index file: %v", err)
	}

	for _, sticker := range index.Stickers {
		stickerPath := filepath.Join(packDir, sticker.FileName)
		stickerURL := index.ResourceURLPrefix + sticker.FileName

		log.Printf("downloading sticker: %s", sticker.FileName)
		if err := DownloadImage(stickerURL, stickerPath); err != nil {
			log.Fatalf("failed to download sticker: %v", err)
		}
	}
}

func DownloadImage(url, path string) error {
	client := http.Client{}

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return fmt.Errorf("creating new download resest: %w", err)
	}

	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Host", Host)

	res, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("doing download request: %w", err)
	}

	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("creating image file: %w", err)
	}
	defer file.Close()

	_, err = io.Copy(file, res.Body)
	if err != nil {
		return fmt.Errorf("writing image file: %w", err)
	}

	return nil
}

func GetStickerIndex(packID string) (*StickerIndex, error) {
	packIndexUrl := fmt.Sprintf(IndexUrl, packID)
	client := http.Client{}

	req, err := http.NewRequest(http.MethodGet, packIndexUrl, nil)
	if err != nil {
		return nil, fmt.Errorf("creating index request: %w", err)
	}

	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Host", Host)

	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("doing index request: %w", err)
	}

	var indexResult StickerIndexResult

	if err := json.NewDecoder(res.Body).Decode(&indexResult); err != nil {
		return nil, fmt.Errorf("decoding index: %w", err)
	}

	return indexResult.Result, nil
}
