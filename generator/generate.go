package generator

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"os"
	"zetsu/builtin"
	"zetsu/compiler"
	"zetsu/errrs"
	"zetsu/global"
	"zetsu/lexer"
	"zetsu/mutil"
	"zetsu/object"
	"zetsu/parser"
	"zetsu/security"
)

// Generate function takes a `string`, it's the path for the source code
func Generate(srcpath, dstpath, goos, goarch string, release bool) (error, errrs.ErrorType, []string) {
	data, err := os.ReadFile(srcpath)
	if err != nil {
		return err, errrs.ERROR, nil
	}

	bytecode, err, errtype, errors := compile(data)
	if err != nil {
		return err, errtype, errors
	}

	if release {

		if goos == WINDOWS {
			dstpath += global.WindowsPE32ExecutableExtension
		}

		if err := writeBinary(dstpath, goos, goarch, bytecode); err != nil {
			return err, errrs.ERROR, nil
		}

		return nil, "", nil
	}

	if err := os.WriteFile(dstpath+global.ZetsuByteCodeCompiledFileExtension, bytecode, 0644); err != nil {
		return err, errrs.ERROR, nil
	}

	return nil, "", nil
}

func compile(data []byte) ([]byte, error, errrs.ErrorType, []string) {
	constants := []object.Object{}
	symbolTable := compiler.NewSymbolTable()
	for i, v := range builtin.Builtins {
		symbolTable.DefineBuiltin(i, v.Name)
	}

	l := lexer.New(string(data))
	p := parser.New(l)
	program := p.ParseProgram()

	if len(p.Errors()) != 0 {
		return nil, fmt.Errorf("pareser error"), errrs.PARSER_ERROR, p.Errors()
	}

	comp := compiler.NewWithState(symbolTable, constants)
	if err := comp.Compile(program); err != nil {
		return nil, err, errrs.COMPILER_ERROR, nil
	}

	encodedByteCode, err := encode(comp.ByteCode())
	if err != nil {
		return nil, err, errrs.ERROR, nil
	}

	return encodedByteCode, nil, "", nil
}

func encode(compByteCode *compiler.ByteCode) ([]byte, error) {
	var content bytes.Buffer

	compByteCode = mutil.EncryptByteCode(compByteCode)

	registerTypes()
	enc := gob.NewEncoder(&content)
	if err := enc.Encode(compByteCode); err != nil {
		return nil, err
	}

	byteCode := content.Bytes()
	return encryptCode(byteCode)
}

func encryptCode(b64ByteCode []byte) ([]byte, error) {
	xorByteCode := security.XOR(b64ByteCode, len(b64ByteCode))
	encodedByteCode, err := security.AESEncrypt(xorByteCode)
	if err != nil {
		return nil, err
	}
	signedCode := security.SignCode(encodedByteCode)
	return signedCode, nil
}

func registerTypes() {
	gob.Register(&object.Integer{})
	gob.Register(&object.Boolean{})
	gob.Register(&object.Null{})
	gob.Register(&object.ReturnValue{})
	gob.Register(&object.Error{})
	gob.Register(&object.Function{})
	gob.Register(&object.String{})
	gob.Register(&builtin.BuiltIn{})
	gob.Register(&object.Array{})
	gob.Register(&object.Hash{})
	gob.Register(&object.Quote{})
	gob.Register(&object.Macro{})
	gob.Register(&object.CompiledFunction{})
	gob.Register(&object.Closure{})
	gob.Register(&object.Encrypted{})
}
