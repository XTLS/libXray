package nodep

import (
	"os"
)

func WriteBytes(bytes []byte, path string) error {
	fi, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0664)
	if err != nil {
		return err
	}
	defer fi.Close()

	_, err = fi.Write(bytes)
	if err != nil {
		return err
	}
	return nil
}

func WriteText(text string, path string) error {
	fi, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0664)
	if err != nil {
		return err
	}
	defer fi.Close()

	_, err = fi.WriteString(text)
	if err != nil {
		return err
	}
	return nil
}
