package app

import (
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
)

func Retry(workFunc func() error, cnt int) error {
	var err error
	for i := 0; i < cnt; i++ {
		err = workFunc()
		if err == nil {
			break
		}
		if i+1 < cnt {
			time.Sleep(1 * time.Second)
		}
	}
	return err
}

func CheckAndMkdir(path string) error {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return os.MkdirAll(path, 0755)
	}
	return nil
}

var illegalFileNameChar = [...]string{"*", "\"", "<", ">", "?", "\\", "|", "/", ":", " "}

func StandardizeFileName(name string) string {
	newName := name
	for _, c := range illegalFileNameChar {
		newName = strings.Replace(newName, c, "_", -1)
	}
	return newName
}

func FileSha1Sum(filename string) (string, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = file.Close()
	}()

	h := sha1.New()
	if _, err := io.Copy(h, file); err != nil {
		return "", nil
	}

	sum := fmt.Sprintf("%x", h.Sum(nil))
	return sum, nil
}
