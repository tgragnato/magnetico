package persistence

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net/url"
	"strconv"
	"strings"
	"text/template"
	"time"
	"unicode/utf8"

	_ "github.com/jackc/pgx/v5"
	_ "github.com/jackc/pgx/v5/stdlib"
)

type postgresDatabase struct {
	conn *sql.DB
}

func makePostgresDatabase(url_ *url.URL) (Database, error) {
	db := new(postgresDatabase)

	if url_.Scheme == "cockroach" {
		url_.Scheme = "postgres"
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

	// https://github.com/mattn/go-sqlite3/issues/618
	db.conn.SetConnMaxLifetime(0) // https://golang.org/pkg/database/sql/#DB.SetConnMaxLifetime
	db.conn.SetMaxOpenConns(3)
	db.conn.SetMaxIdleConns(3)

	if err := db.setupDatabase(); err != nil {
		return nil, errors.New("setupDatabase " + err.Error())
	}

	return db, nil
}

func (db *postgresDatabase) Engine() databaseEngine {
	return Postgres
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
		log.Printf("Ignoring a torrent whose name is not UTF-8 compliant. infoHash: %s", infoHash)
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
			log.Printf("Ignoring a file whose path is not UTF-8 compliant. %s", file.Path)

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
	rows, err := db.conn.Query("SELECT COUNT(*)::BIGINT AS exact_count FROM torrents;")
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	if !rows.Next() {
		return 0, errors.New("no rows returned from `SELECT COUNT(*)::BIGINT AS exact_count FROM torrents;`")
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

func (db *postgresDatabase) QueryTorrents(
	query string,
	epoch int64,
	orderBy OrderingCriteria,
	ascending bool,
	limit uint,
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
			name LIKE CONCAT('%',$1::text,'%') AND
			discovered_on <= $2 AND
			{{.OrderOn}} {{GTEorLTE .Ascending}} $3 AND
			id {{GTEorLTE .Ascending}} $4
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
	defer db.closeRows(rows)
	if err != nil {
		return nil, err
	}

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
	defer db.closeRows(rows)
	if err != nil {
		return nil, err
	}

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

	var toTime time.Time
	var timef string

	switch gran {
	case Year:
		toTime = fromTime.AddDate(int(n), 0, 0)
		timef = "2006"
	case Month:
		toTime = fromTime.AddDate(0, int(n), 0)
		timef = "2006-01"
	case Week:
		toTime = fromTime.AddDate(0, 0, int(n)*7)
		timef = "2006-01-02"
	case Day:
		toTime = fromTime.AddDate(0, 0, int(n))
		timef = "2006-01-02"
	case Hour:
		toTime = fromTime.Add(time.Duration(n) * time.Hour)
		timef = "2006-01-02 15:04"
	}

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
			if err := rows.Close(); err != nil {
				panic(err.Error())
			}
			return nil, err
		}

		epoch, _ := strconv.ParseInt(dT, 10, 64)
		dT = time.Unix(epoch, 0).Format(timef)

		stats.NDiscovered[dT] = nD
		stats.TotalSize[dT] = tS
		stats.NFiles[dT] = nF
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

	// Uncomment for future migrations:
	//switch schemaVersion {
	//case 0: // FROZEN.
	//	log.Println("Updating (fake) database schema from 0 to 1...")
	//	_, err = tx.Exec(`INSERT INTO migrations (schema_version) VALUES (1);`)
	//	if err != nil {
	//		return errors.Wrap(err, "sql.Tx.Exec (v0 -> v1)")
	//	}
	//	//fallthrough
	//}

	if err = tx.Commit(); err != nil {
		return errors.New("sql.Tx.Commit " + err.Error())
	}

	return nil
}

func (db *postgresDatabase) closeRows(rows *sql.Rows) {
	if err := rows.Close(); err != nil {
		log.Printf("could not close row %v", err)
	}
}

func (db *postgresDatabase) rollback(tx *sql.Tx) {
	if err := tx.Rollback(); err != nil &&
		!strings.Contains(err.Error(), "transaction has already been committed") {
		log.Printf("could not rollback transaction %v", err)
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

func (db *postgresDatabase) executeTemplate(text string, data interface{}, funcs template.FuncMap) string {
	t := template.Must(template.New("anon").Funcs(funcs).Parse(text))

	var buf bytes.Buffer
	err := t.Execute(&buf, data)
	if err != nil {
		panic(err.Error())
	}
	return buf.String()
}
