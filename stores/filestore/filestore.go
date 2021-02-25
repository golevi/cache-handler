package filestore

import (
	"crypto/sha256"
	"encoding/gob"
	"encoding/hex"
	"log"
	"os"
	"strings"
	"time"

	"github.com/caddyserver/caddy/v2"
)

// FileStore uses the local filesystem.
type FileStore struct {
	Level int
	Dir   string
}

type fileCache struct {
	Data    []byte
	Created time.Time
	Expires time.Duration
}

// NewFileStore uses `caddy.AppDataDir()` to get a location to store the cached
// files.
func NewFileStore() FileStore {
	f := FileStore{
		Level: 2,
		Dir:   caddy.AppDataDir() + "/cache",
	}

	return f
}

// Get value from file.
func (f FileStore) Get(key string) (interface{}, error) {
	key = hashKey(key)

	path := f.path(key)
	filepath := path + key
	log.Println(filepath)

	file, err := os.Open(filepath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	fc := fileCache{}
	dec := gob.NewDecoder(file)
	err = dec.Decode(&fc)
	if err != nil {
		return nil, err
	}

	if time.Now().Sub(fc.Created) > fc.Expires {
		os.Remove(filepath)
	}

	return fc.Data, nil
}

// Has checks if the key exists.
func (f FileStore) Has(key string) bool {
	key = hashKey(key)
	path := f.path(key)
	filepath := path + key

	if _, err := os.Stat(filepath); os.IsNotExist(err) {
		return false
	}

	return true
}

// Put the value in a file.
func (f FileStore) Put(key string, value interface{}, expiration time.Duration) {
	key = hashKey(key)

	fc := fileCache{
		Data:    value.([]byte),
		Created: time.Now(),
		Expires: expiration,
	}

	path := f.path(key)

	if _, err := os.Stat(path); os.IsNotExist(err) {
		os.MkdirAll(path, 0700)
	}

	filepath := path + key
	file, _ := os.Create(filepath)

	defer file.Close()

	enc := gob.NewEncoder(file)
	enc.Encode(fc)
}

func (f FileStore) path(key string) string {
	s := strings.Split(key, "")
	folders := ""
	for i, d := range s {
		folders += d + "/"
		if i >= f.Level-1 {
			break
		}
	}
	return f.Dir + "/" + folders
}

func hashKey(input string) string {
	h := sha256.New()
	h.Write([]byte(input))
	o := h.Sum(nil)

	return hex.EncodeToString(o)
}
