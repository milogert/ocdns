package main

import (
  "github.com/go-martini/martini"
  "github.com/martini-contrib/render"
  "github.com/mattn/go-sqlite3"
  "time"
  "encoding/json"
  "strconv"
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
    aConn, err := sql.Open("ocdns.db")

    // Set up the arguments.
    aTime = time.Now()
    const aLayout = "Jan 2, 2006 at 3:04pm (MST)"
    //aArgs := sql.NamedArgs{"$time": aTime.format(aLayout)}

    // Query the database.
    // aConn.Exec("SELECT * FROM submissions WHERE submitted >= $time", aArgs)
    rs, aErr := aConn.Query("SELECT * FROM `Submissions` WHERE `Submissions`.`timestamp` >= " +  aTime.format(aLayout))
    if err != nil {
      log.Fatal(err)
    }
    defer rs.Close()

    // Return the results in a json object
    var aRet [][4]string  
    var i int 
    i := 0
    for rs.Next() {
      var submission_id int
      var team_id int
      var problem_id int
      var judge_id int
      var judged int 
      var submitTime Time 

      rs.Scan(&submission_id, &team_id, &problem_id, &judge_id, &judged, &submitTime)

      aRet[i][0] = Itoa(submission_id)
      aRet[i][1] = Itoa(team_id)
      aRet[i][2] = Itoa(problem_id)
      aRet[i][3] = Itoa(judge_id)
      aRet[i][4] = Itoa(judged)
      aRet[i][5] = submitTime.format(aLayout)

      i++
    }
    aJson, aErr := json.Marshal(aRet)
    return aJson
  })

  m.Run()
}
