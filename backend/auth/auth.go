package auth

import (
    "os"
    "fmt"
    "strings"
    "io/ioutil"
    "net/http"
    "database/sql"

    "crypto/sha512"
    "crypto/rand"

    "encoding/hex"
    "encoding/json"

    "github.com/gorilla/mux"
    "github.com/gorilla/sessions"
    "github.com/gorilla/securecookie"

    _ "github.com/bmizerany/pq"
)


var SECRET_HASH_KEY string
var CookieStore = sessions.NewCookieStore(securecookie.GenerateRandomKey(32))

func SaltPassword(password, salt string) string {
    key := fmt.Sprintf("%s:%s:%s", password, salt, SECRET_HASH_KEY)

    sha := sha512.New()
    sha.Write([]byte(key))

    return hex.EncodeToString(sha.Sum(nil))
}

func GenerateSalt() string {
    salt := make([]byte, 16)
    rand.Read(salt)
    return hex.EncodeToString(salt)
}

type User struct {
    Email string
    Salt string
    Password string
}

type LoginForm struct {
    Email string
    Password string
}

type AuthConfig struct {
    Admins []User
    SecretKey string
}

type Users struct {
    db *sql.DB
}

func Connect() *Users {
    host := os.Getenv("POSTGRES_PORT_5432_TCP_ADDR")
    var postgresOptions string

    if host == "" { // if running locally
        postgresOptions = "sslmode=disable"
    } else { // if running in docker
        postgresOptions = fmt.Sprintf(
            "host=%s sslmode=disable user=postgres", host)
    }

    postgres, err := sql.Open("postgres", postgresOptions)

    if err != nil {
        fmt.Println(err.Error())
    }

    return &Users{postgres}
}

func (this *Users) Init() *Users {
    this.db.Exec("CREATE TABLE users (email varchar(40), salt char(32), password char(128))")
    return this
}

func (this *Users) Drop() *Users {
    this.db.Exec("DROP TABLE users")
    return this
}

func (this *Users) CreateUser(email, password string) *Users {
    salt := GenerateSalt()
    saltedPassword := SaltPassword(password, salt)

    insertUsers := fmt.Sprintf("INSERT INTO users VALUES ('%s', '%s', '%s')",
        email, salt, saltedPassword)

    this.db.Exec(insertUsers)
    return this
}

func (this *Users) GetUser(email string) *User {
    userQuery := fmt.Sprintf(
        "SELECT email, salt, password FROM users WHERE email = '%s'", email)
    row := this.db.QueryRow(userQuery)

    user := &User{}
    err := row.Scan(&user.Email, &user.Salt, &user.Password)

    if err != nil {
        fmt.Println(err.Error())
    }

    return user
}

func LoadAuthConfig() *AuthConfig{
    var fileName string
    if _, err := os.Stat("/config/auth.json"); os.IsNotExist(err) {
        fileName = "../config/auth.json"
    } else {
        fileName = "/config/auth.json"
    }
    configFile, _ := ioutil.ReadFile(fileName)

    var config AuthConfig
    json.Unmarshal(configFile, &config)
    return &config
}

func AuthMiddleware(h http.Handler, ignorePaths []string) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        if strings.HasPrefix(r.URL.Path, "/static") {
            h.ServeHTTP(w, r)
            return
        }

        for _, path := range ignorePaths {
            if path == r.URL.Path {
                h.ServeHTTP(w, r)
                return
            }
        }

        session, _ := CookieStore.Get(r, "auth")
        if loggedIn, ok := session.Values["logged_in"].(bool); loggedIn && ok {
            h.ServeHTTP(w, r)
        } else {
            w.WriteHeader(http.StatusUnauthorized)
            fmt.Fprintf(w, "unauthorized")
        }
    })
}

func Handle(r *mux.Router) {
    users := Connect().Drop().Init()
    config := LoadAuthConfig()

    SECRET_HASH_KEY = config.SecretKey

    for _, admin := range config.Admins {
        users.CreateUser(admin.Email, admin.Password)
    }

    r.HandleFunc("/login", func(w http.ResponseWriter, r *http.Request) {
        session, _ := CookieStore.Get(r, "auth")

        loginForm := &LoginForm{}
        body, _ := ioutil.ReadAll(r.Body)
        json.Unmarshal(body, &loginForm)

        user := users.GetUser(loginForm.Email)
        password := SaltPassword(loginForm.Password, user.Salt)
        session.Values["logged_in"] = password == user.Password

        session.Save(r, w)

        fmt.Fprintf(w, "%t", session.Values["logged_in"].(bool))
    }).Methods("POST")


    r.HandleFunc("/logout", func(w http.ResponseWriter, r *http.Request) {
        session, _ := CookieStore.Get(r, "auth")
        session.Values["logged_in"] = false
        session.Save(r, w)
    }).Methods("GET")
}
