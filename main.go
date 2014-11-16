package main

import (
  "github.com/go-martini/martini"
  "github.com/martini-contrib/render"
  "github.com/martini-contrib/sessions"
  "github.com/mattn/go-sqlite3"
  "time"
  "encoding/json"
  "os"
  "os/exec"
  "fmt"
)

func resetDb() {
  // Remove the old database.
  fmt.Println("ii Removing the old database.")
  os.Remove("./db.db")

  // Re-create the database.
  fmt.Println("ii Re-creating the database.")
  cmd := exec.Command("/usr/bin/sqlite3", "ocdns.db", "<", "db.sql")
  cmd.Stdin = os.Stdin
  cmd.Stdout = os.Stdout
  cmd.Stderr = os.Stderr
  err := cmd.Run()

  // Check the output.
  if err != nil {
    fmt.Println("-- Failed to re-create the database.")
    //fmt.Println("Output: " + err)
    os.Exit(1)
  }

  fmt.Println("ii Successfully created the new file.")
}

func main() {
  // Reset the database.
  resetDb()

  // Get the Martini instance.
  m := martini.Classic()

  // Create session store.
  store := sessions.NewCookieStore([]byte("doublesecret"))
  m.Use(sessions.Sessions("ocdns", store))

  // Set a layout.
  m.Use(render.Renderer(render.Options{
    Layout: "layout",
  }))

  m.Get("/", func(r render.Render) {
    // TODO: Perform checks to see if the user is logged in. If they are, then
    // render a specific page based on their role.
    role := session.Get("role")

    if role == "Admin" {
      r.HTML(302, "admin", "")
    } else if role == "Judge" {
      r.HTML(302, "judge", "")
    } else if role == "Player" {
      r.HTML(302, "player", "")
    } else {
      r.HTML(200, "index", "")
    }
  })

  m.Get("/judge", func(r render.Render) {
    role := session.Get("role")

    if role == "Judge" {
      r.HTML(200, "judge", "test")
    } else {
      r.HTML(302, "index", "test")
    }
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
    for aRes.Next() {
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
