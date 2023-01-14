package app

import (
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

var illegalFileNameChar = [...]string{"*", "\"", "<", ">", "?", "\\", "|", "/", ":"}

func StandardizeFileName(name string) string {
	newName := name
	for _, c := range illegalFileNameChar {
		newName = strings.Replace(newName, c, "_", -1)
	}
	return newName
}

func GetHttpProxy() string {
	envKeys := []string{"HTTPS_PROXY", "https_proxy", "HTTP_PROXY", "http_proxy"}
	var proxy string
	for _, key := range envKeys {
		if len(proxy) > 0 {
			break
		}
		proxy = os.Getenv(key)
	}
	return proxy
}
