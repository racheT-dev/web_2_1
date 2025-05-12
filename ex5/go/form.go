package main

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"math/rand"
	"net/http"
	"regexp"
	"strings"
	"text/template"

	"github.com/golang-jwt/jwt/v5"
)

func validate(appl Application) Errors {
	var re *regexp.Regexp

	var errors Errors

	pattern := `^([А-ЯA-Z][а-яa-z]+ ){2}[А-ЯA-Z][а-яa-z]+$`
	re = regexp.MustCompile(pattern)

	if appl.Fio == "" {
		errors.Fio = "Поле должно быть заполнено"
	} else if !re.MatchString(appl.Fio) {
		errors.Fio = "Поле должно быть заполнено в формате: Фамилия Имя Отчество"
	}

	pattern = `^(\+7|8)9\d{9}$`
	re = regexp.MustCompile(pattern)

	if appl.Phone == "" {
		errors.Phone = "Поле должно быть заполнено"
	} else if !re.MatchString(appl.Phone) {
		errors.Phone = "Поле должно быть заполнено в формате: +79XXXXXXXXX или 89XXXXXXXXX"
	}

	pattern = `^[A-Za-z][\w\.-_]+@\w+(\.[a-z]{2,})+$`
	re = regexp.MustCompile(pattern)

	if appl.Email == "" {
		errors.Email = "Поле должно быть заполнено"
	} else if !re.MatchString(appl.Email) {
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

	return errors
}

func (e Errors) Count() int {
	count := 0

	if e.Fio != "" {
		count++
	}
	if e.Phone != "" {
		count++
	}
	if e.Email != "" {
		count++
	}
	if e.Birthdate != "" {
		count++
	}
	if e.Gender != "" {
		count++
	}
	if e.Bio != "" {
		count++
	}
	if e.Langs != "" {
		count++
	}

	return count
}

func isAuthorized(r *http.Request) bool {
	cookie, err := r.Cookie("accessToken")

	if err != nil {
		return false
	}

	token, err := jwt.ParseWithClaims(cookie.Value, &jwt.RegisteredClaims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte("access-token-secret-key"), nil
	})

	return err == nil && token.Valid
}

func updateApplication(id string, appl Application) error {
	db, err := sql.Open("mysql", "u68873:1518909@/u68873")

	if err != nil {
		return err
	}

	defer db.Close()

	_, err = db.Exec(`
		UPDATE APPLICATION
		SET NAME = ?, PHONE = ?, EMAIL = ?, BIRTHDATE = ?, GENDER = ?, BIO = ?
		WHERE ID = ?;
	`, appl.Fio, appl.Phone, appl.Email, appl.Birthdate, appl.Gender, appl.Bio, id)

	if err != nil {
		return err
	}

	_, err = db.Exec(`
		DELETE FROM FAVORITE_PL
		WHERE APPLICATION_ID = ?;
	`, id)

	if err != nil {
		return err
	}

	err = insertPL(appl.Langs, id)

	if err != nil {
		return err
	}

	return nil
}

func insertPL(langs []string, id string) error {
	db, err := sql.Open("mysql", "u68873:1518909@/u68873")

	if err != nil {
		return err
	}

	defer db.Close()

	for _, pl := range langs {
		plid := ""

		sel, err := db.Query(`
			SELECT ID
			FROM PL
			WHERE NAME = ?;
		`, pl)

		if err != nil {
			return err
		}

		defer sel.Close()

		for sel.Next() {
			err := sel.Scan(&plid)

			if err != nil {
				return err
			}
		}

		_, err = db.Exec(`
			INSERT INTO FAVORITE_PL
			VALUES (?, ?);
		`, id, plid)

		if err != nil {
			return err
		}
	}

	return nil
}

func insertApplication(id string, appl Application) error {
	db, err := sql.Open("mysql", "u68873:1518909@/u68873")

	if err != nil {
		return err
	}

	defer db.Close()

	_, err = db.Exec(`
       INSERT INTO APPLICATION
	   VALUES (?, ?, ?, ?, ?, ?, ?);
	`, id, appl.Fio, appl.Phone, appl.Email, appl.Birthdate, appl.Gender, appl.Bio)

	if err != nil {
		return err
	}

	err = insertPL(appl.Langs, id)

	if err != nil {
		return err
	}

	return nil
}

func insertUser(login string, password string) error {
	db, err := sql.Open("mysql", "u68873:1518909@/u68873")

	if err != nil {
		return err
	}

	defer db.Close()

	_, err = db.Exec(`
		INSERT INTO USER
		VALUES (?, ?);
	`, login, fmt.Sprintf("%x", sha256.Sum256([]byte(password))))

	if err != nil {
		return err
	}

	return nil
}

func generateLAP() (string, string, error) {
	var login = "u0000000"
	var password string

	db, err := sql.Open("mysql", "u68873:1518909@/u68873")

	if err != nil {
		return login, password, err
	}

	defer db.Close()

	sel, err := db.Query(`
		SELECT LOGIN
		FROM USER
		ORDER BY LOGIN DESC 
		LIMIT 1;
	`)

	if err != nil {
		return login, password, err
	}

	for sel.Next() {
		err := sel.Scan(&login)

		if err != nil {
			return login, password, err
		}
	}

	chars := "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()_+-=[]{}|;':\",./<>?`~"

	for i := 0; i < 8; i++ {
		password += string(chars[rand.Intn(len(chars))])
	}

	return increaseByOne(login), password, nil
}

func increaseByOne(str string) string {
	digits := "0123456789"
	res := ""

	p := len(str) - 1

	for str[p] == '9' {
		res += "0"
		p--
	}

	q := strings.Index(digits, string(str[p])) + 1
	res = string(digits[q]) + res

	return str[:p] + res
}

func formHandler(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("form.html")

	if err != nil {
		fmt.Fprintf(w, "Template error: %v", err)
		return
	}

	response := FormResponse{}

	if r.Method == http.MethodPost {
		err := r.ParseForm()

		if err != nil {
			fmt.Fprintf(w, "Form parsing error: %v", err)
			return
		}

		id := r.URL.Query().Get("id")
		appl := Application{
			Fio:       r.FormValue("fio"),
			Phone:     r.FormValue("phone"),
			Email:     r.FormValue("email"),
			Birthdate: r.FormValue("birthdate"),
			Gender:    r.FormValue("gender"),
			Bio:       r.FormValue("bio"),
			Langs:     r.PostForm["langs[]"],
		}
		errors := validate(appl)

		response = FormResponse{
			ID:          id,
			Application: appl,
			Errors:      errors,
		}

		if errors.Count() != 0 {
			tmpl.Execute(w, response)
			return
		}

		if id != "" && isAuthorized(r) {
			err := updateApplication(id, appl)

			if err != nil {
				fmt.Fprintf(w, "MySQL error: %v", err)
				return
			}

			response.Message = "Ваши данные успешно изменены!"

			tmpl.Execute(w, response)
			return
		}

		response := LoginResponse{}

		if id != "" {
			response.Type = "warning_red"
			response.Message = "Для выполнения этого действия необходимо авторизоваться"
		} else {
			login, password, err := generateLAP()

			if err != nil {
				fmt.Fprintf(w, "MySQL error: %v", err)
				return
			}

			err = insertUser(login, password)

			if err != nil {
				fmt.Fprintf(w, "MySQL error: %v", err)
				return
			}

			err = insertApplication(extractID(login), appl)

			if err != nil {
				fmt.Fprintf(w, "MySQL error: %v", err)
				return
			}

			response.Login = login
			response.Password = password
			response.Type = "warning_green"
			response.Message = "Вы успешно зарегистрировались. Перед нажатием на кнопку Войти сохраните ваш логин и пароль!"
		}

		tmpl, err := template.ParseFiles("login.html")

		if err != nil {
			fmt.Fprintf(w, "Template error: %v", err)
			return
		}

		tmpl.Execute(w, response)
		return
	}

	tmpl.Execute(w, response)
}
