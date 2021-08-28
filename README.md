# go src reading

std
- [x] fmt.Print
- [x] log.Print
- [x] strings.Builder, strings.Join
- [x] bufio(reader,writer)
- [ ] sort
- [ ] heap
- [x] strconv.Itoa, strconv.Atoi
- [x] rand.Shuffle

sync
- [x] sync.Once
- [x] sync.RWMutex
- [x] sync.WaitGroup
- [x] sync.Cond
- [x] sync.Map
- [ ] sync.Pool
- [ ] SingleFlight

internal/runtime
- [x] netpoll
- [ ] memory

third party
- [x] gin
- [x] fasthttp(workerpool)
- [ ] zap
- [ ] etcd



```go
xfmt.Print([]byte("hello\n"))

xlog.Print("hello")

var builder xstrings.Builder
builder.Write([]byte("hello"))
xlog.Print(builder.String())

s := xstrings.Join([]string{"1", "2"}, ",")
xlog.Print(s)

r := xbufio.NewReader(os.Stdin)
w := xbufio.NewWriter(os.Stdout)
var b = make([]byte, 10)
r.Read(b)
w.Write(b)
w.Flush()

// strconv.Itoa, strconv.Itoa
fmt.Println(xstrconv.Atoi("-11111111111111111"))
fmt.Println(xstrconv.Itoa(11111111111))
```

sync.Map
```go
func main() {
	var m xsync.Map
	m.Store("1", "1")
	m.Store("2", "2")
	m.Store("3", "3")
	// read:
	// dirty: 1 2 3

	for i := 0; i < 100; i++ {
		m.Load("4")
	}
	// 发生dirty -> read
	// read: 1 2 3
	// dirty:

	m.Store("4", "4")
	// read: 1 2 3
	// dirty: 1 2 3 4

	m.Delete("2")
	// read: 1 nil 3
	// dirty: 1 nil 3 4

	for i := 0; i < 100; i++ {
		m.Load("4")
	}
	// 发生dirty -> read
	// read: 1 nil 3 4
	// dirty:

	fmt.Println(m.Load("2")) // <nil> false
}
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

fasthttp(workerpool)
```go
func main() {
	xfasthttp.ListenAndServe(":3000")
}
```

