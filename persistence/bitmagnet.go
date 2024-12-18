package persistence

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

type bitmagnet struct {
	url        string
	debug      bool
	sourceName string
	cache      map[string]time.Time
	sync.Mutex
}

func makeBitmagnet(url_ *url.URL) (Database, error) {
	b := new(bitmagnet)

	url_.Scheme = strings.Replace(url_.Scheme, "bitmagnet", "http", 1)

	url_.ForceQuery = false
	if url_.Query().Get("debug") == "true" {
		b.debug = true
	}
	b.sourceName = url_.Query().Get("source")
	if b.sourceName == "" {
		b.sourceName = "magnetico"
	}
	url_.RawQuery = ""

	url_.Fragment = ""
	url_.RawFragment = ""

	b.url = url_.String()

	b.cache = map[string]time.Time{}
	go func() {
		for range time.NewTicker(10 * time.Minute).C {
			go b.cleanup()
		}
	}()

	return b, nil
}

func (b *bitmagnet) cleanup() {
	b.Lock()
	defer b.Unlock()

	for key, value := range b.cache {
		if time.Now().After(value) {
			delete(b.cache, key)
		}
	}
}

func (b *bitmagnet) Engine() databaseEngine {
	return Bitmagnet
}

func (b *bitmagnet) DoesTorrentExist(infoHash []byte) (bool, error) {
	b.Lock()
	defer b.Unlock()
	_, found := b.cache[string(infoHash)]
	return found, nil
}

func (b *bitmagnet) AddNewTorrent(infoHash []byte, name string, files []File) error {
	totalSize := int64(0)
	for _, file := range files {
		totalSize += file.Size
	}
	data, err := json.Marshal(map[string]interface{}{
		"infoHash":    hex.EncodeToString(infoHash),
		"name":        name,
		"size":        totalSize,
		"publishedAt": time.Now().UTC().Format(time.RFC3339),
		"source":      b.sourceName,
	})
	if err != nil {
		return errors.New("failed to encode metadata " + err.Error())
	}

	b.Lock()
	defer b.Unlock()

	if _, found := b.cache[string(infoHash)]; found {
		return errors.New("torrent already exists")
	}

	dataBuffer := bytes.NewBuffer(data)
	dataBuffer.Write([]byte("\n"))
	resp, err := http.Post(b.url, "application/json", dataBuffer)
	if err != nil {
		return errors.New("failed to post metadata " + err.Error())
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if b.debug {
		log.Printf("Response: %s\n", string(body))
	}

	b.cache[string(infoHash)] = time.Now().Add(10 * time.Minute)
	return nil
}

func (b *bitmagnet) Close() error {
	return nil
}

func (b *bitmagnet) GetNumberOfTorrents() (uint, error) {
	return 0, nil
}

func (b *bitmagnet) GetNumberOfQueryTorrents(query string, epoch int64) (uint64, error) {
	return 0, nil
}

func (b *bitmagnet) QueryTorrents(query string, epoch int64, orderBy OrderingCriteria, ascending bool, limit uint64, lastOrderedValue *float64, lastID *uint64) ([]TorrentMetadata, error) {
	return nil, errors.New("query not supported")
}

func (b *bitmagnet) GetTorrent(infoHash []byte) (*TorrentMetadata, error) {
	return nil, errors.New("fetch not supported")
}

func (b *bitmagnet) GetFiles(infoHash []byte) ([]File, error) {
	return nil, errors.New("file fetch not supported")
}

func (b *bitmagnet) GetStatistics(from string, n uint) (*Statistics, error) {
	return nil, errors.New("statistics not supported")
}
