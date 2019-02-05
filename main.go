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
	"strings"

	"github.com/docopt/docopt-go"
)

var (
	version = "1.0"
	usage   = `cryptoedit ` + version + `

Usage:
    cryptoedit [options] -s <file>
    cryptoedit [options] <file> [-r <recipient>...]
    cryptoedit --help
    cryptoedit --version

Options:
    -s --symmetrical  use --symmetrical gpg flag for encription
    -r <recipient>    use targeted gpg encription. If no one is specified,
                      git email would be used to encrypt just for yourself.
    -g <gpg>          gpg binary to use [default: gpg2]
    <file>            encrypted file path to edit
    -v --version      show version
    -h --help         show this screen
`
)

func decrypt(encryptedPath, gpgBinary string) (string, string, error) {
	tmpFile, err := ioutil.TempFile(os.TempDir(), "cryptoedit")
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

	cmd := exec.Command(gpgBinary, "-d", encryptedPath)

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
	os.Remove(path)
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
		gpgBinary     = args["-g"].(string)
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
		recipients, err := makeRecipients(args["-r"].([]string))
		if err != nil {
			return fmt.Errorf("can't determine email to encrypt")
		}

		cmdArgs = append(cmdArgs, "--encrypt")

		for _, recipient := range recipients {
			cmdArgs = append(cmdArgs, "--recipient", recipient)
		}

		cmdArgs = append(cmdArgs, decryptedPath)
	}

	cmd := exec.Command(gpgBinary, cmdArgs...)
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

func makeRecipients(recipients []string) ([]string, error) {
	if len(recipients) > 0 {
		return recipients, nil
	}

	cmd := exec.Command("git", "config", "--get", "user.email")
	buff := bytes.NewBuffer(nil)
	cmd.Stdout = buff
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		return nil, fmt.Errorf("can't get git email: %s", err)
	}

	email := strings.Trim(buff.String(), " \n")

	return []string{email}, nil
}

func main() {
	args, err := docopt.Parse(
		usage, nil, true, "cryptoedit "+version, false, true,
	)
	if err != nil {
		log.Fatalf("can't parse usage: %s", err)
	}

	encryptedPath := args["<file>"].(string)

	tmpPath, md5Sum, err := decrypt(encryptedPath, args["-g"].(string))
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
