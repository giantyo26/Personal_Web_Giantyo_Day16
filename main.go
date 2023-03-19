package main

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"log"
	"math"
	"net/http"
	"personal-web/connection"
	"personal-web/middleware"
	"strconv"
	"time"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
	"golang.org/x/crypto/bcrypt"
)

// Global struct
type Template struct {
	templates *template.Template
}

type User struct {
	Id       int
	Name     string
	Email    string
	Password string
}

type Project struct {
	Id              int
	ProjectName     string
	StartDate       time.Time
	EndDate         time.Time
	StartDateString string
	EndDateString   string
	Duration        string
	Description     string
	Technologies    []string
	Image           string
	Author          string
}

func (t *Template) Render(w io.Writer, name string, data interface{}, c echo.Context) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func main() {
	connection.ConnectDatabase()
	// Echo instance
	e := echo.New()
	// Access public folder
	e.Static("public", "public")
	e.Static("upload", "upload")

	e.Use(session.Middleware(sessions.NewCookieStore([]byte("session"))))

	t := &Template{
		templates: template.Must(template.ParseGlob("views/*.html")),
	}

	e.Renderer = t

	// Routes for rendering HTML
	e.GET("/", home)
	e.GET("/project", projectPage)
	e.GET("/contact", contactPage)
	e.GET("/project-detail/:id", projectDetailPage)
	e.GET("/project-edit/:id", projectEditPage)
	e.GET("/login", loginPage)
	e.GET("/register", registerPage)

	// Routes for helper function
	e.POST("/add-project", middleware.UploadFile(addProject))
	e.POST("/update-project/:id", middleware.UploadFile(updateProject))
	e.GET("/delete-project/:id", deleteProject)
	e.POST("/to-login", toLogin)
	e.POST("/to-register", toRegister)
	e.GET("/logout", logout)

	fmt.Println("Server is running on port 5000")

	// Start server
	e.Logger.Fatal(e.Start("localhost:5000"))
}

func loginPage(c echo.Context) error {
	sess, _ := session.Get("session", c)

	flash := map[string]interface{}{
		"FlashStatus":  sess.Values["isLogin"],
		"FlashMessage": sess.Values["message"],
		"FlashName":    sess.Values["name"],
	}

	delete(sess.Values, "message")
	delete(sess.Values, "status")
	sess.Save(c.Request(), c.Response())

	return c.Render(http.StatusOK, "login.html", flash)
}

func toLogin(c echo.Context) error {
	err := c.Request().ParseForm()
	if err != nil {
		log.Fatal(err)
	}

	email := c.FormValue("email")
	password := c.FormValue("password")

	user := User{}
	err = connection.Conn.QueryRow(context.Background(), "SELECT * FROM tb_user WHERE email = $1", email).Scan(&user.Id, &user.Name, &user.Email, &user.Password)
	if err != nil {
		return redirectWithMessage(c, "Wrong email", false, "/login")
	}

	// Check if two passwords match using Bcrypt's CompareHashAndPassword
	// which return nil on success and an error on failure.
	err = bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password))
	if err != nil {
		return redirectWithMessage(c, "Wrong password!", false, "/login")
	}

	sess, _ := session.Get("session", c)
	sess.Options.MaxAge = 7200
	sess.Values["message"] = "Login Success"
	sess.Values["status"] = true
	sess.Values["name"] = user.Name
	sess.Values["id"] = user.Id
	sess.Values["isLogin"] = true

	sess.Save(c.Request(), c.Response())

	return c.Redirect(http.StatusMovedPermanently, "/")
}

func registerPage(c echo.Context) error {
	return c.Render(http.StatusOK, "register.html", nil)
}

func toRegister(c echo.Context) error {
	err := c.Request().ParseForm()
	if err != nil {
		log.Fatal(err)
	}

	name := c.FormValue("name")
	email := c.FormValue("email")
	password := c.FormValue("password")

	// Hash password (Convert password string to byte slice, cost => represents the number of hash iteration)
	passwordHash, _ := bcrypt.GenerateFromPassword([]byte(password), 10)

	_, err = connection.Conn.Exec(context.Background(), "INSERT INTO tb_user (name, email, password) VALUES ($1, $2, $3)", name, email, passwordHash)
	if err != nil {
		redirectWithMessage(c, "Register failed, please try again", false, "/register")
	}
	return redirectWithMessage(c, "Succesfully registered, now you can login!", true, "/login")
}

// Render Homepage
func home(c echo.Context) error {
	sess, _ := session.Get("session", c)

	flash := map[string]interface{}{
		"FlashStatus":  sess.Values["isLogin"],
		"FlashMessage": sess.Values["message"],
		"FlashName":    sess.Values["name"],
	}
	delete(sess.Values, "message")
	sess.Save(c.Request(), c.Response())

	data, _ := connection.Conn.Query(context.Background(), "SELECT id, project_name, start_date, end_date, description, technologies, image FROM tb_project")

	var projects []Project

	for data.Next() {
		var project = Project{}

		err := data.Scan(&project.Id, &project.ProjectName, &project.StartDate, &project.EndDate, &project.Description, &project.Technologies, &project.Image)

		if err != nil {
			return c.JSON(http.StatusInternalServerError, map[string]string{"Message": err.Error()})
		}

		project.Duration = countDuration(project.StartDate, project.EndDate)
		projects = append(projects, project)
	}

	projectMap := map[string]interface{}{
		"Projects": projects,
		"Flash":    flash,
	}
	return c.Render(http.StatusOK, "index.html", projectMap)
}

// Render project page
func projectPage(c echo.Context) error {
	sess, _ := session.Get("session", c)
	flash := map[string]interface{}{
		"FlashStatus":  sess.Values["status"],
		"FlashMessage": sess.Values["message"],
		"FlashName":    sess.Values["name"],
	}
	delete(sess.Values, "message")

	return c.Render(http.StatusOK, "project.html", flash)
}

// Add project handler
func addProject(c echo.Context) error {
	projectName := c.FormValue("project-name")
	startDate := c.FormValue("start-date")
	endDate := c.FormValue("end-date")
	description := c.FormValue("description")
	technologies := c.Request().Form["technologies"]
	image := c.Get("dataFile").(string)

	startDateObj, _ := time.Parse("2006-01-02", startDate)
	endDateObj, _ := time.Parse("2006-01-02", endDate)

	_, err := connection.Conn.Exec(context.Background(), "INSERT INTO tb_project (project_name, start_date, end_date, description, technologies, image) VALUES ($1, $2, $3, $4, $5, $6)", projectName, startDateObj, endDateObj, description, technologies, image)

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"Message ": err.Error()})
	}

	return c.Redirect(http.StatusSeeOther, "/?msg=Project+added+successfully")
}

// Render project Detail page
func projectDetailPage(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id")) // url paramater => convert string to int

	sess, _ := session.Get("session", c)
	flash := map[string]interface{}{
		"FlashStatus":  sess.Values["isLogin"],
		"FlashMessage": sess.Values["message"],
		"FlashName":    sess.Values["name"],
	}
	delete(sess.Values, "message")

	sess.Save(c.Request(), c.Response())

	var projectDetail = Project{}

	err := connection.Conn.QueryRow(context.Background(), "SELECT id, project_name, start_date, end_date, description, technologies, image FROM tb_project WHERE id=$1", id).Scan(
		&projectDetail.Id, &projectDetail.ProjectName, &projectDetail.StartDate, &projectDetail.EndDate, &projectDetail.Description, &projectDetail.Technologies, &projectDetail.Image)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"Message": err.Error()})
	}

	projectDetail.StartDateString = projectDetail.StartDate.Format("02 Jan 2006")
	projectDetail.EndDateString = projectDetail.EndDate.Format("02 Jan 2006")

	projectDetail.Duration = countDuration(projectDetail.StartDate, projectDetail.EndDate)

	// data yang akan digunakan/dikirimkan ke html menggunakan map interface
	projectDetailMap := map[string]interface{}{
		"Project": projectDetail,
		"Flash":   flash,
	}

	return c.Render(http.StatusOK, "project-detail.html", projectDetailMap)
}

// Render project edit page
func projectEditPage(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))

	var Javascript, Go, Python, C, React, Postgresql bool

	sess, _ := session.Get("session", c)
	flash := map[string]interface{}{
		"FlashStatus":  sess.Values["isLogin"],
		"FlashMessage": sess.Values["message"],
		"FlashName":    sess.Values["name"],
	}
	delete(sess.Values, "message")

	projectEdit := Project{}

	err := connection.Conn.QueryRow(context.Background(), "SELECT id, project_name, start_date, end_date, description, technologies, image FROM tb_project WHERE id=$1", id).Scan(
		&projectEdit.Id, &projectEdit.ProjectName, &projectEdit.StartDate, &projectEdit.EndDate, &projectEdit.Description, &projectEdit.Technologies, &projectEdit.Image)
	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"Message": err.Error()})
	}

	for _, technology := range projectEdit.Technologies {
		if technology == "javascipt" {
			Javascript = true
		}
		if technology == "go" {
			Go = true
		}
		if technology == "python" {
			Python = true
		}
		if technology == "c" {
			C = true
		}
		if technology == "react" {
			React = true
		}
		if technology == "postgresql" {
			Postgresql = true
		}

	}

	projectEdit.StartDateString = projectEdit.StartDate.Format("2006-01-02")
	projectEdit.EndDateString = projectEdit.EndDate.Format("2006-01-02")

	projectEditMap := map[string]interface{}{
		"Project":    projectEdit,
		"Flash":      flash,
		"Javascript": Javascript,
		"Go":         Go,
		"Python":     Python,
		"C":          C,
		"React":      React,
		"Postgresql": Postgresql,
	}

	return c.Render(http.StatusOK, "project-edit.html", projectEditMap)
}

// Update/Edit project handler
func updateProject(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))

	projectName := c.FormValue("project-name")
	startDate := c.FormValue("start-date")
	endDate := c.FormValue("end-date")
	description := c.FormValue("description")
	technologies := c.Request().Form["technologies"]
	image := c.Get("dataFile").(string)

	startDateObj, _ := time.Parse("2006-01-02", startDate)
	endDateObj, _ := time.Parse("2006-01-02", endDate)

	_, err := connection.Conn.Exec(context.Background(), "UPDATE tb_project SET project_name = $1, start_date = $2, end_date = $3, description = $4, technologies = $5, image = $6 WHERE id = $7",
		projectName, startDateObj, endDateObj, description, technologies, image, id)

	if err != nil {
		return c.JSON(http.StatusInternalServerError, map[string]string{"Message ": err.Error()})
	}

	return c.Redirect(http.StatusMovedPermanently, "/")
}

// Delete project handler
func deleteProject(c echo.Context) error {
	id, _ := strconv.Atoi(c.Param("id"))

	connection.Conn.Exec(context.Background(), "DELETE FROM tb_project WHERE id = $1", id)

	return c.Redirect(http.StatusMovedPermanently, "/")
}

// render Contact page
func contactPage(c echo.Context) error {
	return c.Render(http.StatusOK, "contact-me.html", nil)
}

func redirectWithMessage(c echo.Context, message string, status bool, path string) error {
	sess, _ := session.Get("session", c)
	sess.Values["message"] = message
	sess.Values["status"] = status
	sess.Save(c.Request(), c.Response())

	return c.Redirect(http.StatusMovedPermanently, path)
}

func logout(c echo.Context) error {
	sess, _ := session.Get("session", c)
	sess.Options.MaxAge = -1
	sess.Save(c.Request(), c.Response())

	return c.Redirect(http.StatusTemporaryRedirect, "/")
}

func countDuration(StartDate, EndDate time.Time) string {
	subtract := EndDate.Sub(StartDate)

	month := math.Floor(subtract.Hours() / (24 * 30))
	week := math.Floor(subtract.Hours() / (24 * 7))
	day := math.Floor(subtract.Hours() / 24)

	switch {
	case month >= 1:
		return fmt.Sprintf("%v Bulan", month)
	case week >= 1:
		return fmt.Sprintf("%v Minggu", week)
	case day >= 1:
		return fmt.Sprintf("%v Hari", day)
	case month < 0 || week < 0 || day < 0:
		return "Operation error"
	default:
		return "Within a day"
	}
}
