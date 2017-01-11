package main

import (
	"bytes"
	"fmt"
	"io"
	// "io/ioutil"
	"crypto/md5"
	"log"
	// "mime/multipart"
	"database/sql"
	_ "github.com/lib/pq"
	"net/http"
	// "os"
	"strconv"
	"text/template"
	"time"
)

const (
	DB_USER     = "md101"
	DB_PASSWORD = ""
	DB_NAME     = "md101"
	DB_HOST     = "localhost"
	DB_PORT     = "5432"
)

// hello world, the web server test
func HelloServer(w http.ResponseWriter, req *http.Request) {
	io.WriteString(w, "hello, world!\n")
}

func main() {
	http.HandleFunc("/upload", upload)
	http.HandleFunc("/getfile", getfile)
	http.HandleFunc("/hello2", HelloServer)
	http.HandleFunc("/crud", crud)
	log.Fatal(http.ListenAndServe(":9090", nil))
}

func getfile(w http.ResponseWriter, r *http.Request) {

	id := r.URL.Query().Get("id")
	//insert to db
	con := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME)
	db, err := sql.Open("postgres", con)
	if err != nil {
		log.Fatal(err)
	}

	q := fmt.Sprintf("SELECT file FROM userinfo where uid=%s", id)
	rows, err := db.Query(q)
	checkErr(err)

	w.Header().Set("Content-Disposition", "attachment; filename=file.xlsx")
	w.Header().Set("Content-Type", r.Header.Get("Content-Type"))
	for rows.Next() {
		//If a dest argument has type *[]byte, Scan saves in that argument a copy of the corresponding data. The copy is owned by the caller and can be modified and held indefinitely. The copy can be avoided by using an argument of type *RawBytes instead; see the documentation for RawBytes for restrictions on its use. source: https://golang.org/pkg/database/sql/#Rows
		var bt sql.RawBytes
		err = rows.Scan(&bt)

		//stream the body to the client without fully loading it into memory source:https://play.golang.org/p/v9IAu2Xu3_
		io.Copy(w, bytes.NewReader(bt))
		// fmt.Fprintf(w, "%s", bt)
	}
}

// upload logic
func upload(w http.ResponseWriter, r *http.Request) {
	fmt.Println("method:", r.Method)
	if r.Method == "GET" {
		crutime := time.Now().Unix()
		h := md5.New()
		io.WriteString(h, strconv.FormatInt(crutime, 10))
		token := fmt.Sprintf("%x", h.Sum(nil))

		t, _ := template.ParseFiles("upload.html")
		t.Execute(w, token)
	} else {
		r.ParseMultipartForm(32 << 20)
		file, handler, err := r.FormFile("uploadfile")
		if err != nil {
			fmt.Println(err)
			return
		}
		defer file.Close()
		// fmt.Fprintf(w, "%v", handler.Header)
		// f, err := os.OpenFile("./test/"+handler.Filename, os.O_WRONLY|os.O_CREATE, 0666)
		// if err != nil {
		// 	fmt.Println(err)
		// 	return
		// }
		// defer f.Close()
		//f is out writer, file is a reader
		// io.Copy(f, file)

		//copy to buffer
		var buff bytes.Buffer
		buff.ReadFrom(file)

		//insert to db
		con := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable", DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME)
		db, err := sql.Open("postgres", con)
		if err != nil {
			log.Fatal(err)
		}
		// db.Exec("INSERT INTO userinfo(username,departname,created,file) VALUES($1,$2,$3,$4)", handler.Filename, "IT", "2012-12-09", buff.Bytes())
		var lastInsertId int
		err = db.QueryRow("INSERT INTO userinfo(username,departname,created,file) VALUES($1,$2,$3,$4) returning uid;", handler.Filename, "IT", "2017-01-11", buff.Bytes()).Scan(&lastInsertId)
		fmt.Fprintf(w, "File %s telah terupload id=%d", handler.Filename, lastInsertId)
	}
}

func crud(w http.ResponseWriter, r *http.Request) {
	dbinfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		DB_HOST, DB_PORT, DB_USER, DB_PASSWORD, DB_NAME)
	db, err := sql.Open("postgres", dbinfo)
	checkErr(err)
	defer db.Close()

	fmt.Println("# Inserting values")

	var lastInsertId int
	err = db.QueryRow("INSERT INTO userinfo(username,departname,created) VALUES($1,$2,$3) returning uid;", "astaxie", "研发部门", "2012-12-09").Scan(&lastInsertId)
	checkErr(err)
	fmt.Println("last inserted id =", lastInsertId)

	fmt.Println("# Updating")
	stmt, err := db.Prepare("update userinfo set username=$1 where uid=$2")
	checkErr(err)

	res, err := stmt.Exec("astaxieupdate", lastInsertId)
	checkErr(err)

	affect, err := res.RowsAffected()
	checkErr(err)

	fmt.Println(affect, "rows changed")

	fmt.Println("cek function in psql")
	rows2, err := db.Query("select username,departname,totalrecords() from userinfo")

	checkErr(err)
	fmt.Println("Check function-------")
	for rows2.Next() {
		var user string
		var dep string
		var totalUSer int
		err = rows2.Scan(&user, &dep, &totalUSer)
		checkErr(err)

		fmt.Printf("%3v | %4v | %2v", user, dep, totalUSer)
		fmt.Println("")
	}

	fmt.Println("Check transaction-------")
	tx, err := db.Begin()
	if err != nil {
		fmt.Print(err)
	}
	defer tx.Rollback()
	stmt2, err := tx.Prepare("INSERT INTO userinfo(username,departname,created) VALUES ($1,$2,to_date($3,'DD/MM/YYYY'))")
	if err != nil {
		fmt.Print(err)
	}
	// defer stmt2.Close() // danger!

	name := []string{"a", "b", "c", "d"}

	for i := 0; i < len(name); i++ {
		_, err = stmt2.Exec(name[i], "test", "03/01/2017")
		if err != nil {
			fmt.Print(err)
		}
	}
	err = tx.Commit()
	if err != nil {
		fmt.Print(err)
	}
	stmt2.Close()
	fmt.Println("Check transaction end-------")

	fmt.Println("# Querying")
	rows, err := db.Query("SELECT * FROM userinfo")
	checkErr(err)

	for rows.Next() {
		var uid int
		var username string
		var department string
		var created time.Time
		err = rows.Scan(&uid, &username, &department, &created)
		checkErr(err)
		fmt.Println("uid | username | department | created ")
		fmt.Printf("%3v | %8v | %6v | %6v\n", uid, username, department, created)
	}

	// fmt.Println("# Deleting")
	// stmt, err = db.Prepare("delete from userinfo where uid=$1")
	// checkErr(err)

	// res, err = stmt.Exec(lastInsertId)
	// checkErr(err)

	// affect, err = res.RowsAffected()
	// checkErr(err)

	// fmt.Println(affect, "rows changed")
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}
