package persistence

import (
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	mrand "math/rand"
	"reflect"
	"testing"
	"text/template"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
)

func TestPostgresDatabase_ExecuteTemplate(t *testing.T) {
	t.Parallel()

	db := &postgresDatabase{}

	text := "Hello, {{.Name}}!"
	data := struct {
		Name string
	}{
		Name: "World",
	}

	expected := "Hello, World!"

	result := db.executeTemplate(text, data, template.FuncMap{})
	if result != expected {
		t.Errorf("Expected result to be %q, but got %q", expected, result)
	}
}

func TestPostgresDatabase_OrderOn(t *testing.T) {
	t.Parallel()

	db := &postgresDatabase{}

	testCases := []struct {
		orderBy  OrderingCriteria
		expected string
	}{
		{ByRelevance, "discovered_on"},
		{ByTotalSize, "total_size"},
		{ByDiscoveredOn, "discovered_on"},
		{ByNFiles, "n_files"},
	}

	for _, tc := range testCases {
		result := db.orderOn(tc.orderBy)
		if result != tc.expected {
			t.Errorf("Expected orderOn(%v) to return %q, but got %q", tc.orderBy, tc.expected, result)
		}
	}
}

func TestDoesTorrentExist(t *testing.T) {
	t.Parallel()

	conn, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}

	var random [100]byte
	_, err = rand.Read(random[:])
	if err != nil {
		for i := 0; i < 100; i++ {
			random[i] = byte(mrand.Intn(256))
		}
	}
	infohash := sha1.Sum(random[:])

	rows := sqlmock.NewRows([]string{"1"}).AddRow("1")
	mock.ExpectQuery("SELECT 1 FROM torrents WHERE info_hash = \\$1;").WithArgs(infohash[:]).WillReturnRows(rows)

	db := &postgresDatabase{conn: conn}
	found, err := db.DoesTorrentExist(infohash[:])
	if err != nil {
		t.Error(err)
	}
	if !found {
		t.Error("row returned but no result found")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}

	rows = sqlmock.NewRows([]string{"1"})
	mock.ExpectQuery("SELECT 1 FROM torrents WHERE info_hash = \\$1;").WithArgs(infohash[:]).WillReturnRows(rows)
	found, err = db.DoesTorrentExist(infohash[:])
	if err != nil {
		t.Error(err)
	}
	if found {
		t.Error("no row returned but result found")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func TestPostgresDatabase_GetNumberOfTorrents(t *testing.T) {
	t.Parallel()

	conn, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	rows := sqlmock.NewRows([]string{"exact_count"}).AddRow(10)
	mock.ExpectQuery("SELECT COUNT\\(\\*\\)::BIGINT AS exact_count FROM torrents;").WillReturnRows(rows)

	db := &postgresDatabase{conn: conn}
	count, err := db.GetNumberOfTorrents()
	if err != nil {
		t.Error(err)
	}
	if count != 10 {
		t.Errorf("Expected count to be 10, but got %d", count)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}

	rows = sqlmock.NewRows([]string{"exact_count"})
	mock.ExpectQuery("SELECT COUNT\\(\\*\\)::BIGINT AS exact_count FROM torrents;").WillReturnRows(rows)

	count, err = db.GetNumberOfTorrents()
	if err == nil {
		t.Error("no rows returned for query without corresponding error")
	}
	if count != 0 {
		t.Errorf("Expected count to be 0, but got %d", count)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}

	rows = sqlmock.NewRows([]string{"exact_count"}).AddRow(nil)
	mock.ExpectQuery("SELECT COUNT\\(\\*\\)::BIGINT AS exact_count FROM torrents;").WillReturnRows(rows)

	count, err = db.GetNumberOfTorrents()
	if err != nil {
		t.Error(err)
	}
	if count != 0 {
		t.Errorf("Expected count to be 0, but got %d", count)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func TestPostgresDatabase_GetNumberOfQueryTorrents(t *testing.T) {
	t.Parallel()

	conn, mock, err := sqlmock.New()
	if err != nil {
		t.Fatalf("An error '%s' was not expected when opening a stub database connection", err)
	}
	defer conn.Close()

	pgDb := &postgresDatabase{conn: conn}

	query := "test-query"
	epoch := int64(1609459200) // 2021-01-01 00:00:00 UTC

	rows := sqlmock.NewRows([]string{"count"}).AddRow(int64(10))
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM torrents WHERE name ILIKE CONCAT\('%',\$1::text,'%'\) AND discovered_on <= \$2;`).
		WithArgs(query, epoch).
		WillReturnRows(rows)

	result, err := pgDb.GetNumberOfQueryTorrents(query, epoch)

	if err != nil {
		t.Errorf("Expected no error, but got %v", err)
	}

	if result != uint64(10) {
		t.Errorf("Expected result to be 10, but got %d", result)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("There were unmet expectations: %s", err)
	}

	rows = sqlmock.NewRows([]string{"count"})
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM torrents WHERE name ILIKE CONCAT\('%',\$1::text,'%'\) AND discovered_on <= \$2;`).
		WithArgs(query, epoch).
		WillReturnRows(rows)

	result, err = pgDb.GetNumberOfQueryTorrents(query, epoch)

	if err == nil {
		t.Error("Expected an error, but got none")
	}

	if result != uint64(0) {
		t.Errorf("Expected result to be 0, but got %d", result)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("There were unmet expectations: %s", err)
	}

	rows = sqlmock.NewRows([]string{"count"}).AddRow(nil)
	mock.ExpectQuery(`SELECT COUNT\(\*\) FROM torrents WHERE name ILIKE CONCAT\('%',\$1::text,'%'\) AND discovered_on <= \$2;`).
		WithArgs(query, epoch).
		WillReturnRows(rows)

	result, err = pgDb.GetNumberOfQueryTorrents(query, epoch)

	if err != nil {
		t.Errorf("Expected no error, but got %v", err)
	}

	if result != uint64(0) {
		t.Errorf("Expected result to be 0, but got %d", result)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("There were unmet expectations: %s", err)
	}

}

func TestPostgresDatabase_Close(t *testing.T) {
	t.Parallel()

	conn, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	db := &postgresDatabase{conn: conn}

	mock.ExpectClose()

	err = db.Close()
	if err != nil {
		t.Error(err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func TestPostgresDatabase_GetTorrent(t *testing.T) {
	t.Parallel()

	conn, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	var random [100]byte
	_, err = rand.Read(random[:])
	if err != nil {
		for i := 0; i < 100; i++ {
			random[i] = byte(mrand.Intn(256))
		}
	}
	infohash := sha1.Sum(random[:])
	name := "Test Torrent"
	size := uint64(1024)
	discoveredOn := time.Now().Unix()

	rows := sqlmock.NewRows([]string{"info_hash", "name", "total_size", "discovered_on", "n_files"}).
		AddRow(infohash[:], name, size, discoveredOn, 5)
	mock.ExpectQuery("SELECT t.info_hash, t.name, t.total_size, t.discovered_on, \\(SELECT COUNT\\(\\*\\) FROM files f WHERE f.torrent_id = t.id\\) AS n_files FROM torrents t WHERE t.info_hash = \\$1;").
		WithArgs(infohash[:]).
		WillReturnRows(rows)

	db := &postgresDatabase{conn: conn}
	torrent, err := db.GetTorrent(infohash[:])
	if err != nil {
		t.Error(err)
	}
	if torrent == nil {
		t.Fatal("Expected torrent to be found, but got nil")
	}

	if !bytes.Equal(torrent.InfoHash, infohash[:]) {
		t.Errorf("Expected InfoHash to be %v, but got %v", infohash, torrent.InfoHash)
	}
	if torrent.Name != name {
		t.Errorf("Expected Name to be %q, but got %q", name, torrent.Name)
	}
	if torrent.Size != size {
		t.Errorf("Expected Size to be %d, but got %d", size, torrent.Size)
	}
	if torrent.DiscoveredOn != discoveredOn {
		t.Errorf("Expected DiscoveredOn to be %v, but got %v", discoveredOn, torrent.DiscoveredOn)
	}
	if torrent.NFiles != 5 {
		t.Errorf("Expected NFiles to be 5, but got %d", torrent.NFiles)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}

	rows = sqlmock.NewRows([]string{"info_hash", "name", "total_size", "discovered_on", "n_files"})
	mock.ExpectQuery("SELECT t.info_hash, t.name, t.total_size, t.discovered_on, \\(SELECT COUNT\\(\\*\\) FROM files f WHERE f.torrent_id = t.id\\) AS n_files FROM torrents t WHERE t.info_hash = \\$1;").
		WithArgs(infohash[:]).
		WillReturnRows(rows)

	torrent, err = db.GetTorrent(infohash[:])
	if err != nil {
		t.Error(err)
	}
	if torrent != nil {
		t.Error("Expected torrent to be not found, but got a result")
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func TestPostgresDatabase_GetFiles(t *testing.T) {
	t.Parallel()

	conn, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	var random [100]byte
	_, err = rand.Read(random[:])
	if err != nil {
		for i := 0; i < 100; i++ {
			random[i] = byte(mrand.Intn(256))
		}
	}
	infohash := sha1.Sum(random[:])

	rows := sqlmock.NewRows([]string{"size", "path"}).
		AddRow(1024, "/path/to/file1").
		AddRow(2048, "/path/to/file2")
	mock.ExpectQuery("SELECT f.size, f.path FROM files f, torrents t WHERE f.torrent_id = t.id AND t.info_hash = \\$1;").
		WithArgs(infohash[:]).
		WillReturnRows(rows)

	db := &postgresDatabase{conn: conn}
	files, err := db.GetFiles(infohash[:])
	if err != nil {
		t.Error(err)
	}
	if len(files) != 2 {
		t.Errorf("Expected 2 files, but got %d", len(files))
	}

	expectedFiles := []File{
		{Size: 1024, Path: "/path/to/file1"},
		{Size: 2048, Path: "/path/to/file2"},
	}
	for i, file := range files {
		if file.Size != expectedFiles[i].Size {
			t.Errorf("Expected file size to be %d, but got %d", expectedFiles[i].Size, file.Size)
		}
		if file.Path != expectedFiles[i].Path {
			t.Errorf("Expected file path to be %q, but got %q", expectedFiles[i].Path, file.Path)
		}
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}

	rows = sqlmock.NewRows([]string{"size", "path"})
	mock.ExpectQuery("SELECT f.size, f.path FROM files f, torrents t WHERE f.torrent_id = t.id AND t.info_hash = \\$1;").
		WithArgs(infohash[:]).
		WillReturnRows(rows)

	files, err = db.GetFiles(infohash[:])
	if err != nil {
		t.Error(err)
	}
	if len(files) != 0 {
		t.Errorf("Expected 0 files, but got %d", len(files))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func TestPostgresDatabase_GetStatistics(t *testing.T) {
	t.Parallel()

	conn, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	from := "2023-08-31"
	n := uint(1)

	rows := sqlmock.NewRows([]string{"dT", "tS", "nD", "nF"}).
		AddRow("1640995200", uint64(1024), uint64(10), uint64(5)).
		AddRow("1641081600", uint64(2048), uint64(20), uint64(10))
	mock.ExpectQuery("SELECT discovered_on AS dT, sum\\(files.size\\) AS tS, count\\(DISTINCT torrents.id\\) AS nD, count\\(DISTINCT files.id\\) AS nF FROM torrents, files WHERE torrents.id = files.torrent_id AND discovered_on >= \\$1 AND discovered_on <= \\$2 GROUP BY dt;").
		WithArgs(1693526399, 1693612799).
		WillReturnRows(rows)

	db := &postgresDatabase{conn: conn}
	stats, err := db.GetStatistics(from, n)
	if err != nil {
		t.Error(err)
	}
	if stats == nil {
		t.Fatal("Expected statistics to be found, but got nil")
	}

	expectedStats := &Statistics{
		NDiscovered: map[string]uint64{
			"2022-01-01": 10,
			"2022-01-02": 20,
		},
		TotalSize: map[string]uint64{
			"2022-01-01": 1024,
			"2022-01-02": 2048,
		},
		NFiles: map[string]uint64{
			"2022-01-01": 5,
			"2022-01-02": 10,
		},
	}
	if !reflect.DeepEqual(stats, expectedStats) {
		t.Errorf("Expected statistics to be %v, but got %v", expectedStats, stats)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}

	rows = sqlmock.NewRows([]string{"dT", "tS", "nD", "nF"})
	mock.ExpectQuery("SELECT discovered_on AS dT, sum\\(files.size\\) AS tS, count\\(DISTINCT torrents.id\\) AS nD, count\\(DISTINCT files.id\\) AS nF FROM torrents, files WHERE torrents.id = files.torrent_id AND discovered_on >= \\$1 AND discovered_on <= \\$2 GROUP BY dt;").
		WithArgs(1693526399, 1693612799).
		WillReturnRows(rows)

	stats, err = db.GetStatistics(from, n)
	if err != nil {
		t.Error(err)
	}
	expectedStats = NewStatistics()
	if !reflect.DeepEqual(stats, expectedStats) {
		t.Errorf("Expected statistics to be %v, but got %v", expectedStats, stats)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func TestPostgresDatabase_QueryTorrents(t *testing.T) {
	t.Parallel()

	conn, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	db := &postgresDatabase{conn: conn}

	query := "test"
	epoch := int64(1693526399)
	orderBy := ByTotalSize
	ascending := true
	limit := uint64(10)
	lastOrderedValue := float64(100)
	lastID := uint64(5)

	rows := sqlmock.NewRows([]string{"id", "info_hash", "name", "total_size", "discovered_on", "n_files", "relevance"}).
		AddRow(1, []byte("infohash1"), "Torrent 1", uint64(1024), int64(1640995200), uint64(5), float64(0.5)).
		AddRow(2, []byte("infohash2"), "Torrent 2", uint64(2048), int64(1641081600), uint64(10), float64(0.8))
	mock.ExpectQuery(`
			SELECT
				id,
				info_hash,
				name,
				total_size,
				discovered_on,
				\(SELECT COUNT\(\*\) FROM files WHERE torrents.id = files.torrent_id\) AS n_files,
				0
			FROM torrents
			WHERE
				\(\$1::text = '' OR name ILIKE CONCAT\('%',\$1::text,'%'\)\) AND
				discovered_on <= \$2 AND
				\(\$3 = 0 OR total_size > \$3\) AND
				\(\$4 = 0 OR id > \$4\)
			ORDER BY total_size ASC, id ASC
			LIMIT \$5;
		`).
		WithArgs(query, epoch, lastOrderedValue, lastID, limit).
		WillReturnRows(rows)

	torrents, err := db.QueryTorrents(query, epoch, orderBy, ascending, limit, &lastOrderedValue, &lastID)
	if err != nil {
		t.Error(err)
	}
	if len(torrents) != 2 {
		t.Errorf("Expected 2 torrents, but got %d", len(torrents))
	}

	expectedTorrents := []TorrentMetadata{
		{
			ID:           1,
			InfoHash:     []byte("infohash1"),
			Name:         "Torrent 1",
			Size:         1024,
			DiscoveredOn: 1640995200,
			NFiles:       5,
			Relevance:    0.5,
		},
		{
			ID:           2,
			InfoHash:     []byte("infohash2"),
			Name:         "Torrent 2",
			Size:         2048,
			DiscoveredOn: 1641081600,
			NFiles:       10,
			Relevance:    0.8,
		},
	}
	for i, torrent := range torrents {
		if !reflect.DeepEqual(torrent, expectedTorrents[i]) {
			t.Errorf("Expected torrent to be %v, but got %v", expectedTorrents[i], torrent)
		}
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}

	rows = sqlmock.NewRows([]string{"id", "info_hash", "name", "total_size", "discovered_on", "n_files", "relevance"})
	mock.ExpectQuery(`
			SELECT
				id,
				info_hash,
				name,
				total_size,
				discovered_on,
				\(SELECT COUNT\(\*\) FROM files WHERE torrents.id = files.torrent_id\) AS n_files,
				0
			FROM torrents
			WHERE
				\(\$1::text = '' OR name ILIKE CONCAT\('%',\$1::text,'%'\)\) AND
				discovered_on <= \$2 AND
				\(\$3 = 0 OR total_size > \$3\) AND
				\(\$4 = 0 OR id > \$4\)
			ORDER BY total_size ASC, id ASC
			LIMIT \$5;
		`).
		WithArgs(query, epoch, lastOrderedValue, lastID, limit).
		WillReturnRows(rows)

	torrents, err = db.QueryTorrents(query, epoch, orderBy, ascending, limit, &lastOrderedValue, &lastID)
	if err != nil {
		t.Error(err)
	}
	if len(torrents) != 0 {
		t.Errorf("Expected 0 torrents, but got %d", len(torrents))
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func TestPostgresDatabase_AddNewTorrent(t *testing.T) {
	t.Parallel()

	conn, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	db := &postgresDatabase{conn: conn}

	infoHash := []byte("infohash")
	name := "Test Torrent"
	files := []File{
		{Size: 1024, Path: "/path/to/file1"},
		{Size: 2048, Path: "/path/to/file2"},
	}

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT 1 FROM torrents WHERE info_hash = \\$1;").
		WithArgs(infoHash).
		WillReturnRows(sqlmock.NewRows([]string{"1"}))
	mock.ExpectQuery(`
		INSERT INTO torrents \(
			info_hash,
			name,
			total_size,
			discovered_on
		\) VALUES \(\$1, \$2, \$3, \$4\)
		RETURNING id;
	`).
		WithArgs(infoHash, name, uint64(3072), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(1))
	mock.ExpectExec("INSERT INTO files \\(torrent_id, size, path\\) VALUES \\(\\$1, \\$2, \\$3\\);").
		WithArgs(1, 1024, "/path/to/file1").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectExec("INSERT INTO files \\(torrent_id, size, path\\) VALUES \\(\\$1, \\$2, \\$3\\);").
		WithArgs(1, 2048, "/path/to/file2").
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = db.AddNewTorrent(infoHash, name, files)
	if err != nil {
		t.Error(err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}

func TestPostgresDatabase_Engine(t *testing.T) {
	t.Parallel()

	instance := &postgresDatabase{}
	if got := instance.Engine(); got != Postgres {
		t.Errorf("zeromq.Engine() = %v, want %v", got, Postgres)
	}
}

func TestPostgresDatabase_SetupDatabase(t *testing.T) {
	t.Parallel()

	conn, mock, err := sqlmock.New()
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	db := &postgresDatabase{conn: conn}

	mock.ExpectBegin()
	mock.ExpectQuery("SELECT 1 FROM pg_extension WHERE extname = 'pg_trgm';").
		WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow("1"))
	mock.ExpectExec(`
		-- Torrents ID sequence generator
		CREATE SEQUENCE IF NOT EXISTS seq_torrents_id;
		-- Files ID sequence generator
		CREATE SEQUENCE IF NOT EXISTS seq_files_id;

		CREATE TABLE IF NOT EXISTS torrents \(
			id             INTEGER PRIMARY KEY DEFAULT nextval\('seq_torrents_id'\),
			info_hash      bytea NOT NULL UNIQUE,
			name           TEXT NOT NULL,
			total_size     BIGINT NOT NULL CHECK\(total_size > 0\),
			discovered_on  INTEGER NOT NULL CHECK\(discovered_on > 0\)
		\);

		-- Indexes for search sorting options
		CREATE INDEX IF NOT EXISTS idx_torrents_total_size ON torrents \(total_size\);
		CREATE INDEX IF NOT EXISTS idx_torrents_discovered_on ON torrents \(discovered_on\);

		-- Using pg_trgm GIN index for fast ILIKE queries
		-- You need to execute "CREATE EXTENSION pg_trgm" on your database for this index to work
		-- Be aware that using this type of index implies that making ILIKE queries with less that
		-- 3 character values will cause full table scan instead of using index.
		-- You can try to avoid that by doing 'SET enable_seqscan=off'.
		CREATE INDEX IF NOT EXISTS idx_torrents_name_gin_trgm ON torrents USING GIN \(name gin_trgm_ops\);

		CREATE TABLE IF NOT EXISTS files \(
			id          INTEGER PRIMARY KEY DEFAULT nextval\('seq_files_id'\),
			torrent_id  INTEGER REFERENCES torrents ON DELETE CASCADE ON UPDATE RESTRICT,
			size        BIGINT NOT NULL,
			path        TEXT NOT NULL
		\);

		CREATE INDEX IF NOT EXISTS idx_files_torrent_id ON files \(torrent_id\);

		CREATE TABLE IF NOT EXISTS migrations \(
			schema_version		SMALLINT NOT NULL UNIQUE 
		\);

		INSERT INTO migrations \(schema_version\) VALUES \(0\) ON CONFLICT DO NOTHING;
	`).
		WillReturnResult(sqlmock.NewResult(0, 0))
	mock.ExpectQuery("SELECT MAX\\(schema_version\\) FROM migrations;").
		WillReturnRows(sqlmock.NewRows([]string{"MAX(schema_version)"}).AddRow(0))
	mock.ExpectCommit()

	err = db.setupDatabase()
	if err != nil {
		t.Error(err)
	}

	if err := mock.ExpectationsWereMet(); err != nil {
		t.Error(err)
	}
}
