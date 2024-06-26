package main

import (
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

const defaultRootFolderName = "ggnetwork"

func CASPathTransformFunc(key string) PathKey {
	hash := sha1.Sum([]byte(key))
	hashStr := hex.EncodeToString(hash[:])

	blocksize := 5

	sliceLen := len(hashStr) / blocksize

	paths := make([]string, sliceLen)
	for i := 0; i < sliceLen; i++ {
		from, to := i*blocksize, i*blocksize+blocksize
		paths[i] = hashStr[from:to]
	}

	return PathKey{
		Pathname: strings.Join(paths, "/"),
		Filename: hashStr,
	}
}

var DefaultPathTransformFunc = func(key string) PathKey {
	return PathKey{
		Pathname: key,
		Filename: key,
	}
}

type PathTransformFunc func(string) PathKey

type PathKey struct {
	Pathname string
	Filename string
}

func (p PathKey) FirstPathname() string {
	paths := strings.Split(p.Pathname, "/")
	if len(paths) == 0 {
		return ""
	}
	return paths[0]
}

func (p PathKey) Fullpath() string {
	return fmt.Sprintf("%s/%s", p.Pathname, p.Filename)
}

type StoreOps struct {
	// root is the folder that contains name of root directory containing folders/files of the system
	Root              string
	PathTransformFunc PathTransformFunc
}

type Store struct {
	StoreOps
}

func NewStore(opts StoreOps) *Store {
	if opts.PathTransformFunc == nil {
		opts.PathTransformFunc = DefaultPathTransformFunc
	}

	if len(opts.Root) == 0 {
		opts.Root = defaultRootFolderName
	}
	return &Store{StoreOps: opts}
}

func (s *Store) Clear() error {
	return os.RemoveAll(s.Root)
}

func (s *Store) Has(key string) bool {
	pathKey := s.PathTransformFunc(key)
	fullpathWithRoot := fmt.Sprintf("%s/%s", s.Root, pathKey.Fullpath())
	_, err := os.Stat(fullpathWithRoot)
	return !errors.Is(err, os.ErrNotExist)
}

func (s *Store) Delete(key string) error {
	pathKey := s.PathTransformFunc(key)
	defer func() {

		fmt.Printf("deleted [%s] from disk", pathKey.Filename)

	}()
	firstPathnameWithRoot := fmt.Sprintf("%s/%s", s.Root, pathKey.FirstPathname())
	return os.RemoveAll(firstPathnameWithRoot)
}

func (s *Store) Read(key string) (int64, io.Reader, error) {
	return s.readStream(key)
	// n, f, err := s.readStream(key)
	// if err != nil {
	// 	return 0, nil, err
	// }
	// defer f.Close()

	// buf := new(bytes.Buffer)
	// _, err = io.Copy(buf, f)

	// return n, buf, err
}

func (s *Store) Write(key string, r io.Reader) (int64, error) {
	return s.writeStream(key, r)
}

func (s *Store) readStream(key string) (int64, io.ReadCloser, error) {
	pathkey := s.PathTransformFunc(key)

	pathAndFileName := pathkey.Fullpath()
	fullpathWithRoot := fmt.Sprintf("%s/%s", s.Root, pathAndFileName)
	// fi,err:=os.Stat(fullpathWithRoot)
	// if err!=nil{
	// 	return 0,nil,nil
	// }
	f, err := os.Open(fullpathWithRoot)
	if err != nil {
		return 0, nil, nil
	}
	fi, err := f.Stat()
	if err != nil {
		return 0, nil, nil
	}
	return fi.Size(), f, nil

}

func (s *Store) WriteDecrypt(enKey []byte, key string, r io.Reader) (int64, error) {
	f, err := s.openFileForWriting(key)
	if err != nil {
		return 0, err
	}
	n, err := copyDecrypt(enKey, r, f)

	if err != nil {
		return 0, err
	}

	return int64(n), nil

}

func (s *Store) openFileForWriting(key string) (*os.File, error) {
	pathname := s.PathTransformFunc(key)
	pathNameWithRoot := fmt.Sprintf("%s/%s", s.Root, pathname.Pathname)
	if err := os.MkdirAll(pathNameWithRoot, os.ModePerm); err != nil {
		return nil, err
	}

	fullPath := pathname.Fullpath()
	fullpathWithRoot := fmt.Sprintf("%s/%s", s.Root, fullPath)
	return os.Create(fullpathWithRoot)
}

func (s *Store) writeStream(key string, r io.Reader) (int64, error) {
	f, err := s.openFileForWriting(key)
	if err != nil {
		return 0, err
	}
	n, err := io.Copy(f, r)

	if err != nil {
		return 0, err
	}

	// fmt.Printf("written (%d) bytes to disk: %s\n", n, fullpathWithRoot)

	return n, nil

}
