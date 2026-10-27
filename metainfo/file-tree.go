package metainfo

import (
	"bytes"
	"errors"
	"fmt"
	"iter"
	"maps"
	"slices"

	"tgragnato.it/magnetico/v2/bencode"
	infohash_v2 "tgragnato.it/magnetico/v2/types/infohash-v2"
)

const FileTreePropertiesKey = ""

type FileTreeFile struct {
	Length     int64  `bencode:"length"`
	PiecesRoot string `bencode:"pieces root"`
}

// The fields here don't need bencode tags as the marshalling is done manually.
type FileTree struct {
	File FileTreeFile
	Dir  map[string]FileTree
}

// UnmarshalBencode parses the tree in a single pass over data, iteratively, so that cost is linear
// in the input size and independent of nesting depth.
func (ft *FileTree) UnmarshalBencode(data []byte) error {
	type frame struct {
		tree FileTree
		key  string // key of the child currently being parsed
	}

	if len(data) == 0 || data[0] != 'd' {
		return errors.New("file tree: expected dictionary")
	}
	stack := []frame{{tree: FileTree{Dir: make(map[string]FileTree)}}}
	pos := 1

	for {
		if pos >= len(data) {
			return errors.New("file tree: unexpected end of data")
		}
		top := &stack[len(stack)-1]

		if data[pos] == 'e' {
			pos++
			done := top.tree
			stack = stack[:len(stack)-1]
			if len(stack) == 0 {
				if pos != len(data) {
					return fmt.Errorf("file tree: %d unused trailing bytes", len(data)-pos)
				}
				*ft = done
				return nil
			}
			parent := &stack[len(stack)-1]
			parent.tree.Dir[parent.key] = done
			continue
		}

		keyBytes, next, err := readBencodeString(data, pos)
		if err != nil {
			return err
		}
		pos = next
		if pos >= len(data) {
			return errors.New("file tree: unexpected end of data")
		}

		if len(keyBytes) == 0 {
			end, err := skipBencodeValue(data, pos)
			if err != nil {
				return err
			}
			if err := bencode.Unmarshal(data[pos:end], &top.tree.File); err != nil {
				return err
			}
			pos = end
			continue
		}

		if data[pos] != 'd' {
			return errors.New("file tree: expected dictionary")
		}
		top.key = string(keyBytes)
		stack = append(stack, frame{tree: FileTree{Dir: make(map[string]FileTree)}})
		pos++
	}
}

// readBencodeString reads a length-prefixed string at pos and returns it with the following offset.
func readBencodeString(data []byte, pos int) ([]byte, int, error) {
	n, i := 0, pos
	for ; i < len(data) && data[i] >= '0' && data[i] <= '9'; i++ {
		n = n*10 + int(data[i]-'0')
		if n > len(data) {
			return nil, 0, errors.New("file tree: string length out of range")
		}
	}
	if i == pos || i >= len(data) || data[i] != ':' {
		return nil, 0, errors.New("file tree: expected string")
	}
	i++
	if n > len(data)-i {
		return nil, 0, errors.New("file tree: unexpected end of data")
	}
	return data[i : i+n], i + n, nil
}

// skipBencodeValue returns the offset just past the value starting at pos, without recursion.
func skipBencodeValue(data []byte, pos int) (int, error) {
	depth := 0
	for {
		if pos >= len(data) {
			return 0, errors.New("file tree: unexpected end of data")
		}
		switch c := data[pos]; {
		case c == 'i':
			e := bytes.IndexByte(data[pos:], 'e')
			if e < 0 {
				return 0, errors.New("file tree: unterminated integer")
			}
			pos += e + 1
		case c >= '0' && c <= '9':
			_, next, err := readBencodeString(data, pos)
			if err != nil {
				return 0, err
			}
			pos = next
		case c == 'l' || c == 'd':
			depth++
			pos++
		case c == 'e' && depth > 0:
			depth--
			pos++
		default:
			return 0, errors.New("file tree: invalid bencode value")
		}
		if depth == 0 {
			return pos, nil
		}
	}
}

var _ bencode.Unmarshaler = (*FileTree)(nil)

func (ft *FileTree) MarshalBencode() (bytes []byte, err error) {
	if ft.IsDir() {
		dir := make(map[string]bencode.Bytes, len(ft.Dir))
		for _, key := range ft.orderedKeys() {
			if key == FileTreePropertiesKey {
				continue
			}
			sub, ok := ft.Dir[key]
			if !ok {
				return nil, errors.New("missing subfile")
			}
			subBytes, err := sub.MarshalBencode()
			if err != nil {
				return nil, err
			}
			dir[key] = subBytes
		}
		return bencode.Marshal(dir)
	} else {
		fileBytes, err := bencode.Marshal(ft.File)
		if err != nil {
			return nil, err
		}
		res := map[string]bencode.Bytes{
			"": fileBytes,
		}
		return bencode.Marshal(res)
	}
}

var _ bencode.Marshaler = (*FileTree)(nil)

func (ft *FileTree) NumEntries() (num int) {
	num = len(ft.Dir)
	if _, ok := ft.Dir[FileTreePropertiesKey]; ok {
		num--
	}
	return
}

func (ft *FileTree) IsDir() bool {
	return ft.NumEntries() != 0
}

func (ft *FileTree) orderedKeys() []string {
	return slices.Sorted(maps.Keys(ft.Dir))
}

func (ft *FileTree) upvertedFiles(pieceLength int64) iter.Seq[FileInfo] {
	var offset int64
	return ft.upvertedFilesInner(pieceLength, nil, &offset)
}

func (ft *FileTree) upvertedFilesInner(
	pieceLength int64,
	path []string,
	offset *int64,
) iter.Seq[FileInfo] {
	return func(yield func(FileInfo) bool) {
		if ft.IsDir() {
			for _, key := range ft.orderedKeys() {
				if key == FileTreePropertiesKey {
					continue
				}
				sub, ok := ft.Dir[key]
				if !ok {
					return
				}
				for fi := range sub.upvertedFilesInner(pieceLength, append(path, key), offset) {
					if !yield(fi) {
						return
					}
				}
			}
		} else {
			yield(FileInfo{
				Length: ft.File.Length,
				Path:   append([]string(nil), path...),
				// BEP 52 requires paths be UTF-8 if possible.
				PathUtf8:      append([]string(nil), path...),
				PiecesRoot:    ft.PiecesRootAsByteArray(),
				TorrentOffset: *offset,
			})
			// v2 files are piece aligned. This bumps up the offset to the next piece boundary.
			*offset += (ft.File.Length + pieceLength - 1) / pieceLength * pieceLength
		}
	}
}

func (ft *FileTree) Walk(path []string, f func(path []string, ft *FileTree)) {
	f(path, ft)
	for key, sub := range ft.Dir {
		if key == FileTreePropertiesKey {
			continue
		}
		sub.Walk(append(path, key), f)
	}
}

func (ft *FileTree) PiecesRootAsByteArray() (ret infohash_v2.T) {
	if ft.File.PiecesRoot == "" {
		return [32]byte{}
	}
	n := copy(ret[:], ft.File.PiecesRoot)
	if n != 32 {
		// Must be 32 bytes for meta version 2 and non-empty files. See BEP 52.
		return [32]byte{}
	}
	return
}
