package repl

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"runtime"
	"zetsu/builtin"
	"zetsu/compiler"
	"zetsu/errrs"
	"zetsu/global"
	"zetsu/lexer"
	"zetsu/mutil"
	"zetsu/object"
	"zetsu/parser"
	"zetsu/vm"
)

const banner = `
==========================================
▒███████▒▓█████▄▄▄█████▓  ██████  █    ██
▒ ▒ ▒ ▄▀░▓█   ▀▓  ██▒ ▓▒▒██    ▒  ██  ▓██▒
░ ▒ ▄▀▒░ ▒███  ▒ ▓██░ ▒░░ ▓██▄   ▓██  ▒██░
  ▄▀▒   ░▒▓█  ▄░ ▓██▓ ░   ▒   ██▒▓▓█  ░██░
▒███████▒░▒████▒ ▒██▒ ░ ▒██████▒▒▒▒█████▓
░▒▒ ▓░▒░▒░░ ▒░ ░ ▒ ░░   ▒ ▒▓▒ ▒ ░░▒▓▒ ▒ ▒
░░▒ ▒ ░ ▒ ░ ░  ░   ░    ░ ░▒  ░ ░░░▒░ ░ ░
░ ░ ░ ░ ░   ░    ░      ░  ░  ░   ░░░ ░ ░
  ░ ░       ░  ░              ░     ░
░
==========================================
`

// PROMPT is the constant for showing REPL prompt
const PROMPT = ">> "

// Start function is the entrypoint of our repl
func Start(in io.Reader, out io.Writer) {
	welcome()
	scanner := bufio.NewScanner(in)
	// env := object.NewEnvironment()
	// macroEnv := object.NewEnvironment()

	constants := []object.Object{}
	globals := make([]object.Object, global.GlobalSize)
	symbolTable := compiler.NewSymbolTable()
	for i, v := range builtin.Builtins {
		symbolTable.DefineBuiltin(i, v.Name)
	}

	for {
		fmt.Printf("\n\n%s", PROMPT)
		scanned := scanner.Scan()
		if !scanned {
			return
		}

		line := scanner.Text()
		if vanity(line, out) {
			continue
		}

		l := lexer.New(line)
		p := parser.New(l)
		program := p.ParseProgram()

		if len(p.Errors()) != 0 {
			errrs.PrintParseErrors(out, p.Errors())
			continue
		}

		comp := compiler.NewWithState(symbolTable, constants)
		if err := comp.Compile(program); err != nil {
			errrs.PrintCompilerError(out, err.Error())
			continue
		}

		byteCode := comp.ByteCode()
		byteCode = mutil.EncryptByteCode(byteCode)

		machine := vm.NewWithGlobalStore(byteCode, globals)
		if err := machine.Run(); err != nil {
			errrs.PrintMachineError(out, err.Error())
			continue
		}

		last := machine.LastPoppedStackElement()
		io.WriteString(out, last.Inspect())
		io.WriteString(out, "\n")
	}
}

func welcome() {
	fmt.Print(banner)

	user, err := user.Current()
	if err != nil {
		panic(err)
	}
	fmt.Printf("Hello %s! Let's investigate with zetsu\n", user.Name)
	fmt.Printf("Please get started by using this REPL")
}

func vanity(line string, out io.Writer) bool {
	if line == "" {
		return true
	}

	if line == "clear" || line == "cls" {
		clear := make(map[string]func())
		clear["linux"] = func() {
			cmd := exec.Command("clear")
			cmd.Stdout = os.Stdout
			cmd.Run()
		}
		clear["darwin"] = func() {
			cmd := exec.Command("clear")
			cmd.Stdout = os.Stdout
			cmd.Run()
		}
		clear["windows"] = func() {
			cmd := exec.Command("cmd", "/c", "cls")
			cmd.Stdout = os.Stdout
			cmd.Run()
		}

		if value, ok := clear[runtime.GOOS]; ok {
			value()
		} else {
			io.WriteString(out, "\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n\n")
		}
		return true
	}

	if line == "exit" {
		GracefulExit()
	}

	return false
}

func GracefulExit() {
	fmt.Printf("\n\n")
	fmt.Println("---- Leaving for a byte? I'll see you later! ----")
	fmt.Printf("\n\n")
	os.Exit(0)
}
