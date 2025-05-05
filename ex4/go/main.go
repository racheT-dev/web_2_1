package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"net/http/cgi"
	"net/url"
	"regexp"

	_ "github.com/go-sql-driver/mysql"
)

type Application struct {
	Fio       string   `json:"fio"`
	Phone     string   `json:"phone"`
	Email     string   `json:"email"`
	Birthdate string   `json:"birthdate"`
	Gender    string   `json:"gender"`
	Bio       string   `json:"bio"`
	Langs     []string `json:"langs"`
}

type Errors struct {
	Fio       string `json:"fio"`
	Phone     string `json:"phone"`
	Email     string `json:"email"`
	Birthdate string `json:"birthdate"`
	Gender    string `json:"gender"`
	Bio       string `json:"bio"`
	Langs     string `json:"langs"`
}

func (e Errors) ToArray() []string {
	var res []string

	if e.Fio != "" {
		res = append(res, e.Fio)
	}

	if e.Phone != "" {
		res = append(res, e.Phone)
	}

	if e.Email != "" {
		res = append(res, e.Email)
	}

	if e.Birthdate != "" {
		res = append(res, e.Birthdate)
	}

	if e.Gender != "" {
		res = append(res, e.Gender)
	}

	if e.Bio != "" {
		res = append(res, e.Bio)
	}

	if e.Langs != "" {
		res = append(res, e.Langs)
	}

	return res
}

type Response struct {
	Errors      Errors      `json:"errors"`
	Application Application `json:"application"`
	Succeed     bool        `json:"succeed"`
}

func (r Response) Contains(lang string) bool {
	for _, s := range r.Application.Langs {
		if s == lang {
			return true
		}
	}

	return false
}

func IsSucceed(errors Errors) bool {
	return len(errors.ToArray()) == 0
}

func validate(appl Application) Response {
	var re *regexp.Regexp

	var errors Errors

	pattern := `^([А-ЯA-Z][а-яa-z]+ ){2}[А-ЯA-Z][а-яa-z]+$`
	re = regexp.MustCompile(pattern)

	if !re.MatchString(appl.Fio) {
		errors.Fio = "Поле должно быть заполнено в формате: Фамилия Имя Отчество"
	}

	pattern = `^(\+7|8)9\d{9}$`
	re = regexp.MustCompile(pattern)

	if !re.MatchString(appl.Phone) {
		errors.Phone = "Поле должно быть заполнено в формате: +79XXXXXXXXX или 89XXXXXXXXX"
	}

	pattern = `^[A-Za-z][\w\.-_]+@\w+(\.[a-z]{2,})+$`
	re = regexp.MustCompile(pattern)

	if !re.MatchString(appl.Email) {
		errors.Email = "Поле должно быть заполнено в формате: имя@домен"
	}

	if appl.Birthdate == "" {
		errors.Birthdate = "Поле должно быть заполнено"
	}

	if appl.Gender == "" {
		errors.Gender = "Поле должно быть заполнено"
	}

	if appl.Bio == "" {
		errors.Bio = "Поле должно быть заполнено"
	}

	if len(appl.Langs) == 0 {
		errors.Langs = "Поле должно быть заполнено"
	}

	return Response{errors, appl, IsSucceed(errors)}
}

func insertData(appl Application) {
	db, _ := sql.Open("mysql", "u68873:1518909@/u68873")
	defer db.Close()

	insert, _ := db.Query(fmt.Sprintf("INSERT INTO APPLICATION(NAME, PHONE, EMAIL, BIRTHDATE, GENDER, BIO) VALUES ('%s', '%s', '%s', '%s', '%s', '%s')", appl.Fio, appl.Phone, appl.Email, appl.Birthdate, appl.Gender, appl.Bio))
	defer insert.Close()

	sel, _ := db.Query("SELECT ID FROM APPLICATION ORDER BY ID DESC LIMIT 1")
	defer sel.Close()

	var id int
	for sel.Next() {
		sel.Scan(&id)
	}

	for _, name := range appl.Langs {
		sel, _ := db.Query(fmt.Sprintf("SELECT ID FROM PL WHERE NAME='%s'", name))
		defer sel.Close()

		var plId int
		for sel.Next() {
			sel.Scan(&plId)
		}

		insert, _ := db.Query(fmt.Sprintf("INSERT INTO FAVORITE_PL (APPLICATION_ID, PL_ID) VALUES ('%d', '%d')", id, plId))
		defer insert.Close()
	}
}

func setCookies(w http.ResponseWriter, response Response) *http.Cookie {
	responseJSON, _ := json.Marshal(response)
	responseEncoded := url.QueryEscape(string(responseJSON))

	cookie := &http.Cookie{
		Name:  "application",
		Value: responseEncoded,
	}

	http.SetCookie(w, cookie)

	return cookie
}

func getCookies(r *http.Request) (*http.Cookie, error) {
	cookie, err := r.Cookie("application")

	if err != nil {
		return cookie, err
	}

	return cookie, nil
}

func applicationHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, _ := template.ParseFiles("index.html")

	var response Response

	cookie, err := getCookies(r)

	if err == nil {
		responseJSON, _ := url.QueryUnescape(cookie.Value)
		json.Unmarshal([]byte(responseJSON), &response)
	}

	if r.Method == http.MethodPost {
		r.ParseForm()

		appl := Application{
			Fio:       r.FormValue("fio"),
			Phone:     r.FormValue("phone"),
			Email:     r.FormValue("email"),
			Birthdate: r.FormValue("birthdate"),
			Gender:    r.FormValue("gender"),
			Langs:     r.PostForm["langs[]"],
			Bio:       r.FormValue("bio")}

		response = validate(appl)

		cookie := setCookies(w, response)

		if response.Succeed {
			insertData(appl)

			cookie.MaxAge = 3600 * 24 * 365
			http.SetCookie(w, cookie)
		}
	}

	tmpl.Execute(w, response)
}

func main() {
	cgi.Serve(http.HandlerFunc(applicationHandler))
}
