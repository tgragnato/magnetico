package persistence

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"net"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"text/template"
	"time"
	"unicode/utf8"

	_ "github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
	"tgragnato.it/magnetico/v2/stats"
)

type postgresDatabase struct {
	conn            *sql.DB
	isCockroach     bool
	dhtMu           sync.Mutex
	dhtMaxNeighbors uint
	dhtFilterNodes  []net.IPNet
	dhtConfigured   bool
}

func makePostgresDatabase(url_ *url.URL) (Database, error) {
	db := new(postgresDatabase)
	db.isCockroach = false

	if url_.Scheme == "cockroach" {
		url_.Scheme = "postgres"
		db.isCockroach = true
	}

	var err error
	db.conn, err = sql.Open("pgx", url_.String())
	if err != nil {
		return nil, errors.New("sql.Open " + err.Error())
	}

	// > Open may just validate its arguments without creating a connection to the database. To
	// > verify that the data source Name is valid, call Ping.
	// https://golang.org/pkg/database/sql/#Open
	if err = db.conn.Ping(); err != nil {
		return nil, errors.New("sql.DB.Ping " + err.Error())
	}

	if err := db.setupDatabase(); err != nil {
		return nil, errors.New("setupDatabase " + err.Error())
	}

	return db, nil
}

func (db *postgresDatabase) Engine() databaseEngine {
	return Postgres
}

func (db *postgresDatabase) ConfigureDHTRoutingTable(maxNeighbors uint, filterNodes []net.IPNet) error {
	db.dhtMu.Lock()
	defer db.dhtMu.Unlock()
	db.dhtMaxNeighbors = maxNeighbors
	db.dhtFilterNodes = append([]net.IPNet(nil), filterNodes...)
	db.dhtConfigured = true
	return nil
}

func (db *postgresDatabase) AddDHTNodes(ctx context.Context, nodes []net.UDPAddr) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	db.dhtMu.Lock()
	defer db.dhtMu.Unlock()
	if !db.dhtConfigured {
		return errors.New("DHT routing table is not configured")
	}
	filtered := make([]string, 0, len(nodes))
	for _, node := range nodes {
		if postgresDHTNodeAllowed(node, db.dhtFilterNodes) {
			filtered = append(filtered, node.String())
		}
	}
	if len(filtered) == 0 {
		return nil
	}
	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("postgres: begin DHT node update: %w", err)
	}
	defer db.rollback(tx)
	var count uint64
	if err := tx.QueryRowContext(ctx, "SELECT COUNT(*) FROM dht_nodes").Scan(&count); err != nil {
		return fmt.Errorf("postgres: count DHT nodes: %w", err)
	}
	if count+uint64(len(filtered)) > uint64(db.dhtMaxNeighbors)*uint64(db.dhtMaxNeighbors) {
		if _, err := tx.ExecContext(ctx, "DELETE FROM dht_nodes"); err != nil {
			return fmt.Errorf("postgres: clear DHT nodes: %w", err)
		}
	}
	for _, addr := range filtered {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO dht_nodes (addr, sightings) VALUES ($1, 1)
			ON CONFLICT (addr) DO UPDATE SET sightings = dht_nodes.sightings + 1
		`, addr); err != nil {
			return fmt.Errorf("postgres: add DHT node: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("postgres: commit DHT node update: %w", err)
	}
	return nil
}

func (db *postgresDatabase) GetDHTNodes(ctx context.Context) ([]net.UDPAddr, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	db.dhtMu.Lock()
	defer db.dhtMu.Unlock()
	if !db.dhtConfigured {
		return nil, errors.New("DHT routing table is not configured")
	}
	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("postgres: begin DHT node selection: %w", err)
	}
	defer db.rollback(tx)
	rows, err := tx.QueryContext(ctx, "SELECT addr, sightings FROM dht_nodes LIMIT $1 FOR UPDATE", db.dhtMaxNeighbors)
	if err != nil {
		return nil, fmt.Errorf("postgres: select DHT nodes: %w", err)
	}
	type node struct {
		addr      string
		sightings int64
	}
	selected := make([]node, 0, db.dhtMaxNeighbors)
	for rows.Next() {
		var current node
		if err := rows.Scan(&current.addr, &current.sightings); err != nil {
			db.closeRows(rows)
			return nil, fmt.Errorf("postgres: scan DHT node: %w", err)
		}
		selected = append(selected, current)
	}
	if err := rows.Err(); err != nil {
		db.closeRows(rows)
		return nil, fmt.Errorf("postgres: iterate DHT nodes: %w", err)
	}
	db.closeRows(rows)
	result := make([]net.UDPAddr, 0, len(selected))
	for _, current := range selected {
		addr, err := postgresDHTAddrFromString(current.addr)
		if err != nil {
			continue
		}
		result = append(result, addr)
		if current.sightings > 1 {
			if _, err := tx.ExecContext(ctx, "UPDATE dht_nodes SET sightings = sightings - 1 WHERE addr = $1", current.addr); err != nil {
				return nil, fmt.Errorf("postgres: decrement DHT node sightings: %w", err)
			}
		} else if _, err := tx.ExecContext(ctx, "DELETE FROM dht_nodes WHERE addr = $1", current.addr); err != nil {
			return nil, fmt.Errorf("postgres: consume DHT node: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("postgres: commit DHT node selection: %w", err)
	}
	return result, nil
}

func (db *postgresDatabase) IsDHTRoutingEmpty(ctx context.Context) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	db.dhtMu.Lock()
	defer db.dhtMu.Unlock()
	if !db.dhtConfigured {
		return false, errors.New("DHT routing table is not configured")
	}
	var count uint64
	if err := db.conn.QueryRowContext(ctx, "SELECT COUNT(*) FROM dht_nodes").Scan(&count); err != nil {
		return false, fmt.Errorf("postgres: count DHT nodes: %w", err)
	}
	return count == 0, nil
}

func (db *postgresDatabase) DumpDHTNodes(ctx context.Context, ipv4 bool) ([]net.UDPAddr, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	db.dhtMu.Lock()
	defer db.dhtMu.Unlock()
	if !db.dhtConfigured {
		return nil, errors.New("DHT routing table is not configured")
	}
	rows, err := db.conn.QueryContext(ctx, "SELECT addr FROM dht_nodes")
	if err != nil {
		return nil, fmt.Errorf("postgres: select DHT node dump: %w", err)
	}
	defer db.closeRows(rows)
	result := make([]net.UDPAddr, 0, 100)
	for rows.Next() {
		var value string
		if err := rows.Scan(&value); err != nil {
			return nil, fmt.Errorf("postgres: scan DHT node dump: %w", err)
		}
		addr, err := postgresDHTAddrFromString(value)
		if err != nil || ipv4 != (addr.IP.To4() != nil) {
			continue
		}
		result = append(result, addr)
		if len(result) == 100 {
			break
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("postgres: iterate DHT node dump: %w", err)
	}
	return result, nil
}

func (db *postgresDatabase) AddDHTHashes(ctx context.Context, hashes [][20]byte) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	db.dhtMu.Lock()
	defer db.dhtMu.Unlock()
	if !db.dhtConfigured {
		return errors.New("DHT routing table is not configured")
	}
	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("postgres: begin DHT hash update: %w", err)
	}
	defer db.rollback(tx)
	for i := 0; i < len(hashes) && i < 10; i++ {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO dht_hashes (slot, hash) VALUES ($1, $2)
			ON CONFLICT (slot) DO UPDATE SET hash = EXCLUDED.hash
		`, i, hashes[i][:]); err != nil {
			return fmt.Errorf("postgres: update DHT hash slot %d: %w", i, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("postgres: commit DHT hash update: %w", err)
	}
	return nil
}

func (db *postgresDatabase) GetDHTHashes(ctx context.Context) ([10][20]byte, error) {
	if err := ctx.Err(); err != nil {
		return [10][20]byte{}, err
	}
	db.dhtMu.Lock()
	defer db.dhtMu.Unlock()
	if !db.dhtConfigured {
		return [10][20]byte{}, errors.New("DHT routing table is not configured")
	}
	var result [10][20]byte
	rows, err := db.conn.QueryContext(ctx, "SELECT slot, hash FROM dht_hashes")
	if err != nil {
		return result, fmt.Errorf("postgres: select DHT hashes: %w", err)
	}
	defer db.closeRows(rows)
	for rows.Next() {
		var slot int
		var hash []byte
		if err := rows.Scan(&slot, &hash); err != nil {
			return result, fmt.Errorf("postgres: scan DHT hash: %w", err)
		}
		if slot >= 0 && slot < len(result) && len(hash) == len(result[slot]) {
			copy(result[slot][:], hash)
		}
	}
	if err := rows.Err(); err != nil {
		return result, fmt.Errorf("postgres: iterate DHT hashes: %w", err)
	}
	return result, nil
}

func (db *postgresDatabase) EnqueueDHTMessage(ctx context.Context, message DHTOutboundMessage) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	message, err := prepareDHTOutboundMessage(message)
	if err != nil {
		return err
	}
	_, err = db.conn.ExecContext(ctx, `
		INSERT INTO dht_outbox (message_id, queued_at, destination_ip, destination_port, destination_zone, payload)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, message.ID, message.QueuedAt, message.Destination.IP.String(), message.Destination.Port, message.Destination.Zone, message.Payload)
	if err != nil {
		return fmt.Errorf("postgres: enqueue DHT message: %w", err)
	}
	return nil
}

func (db *postgresDatabase) NextDHTMessage(ctx context.Context) (DHTOutboundMessage, bool, error) {
	var message DHTOutboundMessage
	var destinationIP string
	err := db.conn.QueryRowContext(ctx, `
		SELECT message_id, queued_at, destination_ip, destination_port, destination_zone, payload
		FROM dht_outbox ORDER BY queued_at, message_id LIMIT 1
	`).Scan(&message.ID, &message.QueuedAt, &destinationIP, &message.Destination.Port, &message.Destination.Zone, &message.Payload)
	if errors.Is(err, sql.ErrNoRows) {
		return DHTOutboundMessage{}, false, nil
	}
	if err != nil {
		return DHTOutboundMessage{}, false, fmt.Errorf("postgres: read DHT outbox: %w", err)
	}
	message.Destination.IP = net.ParseIP(destinationIP)
	if message.Destination.IP == nil {
		return DHTOutboundMessage{}, false, fmt.Errorf("postgres: invalid DHT outbox IP %q", destinationIP)
	}
	return message, true, nil
}

func (db *postgresDatabase) AckDHTMessage(ctx context.Context, messageID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, err := db.conn.ExecContext(ctx, "DELETE FROM dht_outbox WHERE message_id = $1", messageID); err != nil {
		return fmt.Errorf("postgres: acknowledge DHT message: %w", err)
	}
	return nil
}

func (db *postgresDatabase) ReleaseDHTMessage(ctx context.Context, messageID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if _, err := db.conn.ExecContext(ctx, `
		UPDATE dht_outbox
		SET queued_at = (SELECT COALESCE(MAX(queued_at), 0) + 1 FROM dht_outbox)
		WHERE message_id = $1
	`, messageID); err != nil {
		return fmt.Errorf("postgres: release DHT message: %w", err)
	}
	return nil
}

func postgresDHTNodeAllowed(node net.UDPAddr, filterNodes []net.IPNet) bool {
	if len(filterNodes) > 0 {
		for _, filterNode := range filterNodes {
			if filterNode.Contains(node.IP) {
				return true
			}
		}
		return false
	}
	if !node.IP.IsGlobalUnicast() || node.IP.IsPrivate() {
		return false
	}
	return node.Port == 80 || node.Port == 443 || node.Port >= 1024 && node.Port <= 65535
}

func postgresDHTAddrFromString(value string) (net.UDPAddr, error) {
	host, port, err := net.SplitHostPort(value)
	if err != nil {
		return net.UDPAddr{}, err
	}
	portNumber, err := strconv.Atoi(port)
	if err != nil {
		return net.UDPAddr{}, err
	}
	return net.UDPAddr{IP: net.ParseIP(host), Port: portNumber}, nil
}

func (db *postgresDatabase) DoesTorrentExist(infoHash []byte) (bool, error) {
	rows, err := db.conn.Query("SELECT 1 FROM torrents WHERE info_hash = $1;", infoHash)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	exists := rows.Next()
	if rows.Err() != nil {
		return false, err
	}

	return exists, nil
}

func (db *postgresDatabase) AddNewTorrent(infoHash []byte, name string, files []File) error {
	if !utf8.ValidString(name) {
		go stats.GetInstance().IncNonUTF8()
		// Returning nil so deferred tx.Rollback() will be called and transaction will be canceled.
		return nil
	}
	name = strings.ReplaceAll(name, "\x00", "")

	tx, err := db.conn.Begin()
	if err != nil {
		return errors.New("conn.Begin " + err.Error())
	}
	// If everything goes as planned and no error occurs, we will commit the transaction before
	// returning from the function so the tx.Rollback() call will fail, trying to rollback a
	// committed transaction. BUT, if an error occurs, we'll get our transaction rollback'ed, which
	// is nice.
	defer db.rollback(tx)

	var totalSize uint64 = 0
	for _, file := range files {
		totalSize += uint64(file.Size)
	}

	// This is a workaround for a bug: the database will not accept total_size to be zero.
	if totalSize == 0 {
		return nil
	}

	if exist, err := db.DoesTorrentExist(infoHash); exist || err != nil {
		return err
	}

	var lastInsertId int64

	err = tx.QueryRow(`
		INSERT INTO torrents (
			info_hash,
			name,
			total_size,
			discovered_on
		) VALUES ($1, $2, $3, $4)
		RETURNING id;
	`, infoHash, name, totalSize, time.Now().Unix()).Scan(&lastInsertId)
	if err != nil {
		return errors.New("tx.QueryRow (INSERT INTO torrents) " + err.Error())
	}

	for _, file := range files {
		if !utf8.ValidString(file.Path) {
			go stats.GetInstance().IncNonUTF8()
			// Returning nil so deferred tx.Rollback() will be called and transaction will be canceled.
			return nil
		}

		_, err = tx.Exec("INSERT INTO files (torrent_id, size, path) VALUES ($1, $2, $3);",
			lastInsertId, file.Size, file.Path,
		)
		if err != nil {
			return errors.New("tx.Exec (INSERT INTO files) " + err.Error())
		}
	}

	err = tx.Commit()
	if err != nil {
		return errors.New("tx.Commit " + err.Error())
	}

	return nil
}

func (db *postgresDatabase) Close() error {
	return db.conn.Close()
}

func (db *postgresDatabase) GetNumberOfTorrents() (uint, error) {
	if rows, err := db.getExactCount(); err == nil {
		return rows, nil
	}

	return db.getFuzzyCount()
}

func (db *postgresDatabase) getExactCount() (uint, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second/2)
	defer cancel()

	rows, err := db.conn.QueryContext(ctx, "SELECT last_value::BIGINT AS exact_count FROM seq_torrents_id;")
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	if !rows.Next() {
		return 0, errors.New("no rows returned from `SELECT last_value::BIGINT AS exact_count FROM seq_torrents_id;`")
	}

	// Returns int64: https://godoc.org/github.com/lib/pq#hdr-Data_Types
	var n *int64
	if err = rows.Scan(&n); err != nil {
		return 0, err
	}

	// If the database is empty (i.e. 0 entries in 'torrents') then the query will return nil.
	if n == nil {
		return 0, nil
	} else {
		return uint(*n), nil
	}
}

func (db *postgresDatabase) getFuzzyCount() (uint, error) {
	rows, err := db.conn.Query("SELECT reltuples::BIGINT AS estimate_count FROM pg_class WHERE relname='torrents';")
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	if !rows.Next() {
		return 0, errors.New("no rows returned from `SELECT reltuples::BIGINT AS estimate_count FROM pg_class WHERE relname='torrents';`")
	}

	// Returns int64: https://godoc.org/github.com/lib/pq#hdr-Data_Types
	var n *int64
	if err = rows.Scan(&n); err != nil {
		return 0, err
	}

	// If the database is empty (i.e. 0 entries in 'torrents') then the query will return nil.
	if n == nil {
		return 0, nil
	} else {
		return uint(*n), nil
	}
}

func (db *postgresDatabase) GetNumberOfQueryTorrents(query string, epoch int64) (uint64, error) {

	var querySkeleton = `SELECT COUNT(*)
		FROM torrents
		WHERE
    		name ILIKE CONCAT('%',$1::text,'%') AND
			discovered_on <= $2;
	`

	rows, err := db.conn.Query(querySkeleton, query, epoch)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	if !rows.Next() {
		return 0, errors.New("no rows returned from `SELECT COUNT(*) FROM torrents WHERE name ILIKE CONCAT('%%',$1::text,'%%') AND discovered_on <= $2;`")
	}

	var n *int64
	if err = rows.Scan(&n); err != nil {
		return 0, err
	}

	// If the database is empty (i.e. 0 entries in 'torrents') then the query will return nil.
	if n == nil || *n < 0 {
		return 0, nil
	} else {
		return uint64(*n), nil
	}
}

func (db *postgresDatabase) QueryTorrents(
	query string,
	epoch int64,
	orderBy OrderingCriteria,
	ascending bool,
	limit uint64,
	lastOrderedValue *float64,
	lastID *uint64,
) ([]TorrentMetadata, error) {
	var (
		safeLastID           uint64  = 0
		safeLastOrderedValue float64 = 0
		querySkeleton                = `
		SELECT
			id,
			info_hash,
			name,
			total_size,
			discovered_on,
			(SELECT COUNT(*) FROM files WHERE torrents.id = files.torrent_id) AS n_files,
			0
		FROM torrents
		WHERE
			(
				$1::text = ''
				OR name ILIKE CONCAT('%',$1::text,'%')
				OR EXISTS (
					SELECT 1 FROM files
					WHERE files.torrent_id = torrents.id
					AND files.path ILIKE CONCAT('%',$1::text,'%')
				)
			) AND
			discovered_on <= $2 AND
			($3 = 0 OR {{.OrderOn}} {{GTEorLTE .Ascending}} $3) AND
			($4 = 0 OR id {{GTEorLTE .Ascending}} $4)
		ORDER BY {{.OrderOn}} {{AscOrDesc .Ascending}}, id {{AscOrDesc .Ascending}}
		LIMIT $5;
	`
	)

	if (lastOrderedValue == nil) != (lastID == nil) {
		return nil, fmt.Errorf("lastOrderedValue and lastID should be supplied together, if supplied")
	}
	firstPage := lastID == nil
	if !firstPage {
		safeLastID = *lastID
		safeLastOrderedValue = *lastOrderedValue
	}

	sqlQuery := db.executeTemplate(
		querySkeleton,
		struct {
			OrderOn   string
			Ascending bool
		}{
			OrderOn:   db.orderOn(orderBy),
			Ascending: ascending,
		}, template.FuncMap{
			"GTEorLTE": func(ascending bool) string {
				if ascending {
					return ">"
				} else {
					return "<"
				}
			},
			"AscOrDesc": func(ascending bool) string {
				if ascending {
					return "ASC"
				} else {
					return "DESC"
				}
			},
		},
	)

	rows, err := db.conn.Query(
		sqlQuery,
		query,
		epoch,
		safeLastOrderedValue,
		safeLastID,
		limit,
	)

	if err != nil {
		return nil, errors.New("query error " + err.Error())
	}
	defer db.closeRows(rows)

	torrents := make([]TorrentMetadata, 0)
	for rows.Next() {
		var torrent TorrentMetadata
		err = rows.Scan(
			&torrent.ID,
			&torrent.InfoHash,
			&torrent.Name,
			&torrent.Size,
			&torrent.DiscoveredOn,
			&torrent.NFiles,
			&torrent.Relevance,
		)
		if err != nil {
			return nil, err
		}
		torrents = append(torrents, torrent)
	}

	return torrents, nil
}

func (db *postgresDatabase) GetTorrent(infoHash []byte) (*TorrentMetadata, error) {
	rows, err := db.conn.Query(`
		SELECT
			t.info_hash,
			t.name,
			t.total_size,
			t.discovered_on,
			(SELECT COUNT(*) FROM files f WHERE f.torrent_id = t.id) AS n_files
		FROM torrents t
		WHERE t.info_hash = $1;`,
		infoHash,
	)
	if err != nil {
		return nil, err
	}
	defer db.closeRows(rows)

	if !rows.Next() {
		return nil, nil
	}

	var tm TorrentMetadata
	if err = rows.Scan(&tm.InfoHash, &tm.Name, &tm.Size, &tm.DiscoveredOn, &tm.NFiles); err != nil {
		return nil, err
	}

	return &tm, nil
}

func (db *postgresDatabase) GetFiles(infoHash []byte) ([]File, error) {
	rows, err := db.conn.Query(`
		SELECT
       		f.size,
       		f.path 
		FROM
			files f,
			torrents t
		WHERE
			f.torrent_id = t.id AND
			t.info_hash = $1;`,
		infoHash,
	)
	if err != nil {
		return nil, err
	}
	defer db.closeRows(rows)

	var files []File
	for rows.Next() {
		var file File
		if err = rows.Scan(&file.Size, &file.Path); err != nil {
			return nil, err
		}
		files = append(files, file)
	}

	return files, nil
}

func (db *postgresDatabase) GetStatistics(from string, n uint) (*Statistics, error) {
	fromTime, gran, err := ParseISO8601(from)
	if err != nil {
		return nil, errors.New("parsing ISO8601 error " + err.Error())
	}

	var timef string

	switch gran {
	case Year:
		timef = "2006"
	case Month:
		timef = "2006-01"
	case Week:
		timef = "2006-01-02"
	case Day:
		timef = "2006-01-02"
	case Hour:
		timef = "2006-01-02 15:04"
	}
	toTime := statisticsToTime(*fromTime, gran, n)

	rows, err := db.conn.Query(`
		SELECT
			discovered_on AS dT,
			sum(files.size) AS tS,
			count(DISTINCT torrents.id) AS nD,
			count(DISTINCT files.id) AS nF
		FROM
			torrents,
			files
		WHERE
			torrents.id = files.torrent_id AND
			discovered_on >= $1 AND
			discovered_on <= $2
		GROUP BY dt;`,
		fromTime.Unix(),
		toTime.Unix(),
	)
	if err != nil {
		return nil, err
	}
	defer db.closeRows(rows)

	stats := NewStatistics()

	for rows.Next() {
		var dT string
		var tS, nD, nF uint64
		if err := rows.Scan(&dT, &tS, &nD, &nF); err != nil {
			return nil, err
		}

		epoch, _ := strconv.ParseInt(dT, 10, 64)
		dT = time.Unix(epoch, 0).Format(timef)

		stats.NDiscovered[dT] += nD
		stats.TotalSize[dT] += tS
		stats.NFiles[dT] += nF
	}

	return stats, nil
}

func (db *postgresDatabase) setupDatabase() error {
	tx, err := db.conn.Begin()
	if err != nil {
		return errors.New("sql.DB.Begin " + err.Error())
	}
	defer db.rollback(tx)

	rows, err := db.conn.Query("SELECT 1 FROM pg_extension WHERE extname = 'pg_trgm';")
	if err != nil {
		return err
	}
	defer db.closeRows(rows)

	trgmInstalled := rows.Next()
	if rows.Err() != nil {
		return err
	}
	if !trgmInstalled {
		log.Println("pg_trgm extension is not enabled. You need to execute 'CREATE EXTENSION pg_trgm' on this database")
	}

	// Initial Setup for schema version 0:
	// FROZEN.
	_, err = tx.Exec(`
		-- Torrents ID sequence generator
		CREATE SEQUENCE IF NOT EXISTS seq_torrents_id;
		-- Files ID sequence generator
		CREATE SEQUENCE IF NOT EXISTS seq_files_id;

		CREATE TABLE IF NOT EXISTS torrents (
			id             INTEGER PRIMARY KEY DEFAULT nextval('seq_torrents_id'),
			info_hash      bytea NOT NULL UNIQUE,
			name           TEXT NOT NULL,
			total_size     BIGINT NOT NULL CHECK(total_size > 0),
			discovered_on  INTEGER NOT NULL CHECK(discovered_on > 0)
		);

		-- Indexes for search sorting options
		CREATE INDEX IF NOT EXISTS idx_torrents_total_size ON torrents (total_size);
		CREATE INDEX IF NOT EXISTS idx_torrents_discovered_on ON torrents (discovered_on);

		-- Using pg_trgm GIN index for fast ILIKE queries
		-- You need to execute "CREATE EXTENSION pg_trgm" on your database for this index to work
		-- Be aware that using this type of index implies that making ILIKE queries with less that
		-- 3 character values will cause full table scan instead of using index.
		-- You can try to avoid that by doing 'SET enable_seqscan=off'.
		CREATE INDEX IF NOT EXISTS idx_torrents_name_gin_trgm ON torrents USING GIN (name gin_trgm_ops);

		CREATE TABLE IF NOT EXISTS files (
			id          INTEGER PRIMARY KEY DEFAULT nextval('seq_files_id'),
			torrent_id  INTEGER REFERENCES torrents ON DELETE CASCADE ON UPDATE RESTRICT,
			size        BIGINT NOT NULL,
			path        TEXT NOT NULL
		);

		CREATE INDEX IF NOT EXISTS idx_files_torrent_id ON files (torrent_id);

		CREATE TABLE IF NOT EXISTS migrations (
		    schema_version		SMALLINT NOT NULL UNIQUE 
		);

		INSERT INTO migrations (schema_version) VALUES (0) ON CONFLICT DO NOTHING;
	`)
	if err != nil {
		return errors.New("sql.Tx.Exec (v0) " + err.Error())
	}

	// Get current schema version
	rows, err = tx.Query("SELECT MAX(schema_version) FROM migrations;")
	if err != nil {
		return errors.New("sql.Tx.Query (SELECT MAX(version) FROM migrations) " + err.Error())
	}
	defer db.closeRows(rows)

	var schemaVersion int
	if !rows.Next() {
		return fmt.Errorf("sql.Rows.Next (SELECT MAX(version) FROM migrations): Query did not return any rows")
	}
	if err = rows.Scan(&schemaVersion); err != nil {
		return errors.New("sql.Rows.Scan (MAX(version)) " + err.Error())
	}
	// If next line is removed we're getting error on sql.Tx.Commit: unexpected command tag SELECT
	// https://stackoverflow.com/questions/36295883/golang-postgres-commit-unknown-command-error/36866993#36866993
	db.closeRows(rows)

	switch schemaVersion {
	case 0: // FROZEN.
		log.Println("Updating database schema from 0 to 1... (this might take a while)")
		migrationStmt := `
				-- Rename the existing table
				ALTER TABLE files RENAME TO files_old;

				-- Create the new partitioned table
				CREATE TABLE files (
					id          INTEGER PRIMARY KEY DEFAULT nextval('seq_files_id'),
					torrent_id  INTEGER REFERENCES torrents ON DELETE CASCADE ON UPDATE RESTRICT,
					size        BIGINT NOT NULL,
					path        TEXT NOT NULL
				) PARTITION BY HASH (id);

				-- Create the partitions
				DO $$
				BEGIN
					FOR i IN 0..99 LOOP
						EXECUTE format(
							'CREATE TABLE files_p%s PARTITION OF files FOR VALUES WITH (MODULUS 100, REMAINDER %s);',
							i, i
						);
						EXECUTE format(
							'CREATE INDEX IF NOT EXISTS idx_files_torrent_id_p%s ON files_p%s (torrent_id);',
							i, i
						);
						EXECUTE format(
							'CREATE INDEX IF NOT EXISTS idx_files_path_gin_trgm_p%s ON files_p%s USING GIN (path gin_trgm_ops);',
							i, i
						);
					END LOOP;
				END$$;

				-- Copy data from the old table to the new partitioned table
				INSERT INTO files (id, torrent_id, size, path) SELECT id, torrent_id, size, path FROM files_old;

				-- Drop the old indexes, table and finalize the migration
				DROP INDEX IF EXISTS idx_files_torrent_id;
				DROP TABLE IF EXISTS files_old;
				INSERT INTO migrations (schema_version) VALUES (1);
			`
		if db.isCockroach {
			migrationStmt = `
				CREATE INDEX IF NOT EXISTS idx_files_path_gin_trgm ON files USING GIN (path gin_trgm_ops);
				INSERT INTO migrations (schema_version) VALUES (1);
			`
		}

		if _, err := tx.Exec(migrationStmt); err != nil {
			return errors.New("sql.Tx.Exec (v0 -> v1) " + err.Error())
		}
		fallthrough

	case 1:
		log.Println("Updating database schema from 1 to 2...")
		_, err = tx.Exec(`
			CREATE TABLE IF NOT EXISTS dht_nodes (
				addr TEXT PRIMARY KEY,
				sightings BIGINT NOT NULL
			);
			CREATE TABLE IF NOT EXISTS dht_hashes (
				slot INTEGER PRIMARY KEY,
				hash BYTEA NOT NULL
			);
			CREATE TABLE IF NOT EXISTS dht_outbox (
				message_id TEXT PRIMARY KEY,
				queued_at BIGINT NOT NULL,
				destination_ip TEXT NOT NULL,
				destination_port INTEGER NOT NULL,
				destination_zone TEXT NOT NULL,
				payload BYTEA NOT NULL
			);
			CREATE INDEX IF NOT EXISTS dht_outbox_queued ON dht_outbox (queued_at, message_id);
			INSERT INTO migrations (schema_version) VALUES (2);
		`)
		if err != nil {
			return errors.New("sql.Tx.Exec (v1 -> v2) " + err.Error())
		}
	}

	if err = tx.Commit(); err != nil {
		return errors.New("sql.Tx.Commit " + err.Error())
	}

	return nil
}

func (db *postgresDatabase) closeRows(rows *sql.Rows) {
	if err := rows.Close(); err != nil {
		panic("postgres: could not close row " + err.Error())
	}
}

func (db *postgresDatabase) rollback(tx *sql.Tx) {
	if err := tx.Rollback(); err != nil &&
		!strings.Contains(err.Error(), "transaction has already been committed") {
		panic("postgres: could not rollback transaction " + err.Error())
	}
}

func (db *postgresDatabase) orderOn(orderBy OrderingCriteria) string {
	switch orderBy {
	case ByRelevance:
		return "discovered_on"

	case ByTotalSize:
		return "total_size"

	case ByDiscoveredOn:
		return "discovered_on"

	case ByNFiles:
		return "n_files"

	default:
		panic(fmt.Sprintf("unknown orderBy: %v", orderBy))
	}
}

func (db *postgresDatabase) executeTemplate(text string, data any, funcs template.FuncMap) string {
	t := template.Must(template.New("anon").Funcs(funcs).Parse(text))

	var buf bytes.Buffer
	err := t.Execute(&buf, data)
	if err != nil {
		panic(err.Error())
	}
	return buf.String()
}

func (db *postgresDatabase) Export() (chan SimpleTorrentSummary, error) {
	out := make(chan SimpleTorrentSummary)
	rows, err := db.conn.Query("SELECT info_hash, name, id FROM torrents;")
	if err != nil {
		return nil, err
	}

	go func(out chan SimpleTorrentSummary, rows *sql.Rows) {
		defer close(out)
		defer rows.Close()
		for rows.Next() {
			var infoHash []byte
			var name string
			var id int64

			err = rows.Scan(&infoHash, &name, &id)
			if err != nil {
				log.Fatalln("Error scanning row:", err.Error())
			}

			files, err := db.GetFiles(infoHash)
			if err != nil {
				log.Fatalln("Error getting files:", err.Error())
			}

			out <- SimpleTorrentSummary{
				InfoHash: hex.EncodeToString(infoHash),
				Name:     name,
				Files:    files,
			}
		}
	}(out, rows)

	return out, nil
}
