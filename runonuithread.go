package qtrt

import (
	"fmt"
	"sync/atomic"

	"github.com/ebitengine/purego"
	"github.com/kitech/gopp"
	"github.com/kitech/gopp/cgopp"
	cmap "github.com/orcaman/concurrent-map/v2"
)

/*
//
*/
import "C"

// todo
type seqfnpair struct {
	np *int64
	f  func()
}

var runuithfns = cmap.New[seqfnpair]()
var runuithseq int64 = 10000

//export qtuithcbfningo
func qtuithcbfningo(n *int64) {
	key := fmt.Sprintf("%d", *n)
	// log.Println(*n, key)
	pair, ok := runuithfns.Get(key)
	if ok {
		pair.f()
		runuithfns.Remove(key)
	}
}

func RunonUithreadfn(f func()) func() {
	return func() { RunonUithread(f) }
}
func RunonUithread(f func()) {

	const name = "QMetaObjectInvokeMethod1"
	sym := dlsym(name)
	sym2 := dlsym("qtuithcbfningo")
	// log.Println(sym, name, sym2)

	seq := new(int64)
	*seq = atomic.AddInt64(&runuithseq, 3)

	key := fmt.Sprintf("%d", *seq)
	runuithfns.Set(key, seqfnpair{seq, f})

	cgopp.Litfficallg(sym, sym2, seq)
}
func RunonUithreadx(fx any, args ...any) {
	RunonUithread(func() { gopp.CallFuncx(fx, args...) })
}

// 这个函数很快，50ns
// current process
func dlsym(name string) voidptr {
	// if sym, ok := symcache.Get(name); ok {
	// 	return sym
	// }
	symi, err := purego.Dlsym(purego.RTLD_DEFAULT, name)
	gopp.ErrPrint(err, name)
	if gopp.ErrHave(err, "symbol not found") {
		// sym, err = purego.Dlsym(purego.RTLD_DEFAULT, "_"+name)
	}
	sym := voidptr(symi)
	// if sym != nil {
	// 	symcache.Set(name, sym)
	// }
	return sym
}
func Dlsym0(name string) voidptr { return dlsym(name) }
