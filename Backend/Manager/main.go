package main

import (
	_"github.com/mattn/go-sqlite3"
	"encoding/json"
	//"encoding/hex"
	"database/sql"
	//"io/ioutil"
	"net/http"
	//"strings"
	"regexp"
	"errors"
	"bytes"
	//"sync"
	//"time"
	"fmt"
	"log"
	//"os"
)
//Remote server URLs
//const backupServer = "http://localhost:9000/reportBackup"
//const updateServer = "http://localhost:9000/reportUpdate"
//const managerPet = "http://localhost:9000/managerPet"
//const managerUserMap = "http://localhost:9000/managerUserMap"
//const managerServer = "http://localhost:8081/Manager"

const backupServer = "http://localhost:9000/reportBackup"
const updateServer = "http://localhost:9000/reportUpdate"
const managerPet = "http://localhost:9000/managerPet"
const managerUserMap = "http://localhost:9000/managerUserMap"
const managerServer = "http://localhost:8081/Manager"

//MUTEX & DATABASE
//var database *sql.DB
//var mutex sync.Mutex

func createTable (t tableDesc) {
	database,_ := sql.Open ("sqlite3","./grupos.db")
	defer database.Close()
	statement,_ := database.Prepare (t.ToString())
	statement.Exec()
}

func stringInsert(ins insertGrpObj) string {
	query := fmt.Sprintf(
		"INSERT INTO %s VALUES ('%s','%s')",ins.Table_name,ins.Device_uuid,ins.Group_id)
	
	fmt.Println(query);
	return query
}

func stringUserInsert(ins insertRepObj) string {
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
/*
func handler(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w,r,"index.html")
}
*/
func reportToManager(w http.ResponseWriter, r *http.Request) {

	if r.Method == "POST" {
		//Decode POST body into JSON

		var jsondata insertRepObj
		buf := new(bytes.Buffer)
		
		buf.ReadFrom(r.Body)

		plaintext,err := decipher(buf.Bytes())
		if err != nil { return }
		
		json.Unmarshal(plaintext,&jsondata)
		fmt.Println(jsondata)

		//If expression is faulty		
		re := regexp.MustCompile(`^[a-f0-9]{16}$`)
		if ! re.Match([]byte(jsondata.Device_uuid)) {
			
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprintf(w, "ERR-ID")
			
			return
		}

		//Respond to user
		nonce := makeNonce()
		userResponse := aes_gcm.Seal(nonce,nonce,[]byte("OK"),nil)
		w.Write(userResponse)

		//VERIFY USER BACKUP
		res,err := SendToServer(buf.Bytes(),managerPet)
		if err != nil {
			return
		}
		
		buf.Reset()
		buf.ReadFrom(res.Body)

		plaintext,err = decipher(buf.Bytes())
		if err != nil {	fmt.Println(err) ; return	}

		status := string(plaintext)

		if status != "OK" {
			fmt.Println("Connection error... Aborting");
			return
		}

		//INSERT USER INTO LOCAL DATABASE
		database,_ := sql.Open ("sqlite3","./grupos.db")
		defer database.Close()

		query := stringUserInsert(jsondata)
		_,err = database.Prepare(query)
		if err != nil { fmt.Println("<User prepare>\n",err);return}
		
		_,err = database.Exec("")
		if err != nil { fmt.Println("<User exec>\n",err);return}

		//SEND USER TO GET NEAR POINTS
		data,_ := json.Marshal(jsondata)

		nonce = makeNonce()
		data = aes_gcm.Seal(nonce,nonce,data,nil)

		res,err = SendToServer(data,managerUserMap)
		if err != nil { fmt.Println("<managerUserMap>\n",err) ; return }

		buf.Reset()
		buf.ReadFrom(res.Body)
		
		pt,err := decipher(buf.Bytes())
		if err != nil { fmt.Println("<Group decipher>\n",err);return}

		var groupdata []insertGrpObj
		
		json.Unmarshal(pt,&groupdata)

		//INSERT NEAR POINTS IN LOCAL DATABASE
		for _,element := range groupdata {
			
			query = stringInsert(element)
			_,err = database.Prepare(query)
			if err != nil { fmt.Println(err);return}
			
			_,err = database.Exec(query)
			if err != nil { fmt.Println(err);return}
		}
	}
}

func getUserMap() ([]byte,error) {

	nonce := makeNonce()
	raw_data := aes_gcm.Seal(nonce,nonce,[]byte("givememap"),nil)

	data := fmt.Sprintf("%x",raw_data)
	
	var getMap bytes.Buffer
	getMap.WriteString(managerUserMap)
	getMap.WriteString("?key=")
	getMap.WriteString(data)

	resp,err :=http.Get(string(getMap.Bytes()))
	if err != nil {return nil,err}
	
	buf := new(bytes.Buffer)
	buf.ReadFrom(resp.Body)

	plaintext,err := decipher(buf.Bytes())
	if err != nil { fmt.Println(err);return nil,err}

	if "MAP-BR" == string(plaintext) {
		fmt.Println("FAIL -- ",string(plaintext))
		return nil,errors.New("MAP-BR")
		}

	return plaintext,nil
	
}

func startMap(w http.ResponseWriter, r *http.Request) {
	
	if r.Method == "GET" {
		raw,err := getUserMap();
		if err != nil {
			fmt.Println("Getting map: ",err)
			switch err.Error() {
			case "MAP-BR":
				w.Write([]byte(err.Error()))
			}
		}
		w.Write(raw)
	}
}

func main () {

	//Init AES Key and GCM
	err := initAES()
	if err != nil { return }

	//Create users table
	td := tableDesc {
		name: "usuario",
		attrs: []definition {
			definition{"device_uuid","text","PRIMARY KEY"},
			definition{"gps_long","real",""},
			definition{"gps_lat","real",""},
			definition{"cdate","integer",""},
			definition{"report_type","integer",""},
		},
	}
	createTable(td)

	//Create groups table
	td = tableDesc {
		name: "groups",
		attrs: []definition {
			definition{"device_uuid","text",""},
			definition{"duct_id","integer",""},
			definition{"PRIMARY KEY (","device_uuid,duct_id",")"},
			definition{"FOREIGN KEY(device_uuid) REFERENCES usuario(","device_uuid",")"},
		},
	}
	createTable(td)
	
	http.HandleFunc("/postToManager",reportToManager)
	http.HandleFunc("/startMap",startMap)
	log.Fatal(http.ListenAndServe(":8081", nil))

}

/*
========== FUNCTIONS AND DESCRIPTIONS ==========

-->func createTable (t tableDesc):
  -Create table with tableDesc info.

-->func stringInsert(ins insertGrpObj) string:
  -Creates an insert query with the info in insertGrpObj.
  -Returns string.

-->func stringUserInsert(ins insertRepObj) string:
  -Creates an insert query with the info in insertRepObj.
  -Returns string.

-->func reportToManager(w http.ResponseWriter, r *http.Request):
  -Receives User data from Web backend ([Auth]WebBack -> [Auth]GolangBack -> [Manager]GolangBack).
  -Verifies existence of User on DB (GolangBack <-> DB).
  -Saves local copy of User data.
  -Verifies that the User point is near to Ducts (GolangBack <-> DB).
  -Stores the points User is near in local copy.

-->func getUserMap() []byte:
  -Asks for points (GolangBack <->)
*/
