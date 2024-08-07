package qtrt

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"unsafe"

	"github.com/ebitengine/purego"
	"github.com/kitech/gopp"
	"github.com/kitech/gopp/cgopp"
)

func InvokeQtFuncByName(symname string, args []uint64, types []int) uint64 {
	return 0
}

// //////
type VRetype = uint64 // interface{}
type FRetype struct {
	H uint64
	L uint64
}

var debugFFICall = false

func init() {
	dbgval := os.Getenv("QTGO_DEBUG_FFI_CALL")
	if strings.ToLower(dbgval) == "true" || dbgval == "1" {
		debugFFICall = true
	}
}

func SetDebugFFICall(on bool) { debugFFICall = on }
func init() {
	log.SetFlags(log.Flags() | log.Lshortfile)
	// isLinkedQtlib = check_linked_qtmod()
	init_ffi_invoke()
	init_so_ffi_call()

	// TODO maybe run when first qtcall
	init_destroyedDynSlot()
	init_callack_inherit() //
}

func init_ffi_invoke() {
	if true {
		loadAllModules()
		log.Println("Loaded", len(qtlibs), "of", len(mainqtmods), gopp.MapKeys(qtlibs))
		gopp.ZeroPrint(len(qtlibs), "Load 0 qtmodule, want", len(mainqtmods), mainqtmods)
		return
	}

	// lib dir prefix
	// go arch name => android lib name
	archs := map[string]string{"386": "x86", "amd64": "x86_64", "arm": "arm", "mips": "mips"}
	oslibexts := map[string]string{"linux": "so", "darwin": "dylib", "windows": "dll"}

	getLibDirp := func() string {
		switch runtime.GOOS {
		case "android":
			bcc, err := ioutil.ReadFile(fmt.Sprintf("/proc/%d/cmdline", os.Getpid()))
			ErrPrint(err)
			appdir := string(bcc[:bytes.IndexByte(bcc, 0)])
			sepos := strings.Index(appdir, ":") // for service process name
			if sepos != -1 {
				appdir = appdir[:sepos]
			}

			for i := 0; i < 9; i++ {
				d := fmt.Sprintf("/data/app/%s-%s/lib/%s/", appdir,
					IfElseStr(i == 0, "", fmt.Sprintf("%d", i)), archs[runtime.GOARCH])
				if FileExist(d) {
					return d
				}
			}
			dirs, err := filepath.Glob(fmt.Sprintf("/data/app/%s-*", appdir))
			ErrPrint(err)
			if len(dirs) > 0 {
				return dirs[0] + fmt.Sprintf("/lib/%s/", archs[runtime.GOARCH])
			}
			if FileExist(fmt.Sprintf("/data/data/%s/lib/", appdir)) {
				return fmt.Sprintf("/data/data/%s/lib/", appdir)
			}
		}
		return ""
	}
	// dirp must endsWith / or ""
	getLibFile := func(dirp, modname string) string {
		switch runtime.GOOS {
		case "darwin":
			return fmt.Sprintf("%slibQt5%s.%s", dirp, modname, oslibexts[runtime.GOOS])
		case "windows": // best put libs in current directory
			return fmt.Sprintf("%sQt5%s.%s", dirp, modname, oslibexts[runtime.GOOS])
		}
		// case "linux", "freebsd", "netbsd", "openbsd", "android", ...:
		return fmt.Sprintf("%slibQt5%s.%s", dirp, modname, oslibexts["linux"])
	}

	mods := []string{"Inline"}
	// TODO auto check static and omit load other module
	if !UseWrapSymbols { // raw c++ symbols
		mods = append([]string{"Core", "Gui", "Widgets", "Network", "Qml", "Quick", "QuickControls2", "QuickWidgets"}, mods...)
	}

	for _, modname := range mods {
		libpath := getLibFile(getLibDirp(), modname)
		loadModule(libpath, modname)
	}

	// log.Println("Loaded", len(qtlibs), "of", len(mods), gopp.MapKeys(qtlibs))
	gopp.ZeroPrint(len(qtlibs), "Load 0 qtmodule, want", len(mods), mods)

}

func InvokeQtFunc6(symname string, retype byte, args ...interface{}) (VRetype, error) {
	addr := GetQtSymAddr(symname)
	if debugFFICall {
		log.Println("FFI Call:", symname, addr, "retype=", retype, "argc=", len(args))
	}

	// argtys, argvals, argrefps := convArgs(args...)
	// _ = argrefps
	// var retval C.uint64_t = 0
	// _, cok := C.ffi_call_ex(addr, C.int(retype), &retval, C.int(len(args)),
	// 	(*C.uint8_t)(&argtys[0]), (*C.uint64_t)(&argvals[0]))
	var cok error
	retval := cgopp.FfiCall[uint64](addr, args...)

	if debugFFICall {
		ErrPrint(cok, symname, retype, len(args))
	}

	onCtorAlloc(symname)
	return uint64(retval), nil
}

// TODO resolve ffi parameters and then forward to C scope execute
// C scope receiver func: void(void*fnptr, uint64_t*retval, void* argtys, void* argvals)
func ForwardFFIFunc(pxysymname string, symname string, args ...interface{}) (VRetype, error) {
	return 0, nil
}

func isUndefinedSymbolErr(err error) bool {
	return err != nil && strings.Contains(err.Error(), ": undefined symbol: ")
}
func isNotfoundSymbolErr(err error) bool {
	return err != nil &&
		(strings.Contains(err.Error(), "Symbol not found:") ||
			// macos???
			strings.Contains(err.Error(), "symbol not found"))
}

// 直接使用封装的C++ symbols。好像在这设置没有用啊，符号不同，因为参数表的处理也不同，还是要改生成的调用代码。
var UseWrapSymbols bool = false // see also qtrt.UseCppSymbols TODO merge

func refmtSymbolName(symname string) string {
	return IfElseStr(UseWrapSymbols && strings.HasPrefix(symname, "_Z"), "C"+symname, symname)
}

func GetQtSymAddr(symname string) unsafe.Pointer {
	rcsymnames := []string{}
	// orig := symname
	if strings.HasPrefix(symname, "__Z") { // why nm got __Z, but need _Z
		symname = symname[1:]
		rcsymnames = append(rcsymnames, symname)
	}
	symname = refmtSymbolName(symname)
	rcsymnames = append(rcsymnames, symname)
	rcsymnames = append(rcsymnames, symname+"_weakwrap")

	var symadr voidptr
	for _, name := range rcsymnames {
		symadr = GetQtSymAddrRaw(name)
		if symadr != nil {
			break
		}
	}
	gopp.NilPrint(symadr, rcsymnames)

	return symadr
}

func GetQtSymAddrRaw(symname string) unsafe.Pointer {
	for _, lib := range qtlibs {
		addr, err := lib.Symbol(symname)
		if !isUndefinedSymbolErr(err) && !isNotfoundSymbolErr(err) {
			ErrPrint(err, lib.Name(), symname)
		}
		if err != nil {
			continue
		}
		return addr
	}

	rv, err := purego.Dlsym(purego.RTLD_DEFAULT, symname)
	if false {
		gopp.ErrPrint(err, symname)
	}

	return voidptr(rv)
}

// TODO
// get method symbol via virtual table offset
// ptr is class instance ptr
// midx is virtual method offset
// bidx is virtual base class offset
// return is the virtual method function pointer
func getSymByVTable(ptr unsafe.Pointer, midx int, bidx int) unsafe.Pointer {
	return ptr
}

func convArgs(args ...interface{}) (argtys []byte, argvals []uint64, argrefps []*reflect.Value) {
	argtys = make([]byte, 20)
	argvals = make([]uint64, 20)
	argrefps = make([]*reflect.Value, 20)
	for i, argx := range args {
		argty, argval, argrefp := convArg(i, argx)
		argtys[i], argvals[i], argrefps[i] = argty, argval, &argrefp
	}
	return
}

var tyconvmap = map[reflect.Kind]byte{
	reflect.Uint64: FFI_TYPE_UINT64, reflect.Int64: FFI_TYPE_SINT64,
	reflect.Uint32: FFI_TYPE_UINT32, reflect.Int32: FFI_TYPE_SINT32,
	reflect.Uint: FFI_TYPE_UINT32, reflect.Int: FFI_TYPE_INT,
	reflect.Uint16: FFI_TYPE_UINT16, reflect.Int16: FFI_TYPE_SINT16,
	reflect.Uint8: FFI_TYPE_UINT8, reflect.Int8: FFI_TYPE_SINT8,
}

// argval should be the value's valid address
//
//	for non-addressable primitive type, a temporary var is created and it's address is returned
//
// argrefp for hold the temporary created var's address's reference, prevent gc for a while
func convArg(idx int, argx interface{}) (argty byte, argval uint64, argrefp reflect.Value) {
	av := reflect.ValueOf(argx)
	aty := av.Type()

	switch aty.Kind() {
	case reflect.Uint64, reflect.Int64, reflect.Uint32, reflect.Int32,
		reflect.Int, reflect.Uint, reflect.Uint16, reflect.Int16,
		reflect.Uint8, reflect.Int8:
		argty = tyconvmap[aty.Kind()]
		if av.CanAddr() {
			argrefp = av
			argval = refvaluint64(&argrefp)
		} else {
			argrefp = reflect.New(aty)
			argrefp.Elem().Set(av)
			argval = refvaluint64(&argrefp)
		}
	case reflect.Bool:
		argty = FFI_TYPE_INT
		argrefp = reflect.New(IntTy)
		argrefp.Elem().Set(reflect.ValueOf(IfElseInt(argx.(bool), 1, 0)))
		argval = refvaluint64(&argrefp)
	case reflect.Float64:
		argty = FFI_TYPE_DOUBLE
		if av.CanAddr() {
			argrefp = av
			argval = refvaluint64(&argrefp)
		} else {
			argrefp = reflect.New(Float64Ty)
			argrefp.Elem().Set(av)
			argval = refvaluint64(&argrefp)
		}

	case reflect.Float32:
		argty = FFI_TYPE_FLOAT
		if av.CanAddr() {
			argrefp = av
			argval = refvaluint64(&argrefp)
		} else {
			argrefp = reflect.New(Float32Ty)
			argrefp.Elem().Set(av)
			argval = refvaluint64(&argrefp)
		}

	case reflect.Ptr:
		argty = FFI_TYPE_POINTER
		argrefp = reflect.New(av.Type())
		argrefp.Elem().Set(av)
		argval = refvaluint64(&argrefp)

	case reflect.UnsafePointer:
		argty = FFI_TYPE_POINTER
		argrefp = reflect.New(VoidpTy())
		argrefp.Elem().Set(av)
		argval = refvaluint64(&argrefp)

	case reflect.String:
		argty = FFI_TYPE_POINTER
		argpv := unsafe.Pointer(cgopp.CString(argx.(string))) // TODO free memory
		argrefp = reflect.New(VoidpTy())
		argrefp.Elem().Set(reflect.ValueOf(argpv))
		argval = refvaluint64(&argrefp)
		//

	default:
		log.Println("Unknown type:", argty, argval, aty.String(), argx)
	}

	return
}

// emulate reflect.Value
type emuValue struct {
	typ *reflect.Value // placeholder struct pointer field
	ptr unsafe.Pointer
	uint8
}

// hacked replacement of flaged depcreated  reflect.Value.Unsafe.Pointer() and reflect.Value.Pointer()
func refvalptr(vp *reflect.Value) unsafe.Pointer  { return (*emuValue)(unsafe.Pointer(vp)).ptr }
func refvaluintptr(vp *reflect.Value) uintptr     { return uintptr(refvalptr(vp)) }
func refvaluint64(vp *reflect.Value) uint64       { return uint64(refvaluintptr(vp)) }
func refvalptr_(vp *reflect.Value) unsafe.Pointer { return unsafe.Pointer(vp.UnsafeAddr()) }
func refvaluintptr_(vp *reflect.Value) uintptr    { return uintptr(refvalptr(vp)) }
func refvaluint64_(vp *reflect.Value) uint64      { return uint64(refvaluintptr(vp)) }

func convRetval(retype byte, retval interface{}) interface{} {
	refv := reflect.ValueOf(retval)
	switch retype {
	case FFI_TYPE_VOID:
	case FFI_TYPE_INT:
		return refv.Convert(IntTy).Interface()
	case FFI_TYPE_UINT8:
		return refv.Convert(Uint8Ty).Interface()
	default:
		log.Println("Unknown type:", refv.Type().String())
	}
	return retval
}

// func KeepMe() {}

var ctorAllocStacks = map[string][]uintptr{}
var ctorAllocStacksMu sync.Mutex

func onCtorAlloc(symname string) {
	f := func(clsname string) {
		var pc [16]uintptr
		n := runtime.Callers(2, pc[:])
		_ = n
		ctorAllocStacksMu.Lock()
		ctorAllocStacks[clsname] = pc[:]
		ctorAllocStacksMu.Unlock()
	}

	if strings.Index(symname, "C2") > 0 {
		tmp1 := strings.Split(symname, "C2")[0]
		if strings.Index(tmp1, "Q") > 0 {
			tmp2 := strings.Split(tmp1, "Q")[1]
			clsname := "Q" + tmp2
			_ = clsname
			// log.Println("ctor alloc:", clsname)

			f(clsname)
		}
	}
}

// 奇怪了，正则怎么就让程序乱了呢？
func onCtorAlloc1(symname string) {
	reg := `_ZN(\d+)(Q.+)C2.*`
	exp := regexp.MustCompile(reg)
	mats := exp.FindAllStringSubmatch(symname, -1)
	if len(mats) > 0 {
		// var pc [16]uintptr
		// n := runtime.Callers(2, pc[:])
		// _ = n
		// log.Println("fill elems:", n, symname)
		// ctorAllocStacksMu.Lock()
		// ctorAllocStacks[mats[0][2]] = pc[:]
		// ctorAllocStacksMu.Unlock()
	} else {
		// log.Println("not match ctor: ", symname)
	}
}

func GetCtorAllocStack(clsname string) []uintptr {
	ctorAllocStacksMu.Lock()
	defer ctorAllocStacksMu.Unlock()
	if stk, ok := ctorAllocStacks[clsname]; ok {
		return stk
	}
	return nil
}

// /
func test() {
	var ret VRetype
	var err error
	ret, err = InvokeQtFunc("_Z5qrandv", 0, nil)
	log.Println(ret, err)
	// ret, err = InvokeQtFunc("_Z6qsrandj", nil, nil)
	// log.Println(ret, err)
}
