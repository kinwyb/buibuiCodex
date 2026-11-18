#！/bin/sh

# 使用方法 build.sh 操作系统 系统架构(amd64)

# 获取操作系统
if [ -n "$1" ]; then
  os=$1
else
  os=$(uname -s | awk '{print tolower($0)}')
fi
# 获取系统架构
if [ -n "$2" ]; then
  arch=$2
else
  arch=amd64
fi

OUTPUT_NAME="buibuiCodex"
if [ "$os" = "linux" ]; then
  OUTPUT_NAME="${OUTPUT_NAME}_linux"
elif [ "$os" = "windows" ]; then
  OUTPUT_NAME="${OUTPUT_NAME}.exe"
fi

echo "编译参数: CGO_ENABLED=0 GOOS=${os} GOARCH=${arch}  go build ."
echo "开始编译"
CGO_ENABLED=0 GOOS=${os} GOARCH=${arch}  go build -o "./${OUTPUT_NAME}" .
echo "编译结束"