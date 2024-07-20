package qtrt

import (
	"fmt"
	"strings"

	"github.com/kitech/gopp"
)

// symbol info from qtlib, contains args type info,
// but lack some info like, return type, static, class size
var QtSymbols = map[string][]ccMethod{} // class name => struct type
var QtClassSizes = map[string]int{}     // todo

type ccMethod struct {
	// reflect.Method

	// 89M => 77M/79M
	// thin version of reflect.Method
	Name string // `json:"N"`
	// Index   int
	// PkgPath string

	CCSym string // `json:"S"` // 存储的去掉classname的部分
	// CCCrc  uint64

	// TODO how
	// Rety string // `json:"R"`
	// Size int    // `json:"Z"`
	St int // `json:"T"` //bool
}

//	func (me ccMethod) Symbol(clz string) string {
//		return qtmthsymrestore(clz, me.Name, me.Sym)
//	}
func (me ccMethod) Static() bool { return me.St == 1 }

// \see https://web.mit.edu/tibbetts/Public/inside-c/www/mangling.html
func (me ccMethod) Const() bool { return strings.HasPrefix(me.CCSym, "_ZNK") }

var Symdedups = map[uint64]int{} // sym crc =>
var Symdedupedcnt = 0

func Addsymrawline(qtmodname string, line string) {
	flds := strings.Split(line, " ")
	// log.Println(line, flds)
	sym := gopp.LastofGv(flds)
	// log.Println(sym)
	addqtsym(qtmodname, sym)
}
func addqtsym(qtmodname, symname string) {
	// log.Println("demangle...", len(symname), symname)
	sgnt, ok := Demangle(symname)
	if strings.HasPrefix(symname, "GCC_except") {
	} else if strings.HasPrefix(symname, "_OBJC_") {
	} else if strings.Contains(symname, "QtPrivate") {
	} else {
		// gopp.FalsePrint(ok, "demangle failed", symname)
	}
	// log.Println(ok, len(symname), "=>", len(sgnt), sgnt, ok)
	if !ok {
		return
	}
	if strings.HasPrefix(sgnt, "typeinfo") {
		return
	}
	if strings.HasPrefix(sgnt, "vtable") {
		return
	}
	if strings.HasPrefix(sgnt, "operator") {
		return
	}
	if strings.Contains(sgnt, "operator+=") {
		return
	}
	if strings.Contains(sgnt, "anonymous namespace") {
		return
	}

	clzname, mthname := SplitMethod(sgnt)
	if clzname == "" || mthname == "" {
		if clzname == "" && mthname != "" {
			// maybe global function
		} else {
			gopp.Warn("somerr", clzname, mthname, sgnt)
		}
		return
	}
	// log.Println(clzname, mthname, sgnt)
	if clzname == "$_0" || clzname == "$_5" {
		// log.Println("wtf", qtmodname, symname)
		return
	}
	if clzname[0] != 'Q' {
		// log.Println("wtf", qtmodname, clzname, mthname, symname)
		return
	}

	symcrc := gopp.Crc64Str(symname)
	if _, ok := Symdedups[symcrc]; ok {
		// log.Println("already have", sgnt, len(dedups))
		Symdedupedcnt++
		return
	}
	Symdedups[symcrc] = 1

	mtho := ccMethod{}
	mtho.CCSym = symname
	// mtho.Sym = qtmthsymshorten(clzname, mthname, symname)
	// mtho.CCCrc = symcrc
	mtho.Name = strings.Title(mthname)
	// mtho.Index = len(mths)
	// mtho.PkgPath = qtmodname

	mths, ok := QtSymbols[clzname]
	mths = append(mths, mtho)
	QtSymbols[clzname] = mths
}

func SplitMethod(s string) (string, string) {
	idx := strings.LastIndexAny(s, " )")
	if idx != -1 {
		s = s[:idx]
	}

	flds := strings.Split(s, "::")
	for i, fld := range flds {
		idx := strings.Index(fld, "(")
		if idx != -1 {
			flds[i] = fld[:idx]
		}
	}
	if len(flds) < 2 {
		return "", flds[0]
	}
	return flds[0], flds[1]
}

func SplitArgs(s string) (rets []string) {
	pos1 := strings.Index(s, "(")
	pos2 := strings.LastIndex(s, ")")

	mid := s[pos1+1 : pos2]
	// gopp.ZeroPrint(mid, "Empty args?", s)
	if mid == "" {
		return
	}

	rets = strings.Split(mid, ", ")

	return
}

// 去掉前缀 __ZN
// 替换类名为.
// 替换方法名为,
func qtmthsymshorten(clz string, mth string, symname string) string {
	ct := fmt.Sprintf("%d%s", len(clz), clz)
	mt := fmt.Sprintf("%d%s", len(mth), mth)
	rv := strings.Replace(symname[4:], ct, ".", 1)
	rv = strings.Replace(rv, mt, ",", 1)
	return rv
}
func qtmthsymrestore(clz string, mth string, symname string) string {
	ct := fmt.Sprintf("%d%s", len(clz), clz)
	mt := fmt.Sprintf("%d%s", len(mth), mth)
	rv := strings.Replace(symname, ".", ct, 1)
	rv = strings.Replace(rv, ",", mt, 1)
	return "__ZN" + rv
}
