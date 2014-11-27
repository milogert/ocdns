package main

import (
  "database/sql"
  "github.com/go-martini/martini"
  "github.com/martini-contrib/render"
  "github.com/martini-contrib/sessions"
  _ "github.com/mattn/go-sqlite3"
  "time"
  "encoding/json"
  "os"
  "os/exec"
  "fmt"
  "log"
  "strconv"
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
  //resetDb()
  
  // TODO: Create default admin account.
  // TODO: Use config file?

  // Get the Martini instance.
  m := martini.Classic()

  // Create session store.
  store := sessions.NewCookieStore([]byte("doublesecret"))
  m.Use(sessions.Sessions("ocdns", store))

  // Set a layout.
  m.Use(render.Renderer(render.Options{
    Layout: "layout",
  }))

  m.Get("/", func(r render.Render, session sessions.Session) {
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

  m.Get("/judge", func(r render.Render, session sessions.Session) {
    //role := session.Get("role")

    //if role == "Judge" {
      r.HTML(200, "judge", "test")
    //} else {
    //  r.HTML(302, "index", "test")
    //}
  })

  m.Get("/admin", func(r render.Render, session sessions.Session) {
    //role := session.Get("role")

    //if role == "Admin" {
      r.HTML(200, "admin", "test")
    //} else {
    //  r.HTML(302, "index", "test")
    //}
  })
  
  m.Get("/api/getSubmissions", func(r render.Render) []byte {
    aConn, err := sql.Open("sqlite3", "ocdns.db")

    // Set up the arguments.
    aTime := time.Now()
    const aLayout = "Jan 2, 2006 at 3:04pm (MST)"
    //aArgs := sql.NamedArgs{"$time": aTime.Format(aLayout)}

    // Query the database.
    // aConn.Exec("SELECT * FROM submissions WHERE submitted >= $time", aArgs)
    rs, aErr := aConn.Query("SELECT * FROM `Submissions` WHERE `Submissions`.`timestamp` >= " +  aTime.Format(aLayout))
    if aErr != nil {
      log.Fatal(err)
    }
    defer rs.Close()

    // Return the results in a json object
    var aRet [][6]string  
    var i int 
    i = 0
    for rs.Next() {
      var submission_id int
      var team_id int
      var problem_id int
      var judge_id int
      var judged int
      var submitTime string

      rs.Scan(&submission_id, &team_id, &problem_id, &judge_id, &judged, &submitTime)

      aRet[i][0] = strconv.Itoa(submission_id)
      aRet[i][1] = strconv.Itoa(team_id)
      aRet[i][2] = strconv.Itoa(problem_id)
      aRet[i][3] = strconv.Itoa(judge_id)
      aRet[i][4] = strconv.Itoa(judged)
      aRet[i][5] = aLayout

      i++
    }
    
    // Conver the array into json.
    aJson, aErr := json.Marshal(aRet)
    
    // Return the json generated.
    return aJson
  })

  m.Run()
}
 	
