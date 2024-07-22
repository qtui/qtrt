package qtrt

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	// "github.com/ebitengine/purego"

	"github.com/kitech/gopp"
	"github.com/kitech/gopp/cgopp"
)

//

var isLinkedQtlib = false
var qtlibs = map[string]FFILibrary{}
var qtlibpaths = map[string]string{}

func check_linked_qtmod() bool {
	// images := cgopp.DyldImages()
	// _QCompileVersion
	rv := cgopp.Dlsym0("_QCompileVersion")
	return rv != nil
}

var mainqtmods = []string{"Core", "Gui", "Widgets", "Network", "Qml", "Quick", "QuickControls2", "QuickWidgets"}

// 1. 从自己的二进制文件中查找链接的qt库
// 2. 如果没有链接，转换到搜索模式
func loadAllModules() {
	soimgs := cgopp.DyldImagesSelf()
	soimgs = filterQtsoimages(soimgs)

	if len(soimgs) > 0 {
		isLinkedQtlib = true
		gopp.Mapdo(soimgs, func(vx any) any {
			mod := (vx.(string))
			log.Println(mod, vx)
			dlh, err := NewFFILibrary(vx.(string))
			gopp.ErrPrint(err, vx)
			// log.Println(dlh)
			if err == nil {
				qtlibs[mod] = dlh
				qtlibpaths[mod] = vx.(string)
			}
			return nil
		})
	} else {
		for _, modname := range mainqtmods {
			// libpath := getLibFile(getLibDirp(), modname)
			loadModule("", modname)
		}
	}
}

func filterQtsoimages(soimgs []string) (rets []string) {
	gopp.Mapdo(soimgs, func(vx any) any {
		v := vx.(string)
		bname := filepath.Base(v)
		if strings.HasPrefix(bname, "Qt") { // macos
			rets = append(rets, v)
		} else if strings.HasPrefix(bname, "libQt") {
			rets = append(rets, v)
		}
		return nil
	})
	return
}

// libQt6Core.so => Core
func qtlibname2mod(nameorpath string) string {
	bname := qtlibname2link(nameorpath)
	if strings.HasPrefix(bname, "Qt") {
		if bname[2] >= '0' && bname[2] <= '9' {
			return bname[3:]
		}
		return bname[2:]
	}
	return bname
}

// libQt6Core.so => Qt6Core
func qtlibname2link(nameorpath string) string {
	// QtCore // mac
	// libQtCore.dylib // mac
	// libQtCore.so // linux/unix
	// libQt5Core.so // linux/unix
	// libQt6Core.so // linux/unix
	// libQtCore.dll // win
	// libQt5Core.dll // win
	// libQt6Core.dll // win
	bname := nameorpath
	bname = filepath.Base(bname)
	pos := strings.Index(bname, ".")
	if pos > 0 {
		bname = bname[:pos]
	}
	if strings.HasPrefix(bname, "lib") {
		bname = bname[3:]
	}
	return bname
}

// nameorpath
func qtmod2rclibnames(nameorpath string, incinline bool) (rets []string) {
	mod := qtlibname2mod(nameorpath)
	rets = gopp.Sliceof("Qt"+mod, "Qt6"+mod,
		"libQt"+mod+".so", "libQt"+mod+".dylib", "libQt"+mod+".dll",
		"libQt6"+mod+".so", "libQt6"+mod+".dylib", "libQt6"+mod+".dll",
	)
	if incinline {
		rets = append(rets,
			"libQt"+mod+"Inline.so", "libQt"+mod+"Inline.dylib", "libQt"+mod+"Inline.dll",
		)
	}
	return
}

func loadModuleFullpath(fullpath string, modname string) {

}

func loadModule(libpath string, modname string) (err error) {
	err = loadModuleImpl(libpath, modname)
	if err == nil {
		err = loadModuleImpl(libpath, modname+"Inline")
	}
	return
}

// func FindModule(modname string) (string, error) {
// 	modname = "Core"
// 	dlh, err := purego.Dlopen(modname, purego.RTLD_LAZY)
// 	gopp.ErrPrint(err, modname)
// 	log.Println(dlh)

// 	return modname, nil
// }

// basename like libQtCore.so
// search in libdirs
func findmoduleBylibname(libname string) string {
	libdirs := getsyslibdirs()
	for _, libdir := range libdirs {
		libfile := filepath.Join(libdir, libname)
		if gopp.FileExist2(libfile) {
			return libfile
		}
	}
	return ""
}

func getsyslibdirs() []string {
	libdirs := []string{"", "./", "/opt/qt/lib/", "/usr/lib/", "/usr/lib64/", "/usr/local/lib/", "/usr/local/opt/qt/lib/", gopp.Mustify1(os.UserHomeDir()) + "/.nix-profile/lib/"}

	for _, envname := range []string{"LD_LIBRARY_PATH", "DYLD_LIBRARY_PATH"} {
		envldpath := os.Getenv(envname)
		if len(envldpath) == 0 {
			continue
		}
		fld := strings.Split(envldpath, ":")
		libdirs = append(libdirs, fld...)
	}

	qmakepath, err := Which("qmake")
	// log.Println(qmakepath, err)
	if err == nil {
		qmakedir := filepath.Dir(qmakepath)
		qmakelibdir1 := filepath.Join(qmakedir, "../lib")
		qmakelibdir2 := filepath.Join(qmakedir, "../lib64")
		libdirs = append(libdirs, qmakelibdir1, qmakelibdir2)

		rets, err := filepath.Glob(qmakelibdir1 + "/Qt*.framework")
		if err == nil {
			libdirs = append(libdirs, rets...)
		}
		// log.Println(rets)
		rets, err = filepath.Glob(qmakelibdir2 + "/Qt*.framework")
		if err == nil {
			libdirs = append(libdirs, rets...)
		}
		// log.Println(rets)
	}

	return libdirs
}

func Which(name string) (string, error) {
	lines, err := gopp.RunCmd(".", "which", name)
	if err != nil {
		return "", err
	}
	if len(lines) == 0 {
		return "", os.ErrNotExist
	}
	return gopp.FirstofGv(lines), nil
}

func loadModuleImpl(libpath string, modname string) error {
	// must endwiths /
	// todo LD_LIBRARY_PATH
	// todo DYLD_LIBRARY_PATH
	// todo windows...
	// todo diffenece os, diffence libdirs/fnames
	libdirs := []string{"", "./", "/opt/qt/lib/", "/usr/lib/", "/usr/lib64/", "/usr/local/lib/", "/usr/local/opt/qt/lib/", gopp.Mustify1(os.UserHomeDir()) + "/.nix-profile/lib/"}
	libpath = gopp.IfElse2(libpath == "", "Qt5"+modname, libpath)
	fnames := []string{libpath, "Qt" + modname,
		fmt.Sprintf("Qt%s.framework/Versions/Qt%s", modname, modname),
		fmt.Sprintf("Qt%s.framework/Versions/5/Qt%s", modname, modname),
		fmt.Sprintf("Qt%s.framework/Versions/6/Qt%s", modname, modname),
		fmt.Sprintf("Qt%s.framework/Versions/7/Qt%s", modname, modname),
		fmt.Sprintf("Qt%s.framework/Versions/A/Qt%s", modname, modname),
	}

	// log.Println(libpath, modname)
	var err error = os.ErrNotExist
	var lib FFILibrary
	for _, dir := range libdirs {
		for _, fname := range fnames {
			rcfile := dir + fname
			lib, err = NewFFILibrary(rcfile)
			if err == nil {
				qtlibs[modname] = lib
				qtlibpaths[modname] = rcfile
				break
			}
		}
		if err == nil {
			break
		}
	}
	if strings.HasPrefix(modname, "Inline") && modname != "Inline" {
		ErrPrint(err, lib, libpath, modname, fnames, libdirs)
	}
	return err
}
