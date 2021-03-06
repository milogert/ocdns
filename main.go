package main

import (
  "crypto/sha1"
  "database/sql"
  "encoding/base64"
  "encoding/json"
  "errors"
  "flag"
  "fmt"
  "github.com/go-martini/martini"
  "github.com/martini-contrib/binding"
  "github.com/martini-contrib/render"
  "github.com/martini-contrib/sessions"
  "io"
  "io/ioutil"
  "log"
  "mime/multipart"
  "os"
  "os/exec"
  "strconv"
  "time"
  _ "github.com/mattn/go-sqlite3"
)

/* Structs ********************************************************************/
type LoginForm struct {
  Username    string    `form:"username" binding:"required"`
  Password    string    `form:"password" binding:"required"`
}

type Context struct {
  Id          int       `json:"id"`
  Username    string    `json:"username"`
  NameFirst   string    `json:"name_first"`
  NameLast    string    `json:"name_last"`
  Role        string    `json:"role"`
  TeamId      int       `json:"team_id"`
}

type FileUpload struct {
  ProblemNum  int                   `form:"problemNum"  binding:"required"`
  Language    int                   `form:"language"    binding:"required"`
  File        *multipart.FileHeader `form:"file"        binding:"required"`
}

type UploadResponse struct {
  ProblemId     string  `json:"problem_id"`
  IsJudged      bool    `json:"is_judged"`
  JudgeMessage  string  `json:"message"`
  Correct       bool    `json:"correct"`
}


/******************************************************************************/
func resetDb() {
  // Remove the old database.
  log.Print("ii Removing the old database.")
  os.Remove("./ocdns.db")

  binary, err := exec.LookPath("sqlite3")
  if err != nil {
    log.Panic(err)
  }

  // Re-create the database.
  log.Print("ii Re-creating the database with " + binary)

  // Create the system call.
  cmd := exec.Command(binary, "ocdns.db")

  // Read the database file in.
  file, err := ioutil.ReadFile("./db.sql")
  if err != nil {
    log.Panic(err)
  }

  // Create pipe and write everything into the pipe.
  stdin, err := cmd.StdinPipe()
  if err != nil {
    log.Panic(err)
  }

  // Set the io stuff.
  cmd.Stdout = os.Stdout
  cmd.Stderr = os.Stderr

  // Start the command.
  if err = cmd.Start(); err != nil {
    log.Panic(err)
  }

  // Write the file in.
  io.WriteString(stdin, string(file))
  stdin.Close()

  log.Print("ii After write.")

  // Wait for the process.
  err = cmd.Wait()
  if err != nil {
    log.Panic(err)
  }

  log.Print("ii Successfully created the new database.")
}


/******************************************************************************/
func genAdmin(s string) error {
  // Create a hasher and generate a password.
  hasher := sha1.New()
  now := fmt.Sprint(time.Now())
  hasher.Write([]byte(now))

  var pwd string
  if s == "" {
    pwd = base64.URLEncoding.EncodeToString(hasher.Sum(nil))
  } else {
    pwd = s
  }

  // Insert the password into the database.
  aConn, _ := sql.Open("sqlite3", "ocdns.db")
  defer aConn.Close()

  // Check to see if the admin account exists
  rs, err := aConn.Query("SELECT password FROM User WHERE username = 'admin'")
  if err != nil {
    log.Fatal(err)
  }
  if b := rs.Next(); b == true {
    var pwd string
    rs.Scan(&pwd)
    log.Print("ii Admin account already exists with password " + pwd)

    return errors.New("account error: admin account already exists")
  } else {
    _, err = aConn.Exec(`
      INSERT INTO User
      VALUES(0, "admin", "` + pwd + `", "Admin", "Admin", "admin", -1);
    `)
    if err != nil {
      log.Fatal(err)
      return errors.New("bad database: admin account not created")
    }

    // Report the password to the user.
    log.Print("!! Admin password: " + pwd)

    return nil
  }
}

/******************************************************************************/
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
  // Define some command line flags.
  var reset bool
  flag.BoolVar(&reset, "r", false, "Resets the database and generates an admin account.")

  var pwd string
  flag.StringVar(&pwd, "p", "", "Provides the password for the admin account.")

  // Parse the flags.
  flag.Parse()

  // Reset the database.
  _, err := os.Stat("ocdns.db")
  if reset || os.IsNotExist(err) {
    resetDb()
  }

  // Create the default admin account.
  if pwd != "" || reset {
    err = genAdmin(pwd)
    if err != nil {
      log.Print(err)
    }
  }


  // TODO: Use config file?


  // Get the Martini instance.
  m := martini.Classic()

  // Create session store.
  store := sessions.NewCookieStore([]byte("ocdns"))
  m.Use(sessions.Sessions("my_session", store))

  // Set a layout.
  m.Use(render.Renderer(render.Options{
    Layout: "layout",
  }))

  /* Index route **************************************************************/
  m.Get("/", func(r render.Render, session sessions.Session) {
    // TODO: Perform checks to see if the user is logged in. If they are, then
    // render a specific page based on their role.
    role := session.Get("role").(string)

    // if role == "admin" {
    //   r.HTML(302, "admin", "")
    // } else if role == "judge" {
    //   r.HTML(302, "judge", "")
    // } else if role == "player" {
    //   r.HTML(302, "player", "")
    // } else {
      r.HTML(200, "index", "")
    // }
  })

  m.Get("/login", binding.Bind(LoginForm{}), func(r render.Render, session sessions.Session, form LoginForm) string {
    // Get info from the database.
    conn, err := sql.Open("sqlite3", "ocdns.db")
    defer conn.Close()

    // Prepare the statement.
    stmt, err := conn.Prepare(`
      SELECT user_id, username, name_first, name_last, role, team_id
      FROM User
      WHERE username = ? AND password = ?
      LIMIT 1;
    `)
    if err != nil {
      log.Fatal(err)
    }
    defer stmt.Close()

    // Query the database and set appropriate items if a row was actually returned.
    var id string
    var username string
    var name_first string
    var name_last string
    var role string
    var team_id string

    err = stmt.QueryRow(form.Username, form.Password).Scan(&id, &username, &name_first, &name_last, &role, &team_id)

    if err != nil {
      log.Print("!! Bad login from " + form.Username + " with " + form.Password)
      log.Fatal(err)
    } else {
      log.Print(">" + id + "<")
      log.Print(">" + username + "<")
      log.Print(">" + name_first + "<")
      log.Print(">" + name_last + "<")
      log.Print(">" + role + "<")
      log.Print(">" + team_id + "<")
      session.Set("id", id)
      session.Set("username", username)
      session.Set("name_first", name_first)
      session.Set("name_last", name_last)
      session.Set("role", role)
      session.Set("team_id", team_id)

      v := session.Get("name_first")
      if v == nil {
        log.Print("!! Uh oh.")
      }
      log.Print(v.(string))

      return "OK"
    }

    return "Bad"
  })

  m.Get("/session", func(session sessions.Session) string {
    var c Context

    i := session.Get("id")
    if i == nil {
      c.Id = -1
    }

    c.Id, _ = strconv.Atoi(i.(string))


    i = session.Get("username")
    if i == nil {
      log.Print("!! username")
    }

    if vs, ok := i.(string); ok {
      c.Username = vs
    } else {
      log.Print(vs)
    }

    // i = s.Get("name_first")
    // if i == nil {
    //   log.Print("!! name_first")
    // }

    // if vs, ok := i.(string); ok {
    //   c.NameFirst = vs
    // } else {
    //   log.Print(string(vs))
    // }

    // i = s.Get("name_last")
    // if i == nil {
    //   log.Print("!! name_last")
    // }

    // if vs, ok := i.(string); ok {
    //   c.NameLast = vs
    // } else {
    //   log.Print(string(vs))
    // }

    // i = s.Get("role")
    // if i == nil {
    //   log.Print("!! role")
    // }

    // if vs, ok := i.(string); ok {
    //   c.Role = vs
    // } else {
    //   log.Print(string(vs))
    // }

    // i = s.Get("team_id")
    // if i == nil {
    //   log.Print("!! team_id")
    // }

    // if vi, ok := i.(int); ok {
    //   c.TeamId = vi
    // } else {
    //   log.Print(string(vi))
    // }

    log.Print(c)

    j, _ := json.Marshal(c)

    return string(j)
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
    role := session.Get("role").(string)

    // if role == "admin" {
      r.HTML(200, "admin", "test")
    // } else {
     r.HTML(302, "index", "test")
    // }
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
