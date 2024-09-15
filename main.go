package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"zetsu/cli"
	"zetsu/global"
)

const RELEASECMD = "release"

func main() {
	if len(os.Args) == 1 {
		cli.RunRepl()
		return
	}

	if len(os.Args) == 2 {
		if os.Args[1] == "-h" || os.Args[1] == "--help" {
			fmt.Println("zetsu - an open source, secure by default programming language")
			fmt.Println()
			fmt.Println("USAGE: zetsu COMMAND [OPTIONS...]")

			fmt.Println("Options:")

			fmt.Println("\tzetsu")
			fmt.Println("\t\tRun zetsu in REPL mode.")
			fmt.Println()

			fmt.Println("\tzetsu -h, --help")
			fmt.Println("\t\tShow this help message.")
			fmt.Println()

			fmt.Println("\tzetsu -v, --version")
			fmt.Println("\t\tShow version information.")
			fmt.Println()

			fmt.Println("\tzetsu <FILENAME>.zeta")
			fmt.Println("\t\tCompile zetsu source code into zetsu bytecode.")
			fmt.Println()

			fmt.Println("\tzetsu <FILENAME>.ze")
			fmt.Println("\t\tRun zetsu bytecode using zetsu VM.")
			fmt.Println()

			fmt.Println("\tzetsu release -src <FILENAME>.mut [-os | -arch]")
			fmt.Println("\t\tCompile zetsu source code into standalone, independent binary executable.")
			fmt.Println("")
			fmt.Println("\t\tPossible values for -os: darwin | linux | windows.")
			fmt.Println("\t\tPossible values for -arch: amd64 | arm64 | arm | 386 | x86. (386 & x86 have same meaning here)")

			return
		}

		if os.Args[1] == "-v" || os.Args[1] == "--version" {
			fmt.Println("Version: 2.0.1")
			return
		}

		if strings.HasSuffix(os.Args[1], global.ZetsuSourceCodeFileExtention) {
			cli.CompileCode(os.Args[1], "", "", false)
			return
		}

		if strings.HasSuffix(os.Args[1], global.ZetsuByteCodeCompiledFileExtension) {
			cli.RunCode(os.Args[1])
			return
		}

	}

	if len(os.Args) >= 2 && os.Args[1] == RELEASECMD {
		src, goos, goarch, err := prepareRelease()
		if err != nil {
			fmt.Println(err)
			os.Exit(1)
		}

		fmt.Println("Compiling Release Build....")
		cli.CompileCode(src, goos, goarch, true)
		return
	}
}

func prepareRelease() (string, string, string, error) {
	var goos, goarch, src string

	releasecmd := flag.NewFlagSet(RELEASECMD, flag.ExitOnError)

	releasecmd.StringVar(&src, "src", "", "zetsu Source Code File Path by using -src flag")
	releasecmd.StringVar(&goos, "os", runtime.GOOS, "Use thie flag to specify target OS for cross-compilation by using -os flag")
	releasecmd.StringVar(&goarch, "arch", runtime.GOARCH, "Use thie flag to specify target Architecture for cross-compilation by using -arch flag")

	if err := releasecmd.Parse(os.Args[2:]); err != nil {
		return "", "", "", err
	}

	if releasecmd.Parsed() {
		if src == "" {
			return "", "", "", errors.New("zetsu source code file path is required, please use -src flag")
		}

		if !strings.HasSuffix(src, global.ZetsuSourceCodeFileExtention) {
			return "", "", "", errors.New("incorrect file extension, this program only works for zetsu source code files")
		}

		absSrc, err := filepath.Abs(src)
		if err != nil {
			return "", "", "", err
		}

		return absSrc, goos, goarch, nil
	}

	return "", "", "", errors.New("could not parse values")
}
