package main

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"encoding/json"
	//"strings"
	//"sync"
	//"time"
	"database/sql"
	//"io/ioutil"
	_ "github.com/mattn/go-sqlite3"
)
//Remote server URLs
//const backupServer = "http://localhost:9000/reportBackup"
//const updateServer = "http://localhost:9000/reportUpdate"
//const managerServer = "http://localhost:8081/postToManager"

//const backupServer = "http://localhost:9000/reportBackup"  
const updateServer = "http://localhost:9000/reportUpdate"  
const managerServer = "http://localhost:8081/postToManager"//MUTEX & DATABASE

//var database *sql.DB
//var mutex sync.Mutex

func createTable (t tableDesc) {
	database,_ := sql.Open ("sqlite3","./reportes.db")
	statement,_ := database.Prepare (t.ToString())
	statement.Exec()
	database.Close()
}

func stringInsert(ins insertRepObj) string {
	query := fmt.Sprintf(
		"INSERT INTO %s VALUES ('%s',%f,%f,%d,%d)",
		ins.Table_name,
		ins.Device_uuid,
		ins.Gps_long,
		ins.Gps_lat,
		ins.Cdate,
		ins.Report_type)
	
	fmt.Println(query);
	return query
}
//WEB HANDLERS

func handler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w,r,"index.html")
}

func receiveReport(w http.ResponseWriter, r *http.Request) {

	if r.Method == "POST" {
		//Decode POST body into JSON
		var jsondata report
		
		decoder := json.NewDecoder(r.Body)
    err := decoder.Decode(&jsondata)
		
    if err != nil {
			fmt.Println("Decode",err)
    }
		fmt.Println(jsondata)
		//Fill insertion object
		var insob insertRepObj
		insob.Fill("usuario",jsondata)

		//If expression is faulty		
		re := regexp.MustCompile(`^[a-f0-9]{16}$`)
		if ! re.Match([]byte(insob.Device_uuid)) {
			
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprintf(w, "ERR-ID")
			
			return
		}

		//mutex.Lock() //START -> [Lock MUTEX]
		
		//Database open
		database,_ := sql.Open ("sqlite3","./reportes.db")
		defer database.Close()

		//Prepare SQL statement with insertion object
		query := stringInsert(insob)
		statement,err := database.Prepare(query)		
		if err != nil {
			fmt.Println("Statement:",err)
			//mutex.Unlock()
			w.Write([]byte("RP-QU"))
			return
		}
		err = nil
		//Execute SQL statement
		_,err = statement.Exec()

		if err != nil {
			fmt.Println("Statement:",err)
			//DESCOMENTAR AL TERMINAR
			//mutex.Unlock() //ESTO NO
			//w.Write([]byte("RP-ST"))
			//return
		}

		//mutex.Unlock()//END -> [Unlock MUTEX]

		//START -> [Send to Main DB and Manager]
		data,_ := json.Marshal(insob)

		nonce := makeNonce()
		data = aes_gcm.Seal(nonce,nonce,data,nil)

		err = CastReport(data,updateServer,managerServer)

		if err != nil {
			fmt.Println("<CAST>\n",err,"\n<---->")
			switch err.Error() {
			case "TIMEOUT":	w.Write([]byte("RP-TO"))
			case "DECRYPT":	w.Write([]byte("RP-CR"))
			}
			
			return
		}
		w.Write([]byte("RP-OK"))
		//END -> [Sent to Main DB and Manager]
	}
}

func main () {
	//Init AES Key and GCM
	err := initAES()
	if err != nil { return }

	//Create table
	td := tableDesc {
		name: "usuario",
		attrs: []definition {
			definition{"device_uuid","text","PRIMARY KEY"},
			definition{"gps_lat","real",""},
			definition{"gps_long","real",""},
			definition{"cdate","integer",""},
			definition{"report_type","integer",""},
		},
	}
	createTable(td)

	http.HandleFunc("/", handler)
	http.HandleFunc("/report",receiveReport)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
