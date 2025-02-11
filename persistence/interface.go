package persistence

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"

	"github.com/klauspost/compress/zstd"
)

type Database interface {
	Engine() databaseEngine
	DoesTorrentExist(infoHash []byte) (bool, error)
	AddNewTorrent(infoHash []byte, name string, files []File) error
	Close() error

	// GetNumberOfTorrents returns the number of torrents saved in the database. Might be an
	// approximation.
	GetNumberOfTorrents() (uint, error)
	// GetNumberOfQueryTorrents returns the total number of data records in a fuzzy query.
	GetNumberOfQueryTorrents(query string, epoch int64) (uint64, error)
	// QueryTorrents returns @pageSize amount of torrents,
	// * that are discovered before @discoveredOnBefore
	// * that match the @query if it's not empty, else all torrents
	// * ordered by the @orderBy in ascending order if @ascending is true, else in descending order
	// after skipping (@page * @pageSize) torrents that also fits the criteria above.
	//
	// On error, returns (nil, error), otherwise a non-nil slice of TorrentMetadata and nil.
	QueryTorrents(
		query string,
		epoch int64,
		orderBy OrderingCriteria,
		ascending bool,
		limit uint64,
		lastOrderedValue *float64,
		lastID *uint64,
	) ([]TorrentMetadata, error)
	// GetTorrents returns the TorrentExtMetadata for the torrent of the given InfoHash. Will return
	// nil, nil if the torrent does not exist in the database.
	GetTorrent(infoHash []byte) (*TorrentMetadata, error)
	GetFiles(infoHash []byte) ([]File, error)
	GetStatistics(from string, n uint) (*Statistics, error)
	// Export returns a channel that will be used to dump all the torrents in the database.
	Export() (chan SimpleTorrentSummary, error)
}

type OrderingCriteria uint8

const (
	ByRelevance OrderingCriteria = iota
	ByTotalSize
	ByDiscoveredOn
	ByNFiles
	ByNSeeders
	ByNLeechers
	ByUpdatedOn
)

// TODO: search `swtich (orderBy)` and see if all cases are covered all the time

type databaseEngine uint8

const (
	Sqlite3 databaseEngine = iota + 1
	Postgres
	ZeroMQ
	RabbitMQ
	Bitmagnet
)

type Statistics struct {
	NDiscovered map[string]uint64 `json:"nDiscovered"`
	NFiles      map[string]uint64 `json:"nFiles"`
	TotalSize   map[string]uint64 `json:"totalSize"`
}

type File struct {
	Size int64  `json:"size"`
	Path string `json:"path"`
}

type TorrentMetadata struct {
	ID           uint64  `json:"id"`
	InfoHash     []byte  `json:"infoHash"` // marshalled differently
	Name         string  `json:"name"`
	Size         uint64  `json:"size"`
	DiscoveredOn int64   `json:"discoveredOn"`
	NFiles       uint    `json:"nFiles"`
	Relevance    float64 `json:"relevance"`
}

type SimpleTorrentSummary struct {
	InfoHash string `json:"infoHash"`
	Name     string `json:"name"`
	Files    []File `json:"files"`
}

func (tm *TorrentMetadata) MarshalJSON() ([]byte, error) {
	type Alias TorrentMetadata
	return json.Marshal(&struct {
		InfoHash string `json:"infoHash"`
		*Alias
	}{
		InfoHash: hex.EncodeToString(tm.InfoHash),
		Alias:    (*Alias)(tm),
	})
}

func MakeDatabase(rawURL string) (Database, error) {
	url_, err := url.Parse(rawURL)
	if err != nil {
		return nil, errors.New("url.Parse " + err.Error())
	}

	switch url_.Scheme {

	case "sqlite", "sqlite3":
		return makeSqlite3Database(url_)

	case "postgres", "cockroach":
		return makePostgresDatabase(url_)

	case "zeromq", "zmq":
		return makeZeroMQ(url_)

	case "amqp", "amqps":
		return makeRabbitMQ(url_)

	case "bitmagnet", "bitmagnets":
		return makeBitmagnet(url_)

	default:
		return nil, fmt.Errorf("unknown URI scheme: `%s`", url_.Scheme)
	}
}

func NewStatistics() (s *Statistics) {
	s = new(Statistics)
	s.NDiscovered = make(map[string]uint64)
	s.NFiles = make(map[string]uint64)
	s.TotalSize = make(map[string]uint64)
	return
}

func MakeExport(db Database, path string, interruptChan chan os.Signal) error {
	torrentsChan, err := db.Export()
	if err != nil {
		return err
	}

	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		if err = file.Close(); err != nil {
			log.Printf("Could not close file! %s\n", err.Error())
		}
	}()

	writer := file.Write
	if strings.HasSuffix(path, ".zstd") {
		zw, err := zstd.NewWriter(file, zstd.WithEncoderLevel(zstd.SpeedBestCompression))
		if err != nil {
			return err
		}
		defer func() {
			if err = zw.Close(); err != nil {
				log.Printf("Could not close zstd writer! %s\n", err.Error())
			}
		}()
		writer = zw.Write
	}

	for {
		select {
		case result, ok := <-torrentsChan:
			if !ok {
				log.Println("Database export completed. Shutting down.")
				return nil
			}

			jsonResult, err := json.Marshal(result)
			if err != nil {
				return err
			}
			jsonResult = append(jsonResult, '\n')

			if _, err = writer(jsonResult); err != nil {
				return err
			}

		case <-interruptChan:
			return nil
		}
	}
}

func MakeImport(db Database, path string, interruptChan chan os.Signal) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer func() {
		if err = file.Close(); err != nil {
			log.Printf("Could not close file! %s\n", err.Error())
		}
	}()

	reader := file.Read
	if strings.HasSuffix(path, ".zstd") {
		zr, err := zstd.NewReader(file)
		if err != nil {
			return err
		}
		defer zr.Close()
		reader = zr.Read
	}

	for {
		select {

		case <-interruptChan:
			return nil

		default:
			var line []byte
			// Read one byte at a time until newline or EOF
			for {
				buf := make([]byte, 1)
				n, err := reader(buf)
				if n == 0 || err != nil {
					if len(line) == 0 {
						return nil // EOF with no data
					}
					break
				}
				if buf[0] == '\n' {
					break
				}
				line = append(line, buf[0])
			}

			var torrent SimpleTorrentSummary
			if err := json.Unmarshal(line, &torrent); err != nil {
				return fmt.Errorf("failed to unmarshal JSON: %v", err)
			}

			infoHash, err := hex.DecodeString(torrent.InfoHash)
			if err != nil {
				return fmt.Errorf("failed to decode infohash: %v", err)
			}

			if err := db.AddNewTorrent(infoHash, torrent.Name, torrent.Files); err != nil {
				log.Printf("failed to add torrent: %v\n", err.Error())
			}
		}
	}
}
