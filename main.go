package main

import (
	"bytes"
	"crypto/md5"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"

	docopt "github.com/docopt/docopt-go"
)

var (
	usage = `cryptovim

Usage:
	cryptovim -s <file>
	cryptovim -r <file> [<recipient>]

Options:
	-s --symmetrical  use --symmetrical gpg flag for encription
	-r --recipient    use targeted gpg encription
	<recipient>       email for targeted encryption [default: git-email]
	<file>            encrypted file path to edit
`
)

func decrypt(encryptedPath string) (string, string, error) {
	tmpFile, err := ioutil.TempFile(os.TempDir(), "cryptovim")
	if err != nil {
		return "", "", fmt.Errorf("can't create tmp file: %s", err)
	}
	defer tmpFile.Close()

	_, err = os.Stat(encryptedPath)
	switch {
	case os.IsNotExist(err):
		log.Printf(
			"file %s not exist, creating new encrypted file",
			encryptedPath,
		)
		return tmpFile.Name(), "", nil
	case err == nil:
		break
	default:
		return "", "", fmt.Errorf("can't stat file %s: %s", encryptedPath, err)
	}

	cmd := exec.Command("gpg", "-d", encryptedPath)

	md5Sum := md5.New()
	writer := io.MultiWriter(tmpFile, md5Sum)

	cmd.Stdout = writer

	err = cmd.Run()
	if err != nil {
		return "", "", err
	}

	return tmpFile.Name(), string(md5Sum.Sum(nil)), nil
}

func rmDecrypted(path string) {
}

func editFile(path string) (string, error) {
	var (
		editor = os.Getenv("EDITOR")
		md5Sum = md5.New()
	)

	if editor == "" {
		editor = "vim"
	}

	cmd := exec.Command(editor, path)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	cmd.Stdin = os.Stdin

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("can't edif file %s: %s", path, err)
	}

	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("can't read edited file %s: %s", path, err)
	}
	defer file.Close()

	_, err = io.Copy(md5Sum, file)
	if err != nil {
		return "", fmt.Errorf("can't calculate md5 of %s: %s", path, err)
	}

	return string(md5Sum.Sum(nil)), nil
}

func encryptFile(decryptedPath string, args map[string]interface{}) error {
	var (
		symmetrical   = args["--symmetrical"].(bool)
		encryptedPath = args["<file>"].(string)
		cmdArgs       = []string{
			"--output",
			encryptedPath,
			"--batch",
			"--yes",
		}
	)

	switch {
	case symmetrical:
		cmdArgs = append(cmdArgs, []string{"-c", decryptedPath}...)
	default:
		realRecipient, err := makeRecipient(args["<recipient>"].(string))
		if err != nil {
			return fmt.Errorf("can't determine email to encrypt")
		}

		cmdArgs = append(cmdArgs, []string{
			"--encrypt", "--recipient", realRecipient, decryptedPath,
		}...)
	}

	cmd := exec.Command("gpg", cmdArgs...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf(
			"can't encrypt file %s into %s: %s",
			decryptedPath, encryptedPath, err,
		)
	}

	return nil
}

func makeRecipient(recipient string) (string, error) {
	if recipient != "git-email" {
		return recipient, nil
	}

	cmd := exec.Command("git", "config", "--get", "user.email")
	buff := bytes.NewBuffer(nil)
	cmd.Stdout = buff
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("can't get git email: %s", err)
	}

	return buff.String(), nil
}

func main() {
	args, err := docopt.Parse(usage, nil, true, "cryptovim 1.0", false, true)
	if err != nil {
		log.Fatalf("can't parse usage: %s", err)
	}

	encryptedPath := args["<file>"].(string)

	tmpPath, md5Sum, err := decrypt(encryptedPath)
	if err != nil {
		log.Fatalf("can't decrypt file %s: %s", encryptedPath, err)
	}
	defer rmDecrypted(tmpPath)

	editedMd5Sum, err := editFile(tmpPath)
	if err != nil {
		log.Printf("can't edit decrypted file: %s", err)
	}

	if editedMd5Sum == md5Sum {
		log.Println("file not changed, not encrypting")
		return
	}

	err = encryptFile(tmpPath, args)
}
