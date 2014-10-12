package main

import (
	"github.com/go-martini/martini"
	"github.com/martini-contrib/render"
)

func main() {
	m := martini.Classic()

	// Set a layout.
	m.Use(render.Renderer(render.Options{
		Layout: "layout",
	}))

	m.Get("/", func(r render.Render) {
		r.HTML(200, "index")
	})

	m.Get("/hello", func(r render.Render) {
		r.HTML(200, "hello", "test")
	})

	m.Get("/login", func())

	m.Run()
}
