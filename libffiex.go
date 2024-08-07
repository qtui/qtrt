package qtrt

/*
///////////
#include <stdio.h>
#include <stdlib.h>
#include <stdint.h>
#include <string.h>

// #include "ffi.h"
#include "libffi_fake.h"

void *lib = 0;

static void* ffi_type_void_dlptr = 0;
static void* ffi_type_pointer_dlptr = 0;
static void* ffi_type_sint_dlptr = 0;
static void* ffi_type_float_dlptr = 0;
static void* ffi_type_double_dlptr = 0;
static void* ffi_type_sint16_dlptr = 0;
static void* ffi_type_sint32_dlptr = 0;
static void* ffi_type_sint64_dlptr = 0;

static int (*ffi_prep_cif_dlptr)(void*, int, int, void*, void*) = 0;
static void (*ffi_call_dlptr)(void*, void*, void*, void*) = 0;
static void (*ffi_call_var_dlptr)() = 0;


static void ffi_call_0(void*fn) {
  ffi_cif cif;
//   ffi_type *args[10];
  void *args[10];
  void *values[10];
  char *s;
  ffi_arg rc;

   args[0] = ffi_type_pointer_dlptr;
   values[0] = &s;

   args[1] = ffi_type_pointer_dlptr;
   s = calloc(1, 256);
   strcpy(s, "dbcdefg");

   int a0 = 3;
   int* a1 = &a0;
   values[1] = &a1;

   args[2] = ffi_type_pointer_dlptr;
   char *a2[] = {"testprog", "hjdskkk"};
   void* a20 = (void*)a2;
   values[2] = &a20; // how too pointer of pointer ???

   args[3] = ffi_type_sint_dlptr;
   values[3] = &a0;

   if (ffi_prep_cif_dlptr(&cif, FFI_DEFAULT_ABI, 4, ffi_type_void_dlptr, args) == FFI_OK) {
       printf("hehehhee: %p\n", fn);
       int64_t n = 0;
       printf("finish: %d, %lld, %p, \n", (int)rc, n, a2);
       printf("finish: %d, %lld, %p, %s\n", (int)rc, n, a2, a2[0]);
       // n = ((int (*)(int, int, int, int))(fn))(s, a1, &a2, a1); // ok
       ffi_call_dlptr(&cif, fn, &rc, values);
       printf("finish: %d, %p\n", (int)rc, (void*)n);
    }
}

extern void ffi_call_ex(void*fn, int retype, uint64_t *rc, int argc, uint8_t* argtys, uint64_t* argvals);
extern void ffi_call_var_ex(void*fn, int retype, uint64_t *rc, int fixedargc, int totalargc, uint8_t* argtys, uint64_t* argvals);
extern void set_so_ffi_call_ex(void* ex_fnptr, void* varex_fnptr);

static void ffi_call_1(void*fn) {

    uint8_t argtys[20];
    uint64_t argvals[20];

    argtys[0] = FFI_TYPE_POINTER;
    void* o = calloc(1, 256);
    argvals[0] = (uint64_t)(&o);

    int argc = 2;
    argtys[1] = FFI_TYPE_POINTER;
    argvals[1] = (uint64_t)(&argc);

    char *a2[] = {"testprog", "hjdskkk"};
    argtys[2] = FFI_TYPE_POINTER;
    argvals[2] = (uint64_t)(void*)(a2);

    int flag = 0;
    argtys[3] = FFI_TYPE_INT;
    argvals[3] = (uint64_t)(&flag);

    uint64_t retval;
    ffi_call_ex(fn, FFI_TYPE_VOID, &retval, 4, argtys, argvals);
}

*/
import "C"
import (
	"fmt"
	"log"
	"unsafe"
)

func init() {

}

// func itype2stype(itype byte) *C.ffi_type {
func itype2stype(itype byte) voidptr {
	switch itype {
	case FFI_TYPE_VOID:
		return C.ffi_type_void_dlptr
	case FFI_TYPE_POINTER:
		return C.ffi_type_pointer_dlptr
	case FFI_TYPE_INT:
		return C.ffi_type_sint32_dlptr
	case FFI_TYPE_FLOAT:
		return C.ffi_type_float_dlptr
	case FFI_TYPE_DOUBLE:
		return C.ffi_type_double_dlptr
	case FFI_TYPE_SINT16:
		return C.ffi_type_sint16_dlptr
	case FFI_TYPE_SINT32:
		return C.ffi_type_sint32_dlptr
	case FFI_TYPE_SINT64:
		return C.ffi_type_sint64_dlptr
	default:
		log.Println("unknown type:", itype)
		break
	}
	return C.ffi_type_void_dlptr
}

/*
TODO

	argtypes int[20]
	argvals uint64_t[20]
*/
func ffi_call_ex(fn unsafe.Pointer, retype int, rc *uint64, argc int, argtys []int, argvals []uint64) {

	var cif C.ffi_cif
	// var ffitys = make([]*C.ffi_type, 20)
	var ffitys = make([]voidptr, 20)
	var ffivals = make([]unsafe.Pointer, 20)
	// var ffirc C.ffi_arg
	var ffirc int64
	_, _, _, _ = cif, ffitys, ffivals, ffirc

	for i := 0; i < argc; i++ {
		switch byte(argtys[i]) {
		case FFI_TYPE_VOID:
			ffitys[i] = C.ffi_type_void_dlptr
			ffivals[i] = nil
		}
	}

	// C.ffi_call(&cif, fn, &ffirc, ffivals)
}

func init_so_ffi_call() {
	ex_fnptr := GetQtSymAddrRaw("ffi_call_ex")
	varex_fnptr := GetQtSymAddrRaw("ffi_call_var_ex")
	if ex_fnptr != nil && varex_fnptr != nil {
		C.set_so_ffi_call_ex(ex_fnptr, varex_fnptr)
	}
}

func deinit() {}

func InvokeQtFunc(symname string, retype byte, types []byte, args ...interface{}) (VRetype, error) {
	for modname, lib := range qtlibs {
		addr, err := lib.Symbol(symname)
		ErrPrint(err)
		if err != nil {
			continue
		}

		log.Println("FFI Call:", modname, symname, addr)
		// C.ffi_call_0(addr)
		C.ffi_call_1(addr)
		return 0, nil
	}
	return 0, fmt.Errorf("Symbol not found: %s", symname)
}

func InvokeQtFunc5(symname string, retype byte, argc int, types []byte, args []uint64) (VRetype, error) {
	addr := GetQtSymAddr(symname)
	log.Println("FFI Call:", symname, addr)

	var retval C.uint64_t = 0
	C.ffi_call_ex(addr, C.int(retype), &retval, C.int(argc),
		(*C.uint8_t)(&types[0]), (*C.uint64_t)(&args[0]))

	return uint64(retval), fmt.Errorf("Symbol not found: %s", symname)
}

// for variadic function call
func InvokeQtFunc6Var(symname string, retype byte, fixedargc int, args ...interface{}) (VRetype, error) {
	addr := GetQtSymAddr(symname)
	if debugFFICall {
		log.Println("FFI Call:", symname, addr, "retype=", retype, "fixedargc=", fixedargc, "totalargc=", len(args))
	}

	argtys, argvals, argrefps := convArgs(args...)
	_ = argrefps
	var retval C.uint64_t = 0
	_, cok := C.ffi_call_var_ex(addr, C.int(retype), &retval, C.int(fixedargc), C.int(len(args)),
		(*C.uint8_t)(&argtys[0]), (*C.uint64_t)(&argvals[0]))
	if debugFFICall {
		ErrPrint(cok, symname, retype, len(args))
	}

	onCtorAlloc(symname)
	return uint64(retval), nil
}

// fix return QSize like pure record, RVO
func InvokeQtFunc7(symname string, args ...interface{}) (VRetype, error) {
	addr := GetQtSymAddr(symname)
	var retype byte = FFI_TYPE_POINTER
	log.Println("FFI Call:", symname, addr, "retype=", retype, "argc=", len(args))

	var retval unsafe.Pointer = C.calloc(1, 256)
	argtys, argvals, argrefps := convArgs(args...)
	_ = argrefps
	// var retval C.uint64_t = 0
	C.ffi_call_ex(addr, C.int(retype), (*C.uint64_t)(retval), C.int(len(args)),
		(*C.uint8_t)(&argtys[0]), (*C.uint64_t)(&argvals[0]))
	return uint64(uintptr(retval)), nil
}

// /Library/Developer/CommandLineTools/SDKs/MacOSX11.sdk/usr/include/ffi/ffi.h

const (
	FFI_TYPE_VOID       = byte(0)  // byte(C.FFI_TYPE_VOID)
	FFI_TYPE_INT        = byte(1)  //byte(C.FFI_TYPE_INT)
	FFI_TYPE_FLOAT      = byte(2)  //byte(C.FFI_TYPE_FLOAT)
	FFI_TYPE_DOUBLE     = byte(3)  //byte(C.FFI_TYPE_DOUBLE)
	FFI_TYPE_LONGDOUBLE = byte(4)  //byte(C.FFI_TYPE_LONGDOUBLE)
	FFI_TYPE_UINT8      = byte(5)  //byte(C.FFI_TYPE_UINT8)
	FFI_TYPE_SINT8      = byte(6)  //byte(C.FFI_TYPE_SINT8)
	FFI_TYPE_UINT16     = byte(7)  //byte(C.FFI_TYPE_UINT16)
	FFI_TYPE_SINT16     = byte(8)  //byte(C.FFI_TYPE_SINT16)
	FFI_TYPE_UINT32     = byte(9)  //byte(C.FFI_TYPE_UINT32)
	FFI_TYPE_SINT32     = byte(10) //byte(C.FFI_TYPE_SINT32)
	FFI_TYPE_UINT64     = byte(11) //byte(C.FFI_TYPE_UINT64)
	FFI_TYPE_SINT64     = byte(12) //byte(C.FFI_TYPE_SINT64)
	FFI_TYPE_STRUCT     = byte(13) //byte(C.FFI_TYPE_STRUCT)
	FFI_TYPE_POINTER    = byte(14) //byte(C.FFI_TYPE_POINTER)
	FFI_TYPE_COMPLEX    = byte(15) //byte(C.FFI_TYPE_COMPLEX)

)
