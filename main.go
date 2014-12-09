package main

import (
  "crypto/sha1"
  "database/sql"
  "encoding/base64"
  "encoding/json"
  "fmt"
  "github.com/go-martini/martini"
  "github.com/martini-contrib/binding"
  "github.com/martini-contrib/render"
  "github.com/martini-contrib/sessions"
  "io"
  "log"
  "mime/multipart"
  "os"
  "os/exec"
  "strconv"
  "time"
  _ "github.com/mattn/go-sqlite3"
)

/* Structs ********************************************************************/
type FileUpload struct {
  ProblemNum  int                   `form:"problemNum"  binding:"required"`
  Language    int                   `form:"language"    binding:"required"`
  File        *multipart.FileHeader `form:"file"        binding:"required"`
}

type UploadResponse struct {
  ProblemId string `json:"problem_id"`
  IsJudged bool `json:"is_judged"`
  JudgeMessage string `json:"message"`
  Correct bool `json:"correct"`
}


/******************************************************************************/
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


/******************************************************************************/
func genAdmin() bool {
  // Create a hasher and generate a password.
  hasher := sha1.New()
  now := fmt.Sprint(time.Now())
  hasher.Write([]byte(now))
  sha := base64.URLEncoding.EncodeToString(hasher.Sum(nil))

  // Insert the password into the database.
  aConn, _ := sql.Open("sqlite3", "ocdns.db")
  defer aConn.Close()
  rs, aErr := aConn.Query(`
    INSERT INTO User
    VALUES(0, "admin", "` + sha + `", "Admin", "Admin", "admin", -1);
  `)
  if aErr != nil {
    log.Fatal(aErr)
    return false
  }
  defer rs.Close()

  // Report the password to the user.
  fmt.Println("!! Password for the admin account is: " + sha)

  return true
}


func getLanguage(language string) (string, string, string, string, string, string) {
  // Open the database.
  aConn, _ := sql.Open("sqlite3", "ocdns.db")
  defer aConn.Close()

  // Get the language.
  rs, query_err := aConn.Query(`
    SELECT compiler, c_flags, c_type, interpreter, i_flags, i_type
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
  var c_type string
  var interpreter string
  var i_flags string
  var i_type string

  // Scan the row and plop them in the variables.
  rs.Scan(&compiler, &c_flags, &c_type, &interpreter, &i_flags, &i_type)

  return compiler, c_flags, c_type, interpreter, i_flags, i_type
}

/******************************************************************************/
func main() {
  // Reset the database.
  //resetDb()

  // Create the default admin account.
  gen := genAdmin()
  if !gen {
    log.Print("-- Admin account not created.")
  }


  // TODO: Use config file?


  // Get the Martini instance.
  m := martini.Classic()

  // Create session store.
  now := fmt.Sprint(time.Now())
  store := sessions.NewCookieStore([]byte(now))
  m.Use(sessions.Sessions("ocdns", store))

  // Set a layout.
  m.Use(render.Renderer(render.Options{
    Layout: "layout",
  }))

  /* Index route **************************************************************/
  m.Get("/", func(r render.Render, session sessions.Session) {
    // TODO: Perform checks to see if the user is logged in. If they are, then
    // render a specific page based on their role.
    role := session.Get("role")

    if role == "admin" {
      r.HTML(302, "admin", "")
    } else if role == "judge" {
      r.HTML(302, "judge", "")
    } else if role == "player" {
      r.HTML(302, "player", "")
    } else {
      r.HTML(200, "index", "")
    }
  })

  /* Judge route **************************************************************/
  m.Get("/judge", func(r render.Render, session sessions.Session) {
    //role := session.Get("role")

    //if role == "judge" {
    r.HTML(200, "judge", "test")
    //} else {
    //  r.HTML(302, "index", "test")
    //}
  })

  /* Admin route **************************************************************/
  m.Get("/admin", func(r render.Render, session sessions.Session) {
    //role := session.Get("role")

    //if role == "admin" {
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
      aJson, _ := json.Marshal(aRet)

      r.HTML(200, "player", string(aJson))
    } else {
     r.HTML(302, "index", nil)
    }
  })

  /* Player: Submit code ******************************************************/
  m.Post("/api/submitCode", binding.MultipartForm(FileUpload{}), func(uf FileUpload) string {
    // Get the team id.
    team_id := string(0)//string(session.Get("team_id"))

    // if team_id == nil {
    //  return
    // }


    // Open the database.
    aConn, _ := sql.Open("sqlite3", "ocdns.db")
    defer aConn.Close()


    // Get form stuff.
    problem_id := string(uf.ProblemNum)
    language := string(uf.Language)
    file, err := uf.File.Open()
    if err != nil {
      return err.Error()
    }
    defer file.Close()


    // Create file path, without extension.
    filepath := "./submissions/" + team_id + "/" + problem_id


    // Get the language.
    compiler, c_flags, c_type, interpreter, i_flags, i_type := getLanguage(language)


    // Save the file.
    dst, create_err := os.Create(filepath + "." + c_type)
    defer dst.Close()
    if create_err != nil {
      return create_err.Error()
    }

    // Copy the file into the destination.
    if _, copy_err := io.Copy(dst, file); copy_err != nil {
      return copy_err.Error()
    }


    // Run the file.
    // var c_out []byte
    var c_err error
    var i_out []byte
    var i_err error

    // Find if it needs a compiler or not.
    if compiler != "" {
      _, c_err = exec.Command(compiler, c_flags, filepath + "." + c_type).Output()
    }

    // Run the interpreter.
    i_out, i_err = exec.Command(interpreter, i_flags, filepath + "." + i_type).Output()


    // Get the problem.
    rs, query_err := aConn.Query(`
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
    if c_err == nil && i_err == nil {
      diff = answer == string(i_out)
    }


    // Declare the struct to return json.
    ret := UploadResponse{}
    var j []byte

    // TODO: Make sure the problem doesn't already exist.

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

      // Put the items in the struct declared above.
      ret.ProblemId = problem_id
      ret.IsJudged = true
      ret.JudgeMessage = "Auto-Judged"
      ret.Correct = true

      // Marshal the JSON.
      j, _ = json.Marshal(ret)

      // Return the JSON as a string.
      return string(j)
    } else if c_err != nil {
      // Compiler returned non 0.
      rs, query_err = aConn.Query(`
        INSERT INTO Submission(team_id, problem_id, judged, correct)
        VALUES(` + team_id + `, ` + problem_id + `, 1, 1);
      `)
      if query_err != nil {
        log.Fatal(query_err)
      }
      defer rs.Close()

      // Put the items in the struct declared above.
      ret.ProblemId = problem_id
      ret.IsJudged = true
      ret.JudgeMessage = "Auto-Judged"
      ret.Correct = false

      // Marshal the JSON.
      j, _ = json.Marshal(ret)

      // Return the JSON as a string.
      return string(j)
    } else if c_err == nil && i_err != nil {
      // Compiler returned 0, interpreter returned non 0.
      rs, query_err = aConn.Query(`
        INSERT INTO Submission(team_id, problem_id, judged)
        VALUES(` + team_id + `, ` + problem_id + `, 0);
      `)
      if query_err != nil {
        log.Fatal(query_err)
      }
      defer rs.Close()

      // Put the items in the struct declared above.
      ret.ProblemId = problem_id
      ret.IsJudged = false
      ret.JudgeMessage = "Being judged currently. Expect a response soon."
      ret.Correct = false

      // Marshal the JSON.
      j, _ = json.Marshal(ret)

      // Return the JSON as a string.
      return string(j)
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
    aJson, _ := json.Marshal(aRet)

    // Return the json generated.
    return string(aJson)
  })

  m.Run()
}
