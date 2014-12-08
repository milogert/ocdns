package main

import (
  "database/sql"
  "encoding/json"
  "fmt"
  "github.com/go-martini/martini"
  "github.com/martini-contrib/render"
  "github.com/martini-contrib/sessions"
  _ "github.com/mattn/go-sqlite3"
  "log"
  "os"
  "os/exec"
  "strconv"
  "time"
  "net/http"
  "io"
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
  rs, aErr := aConn.Query("SELECT * FROM `Submissions` WHERE `Submissions`.`timestamp` >= " + aTime.Format(aLayout))
  if aErr != nil {
    log.Fatal(err)
  }
  defer rs.Close()



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

  /* Judge route **************************************************************/
  m.Get("/judge", func(r render.Render, session sessions.Session) {
    //role := session.Get("role")

    //if role == "Judge" {
    r.HTML(200, "judge", "test")
    //} else {
    //  r.HTML(302, "index", "test")
    //}
  })

  /* Admin route **************************************************************/
  m.Get("/admin", func(r render.Render, session sessions.Session) {
    //role := session.Get("role")

    //if role == "Admin" {
    r.HTML(200, "admin", "test")
    //} else {
    //  r.HTML(302, "index", "test")
    //}
  })

  /* Player route *************************************************************/
  m.Get("/player", func(r render.Render, session sessions.Session) {
    role := ""//session.Get("role")

    if role == "player" || true {
      aConn, err := sql.Open("sqlite3", "ocdns.db")
      defer aConn.Close()

      // Query the database.
      rs, aErr := aConn.Query("SELECT problem_id, name, question FROM `Problems`;")
      if aErr != nil {
        log.Fatal(err)
      }
      defer rs.Close()

      // Return the results in a json object
      var questions = make(map[string]map[string]string)
      var tmp = make(map[string]string)
      for rs.Next() {
        var problem_id int
        var name string
        var question string

        rs.Scan(&problem_id, &name, &question)

        tmp = make(map[string]string)
        tmp["name"] = name
        tmp["question"] = question
        questions[strconv.Itoa(problem_id)] = tmp
      }

      // Query the database.
      rs, aErr = aConn.Query("SELECT language_id, name FROM `Language`;")
      if aErr != nil {
        log.Fatal(err)
      }
      defer rs.Close()

      // Return the results in a json object
      var languages = make(map[string]map[string]string)
      for rs.Next() {
        var language_id int
        var name string

        rs.Scan(&language_id, &name)

        tmp = make(map[string]string)
        tmp["name"] = name
        languages[strconv.Itoa(language_id)] = tmp
      }

      // Create a map to return.
      var aRet = make(map[string]map[string]map[string]string)

      aRet["questions"] = questions
      aRet["languages"] = languages

      // Conver the array into json.
      aJson, aErr := json.Marshal(aRet)

      r.HTML(200, "player", string(aJson))
    } else {
     r.HTML(302, "index", nil)
    }
  })

  /* Player submission route **************************************************/
  m.Get("/api/submitCode", func(r render.Render, h http.Request) string {
    // Get the team id.
    team_id := 0//session.Get("team_id")

    // if team_id == nil {
    //  return
    // }


    // Open the database.
    aConn, _ := sql.Open("sqlite3", "ocdns.db")
    defer aConn.Close()


    // Get form stuff.
    multi_err := h.ParseMultipartForm(100000)
    if multi_err != nil {
      return multi_err.Error()
    }

    problem_id := h.MultipartForm.File["question"]
    language := h.MultipartForm.File["language"]
    file := h.MultipartForm.File["file"]

    var filepath = "./" + string(team_id) + "/" + file[0].Filename


    // Save the file.
    log.Println("getting handle to file")
    file_s, open_err := file[0].Open()
    defer file_s.Close()
    if open_err != nil {
      return open_err.Error()
    }

    log.Println("creating destination file")
    dst, create_err := os.Create("./" + string(team_id) + "/" + file[0].Filename)
    defer dst.Close()
    if create_err != nil {
      return create_err.Error()
    }

    log.Println("copying the uploaded file to the destination file")
    if _, copy_err := io.Copy(dst, file_s); copy_err != nil {
      return copy_err.Error()
    }


    // Get the language.
    rs, query_err := aConn.Query(`
      SELECT compiler, c_flags, interpreter, i_flags
      FROM Language
      WHERE name = ` + language + `
      LIMIT 1;
    `)
    if query_err != nil {
      log.Fatal(query_err)
    }
    defer rs.Close()

    // Get the next (only) row.
    rs.Next()

    // Set up the variables.
    var compiler string
    var c_flags string
    var interpreter string
    var i_flags string

    // Scan the row and plop them in the variables.
    rs.Scan(&compiler, &c_flags, &interpreter, &i_flags)


    // Run the file.
    c_err := 0
    i_err := 0
    // Find if it needs a compiler or not.
    if compiler != "" {
      cmd := exec.Command(compiler, c_flags, filepath)
      cmd.Stdin = os.Stdin
      cmd.Stdout = os.Stdout
      cmd.Stderr = os.Stderr
      c_err := cmd.Run()

      // Capture err and do things with it.
      fmt.Println(err)
    }

    // Run the interpreter.
    // TODO: Find the file path of the output.
    // cmd := exec.Command(interpreter, i_flags, output)
    // cmd.Stdin = os.Stdin
    // cmd.Stdout = os.Stdout
    // cmd.Stderr = os.Stderr
    // i_err := cmd.Run()


    // Get the problem.
    rs, query_err = aConn.Query(`
      SELECT question, answer
      FROM Problem
      WHERE problem_id = ` + problem_id + `
      LIMIT 1;
    `)
    if query_err != nil {
      log.Fatal(query_err)
    }
    defer rs.Close()

    // Get the next (only) row.
    rs.Next()

    // Set up the variables.
    var question string
    var answer string

    // Scan the row and plop them in the variables.
    rs.Scan(&question, &answer)


    // Diff the output of the files.
    diff := false
    if c_err == 0 && i_err == 0 {
      diff = answer == output
    }


    // Decide whether or not to auto judge.
    auto_judge := true

    if diff {
      // The diff of the answer and the submission returned 0.
      rs, query_err = aConn.Query(`
        INSERT INTO Submission(team_id, problem_id, judged)
        VALUES(` + team_id + `, ` + problem_id + `, 1);
      `)
      if query_err != nil {
        log.Fatal(query_err)
      }
      defer rs.Close()

      return problem_id, true, "Auto-judged"
    } else if c_err != 0 {
      // Compiler returned non 0.
      rs, query_err = aConn.Query(`
        INSERT INTO Submission(team_id, problem_id, judged, correct)
        VALUES(` + team_id + `, ` + problem_id + `, 1, 1);
      `)
      if query_err != nil {
        log.Fatal(query_err)
      }
      defer rs.Close()

      return problem_id, true, "Auto-judged"
    } else if c_err == 0 && i_err != 0 {
      // Compiler returned 0, interpreter returned non 0.
      rs, query_err = aConn.Query(`
        INSERT INTO Submission(team_id, problem_id, judged)
        VALUES(` + team_id + `, ` + problem_id + `, 0);
      `)
      if query_err != nil {
        log.Fatal(query_err)
      }
      defer rs.Close()

      return problem_id, false, "Being judged"
    } else {
      panic("uh oh");
    }
  })

  /* Judge: Get submissions ***************************************************/
  m.Get("/api/getSubmissions", func(r render.Render) string {
    aConn, err := sql.Open("sqlite3", "ocdns.db")

    // Set up the arguments.
    aTime := time.Now()
    const aLayout = "Jan 2, 2006 at 3:04pm (MST)"
    //aArgs := sql.NamedArgs{"$time": aTime.Format(aLayout)}

    // Query the database.
    // aConn.Exec("SELECT * FROM submissions WHERE submitted >= $time", aArgs)
    rs, aErr := aConn.Query("SELECT * FROM `Submissions` WHERE `Submissions`.`timestamp` >= " + aTime.Format(aLayout))
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
