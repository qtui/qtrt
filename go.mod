module qtrt

go 1.22.3

require github.com/qtui/qtqt v0.0.0

replace github.com/qtui/qtqt => ../qtqt

require github.com/qtui/miscutil v0.0.0

require (
	github.com/Workiva/go-datastructures v1.1.3 // indirect
	github.com/bitly/go-simplejson v0.5.1 // indirect
	github.com/cheekybits/genny v1.0.0 // indirect
	github.com/dolthub/maphash v0.1.0 // indirect
	github.com/ebitengine/purego v0.7.1
	github.com/emirpasic/gods v1.18.1 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/huandu/xstrings v1.4.0 // indirect
	github.com/kitech/dl v0.0.0-20201225001532-be4f4faa4070
	github.com/kitech/gopp v0.0.0
	github.com/lytics/base62 v0.0.0-20180808010106-0ee4de5a5d6d // indirect
	github.com/pkg/errors v0.9.1 // indirect
	golang.org/x/sys v0.19.0 // indirect
)

replace github.com/qtui/miscutil => ../miscutil

replace github.com/qtui/qtclzsz => ../qtclzsz

replace github.com/qtui/qtsyms => ../qtsyms

require (
	github.com/kitech/gopp/cgopp v0.0.0
	github.com/orcaman/concurrent-map/v2 v2.0.1
	github.com/qtui/qtclzsz v0.0.0
	github.com/qtui/qtsyms v0.0.0
)

replace github.com/kitech/gopp => ../../goplusplus

replace github.com/kitech/gopp/cgopp => ../../goplusplus/cgopp
