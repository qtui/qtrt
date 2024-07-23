package qtrt

import (
	"fmt"
	"log"
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
	&TMCQtptr{}, &TMCToQStrref{}, &TMCToQobjptr{},
	&TMCint2long2{}, &TMCstr2charp{},
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
	// goty := tyo.String()
	if tyo.Kind() == reflect.Pointer {
		ety := tyo.Elem()
		log.Println(ety, ety.Name())
		if ety.Name()+"*" == tystr {
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
			}
			log.Println(tvx, d.Dbgstr())
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
				d.freefn = todoQStringDtor
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
		return true
	}
	return false
}

type TMCint2long2 struct{}

func (me *TMCint2long2) Match(d *TMCData, conv bool) bool {
	if d.gotyo.Kind() == reflect.Int &&
		(d.ctys == "long long" || d.ctys == "long") {
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
	// cthis := cgopp.Mallocgc(123)
	cthis := cgopp.Malloc(qtclzsz.Get("QString"))
	s4c := cgopp.CStringaf(s)
	cgopp.FfiCall[voidptr](sym, cthis, s4c, len(s))
	if cthis != nil {
		// runtime.SetFinalizer(cthis, todoQStringDtor2)
		time.AfterFunc(gopp.DurandSec(3, 3), func() { todoQStringDtor(cthis) })
	}
	return cthis
}

func todoQStringDtor(vx any) {
	name := "QStringDtor"
	sym := GetQtSymAddr(name)
	cgopp.FfiCall[int](sym, vx.(voidptr))
	// log.Println(name, vx)
}
