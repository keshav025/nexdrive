package main

import (
	"bytes"
	"fmt"
	"testing"
)

func TestCopyEncryptDecrypt(t *testing.T) {
	payload := "Foo not Bar"
	src := bytes.NewReader([]byte(payload))
	dst := new(bytes.Buffer)
	key := newEncryptionKey()
	_, err := copyEncrypt(key, src, dst)
	if err != nil {
		t.Error(err)
	}
	fmt.Println(dst.String())

	out := new(bytes.Buffer)
	if _, err := copyDecrypt(key, dst, out); err != nil {
		t.Error(err)
	}

	fmt.Println(out.String())
	if out.String() != payload {
		t.Error("decryption failied !!!")
	}
}
