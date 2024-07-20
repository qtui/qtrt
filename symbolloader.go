package qtrt

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/kitech/gopp"
)

var qtsymbolsloaded = false

// var qtsymbolsraw []string

// TODO 这个非常耗时
// 返回匹配的值
func LoadAllQtSymbols(stub string) []string {
	log.Println(qtlibpaths)
	if qtsymbolsloaded {
		log.Println("Already loaded???", len(Classes))
		return nil
	}
	qtsymbolsloaded = true

	// loadcacheok := loadsymbolsjson()
	loadcacheok := loadsymbolsgob()

	if loadcacheok {
		return nil
	} else {
		rets := implLoadAllQtSymbols(stub)
		savesymbolsjson()
		savesymbolsgob()
		return rets
	}
}

func implLoadAllQtSymbols(stub string) []string {
	// log.Println(qtlibpaths)
	var nowt = time.Now()

	libpfx := gopp.Mustify(os.UserHomeDir())[0].Str() + "/.nix-profile/lib"
	globtmpl := fmt.Sprintf("%s/Qt*.framework/Qt*", libpfx)
	libs, err := filepath.Glob(globtmpl)
	gopp.ErrPrint(err, libs)
	libnames := gopp.Mapdo(libs, func(vx any) any {
		return filepath.Base(vx.(string))
	})
	log.Println(gopp.FirstofGv(libs), libnames, len(libs))
	log.Println("Maybe use about little secs...")
	signtx := gopp.Mapdo(libs, func(idx int, vx any) (rets []any) {
		// log.Println(idx, vx, gopp.Bytes2Humz(gopp.FileSize(vx.(string))))
		tmpfile := "symfiles/" + filepath.Base(vx.(string)) + ".sym"
		var lines []string
		if !gopp.FileExist2(tmpfile) {
			lines, err := gopp.RunCmd(".", "nm", vx.(string))
			gopp.ErrPrint(err, vx)
			log.Println(idx, vx, len(lines))
			// save cache
			gopp.SafeWriteFile(tmpfile, []byte(strings.Join(lines, "\n")), 0644)
		} else {
			bcc, err := os.ReadFile(tmpfile)
			gopp.ErrPrint(err, tmpfile)
			lines = strings.Split(string(bcc), "\n")
		}
		for _, line := range lines {
			if strings.Contains(line, "Private") {
				continue
			}

			if strings.Contains(line, stub) {
				// log.Println(line)
				name := gopp.Lastof(strings.Split(line, " ")).Str()
				signt, ok := Demangle(name)
				log.Println(name, ok, signt)
				rets = append(rets, name, signt)
			}
			Addsymrawline(filepath.Base(vx.(string)), line)
		}
		return
	})
	log.Println(gopp.Lenof(signtx), len(Classes), time.Since(nowt)) // about 1.1s
	signts := gopp.IV2Strings(signtx.([]any))

	// qtsymbolsraw = signts
	return signts
}

func savesymbolsjson() {
	nowt := time.Now()
	bcc, err := json.Marshal(Classes)
	gopp.ErrPrint(err)
	gopp.SafeWriteFile("QtClasses.json", bcc, 0644)
	bcc = nil
	// jsonenc 106.696382ms
	log.Println("jsonenc", time.Since(nowt))
}
func loadsymbolsjson() bool {
	if !gopp.FileExist2("QtClasses.json") {
		return false
	}
	nowt := time.Now()
	bcc, err := os.ReadFile("QtClasses.json")
	gopp.ErrPrint(err)
	Classes = nil
	err = json.Unmarshal(bcc, &Classes)
	gopp.ErrPrint(err)
	log.Println("decode big json", time.Since(nowt)) // about 400ms
	bcc = nil

	return err == nil
}

func savesymbolsgob() {
	nowt := time.Now()
	var buf = bytes.NewBuffer(nil)
	enco := gob.NewEncoder(buf)
	err := enco.Encode(Classes)
	gopp.ErrPrint(err)
	gopp.SafeWriteFile("QtClasses.gob", buf.Bytes(), 0644)
	// gobenc 75.741979ms
	log.Println("gobenc", time.Since(nowt))
}
func loadsymbolsgob() bool {
	if !gopp.FileExist2("QtClasses.gob") {
		return false
	}

	nowt := time.Now()
	bcc, err := os.ReadFile("QtClasses.gob")
	gopp.ErrPrint(err)
	buf := bytes.NewReader(bcc)
	deco := gob.NewDecoder(buf)
	err = deco.Decode(&Classes)
	gopp.ErrPrint(err)
	log.Println("gobdec", time.Since(nowt))

	return err == nil
}
