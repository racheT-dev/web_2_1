package main

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"net/http"
	"regexp"
	"text/template"
	"time"

	_ "github.com/go-sql-driver/mysql"
	jwt "github.com/golang-jwt/jwt/v5"
)

type (
	LoginResponse struct {
		Login    string
		Password string
		Message  string
		Type     string
	}

	Application struct {
		Fio       string
		Phone     string
		Email     string
		Birthdate string
		Gender    string
		Bio       string
		Langs     []string
	}

	Errors struct {
		Fio       string
		Phone     string
		Email     string
		Birthdate string
		Gender    string
		Bio       string
		Langs     string
	}

	FormResponse struct {
		ID          string
		Application Application
		Errors      Errors
		Message     string
	}
)

func (fr FormResponse) Contains(lang string) bool {
	for _, s := range fr.Application.Langs {
		if s == lang {
			return true
		}
	}

	return false
}

func dataIsCorrect(login string, password string) (bool, error) {
	db, err := sql.Open("mysql", "u68873:1518909@/u68873")

	if err != nil {
		return false, err
	}

	defer db.Close()

	p := ""

	sel, err := db.Query(`
		SELECT PASSWORD
		FROM USER
		WHERE LOGIN = ?;
	`, login)

	if err != nil {
		return false, err
	}

	defer sel.Close()

	for sel.Next() {
		err := sel.Scan(&p)

		if err != nil {
			return false, err
		}
	}

	return fmt.Sprintf("%x", sha256.Sum256([]byte(password))) == p, nil
}

func extractID(login string) string {
	re := regexp.MustCompile(`[1-9][0-9]*`)

	return string(re.Find([]byte(login)))
}

func getApplication(id string) (Application, error) {
	db, err := sql.Open("mysql", "u68873:1518909@/u68873")

	if err != nil {
		return Application{}, err
	}

	defer db.Close()

	appl := Application{}

	sel, err := db.Query(`
		SELECT *
		FROM APPLICATION
		WHERE ID = ?;
	`, id)

	if err != nil {
		return appl, err
	}

	defer sel.Close()

	for sel.Next() {
		err := sel.Scan(&id, &appl.Fio, &appl.Phone, &appl.Email, &appl.Birthdate, &appl.Gender, &appl.Bio)

		if err != nil {
			return appl, err
		}
	}

	sel, err = db.Query(`
		SELECT NAME
		FROM FAVORITE_PL fav
		JOIN PL pl ON fav.PL_ID = pl.ID
		WHERE APPLICATION_ID = ?;
	`, id)

	if err != nil {
		return appl, err
	}

	defer sel.Close()

	for sel.Next() {
		var pl string

		err := sel.Scan(&pl)

		if err != nil {
			return appl, err
		}

		appl.Langs = append(appl.Langs, pl)
	}

	return appl, nil
}

func grantAccessToken(w http.ResponseWriter) {
	payload := jwt.RegisteredClaims{
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
	}
	key := []byte("access-token-secret-key")

	accessToken := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)

	t, err := accessToken.SignedString(key)

	if err != nil {
		fmt.Fprintf(w, "Ошибка при создании токена: %v", err)
		return
	}

	cookie := &http.Cookie{
		Name:  "accessToken",
		Value: t,
	}

	http.SetCookie(w, cookie)
}

func deleteCookie(w http.ResponseWriter, r *http.Request) {
	cookie, err := r.Cookie("accessToken")

	if err != nil {
		return
	}

	cookie.MaxAge = -1
	http.SetCookie(w, cookie)
}

func loginHandler(w http.ResponseWriter, r *http.Request) {
	deleteCookie(w, r)

	tmpl, err := template.ParseFiles("login.html")

	if err != nil {
		fmt.Fprintf(w, "Template error: %v", err)
		return
	}

	response := LoginResponse{}

	if r.Method == http.MethodPost {
		login := r.FormValue("login")
		password := r.FormValue("password")

		valid, err := dataIsCorrect(login, password)

		if err != nil {
			fmt.Fprintf(w, "MySQL error: %v", err)
			return
		}

		if !valid {
			response.Type = "warning_red"
			response.Message = "Неверные логин и/или пароль"
			tmpl.Execute(w, response)
			return
		}

		tmpl, err = template.ParseFiles("form.html")

		if err != nil {
			fmt.Fprintf(w, "Template error: %v", err)
			return
		}

		response := FormResponse{ID: extractID(login)}

		response.Application, err = getApplication(response.ID)

		if err != nil {
			fmt.Fprintf(w, "MySQL error: %v", err)
			return
		}

		grantAccessToken(w)

		tmpl.Execute(w, response)
		return
	}

	tmpl.Execute(w, response)
}
