package main

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	age "github.com/bearbin/go-age"
	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
	"github.com/rs/cors"
)

var db *sql.DB

func getDOB(dateOfBirth string) int {
	dobArr := strings.Split(dateOfBirth, "-")
	year, _ := strconv.Atoi(dobArr[0])
	month, _ := strconv.Atoi(dobArr[1])
	day, _ := strconv.Atoi(dobArr[2])

	dob := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
	return age.Age(dob)
}

// struct for person
type Person struct {
	Id        string `json:"id"`
	Firstname string `json:"first_name"`
	Lastname  string `json:"last_name"`
	Dob       string `json:"date_of_birth"`
	Address   string `json:"address"`
	Photoid   string `json:"photo_id"`
	Age       int
}

// functions to be called on specific routes just like we did in nodejs as get('/api', (req, res))

func getPersons(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT * FROM persons")
	if err != nil {
		fmt.Println(err.Error())
	}

	var persons []Person

	// Loop through rows, using Scan to assign column data to struct fields.
	for rows.Next() {
		var person Person
		err = rows.Scan(&person.Id, &person.Firstname, &person.Lastname, &person.Address, &person.Dob, &person.Photoid)
		if err != nil {
			panic(err.Error())
		}
		person.Dob = strings.Split(person.Dob, "T")[0]
		person.Age = getDOB(person.Dob)
		persons = append(persons, person)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(persons)
}

func getPerson(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	var person Person
	rows, err := db.Query("SELECT * FROM persons WHERE id = " + params["id"])
	if err != nil {
		fmt.Println(err.Error())
	}
	rows.Next()
	err = rows.Scan(&person.Id, &person.Firstname, &person.Lastname, &person.Address, &person.Dob, &person.Photoid)
	if err != nil {
		panic(err.Error())
	}
	person.Dob = strings.Split(person.Dob, "T")[0]
	person.Age = getDOB(person.Dob)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(person)
}

func createPerson(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	var person Person
	json.NewDecoder(r.Body).Decode(&person)

	var uid int
	temp_id := ""
	_ = db.QueryRow("INSERT into persons (first_name, last_name, date_of_birth, address, photo_id) values ('" + person.Firstname + "','" + person.Lastname + "','" + person.Dob + "','" + person.Address + "','" + temp_id + "') RETURNING id").Scan(&uid)

	if person.Photoid == "" {
		return
	}
	person.Photoid = uploadFile(person.Photoid)
	insert, err := db.Query(
		"UPDATE persons SET photo_id = '" + person.Photoid + "' WHERE id = '" + strconv.Itoa(uid) + "'")

	if err != nil {
		fmt.Println(err.Error())
	}
	defer insert.Close()
}
func uploadFile(photo_id string) string {
	b64data := photo_id[strings.IndexByte(photo_id, ',')+1:]
	obj, err := base64.StdEncoding.DecodeString(b64data)
	if err != nil {
		fmt.Println("Error Retrieving the File")
		fmt.Println(err)
		return ""
	}

	// Create a temporary file within our temp-images directory that follows
	// a particular naming pattern
	tempFile, err := ioutil.TempFile("temp-images", "upload-*.jpeg")
	if err != nil {
		fmt.Println(err)
	}
	defer tempFile.Close()

	// write this byte array to our temporary file
	tempFile.Write(obj)
	return tempFile.Name()
}

func updatePerson(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	params := mux.Vars(r)
	var person Person
	json.NewDecoder(r.Body).Decode(&person)
	temp_id := ""
	insert, err := db.Query("UPDATE persons SET first_name = '" + person.Firstname + "', last_name = '" + person.Lastname + "', address = '" + person.Address + "', date_of_birth = '" + person.Dob + "', photo_id = '" + temp_id + "' WHERE id = '" + params["id"] + "'")

	if err != nil {
		fmt.Println(err.Error())
	}
	if person.Photoid == "" {
		var old_photo string
		_ = db.QueryRow("SELECT photo_id from persons WHERE person.id = " + params["id"]).Scan(&old_photo)
		_, _ = db.Query(
			"UPDATE persons SET photo_id = '" + old_photo + "' WHERE id = '" + params["id"] + "'")
		return
	}

	person.Photoid = uploadFile(person.Photoid)
	upd, err := db.Query(
		"UPDATE persons SET photo_id = '" + person.Photoid + "' WHERE id = '" + params["id"] + "'")

	if err != nil {
		fmt.Println(err.Error())
	}
	defer insert.Close()
	defer upd.Close()
}

func deletePerson(w http.ResponseWriter, r *http.Request) {
	params := mux.Vars(r)
	_, err := db.Query("DELETE FROM persons WHERE id = " + params["id"])
	if err != nil {
		fmt.Println(err.Error())
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode("DELETED 200 OK")
}

func main() {
	// connecting to db
	db_, err := sql.Open("postgres", "postgresql://hbmkeppsewowpd:bebe59bfb3924f3317bb63a7ac6beab863ea9d49feb14e69f91b56e943501d91@ec2-44-205-64-253.compute-1.amazonaws.com:5432/dfc85j1mm39obt")
	db = db_
	if err != nil {
		panic(err.Error())
	}

	fmt.Println("successfully connected to db")

	// init router
	router := mux.NewRouter()

	// endpoints and router handler
	router.HandleFunc("/api/persons", getPersons).Methods("GET")
	router.HandleFunc("/api/persons/{id}", getPerson).Methods("GET")
	router.HandleFunc("/api/persons", createPerson).Methods("POST")
	router.HandleFunc("/api/persons/{id}", updatePerson).Methods("PUT")
	router.HandleFunc("/api/persons/{id}", deletePerson).Methods("DELETE")
	fs := http.FileServer(http.Dir("./temp-images/"))
	router.PathPrefix("/temp-images/").Handler(http.StripPrefix("/temp-images/", fs))
	// http.Handle("/image", http.StripPrefix("/", http.FileServer(http.Dir("path/to/file"))))

	// liston port (log is to print errors bss)
	// (<port>, <what-to-listen-?>)

	http.Handle("/", router)
	c := cors.New(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowCredentials: true,
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
	})
	handler := c.Handler(router)
	port := os.Getenv("PORT")
	log.Fatal(http.ListenAndServe(":"+port, handler))
}

// type person struct {
// 	name string
// 	age  int
// }

// func main() {
// 	var x int = 6
// 	var y int // default value is zero
// 	z := 3    // also a syntax

// 	per := person{name: "afafa", age: 21}

// 	var arr [5]int       // array with defauls zero
// 	arr2 := [3]int{1, 2} // also a syntax (with 3rd index as '0')

// 	sum := x + y + z

// 	if sum > 0 {
// 		fmt.Println("sum pos\n")
// 	} else if sum == 0 {

// 	} else {
// 		fmt.Println("sum neg\n")
// 	}

// 	for i := 0; i < sum; i++ {
// 		fmt.Println(i, " ")
// 	}
// 	abc, aaa := avg(3, 5)
// 	fmt.Println("\n", sum, abc, aaa, arr, "\n", arr2, per)
// }

// func avg(a int, b int) (int, string) { // this way you can return multiple values
// 	fmt.Println("taking avg now...")
// 	return ((a + b) / 2), "done hehe"
// }
