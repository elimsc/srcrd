# go src reading

- [x] fmt.Print
- [x] log.Print
- [x] strings.Builder, strings.Join
- [ ] bufio

- [x] gin
- [ ] zap


```go
xfmt.Print([]byte("hello\n"))

xlog.Print("hello")

var builder xstrings.Builder
builder.Write([]byte("hello"))
xlog.Print(builder.String())

s := xstrings.Join([]string{"1", "2"}, ",")
xlog.Print(s)
```

gin
```go
func main() {
	r := xgin.Default()
	r.Use(Logger())
	r.GET("/", func(c *xgin.Context) {
		c.String("hello")
	})
	r.Run() // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")
}

func Logger() xgin.HandlerFunc {
	return func(c *xgin.Context) {
		t := time.Now()
		c.Set("k1", "v1")
		c.Next()
		latency := time.Since(t)
		log.Println(latency)
		v, _ := c.Get("k1")
		log.Println(v)
	}
}
```

