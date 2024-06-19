package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"testing"
)

func newStore() *Store {
	opts := StoreOps{
		PathTransformFunc: CASPathTransformFunc,
	}
	return NewStore(opts)
}

func teardown(t testing.T, s *Store) {
	if err := s.Clear(); err != nil {
		t.Error(err)
	}
}

func TestPathTransformFunc(t *testing.T) {
	key := "momsbestpicture"
	pathname := CASPathTransformFunc(key)
	fmt.Println(pathname)
	expectedFilename := "6804429f74181a63c50c3d81d733a12f14a353ff"
	expectedPathname := "68044/29f74/181a6/3c50c/3d81d/733a1/2f14a/353ff"

	if pathname.Pathname != expectedPathname {
		t.Errorf("have %s want %s", pathname.Pathname, expectedPathname)
	}

	if pathname.Filename != expectedFilename {
		t.Errorf("have %s want %s", pathname.Filename, expectedFilename)
	}
}

func TestStoreDeleteKey(t *testing.T) {
	opts := StoreOps{
		PathTransformFunc: CASPathTransformFunc,
	}
	s := NewStore(opts)
	key := "momsbestpicture"
	data := []byte("some jpg bytes")
	dataReader := bytes.NewReader(data)
	if _, err := s.writeStream(key, dataReader); err != nil {
		t.Error(err)
	}

	err := s.Delete(key)
	if err != nil {
		t.Error(err)
	}

}

func TestStore(t *testing.T) {
	opts := StoreOps{
		PathTransformFunc: CASPathTransformFunc,
	}
	s := NewStore(opts)
	key := "momsbestpicture"
	data := []byte("some jpg bytes")
	dataReader := bytes.NewReader([]byte("some jpg bytes"))
	if _, err := s.writeStream(key, dataReader); err != nil {
		t.Error(err)
	}
	if ok := s.Has(key); !ok {
		t.Errorf("expected to have key : %s", key)
	}

	_, r, err := s.Read(key)
	if err != nil {
		t.Error(err)
		return
	}

	b, _ := ioutil.ReadAll(r)

	if string(b) != string(data) {
		t.Errorf("want %s have %s", data, b)
	}
	s.Delete(key)
}
