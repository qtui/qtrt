package qtrt

import (
	"fmt"
	"reflect"
	"strings"
	"time"

	// "github.com/kitech/minqt"

	"github.com/kitech/gopp"
	"github.com/kitech/gopp/cgopp"
	"github.com/qtui/qtclzsz"
)

type TypeMatcher interface {
	Match(d *TMCData, conv bool) bool
}

type TypeConver interface {
	Conv() any
}

type TMCData struct {
	idx    int
	ctys   string
	gotyo  reflect.Type
	goargx any

	// results
	ffiargx any
	freefn  func(any)

	// tmps
	// gotystr string
}

func (me *TMCData) Dbgstr() string {
	return fmt.Sprintf("idx: %v cty: %v goty: %v", me.idx, me.ctys, me.gotyo.String())
}

var typemcers = []TypeMatcher{
	&TMCEQ{}, &TMCTocxref{}, &TMCTocxCharpp{},
	&TMCQtptr{},
	&TMCToQStrview{}, //
	&TMCToQStrref{}, &TMCToQobjptr{},
	&TMCint2long2{}, &TMCstr2charp{}, &TMCf64toreal{},
	&TMCint2qflags{}, &TMCint2qenums{},
}

type TMCint2qenums struct{}

func (me *TMCint2qenums) Match(d *TMCData, conv bool) bool {
	if d.gotyo.Kind() == reflect.Int &&
		(strings.HasPrefix(d.ctys, "Qt::") ||
			// QSizePolicy::
			(strings.HasPrefix(d.ctys, "Q") && strings.Contains(d.ctys, "::"))) {
		if conv {

		}
		return true
	}
	return false
}

type TMCint2qflags struct{}

func (me *TMCint2qflags) Match(d *TMCData, conv bool) bool {
	if d.gotyo.Kind() == reflect.Int && strings.HasPrefix(d.ctys, "QFlags") {
		if conv {

		}
		return true
	}
	return false
}

type TMCf64toreal struct{}

func (me *TMCf64toreal) Match(d *TMCData, conv bool) bool {
	if d.gotyo.Kind() == reflect.Float64 && d.ctys == "double" {
		if conv {

		}
		return true
	}
	return false
}

// ///
type TMCEQ struct{}

func (me *TMCEQ) Match(d *TMCData, conv bool) bool {
	// log.Println(d.gotyo, d.ctys)
	if d.ctys == d.gotyo.String() {
		if conv {
			d.ffiargx = d.goargx
		}
		return true
	}
	return false
}

type TMCTocxref struct{}

// int => int&
func (me *TMCTocxref) Match(d *TMCData, conv bool) bool {
	if d.gotyo.String()+"&" == d.ctys {
		if conv {
			// 只对primitive type可以
			refval := reflect.New(d.gotyo)
			refval.Elem().Set(reflect.ValueOf(d.goargx))
			d.ffiargx = refval.Interface()
		}
		return true
	}
	return false
}

type TMCTocxCharpp struct{}

func (me *TMCTocxCharpp) Match(d *TMCData, conv bool) bool {
	if d.gotyo.String() == "[]string" && d.ctys == "char**" {
		if conv {
			// todo how freeit
			ptr := cgopp.CStrArrFromStrs(d.goargx.([]string))
			d.ffiargx = ptr
		}
		return true
	}
	return false
}

type TMCQtptr struct{}

func isqtptrtymat(tystr string, tyo reflect.Type) bool {
	// QObject* ?<= *main.QObject
	// QVariant const& ?<= *qtcore.QVariant
	// goty := tyo.String()
	if tyo.Kind() == reflect.Pointer {
		ety := tyo.Elem()
		etyname := gopp.LastofGv(strings.Split(ety.Name(), "."))
		// log.Println(ety, etyname, ety.Name())
		if etyname+"*" == tystr {
			return true
		} else if etyname+" const&" == tystr {
			return true
		}
	}

	return false
}
func (me *TMCQtptr) Match(d *TMCData, conv bool) bool {
	tyo := d.gotyo
	tystr := d.ctys
	argx := d.goargx

	if isqtptrtymat(tystr, tyo) {
		if conv {
			tvx := reflect.ValueOf(argx)
			if tvx.IsNil() {
			} else {
				// .Elem().FieldByName("Cthis")
				// log.Println(tyo, tvx, tvx.IsValid(), argx)
				// if obj, ok := argx.(*CObject); ok {
				// 	d.ffiargx = obj.GetCthis()
				// 	log.Println(d.ffiargx)
				// 	gopp.PauseAk()
				// }
			}
			// log.Println(tvx, tvx.IsNil(), d.Dbgstr())
		}
		return true
	}
	return false
}

type TMCToQStrview struct{}

type Fatptr64 struct {
	H int64
	L int64
}
type Fatptr32 struct {
	H int32
	L int32
}

// purego传递结构做，变长的类型不行
func Fatptrof[T any](ptr *T) any {
	var rv any
	if gopp.UintptrTySz == 4 {
		rv = *((*Fatptr32)(voidptr(ptr)))
	} else {
		rv = *((*Fatptr64)(voidptr(ptr)))
	}
	return rv
}

func (me *TMCToQStrview) Match(d *TMCData, conv bool) bool {
	// QAnyStringView ?<= string
	if strings.Contains(d.gotyo.String(), "Fatptr") && d.ctys == "QAnyStringView" {
		if conv {
			// arg := d.goargx.(string)
			// cv := Fatptrof(&arg)
			// d.ffiargx = cv
			// panic("todo")
		}
		return true
	}
	return false
}

type TMCToQStrref struct{}

func (me *TMCToQStrref) Match(d *TMCData, conv bool) bool {
	// QString const& ?<= string
	if "string" == d.gotyo.String() {
		if strings.HasPrefix(d.ctys, "QString ") && strings.HasSuffix(d.ctys, "&") {
			if conv {
				cthis := todoQStringNew(d.goargx.(string))
				d.ffiargx = cthis
				// d.freefn = todoQStringDtor
				// panic("todo")
				//goval := minqt.QStringNew(d.goargx.(string))
				// d.ffiargx = goval.Cthis
			}
			return true
		}
	}
	return false
}

type TMCToQobjptr struct{}

func (me *TMCToQobjptr) Match(d *TMCData, conv bool) bool {
	if (d.gotyo == nil || d.gotyo.Kind() == reflect.UnsafePointer ||
		d.gotyo.Kind() == reflect.Pointer) && strings.HasSuffix(d.ctys, "*") {
		if conv {
		}
		return true
	}
	return false
}

type TMCint2long2 struct{}

func (me *TMCint2long2) Match(d *TMCData, conv bool) bool {
	if d.gotyo.Kind() == reflect.Int &&
		(d.ctys == "long long" || d.ctys == "long" || d.ctys == "int") {
		return true
	}
	return false
}

type TMCstr2charp struct{}

func (me *TMCstr2charp) Match(d *TMCData, conv bool) bool {
	if d.gotyo.Kind() == reflect.String &&
		(d.ctys == "char const*" || d.ctys == "char *") {
		if conv {
			d.ffiargx = cgopp.CStringgc(d.goargx.(string))
		}
		return true
	}
	return false
}

// 用于传递参数
func todoQStringNew(s string) voidptr {
	// name := "__ZN7QStringC1EPKc" // 符号类型为t，dlsym找不到
	name := "__ZN7QString8fromUtf8EPKcx"
	sym := GetQtSymAddr(name)

	clzname := "QString"
	clzsz := qtclzsz.Get(clzname)
	clzsz2 := GetClassSizeByName2(clzname)
	gopp.TruePrint(clzsz2 <= 0, "wtf", clzsz2, clzname)
	gopp.FalsePrint(clzsz2 != clzsz, "ohwtf", clzsz, clzsz2)

	cthis := cgopp.Mallocpg(clzsz)
	s4c := cgopp.CStringaf(s)
	cgopp.FfiCall[voidptr](sym, cthis, s4c, len(s))

	// runtime.SetFinalizer(cthis, todoQStringDtor2)
	time.AfterFunc(gopp.DurandSec(3, 3), func() { todoQStringDtor(cthis) })

	return cthis
}

func todoQStringDtor(vx any) {
	name := "QStringDtor"
	sym := GetQtSymAddr(name)
	cgopp.FfiCall[int](sym, vx.(voidptr))
	// log.Println(name, vx)
}
