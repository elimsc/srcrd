# go src reading

- [x] fmt.Print
- [x] log.Print
- [x] strings.Builder, strings.Join


```go
xfmt.Print([]byte("hello\n"))

xlog.Print("hello")

var builder xstrings.Builder
builder.Write([]byte("hello"))
xlog.Print(builder.String())

s := xstrings.Join([]string{"1", "2"}, ",")
xlog.Print(s)
```

