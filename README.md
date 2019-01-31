# cryptoedit

cryptoedit is a tool to simplify editing gpg encrypted files

## usage

```
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
```

cryptoedit use your editor of choise from `$EDITOR` environment variable (or vim
if not set). It decrypts file to tmp file, open editor, and removes file after
editing. If file is not changed no re encryption is done. File would be encrypted
by rules that given via command line, not by rules used previously.
