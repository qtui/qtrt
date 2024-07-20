package qtrt

import (
	"fmt"
	"os"
	"strings"

	// "github.com/ebitengine/purego"

	"github.com/kitech/gopp"
	"github.com/kitech/gopp/cgopp"
)

//

var isLinkedQtlib = false
var qtlibs = map[string]FFILibrary{}

func check_linked_qtmod() bool {
	// images := cgopp.DyldImages()
	// _QCompileVersion
	rv := cgopp.Dlsym0("_QCompileVersion")
	return rv != nil
}

// func FindModule(modname string) (string, error) {
// 	modname = "Core"
// 	dlh, err := purego.Dlopen(modname, purego.RTLD_LAZY)
// 	gopp.ErrPrint(err, modname)
// 	log.Println(dlh)

// 	return modname, nil
// }

func loadModule(libpath string, modname string) (err error) {
	err = loadModuleImpl(libpath, modname)
	if err == nil {
		err = loadModuleImpl(libpath, modname+"Inline")
	}
	return
}
func loadModuleImpl(libpath string, modname string) error {
	// must endwiths /
	// todo LD_LIBRARY_PATH
	// todo DYLD_LIBRARY_PATH
	// todo windows...
	// todo diffenece os, diffence libdirs/fnames
	libdirs := []string{"", "./", "/opt/qt/lib/", "/usr/lib/", "/usr/lib64/", "/usr/local/lib/", "/usr/local/opt/qt/lib/", gopp.Mustify1(os.UserHomeDir()) + "/.nix-profile/lib/"}
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
