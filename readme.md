# Build
```
CGO_CFLAGS=-I/tmp/grpc CGO_LDFLAGS=-L/tmp/grpc/lib go build  -o server *.go
```

# Install

```
export CGO_LDFLAGS=-L/tmp/grpc/lib
```

# Run

# Reference

`https://www.cnblogs.com/hongjijun/p/13724738.html`
