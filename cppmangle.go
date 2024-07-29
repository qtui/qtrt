package qtrt

import (
	"log"
	"reflect"
	"strings"

	"github.com/kitech/gopp"
)

/*
mangle rule:
int => i
*/
// todo todo tod
func Cppmangle(clz, mth string, args ...any) (rets []string) {
	var isconst, isctor, isdtor bool
	isctor = clz == mth
	isdtor = "~"+clz == mth || mth == "Dtor"

	var sb strings.Builder
	sb.WriteString("_ZN")
	sb.WriteString(gopp.IfElse2(isconst, "K", ""))
	sb.WriteString(gopp.ToStr(len(clz)))
	sb.WriteString(clz)
	if isctor {
		sb.WriteString("C2") // todo how C1
	} else if isdtor {
		sb.WriteString("D2") // // todo how D1
	} else {
	}
	sb.WriteString(gopp.IfElse2(len(rets) == 0, "Ev", "_"))

	argtys := reflecttypes(args...)
	for i, argty := range argtys {
		switch argty.Kind() {
		case reflect.Int:
			sb.WriteRune('i')
		// case reflect.String:
		// todo how QString|char*|constchar*
		case reflect.Float64:
			sb.WriteRune('d') // MSVC D
		case reflect.Float32:
			sb.WriteRune('f') // MSVC F
		case reflect.Int16:
			sb.WriteRune('s') // MSVC Gs
		case reflect.Int8:
			sb.WriteRune('c')

		// case reflect.Pointer:
		// how *A|*B
		// case reflect.UnsafePointer:
		// case reflect.Struct:

		default:
			log.Println("notimpl", i, argty.String(), clz, mth, len(args), args)
		}
	}

	log.Println(len(rets), rets, len(rets))
	return
}
