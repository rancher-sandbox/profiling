package testdata

import (
	"embed"
	"io/fs"
	"path"
)

//go:embed data

var TestDataFS embed.FS

func TestData(filename string) []byte {
	data, err := fs.ReadFile(TestDataFS, path.Join("data", filename))
	if err != nil {
		panic(err)
	}
	return data
}
