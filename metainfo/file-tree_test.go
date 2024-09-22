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
