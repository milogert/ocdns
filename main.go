package main

import (
  "github.com/go-martini/martini"
  "github.com/martini-contrib/render"
  "code.google.com/p/go-sqlite/go1/sqlite3"
  "time"
  "encoding/json"
)

func main() {
  m := martini.Classic()

  // Set a layout.
  m.Use(render.Renderer(render.Options{
    Layout: "layout",
  }))

  m.Get("/", func(r render.Render) {
    // TODO: Perform checks to see if the user is logged in. If they are, then
    // render a specific page based on their role.
    r.HTML(200, "index", "")
  })

  m.Get("/login", func(r render.Render) {
    // Do stuff.

    // TODO: Check what the user's role and return a page depending upon that.

    // r.HTML(200, "/admin", "")
    // r.HTML(200, "/judge", "")
    // r.HTML(200, "/contestant", "")
  })

  m.Get("/judge", func(r render.Render) {
    r.HTML(200, "judge", "test")
  })

  m.Get("/api/getSubmissions", func(r render.Render) {
    aConn, _ := sqlite3.Open("ocdns.db")

    // Set up the arguments.
    aTime = time.Now()
    const aLayout = "Jan 2, 2006 at 3:04pm (MST)"
    aArgs := sqlite3.NamedArgs{"$time": aTime.format(aLayout)}

    // Query the database.
    // aConn.Exec("SELECT * FROM submissions WHERE submitted >= $time", aArgs)
    aRes, aErr := aConn.Query("SELECT * FROM submissions WHERE submitted >= $time", aArgs)

    // Return the results in a json object.
    var aRet map[string]RowMap
    aRow := make(sqlite3.RowMap)
    for aRes, aErr, aErr == nil; err = aRes.next() {
      // var aId int64
      // var aSubmitTime time
      // var aSubmit string
      // var aProblem int64

      aRes.Scan(&aId, aRow)
      aRet[aId] = aRow
    }
    aJson, aErr := json.Marshal(aRet)
    return aJson
  })

  m.Run()
}
