# port consts from old qt.go
# only read and output console
# usage: portconsts.sh <core> > consts.go
basedir=~/bprog/qt.go/
qtmod=$1

echo "package qt${qtmod}"

grep -E "(^const Q|^type Q.*=.int)" ${basedir}/qt${qtmod}/*.go |awk -F: '{print $2}'

# todo 下次生成 const 的时候，搞个实例加成员变量，用起来就友好一点
# usage: pkg.instance.name
#   qtwidgets.QSizePolicy_.Expanding