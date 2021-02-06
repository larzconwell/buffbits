`buffbits`
---

[![License](https://img.shields.io/github/license/larzconwell/buffbits)](/LICENSE)
[![Go Reference](https://pkg.go.dev/badge/github.com/larzconwell/buffbits.svg)](https://pkg.go.dev/github.com/larzconwell/buffbits)
[![Lint/Test](https://github.com/larzconwell/buffbits/workflows/lint-test/badge.svg)](https://github.com/larzconwell/buffbits/actions)

`buffbits` provides a buffered reader/writer that exposes bit level io access.

### Install

```shell
go get github.com/larzconwell/buffbits
```

### Byte Ordering
Data that is being written or read is done so using big-endian ordering. Below is a visual representation of the data being written and read when using the `buffbits` package.

```go
var out bytes.Buffer

writer := buffbits.NewWriter(&out)
writer.Write(0b0001, 4)
writer.Write(0b00000001, 8)
writer.Flush()
err := writer.Err()
if err != nil {
    fmt.Fprintln(os.Stderr, err)
}

// Once the 2  writes and the flush occurs, the buffer `out` will contain the following bytes.
//
//  0  0  0  1  0  0  0  0    0  0  0  1  0  0  0  0
// |        byte 1        |  |        byte 2        |
// |  write 1 ||         write 2        ||  padding |

reader := buffbits.NewReader(&out)
reader.Read(6) // 0 0 0 1 0 0
reader.Read(4) // 0 0 0 0
reader.Read(2) // 0 1
reader.Read(4) // 0 0 0 0
err := reader.Err()
if err != nil {
    fmt.Fprintln(os.Stderr, err)
}
```
