if [ ${GOARCH} == amd64 ]
CC=aarch64-linux-gnu-gcc
fi
CGO_ENABLED=1 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build -tags musl --ldflags "-extldflags -static" -o producer