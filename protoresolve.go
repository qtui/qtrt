package qtrt

import (
	"log"
	"reflect"
	"runtime"
	"strings"

	"github.com/kitech/gopp"
	"github.com/kitech/gopp/cgopp"

	// "github.com/qtui/qtrt"
	"github.com/qtui/qtclzsz"
	"github.com/qtui/qtsyms"
)

// only call by callany
func getclzmthbycaller() (clz string, mth string, isst bool, isctor, isdtor bool) {
	pc, _, _, _ := runtime.Caller(3)
	fno := runtime.FuncForPC(pc)
	fnname := fno.Name()
	// log.Println(fno, fnname, gopp.Retn(fno.FileLine(pc)))

	// todo maybe add number suffix for overloaded
	// main.NewQxx
	if pos := strings.LastIndex(fnname, ".NewQ"); pos > 0 {
		clz = fnname[pos+4:]
		mth = clz
		isctor = true
		return
	}
	// todo maybe add number suffix for overloaded
	// qtcore.QString_FromUtf8
	if pos := strings.Index(fnname, "_"); pos > 0 {
		bpos := strings.Index(fnname, ".Q")
		clz = fnname[bpos+1 : pos]
		mth = fnname[pos+1:]
		isst = true
		return
	}
	// main.(*QObject).Dummy
	funcname := "main.(*QObject).Dummy"
	funcname = fnname
	pos1 := strings.Index(funcname, "(")
	pos2 := strings.Index(funcname, ")")
	clzname := funcname[pos1+2 : pos2]
	mthname := funcname[pos2+2:]

	if mthname == "Dtor" {
		isdtor = true
		mthname = "~" + clzname
		return
	}
	// log.Println(clzname, mthname)

	clz = clzname
	mth = mthname
	return
}

// todo wip
func checkclzmthpropsp(clz string, mth string) (isctor, isdtor, isst bool) {
	isctor = clz == mth
	// isdtor =
	return
}

// static call 用的不多，后续再考虑
const CppStaticCall = 0x3

// static call: cobj == 0x3
// like jit, name jitqt
func Callany(rvop, cobj voidptr, args ...any) gopp.Fatptr {
	return implCallany(rvop, cobj, args...)
}
func implCallany(rovp voidptr, cobj voidptr, args ...any) gopp.Fatptr {
	// log.Println("========", rovp, cobj, len(args), args)
	clzname, mthname, isstatic, isctor, isdtor := getclzmthbycaller()
	log.Println("========", clzname, mthname, rovp, cobj, len(args), args)
	// isctor := clzname == mthname
	// isdtor := mthname == "Dtor"
	// mthname = gopp.IfElse2(isdtor, "~"+clzname, mthname)

	mths, ok := qtsyms.QtSymbols[clzname]
	gopp.FalsePrint(ok, "not found???", clzname, qtsyms.InitLoaded)

	//
	var namercmths []qtsyms.QtMethod // 备份
	var rcmths = mths

	rcmths = resolvebyname(mthname, rcmths)
	namercmths = rcmths

	// 根据参数个数
	rcmths = resolvebyargc(len(args), rcmths)

	//
	argtys := reflecttypes(args...)
	rcmths = resolvebyargty(argtys, rcmths)

	if isctor {
		rcmths = resolvebyctorno(rcmths)
	} else if isdtor {
		rcmths = resolvebydtorno(rcmths)
	}

	log.Println("final rcmths:", len(rcmths), rcmths)
	var ccret gopp.Fatptr
	switch len(rcmths) {
	case 0:
		// sowtfuck
		gopp.Warn("No match, rcmths", len(namercmths), namercmths, len(namercmths))
	case 1:
		// good
		mtho := rcmths[0]
		convedargs := argsconvert(mtho, argtys, args...)
		// log.Println("oriargs", len(args), args, "conved", len(convedargs), convedargs)
		// log.Println("convedargs", len(convedargs), convedargs)
		// fnsym := Libman.Dlsym(mtho.CCSym)
		// if true {
		fnsym := GetQtSymAddr(mtho.CCSym)
		// }
		if isctor {
			clzsz := qtclzsz.Get(clzname)
			// cthis := cgopp.Mallocgc(clzsz) // cannot destruct for free crash
			cthis := cgopp.Malloc(clzsz) // todo, when free?
			ccargs := append([]any{cthis}, convedargs...)
			// log.Println("fficall info", mthname, fnsym, len(args), len(ccargs), ccargs)
			// cpp ctor 函数是没有返回值的
			cgopp.FfiCall[int](fnsym, ccargs...)
			// ccret = cthis
			ccret = gopp.FatptrOf(cthis)
			// log.Println(cthis, ccret, clzsz, clzname)
		} else if isdtor {
			if fnsym == nil {
				fnsym = GetQtSymAddr(clzname + "Dtor")
				log.Println("Dtor weak symbol replace", mtho.CCSym, "=>", clzname+"Dtor", fnsym)
			}
			cgopp.FfiCall[gopp.Fatptr](fnsym, cobj)
		} else if isstatic { // todo ROV
			ccargs := convedargs
			if rovp != nil { // 把 cobj 当作ROV使用
				ccargs = append([]any{rovp}, ccargs...)
			}
			// log.Println("fficall info", clzname, mthname, fnsym, len(args), len(ccargs), ccargs)
			// todo ROV
			ccret = cgopp.FfiCall[gopp.Fatptr](fnsym, ccargs...)
		} else { // common method // todo ROV
			// todo
			ccargs := append([]any{cobj}, convedargs...)
			if rovp != nil {
				ccargs = append([]any{rovp}, ccargs...)
			}
			// log.Println("fficall info", clzname, mthname, fnsym, len(args), len(ccargs), ccargs)
			ccret = cgopp.FfiCall[gopp.Fatptr](fnsym, ccargs...)
		}
		// log.Println("call/ccret", ccret, clzname, mthname, "(", cobj, convedargs, ")")
	default:
		// sowtfuck
	}
	return ccret
}

func resolvebyctorno(mths []qtsyms.QtMethod) (rets []qtsyms.QtMethod) {
	// bye C1E, C2E, C3E?
	c2idx := -1
	c1idx := -1
	for idx, mtho := range mths {
		if strings.Contains(mtho.CCSym, mtho.Name+"C1E") {
			c1idx = idx
		} else if strings.Contains(mtho.CCSym, mtho.Name+"C2E") {
			c2idx = idx
		}
	}
	if c2idx >= 0 {
		rets = append(rets, mths[c2idx])
	} else if c2idx >= 0 {
		rets = append(rets, mths[c1idx])
	} else {
	}
	return
}
func resolvebydtorno(mths []qtsyms.QtMethod) (rets []qtsyms.QtMethod) {
	// bye C1E, C2E, C3E?
	c2idx := -1
	c1idx := -1
	for idx, mtho := range mths {
		if strings.Contains(mtho.CCSym, mtho.Name[1:]+"D1E") {
			c1idx = idx
		} else if strings.Contains(mtho.CCSym, mtho.Name[1:]+"D2E") {
			c2idx = idx
		}
	}
	if c2idx >= 0 {
		rets = append(rets, mths[c2idx])
	} else if c2idx >= 0 {
		rets = append(rets, mths[c1idx])
	} else {
	}
	return
}

// todo todo todo
func qttypemathch(idx int, tystr string, tyo reflect.Type, conv bool, argx any) (any, bool) {
	// log.Println("tymat", idx, tystr, "?<=", tyo.String())
	// goty := tyo.String()

	var rvx = argx
	tymat := false

	mcdata := &TMCData{}
	mcdata.idx = idx
	mcdata.ctys = tystr
	mcdata.gotyo = tyo
	mcdata.goargx = argx
	mcdata.ffiargx = argx // default

	for _, mater := range typemcers {
		tymat = mater.Match(mcdata, conv)
		if tymat {
			if conv {
				rvx = mcdata.ffiargx
			}
			// log.Println("matched", reflect.TypeOf(mater), conv)
			break
		}
	}

	// if goty == tystr {
	// 	tymat = true
	// } else if goty+"&" == tystr {
	// 	tymat = true
	// 	if conv {
	// 		// 只对primitive type可以
	// 		refval := reflect.New(tyo)
	// 		refval.Elem().Set(reflect.ValueOf(argx))
	// 		rvx = refval.Interface()
	// 	}
	// } else if goty == "[]string" && tystr == "char**" {
	// 	tymat = true
	// 	if conv {
	// 		// todo how freeit
	// 		ptr := cgopp.CStrArrFromStrs(argx.([]string))
	// 		rvx = ptr
	// 	}

	// 	// QObject* ?<= *main.QObject
	// } else if isqtptrtymat(tystr, tyo) {
	// 	tymat = true
	// 	if conv {
	// 		tvx := reflect.ValueOf(argx)
	// 		if tvx.IsNil() {

	// 		} else {
	// 			// .Elem().FieldByName("Cthis")
	// 		}
	// 		log.Println(tvx)
	// 	}
	// }
	gopp.FalsePrint(tymat, "tymat", idx, tystr, "?<=", tyo.String(), tymat)

	return rvx, tymat
}

func argsconvert(mtho qtsyms.QtMethod, tys []reflect.Type, args ...any) (rets []any) {
	sgnt, _ := qtsyms.Demangle(mtho.CCSym)
	// log.Println(sgnt, mtho.CCSym)
	vec := qtsyms.SplitArgs(sgnt)
	log.Println("argconving", len(vec), vec, sgnt)

	for j := 0; j < len(vec); j++ {
		argx, mat := qttypemathch(j, vec[j], tys[j], true, args[j])
		if !mat {
			// wtf???
		}
		rets = append(rets, argx)
	}
	return
}

func resolvebyargty(tys []reflect.Type, mths []qtsyms.QtMethod) (rets []qtsyms.QtMethod) {
	for _, mtho := range mths {
		// log.Println(mtho.Name)

		sgnt, _ := qtsyms.Demangle(mtho.CCSym)
		// log.Println(sgnt, mtho.CCSym)
		vec := qtsyms.SplitArgs(sgnt)
		// log.Println(len(vec), vec, sgnt)

		allmat := true
		for j := 0; j < len(vec); j++ {
			_, mat := qttypemathch(j, vec[j], tys[j], false, nil)
			if !mat {
				allmat = false
			}
		}
		if allmat {
			// log.Println(gopp.MyFuncName(), "rc", mtho.CCSym)
			rets = append(rets, mtho)
		}
	}
	return
}

func resolvebyname(mthname string, mths []qtsyms.QtMethod) (rets []qtsyms.QtMethod) {

	for _, mtho := range mths {
		// log.Println(mtho.Name)
		if mtho.Name == mthname {
			// log.Println(gopp.MyFuncName(), "rc", mthname, mtho.CCSym)
			rets = append(rets, mtho)
		}
	}
	return
}
func resolvebyargc(argc int, mths []qtsyms.QtMethod) (rets []qtsyms.QtMethod) {
	for _, mtho := range mths {
		// log.Println(mtho.Name)
		// if mtho.Type.NumIn() == argc {
		sgnt, _ := qtsyms.Demangle(mtho.CCSym)
		// log.Println(sgnt, mtho.CCSym)
		vec := qtsyms.SplitArgs(sgnt)
		// log.Println(vec)
		if len(vec) == argc {
			// log.Println(gopp.MyFuncName(), "rc", mtho.CCSym)
			rets = append(rets, mtho)
		}
		// }
	}
	return
}

func reflecttypes(args ...any) (rets []reflect.Type) {
	for _, argx := range args {
		if argx == nil { // argvx无类型的nil
			ty := gopp.VoidpTy()
			rets = append(rets, ty)
			continue
		}
		ty := reflect.TypeOf(argx)
		rets = append(rets, ty)
	}

	return
}
