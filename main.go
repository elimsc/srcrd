package main

import (
	"log"
	"srcrd/xgin"
	"time"
)

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
		c.Next()
		latency := time.Since(t)
		log.Print(latency)
	}
}
