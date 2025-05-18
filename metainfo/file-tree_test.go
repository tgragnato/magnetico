package metainfo

import (
	"testing"
)

func TestMarshalBencode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		fileTree FileTree
		wantErr  bool
	}{
		{
			name: "Empty directory",
			fileTree: FileTree{
				Dir: make(map[string]FileTree),
			},
			wantErr: false,
		},
		{
			name: "Single file",
			fileTree: FileTree{
				File: FileTreeFile{
					Length:     1234,
					PiecesRoot: "abcdef",
				},
			},
			wantErr: false,
		},
		{
			name: "Nested directories",
			fileTree: FileTree{
				Dir: map[string]FileTree{
					"subdir": {
						Dir: map[string]FileTree{
							"file": {
								File: FileTreeFile{
									Length:     5678,
									PiecesRoot: "ghijkl",
								},
							},
						},
					},
				},
			},
			wantErr: false,
		},
		{
			name: "Missing subfile",
			fileTree: FileTree{
				Dir: map[string]FileTree{
					"subdir": {},
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := tt.fileTree.MarshalBencode()
			if (err != nil) != tt.wantErr {
				t.Errorf("MarshalBencode() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestFileTree_upvertedFilesInner(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		fileTree    FileTree
		pieceLength int64
		wantFiles   []FileInfo
	}{
		{
			name: "Single file",
			fileTree: FileTree{
				File: FileTreeFile{
					Length:     1234,
					PiecesRoot: "abcdef",
				},
			},
			pieceLength: 512,
			wantFiles: []FileInfo{
				{
					Length: 1234,
				},
			},
		},
		{
			name: "Nested directories",
			fileTree: FileTree{
				Dir: map[string]FileTree{
					"subdir": {
						Dir: map[string]FileTree{
							"file": {
								File: FileTreeFile{
									Length:     5678,
									PiecesRoot: "ghijkl",
								},
							},
						},
					},
				},
			},
			pieceLength: 1024,
			wantFiles: []FileInfo{
				{
					Length:   5678,
					Path:     []string{"subdir", "file"},
					PathUtf8: []string{"subdir", "file"},
				},
			},
		},
		{
			name: "Multiple files",
			fileTree: FileTree{
				Dir: map[string]FileTree{
					"file1": {
						File: FileTreeFile{
							Length:     1000,
							PiecesRoot: "mnopqr",
						},
					},
					"file2": {
						File: FileTreeFile{
							Length:     2000,
							PiecesRoot: "stuvwx",
						},
					},
				},
			},
			pieceLength: 512,
			wantFiles: []FileInfo{
				{
					Length:        1000,
					Path:          []string{"file1"},
					PathUtf8:      []string{"file1"},
					TorrentOffset: 0,
				},
				{
					Length:        2000,
					Path:          []string{"file2"},
					PathUtf8:      []string{"file2"},
					TorrentOffset: 1024,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotFiles []FileInfo
			for fi := range tt.fileTree.upvertedFilesInner(tt.pieceLength, nil, new(int64)) {
				gotFiles = append(gotFiles, fi)
			}

			if len(gotFiles) != len(tt.wantFiles) {
				t.Fatalf("expected %d files, got %d", len(tt.wantFiles), len(gotFiles))
			}

			for i, got := range gotFiles {
				want := tt.wantFiles[i]
				if got.Length != want.Length || got.TorrentOffset != want.TorrentOffset || !equalPaths(got.Path, want.Path) || !equalPaths(got.PathUtf8, want.PathUtf8) || got.PiecesRoot != want.PiecesRoot {
					t.Errorf("file %d: got %+v, want %+v", i, got, want)
				}
			}
		})
	}
}

func equalPaths(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
