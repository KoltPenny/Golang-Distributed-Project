package main

import (
	"fmt"
	"log"
	"net/http"
	"regexp"
	"encoding/json"
	"sync"
	"time"
	"database/sql"
	"os"
	_"github.com/mattn/go-sqlite3"
)
//Remote server URLs
const backupServer = "http://localhost:9000/Backup"
const updateServer = "http://localhost:9000/Update"
const managerServer = "http://localhost:8081/Manager"

//MUTEX & DATABASE
var database *sql.DB
var mutex sync.Mutex

func createTable (t tableDesc) {
	database,_ := sql.Open ("sqlite3","./reportes.db")
	statement,_ := database.Prepare (t.ToString())
	statement.Exec()
	database.Close()
}

func stringInsert(ins insertRepObj) string {
	query := fmt.Sprintf(
		"INSERT INTO %s VALUES ('%s',%f,%f,%f,%d)",
		ins.Table_name,
		ins.Device_uuid,
		ins.Gps_lat,
		ins.Gps_long,
		ins.Cdate,
		ins.Report_type)
	
	fmt.Println(query);
	return query
}
//WEB HANDLERS
/*
func handler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w,r,"index.html")
}
*/
func receiveReport(w http.ResponseWriter, r *http.Request) {

	if r.Method == "POST" {
		//Decode POST body into JSON
		var jsondata report
		
		decoder := json.NewDecoder(r.Body)
    err := decoder.Decode(&jsondata)
		
    if err != nil {
			fmt.Println("Decode",err)
    }

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

		fmt.Println("Locking in /report...")
		//START -> [Lock MUTEX]
		mutex.Lock()
		fmt.Println("Locked in /report.")
		
		//Database open
		database,_ := sql.Open ("sqlite3","./reportes.db")
		defer database.Close()

		//Prepare SQL statement with insertion object
		query := stringInsert(insob)
		statement,err := database.Prepare(query)		
		if err != nil {
			fmt.Println("Statement:",err)
			mutex.Unlock()
			w.Write([]byte("RP-QU"))
			return
		}

		//Execute SQL statement
		_,err = statement.Exec()

		if err != nil {
			fmt.Println("Statement:",err)
			mutex.Unlock()
			w.Write([]byte("RP-ST"))
			return
		}

		mutex.Unlock()
		//END -> [Unlock MUTEX]
		fmt.Println("Unlocked.")

		//START -> [Send to Main DB and Manager]
		data,_ := json.Marshal(insob)
		CastReport(data,updateServer,managerServer)
		w.Write([]byte("RP-OK"))
		//END -> [Sent to Main DB and Manager]
	}
}

// periodicBackup
func periodicBackup(minutes int) {

	database,_ := sql.Open ("sqlite3","./reportes.db")
	defer database.Close()

	for {
		time.Sleep(time.Duration(minutes) * time.Second)
		
		fmt.Println("Locking in periodicBackup...")
		//START -> [Lock MUTEX]
		mutex.Lock()
		fmt.Println("Locked in periodicBackup.")
		
		rows,_ := database.Query(`SELECT * FROM usuario`)
		
		stored_rep := report{}
		reports := []report{}

		for rows.Next() {
			err := rows.Scan(	&stored_rep.Uuid,&stored_rep.Lat,&stored_rep.Long,&stored_rep.Cdate,&stored_rep.Rep_t)
			if err != nil {	fmt.Println(err) }
			
			reports = append(reports,stored_rep)
		}

		backup,_ := json.Marshal(reports)
		os.Stdout.Write(backup)

		//SendToServer(backup,backupServer)
		
		mutex.Unlock()
		//END -> [Unlock MUTEX]
		fmt.Println("Unlocked.")

	}	
}
// periodicBackup
/*
func periodicBackup(minutes int, tableName string) {

	database,_ := sql.Open ("sqlite3","./reportes.db")
	defer database.Close()

	for {
		time.Sleep(time.Duration(minutes) * time.Second)

		mutex.Lock() //START -> [Lock MUTEX]

		var str strings.Builder
		str.WriteString("SELECT * FROM ")
		str.WriteString(tableName)

		query := str.String()
		rows,_ := database.Query(query)

		stored_rep := report{}
		reports := []report{}

		for rows.Next() {
			err := rows.Scan(	&stored_rep.Uuid,&stored_rep.Lat,&stored_rep.Long,&stored_rep.Cdate,&stored_rep.Rep_t)
			if err != nil {	fmt.Println(err) }

			reports = append(reports,stored_rep)
		}
		reps := map[string] []report {tableName:reports}

		backup,_ := json.Marshal(reps)
		nonce := makeNonce()
		backup = aes_gcm.Seal(nonce,nonce,backup,nil)

		resp,err := SendToServer(backup,backupServer)

		if err != nil {
			fmt.Println("Couldn't reach main server. Aborting backup...")
			mutex.Unlock()
			continue
		}

		fmt.Println("Backup successful!")
		response,_ := ioutil.ReadAll(resp.Body)
		fmt.Println(string(response))

		mutex.Unlock()//END -> [Unlock MUTEX]

	}	
}
*/
func main () {

	//Create table
	td := tableDesc {
		name: "usuario",
		attrs: []definition {
			definition{"device_uuid","text","PRIMARY KEY"},
			definition{"gps_lat","real",""},
			definition{"gps_long","real",""},
			definition{"cdate","real",""},
			definition{"report_type","integer",""},
		},
	}
	createTable(td)
	
	go periodicBackup(20)

	//http.HandleFunc("/", handler)
	http.HandleFunc("/report",receiveReport)
	log.Fatal(http.ListenAndServe(":8080", nil))
}
