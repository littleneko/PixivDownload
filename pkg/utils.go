package pkg

import (
	"os"
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