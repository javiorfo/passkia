package passc

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"strings"

	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"

	"os"
)

type Encryptor struct {
	MasterPassword string
	FilePath       string
}

var emptyFile = errors.New(passcEmptyFile)

func (e Encryptor) deleteContent() error {
	file, err := os.OpenFile(e.FilePath, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()
	return nil
}

func (e Encryptor) encryptText(text string, isAppend bool) error {
	flag := os.O_CREATE | os.O_WRONLY
	if !isAppend {
		flag = os.O_WRONLY | os.O_TRUNC | os.O_CREATE
	}
	file, err := os.OpenFile(e.FilePath, flag, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	finalText := text
	stat, _ := file.Stat()
	if stat.Size() != 0 {
		if isAppend {
			decryptedText, err := e.readEncryptedText()
			if err != nil {
				return err
			}
			finalText += passcItemSeparator + decryptedText
		}
	}

	block, err := aes.NewCipher([]byte(e.MasterPassword))
	if err != nil {
		return err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return err
	}

	ciphertext := gcm.Seal(nil, nonce, []byte(finalText), nil)

	if _, err := file.Write(append(nonce, ciphertext...)); err != nil {
		return err
	}

	return nil
}

func (e Encryptor) readEncryptedText() (string, error) {
	file, err := os.Open(e.FilePath)
	if err != nil {
		return "", fmt.Errorf("open file %s: %v", e.FilePath, err)
	}
	defer file.Close()

	block, err := aes.NewCipher([]byte(e.MasterPassword))
	if err != nil {
		return "", fmt.Errorf("cipher: %v", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("gcm: %v", err)
	}

	data, err := io.ReadAll(file)
	if err != nil {
		return "", fmt.Errorf("reading file: %v", err)
	}

	nonce := data[:gcm.NonceSize()]
	if len(data) == 0 {
		return "", emptyFile
	}
	ciphertext := data[gcm.NonceSize():]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("extracting plaintext: %v", err)
	}

	return string(plaintext), nil
}

func exportToFile(content string) error {
	items := strings.Split(content, passcItemSeparator)
	file, err := os.OpenFile(passcExportFilename, os.O_WRONLY|os.O_TRUNC|os.O_CREATE, 0644)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.WriteString("[")
	if err != nil {
		return err
	}

	length := len(items)
	for i, v := range items {
		var prettyJSON bytes.Buffer
		err := json.Indent(&prettyJSON, []byte(v), "", "  ")
		if err != nil {
			return err
		}

		coma := ","
		if i == length-1 {
			coma = ""
		}

		_, err = file.WriteString(prettyJSON.String() + coma)
		if err != nil {
			return err
		}
	}
	_, err = file.WriteString("]")
	if err != nil {
		return err
	}
	return nil
}

func makeBackUp() {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("finding user home dir: %v\n", err)
		return
	}
	src := homeDir + "/" + passcPathStoreFile
	srcFile, err := os.Open(src)
	if err != nil {
		log.Printf("opening file: %v\n", err)
		return
	}
	defer srcFile.Close()

	dst := homeDir + "/" + passcDirFolder + "/" + passcBackUpFile
	dstFile, err := os.Create(dst)
	if err != nil {
		log.Printf("creating file: %v\n", err)
		return
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	if err != nil {
		log.Printf("copying to backup file: %v\n", err)
		return
	}
}
