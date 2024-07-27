package qtrt

import (
	"log"
	"os"
	"reflect"
	"runtime"
	"strings"

	"github.com/kitech/gopp"
	"github.com/kitech/gopp/cgopp"

	// "github.com/qtui/qtrt"
	"github.com/qtui/qtclzsz"
	"github.com/qtui/qtsyms"
)

// 本文件的函数也许还可以用于其他的C++库？
// 也许还可以更抽象化一点，支持所有的C/C++库？
// 还需要不少工作的，像得到类结构大小，得到inline的方法/函数符号，enum值等
// 再加上libllvm解析头文件还有点可能。
// project cxjit4go

// todo wip
// static call 用的不多，后续再考虑
const CppStaticCall = 0x3

// Callany with string class name and method name
// just for some case, like test
// todo 怎么支持重载的方法，1. 从传递的参数解析出来，2，调用传递参数信息
// mthname c++ name, first char lower
// for ctor, mth=new or classname
// for dtor, mth=delete or Dtor. delete maybe conflict?
// for static, cobj=nil,
// example:
//
//	Callanystrfy[voidptr]("QWidget", "new", nil)
//	Callanystrfy0("QWidget", "show", w)
func CallanyStrfy[RTY any](clzname, mthname string, cobj GetCthiser, args ...any) RTY {
	var rv RTY
	rv = CallanyStrfyRov[RTY](clzname, mthname,
		nil, cobj, args...)
	return rv
}

// no return, no rov
func CallanyStrfy0(clzname, mthname string, cobj GetCthiser, args ...any) {
	CallanyStrfyRov[int](clzname, mthname,
		nil, cobj, args...)
}

// full signature
func CallanyStrfyRov[RTY any](clzname, mthname string, rovp voidptr, cobj GetCthiser, args ...any) RTY {
	mthname = gopp.Title(mthname)
	isstatic := cobj == nil // todo 区分无类型的nil和有类型的nil？
	isctor := mthname == "new" || mthname == clzname
	isdtor := mthname == "delete" || mthname == "Dtor"
	if isctor {
		mthname = clzname
	} else if isdtor {
		mthname = "Dtor"
	}

	var rv RTY
	rv = implCallany2[RTY](clzname, mthname, isctor, isdtor, isstatic,
		rovp, cobj, args...)
	return rv
}

// todo 支持模板方法，但不是模板类
// static call: cobj == 0x3
// like jit, name jitqt
// no rov
func Callany[RTY any](cobj GetCthiser, args ...any) RTY {
	return implCallany[RTY](nil, cobj, args...)
}

// todo ROV 必定没有返回值了吧
// full signature
func CallanyRov[RTY any](rovp voidptr, cobj GetCthiser, args ...any) RTY {
	return implCallany[RTY](rovp, cobj, args...)
}
func CallanyFull[RTY any](rovp voidptr, cobj GetCthiser, args ...any) RTY {
	return implCallany[RTY](rovp, cobj, args...)
}

// no return, no rov
func Callany0(cobj GetCthiser, args ...any) {
	implCallany[int](nil, cobj, args...)
}
func implCallany[RTY any](rovp voidptr, cobj GetCthiser, args ...any) (ccret RTY) {
	// log.Println("========", rovp, cobj, len(args), args)
	clzname, mthname, isstatic, isctor, isdtor := getclzmthbycaller()
	// log.Println("========", clzname, mthname, rovp, cobj, len(args), args)

	return implCallany2[RTY](clzname, mthname, isctor, isdtor, isstatic,
		rovp, cobj, args...)
}
func implCallany2[RTY any](clzname, mthname string, isctor, isdtor, isstatic bool,
	rovp voidptr, cobj GetCthiser, args ...any) (ccret RTY) {
	// log.Println("========", rovp, cobj, len(args), args)
	// clzname, mthname, isstatic, isctor, isdtor := getclzmthbycaller()
	// this line should be trace level
	log.Printf("== %v.%v rov:%v cobj:%v isst:%v %v %v", clzname, mthname, rovp, cobj, gopp.Toint(isstatic), len(args), args)
	// isctor := clzname == mthname
	// isdtor := mthname == "Dtor"
	// mthname = gopp.IfElse2(isdtor, "~"+clzname, mthname)

	mths, ok := qtsyms.QtSymbols[clzname]
	gopp.FalsePrint(ok, "not found???", clzname, mthname, qtsyms.InitLoaded)
	// log.Println(clzname, mthname, len(mths), mths, len(mths))

	//
	var namercmths []qtsyms.QtMethod // 备份
	var rcmths = mths

	rcmths = resolvebyname(mthname, rcmths)
	namercmths = rcmths
	gopp.ZeroPrint(len(rcmths), "mthname404", clzname, mthname, len(rcmths), rcmths)
	gopp.TrueThen(len(rcmths) == 0, os.Exit, -1)

	// 根据参数个数
	rcmths = resolvebyargc(len(args), rcmths)
	// log.Println(clzname, mthname, len(rcmths), rcmths, len(rcmths))

	//
	argtys := reflecttypes(args...)
	rcmths = resolvebyargty(argtys, rcmths)
	// log.Println(clzname, mthname, len(rcmths), rcmths, len(rcmths))

	// 根据返回的引用类型
	rcmths = resolvebyretrefty(rcmths)
	// log.Println(clzname, mthname, len(rcmths), rcmths, len(rcmths))

	if isctor {
		rcmths = resolvebyctorno(rcmths)
	} else if isdtor {
		rcmths = resolvebydtorno(rcmths)
	}
	// log.Println(clzname, mthname, len(rcmths), rcmths, len(rcmths))

	switch len(rcmths) {
	case 1:
		// good
		mtho := rcmths[0]
		convedargs := argsconvert(mtho, argtys, args...)
		// log.Println("oriargs", len(args), args, "conved", len(convedargs), convedargs)
		// log.Println("convedargs", len(convedargs), convedargs)
		// if true {
		fnsym := GetQtSymAddr(mtho.CCSym)
		// }
		if isctor {
			clzsz := qtclzsz.Get(clzname)
			gopp.TruePrint(clzsz <= 0, "wtf", clzsz, clzname)
			// cthis := cgopp.Mallocgc(clzsz) // cannot destruct for free crash
			cthis := cgopp.Malloc(clzsz) // todo, when free?
			ccargs := append([]any{cthis}, convedargs...)
			// log.Println("fficall info", mthname, fnsym, len(args), len(ccargs), ccargs)
			// cpp ctor 函数是没有返回值的
			cgopp.FfiCall[int](fnsym, ccargs...)
			ccret = any(cthis).(RTY)
			// gopp.Copyx(&ccret, &cthis)
			// log.Println(cthis, ccret, clzsz, clzname)
		} else if isdtor {
			if fnsym == nil {
				fnsym = GetQtSymAddr(clzname + "Dtor")
				log.Println("Dtor weak symbol replace", mtho.CCSym, "=>", clzname+"Dtor", fnsym)
			}
			cgopp.FfiCall[RTY](fnsym, cobj.GetCthis())
		} else if isstatic { // todo ROV
			ccargs := convedargs
			if rovp != nil { // 把 cobj 当作ROV使用
				ccargs = append([]any{rovp}, ccargs...)
			}
			// log.Println("fficall info", clzname, mthname, fnsym, len(args), len(ccargs), ccargs)
			// todo ROV
			ccret = cgopp.FfiCall[RTY](fnsym, ccargs...)
		} else { // common method // todo ROV
			// todo
			ccargs := append([]any{cobj.GetCthis()}, convedargs...)
			if rovp != nil {
				ccargs = append([]any{rovp}, ccargs...)
			}
			// log.Println("fficall info", clzname, mthname, fnsym, len(args), len(ccargs), ccargs, namercmths)
			ccret = cgopp.FfiCall[RTY](fnsym, ccargs...)
		}
		// log.Println("call/ccret", ccret, clzname, mthname, "(", cobj, convedargs, ")")

	case 0:
		// sowtfuck, quit app? panic?
		gopp.Warn("No match, namercmths", clzname, mthname, argtys, len(namercmths), namercmths, len(namercmths))
	default: // >1
		// sowtfuck, quit app? panic?
		log.Println("Final rcmths>1:", clzname, mthname, argtys, len(rcmths), rcmths, len(rcmths))
	}
	gopp.TrueThen(len(rcmths) == 0 || len(rcmths) > 1, os.Exit, -1)
	return
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
		if c2idx >= 0 {
			break
		}
	}
	if c2idx >= 0 {
		rets = append(rets, mths[c2idx])
	} else if c1idx >= 0 {
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
		if c2idx >= 0 {
			break
		}
	}
	if c2idx >= 0 {
		rets = append(rets, mths[c2idx])
	} else if c1idx >= 0 {
		rets = append(rets, mths[c1idx])
	} else {
	}
	return
}

// &, &&
func resolvebyretrefty(mths []qtsyms.QtMethod) (rets []qtsyms.QtMethod) {
	// bye _ZNO, _ZNKR
	c2idx := -1
	c1idx := -1
	for idx, mtho := range mths {
		if strings.Contains(mtho.CCSym, "_ZNO") {
			c1idx = idx
		} else if strings.Contains(mtho.CCSym, "_ZNKR") {
			c2idx = idx
		} else {
			rets = append(rets, mtho)
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

	// gopp.FalsePrint(tymat, "tymat", idx, tystr, "?<=", tyo, tymat)

	return rvx, tymat
}

func argsconvert(mtho qtsyms.QtMethod, tys []reflect.Type, args ...any) (rets []any) {
	sgnt, _ := qtsyms.Demangle(mtho.CCSym)
	// log.Println(sgnt, mtho.CCSym)
	vec := qtsyms.SplitArgs(sgnt)
	// log.Println("argconving", len(vec), vec, sgnt)

	for j := 0; j < len(vec); j++ {
		argx := args[j]
		if arg, ok := argx.(GetCthiser); ok {
			if arg != nil && !reflect.ValueOf(arg).IsNil() {
				argx = arg.GetCthis()
			} else {
				argx = voidptr(nil)
			}
			refval := reflect.ValueOf(arg)
			_ = refval
			// log.Println(reflect.TypeOf(argx), argx, arg, arg == nil, refval.IsNil())
		}

		argx, mat := qttypemathch(j, vec[j], tys[j], true, argx)
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
		} else {
			// log.Println(gopp.MyFuncName(), "rc", mtho.CCSym, )
		}
	}
	return
}

func resolvebyname(mthname string, mths []qtsyms.QtMethod) (rets []qtsyms.QtMethod) {

	for _, mtho := range mths {
		// log.Println(mtho.Name, "want", mthname)
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

// only call by implCallany
func getclzmthbycaller() (clz string, mth string, isst bool, isctor, isdtor bool) {
	pc, _, _, _ := runtime.Caller(3)
	fno := runtime.FuncForPC(pc)
	fnname := fno.Name()
	// log.Println(fno, fnname, gopp.Retn(fno.FileLine(pc)))

	// overload with suffix z*, 日前支持z0-z9
	namelast2c := fnname[len(fnname)-2:]
	if namelast2c[0] == 'z' && namelast2c[1] >= '0' && namelast2c[1] <= '9' {
		fnname = fnname[:len(fnname)-2]
	}

	// todo maybe add number suffix for overloaded
	// main.NewQxx
	if pos := strings.LastIndex(fnname, ".NewQ"); pos > 0 {
		clz = fnname[pos+4:]
		mth = clz
		isctor = true
		return
	}
	if pos := strings.LastIndex(fnname, ".newQ"); pos > 0 {
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
		mth = "~" + clzname
		clz = clzname
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
