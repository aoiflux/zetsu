package exfat

import (
	"fmt"
	"os"
	"path/filepath"
	"zetsu/object"

	"github.com/aoiflux/libxfat"
)

func GetFiles(args ...object.Object) object.Object {
	if len(args) > 2 || len(args) < 1 {
		return newError("wrong number of arguments. got=%d, want=1 or 2", len(args))
	}
	if args[0].Type() != object.STRING_OBJ {
		return newError("first argument to `GetFiles` must be STRING, got=%s", args[0].Type())
	}

	arg := args[0].(*object.String)
	absPath, err := filepath.Abs(arg.Value)
	if err != nil {
		return newError("unable to get absolute path: %v", err.Error())
	}

	_, err = os.Stat(absPath)
	if os.IsNotExist(err) {
		return newError("file not found, please ensure file path is correct")
	}
	if err != nil {
		return newError("error during stat: %v", err)
	}
	fh, err := os.Open(arg.Value)
	if err != nil {
		return newError("unable to read file. got error: %s", err.Error())
	}
	defer fh.Close()

	offset := int64(0)
	if len(args) == 2 {
		if args[1].Type() != object.INTEGER_OBJ {
			return newError("second argument to `GetFiles` must be INTEGER, got=%s", args[0].Type())
		}

		arg2 := args[1].(*object.Integer)
		offset = arg2.Value
	}

	files, err := getFiles(fh, uint64(offset))
	if err != nil {
		return newError("error: %v", err.Error())
	}

	return files
}

func getFiles(fh *os.File, offset uint64) (object.Object, error) {
	xfatObj, err := libxfat.New(fh, true, offset)
	if err != nil {
		return nil, err
	}

	roots, err := xfatObj.ReadRootDir()
	if err != nil {
		return nil, err
	}

	entries, err := xfatObj.GetAllEntries(roots)
	if err != nil {
		return nil, err
	}

	files := &object.Array{}
	for _, entry := range entries {
		pairs := make(map[object.HashKey]object.HashPair)

		pair := getPairStr("name", entry.GetName())
		hkey := getKey("name")
		pairs[hkey] = pair

		pair = getPairInt("size", int64(entry.GetSize()))
		hkey = getKey("size")
		pairs[hkey] = pair

		pair = getPairBool("deleted", entry.IsDeleted())
		hkey = getKey("deleted")
		pairs[hkey] = pair

		pair = getPairBool("indexed", entry.IsIndexed())
		hkey = getKey("indexed")
		pairs[hkey] = pair

		pair = getPairBool("fragmented", entry.HasFatChain())
		hkey = getKey("fragmented")
		pairs[hkey] = pair

		hashObj := &object.Hash{Pairs: pairs}
		files.Elements = append(files.Elements, hashObj)
	}

	return files, nil
}

func getPairStr(key, value string) object.HashPair {
	var pair object.HashPair
	nameKey := &object.String{Value: key}
	pair.Key = nameKey
	pair.Value = &object.String{Value: value}
	return pair
}

func getPairInt(key string, value int64) object.HashPair {
	var pair object.HashPair
	nameKey := &object.String{Value: key}
	pair.Key = nameKey
	pair.Value = &object.Integer{Value: value}
	return pair
}

func getPairBool(key string, value bool) object.HashPair {
	var pair object.HashPair
	nameKey := &object.String{Value: key}
	pair.Key = nameKey
	pair.Value = &object.Boolean{Value: value}
	return pair
}

func getKey(key string) object.HashKey {
	str := &object.String{Value: key}
	return str.HashKey()
}

func newError(format string, a ...interface{}) *object.Error {
	return &object.Error{Message: fmt.Sprintf(format, a...)}
}
