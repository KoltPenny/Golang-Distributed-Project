package main

import (
	_"github.com/go-sql-driver/mysql"
	"encoding/json"
	"encoding/hex"
	"database/sql"
	"net/http"
	"regexp"
	"errors"
	"bytes"
	"fmt"
	"log"
	//"os"
)

//Remote server URLs
const backupServer = "http://localhost:9000/Backup"
const updateServer = "http://localhost:9000/Update"
const managerServer = "http://localhost:8081/Manager"


//MUTEX & DATABASE
var database *sql.DB

func createReportStatement (rep report) error {

	database,_ := sql.Open ("mysql","root:rootpass@/huachicol")
	defer database.Close()
	
	sentence := fmt.Sprintf(
		"call insertUsuario('%s',%f,%f,FROM_UNIXTIME(%d),%d)",
		rep.Uuid,
		rep.Long,
		rep.Lat,
		rep.Cdate,
		rep.Rep_t,
	)
	fmt.Println(sentence)
	query,err := database.Prepare(sentence)
	if err != nil {	fmt.Println(err)	}
	
	_,err = query.Exec()	
	if err != nil {	fmt.Println(err)	}
	
	return err
}

func createGroupPoints (rep report) ([]insertGrpObj,error) {

	database,_ := sql.Open ("mysql","root:rootpass@/huachicol")
	defer database.Close()

	sentence := fmt.Sprintf("call ductNearPoint('%s')",rep.Uuid)

	query,err := database.Prepare(sentence)
	if err != nil {	fmt.Println(err)	}
	
	rows,err := query.Query()	
	if err != nil {	fmt.Println(err)	}

	groupdata := []insertGrpObj{}	
	count := 0
	
	for rows.Next() {
		
		count++
		element := insertGrpObj{"groups",rep.Uuid,""}
		
		rows.Scan(&element.Group_id)
		
		groupdata = append(groupdata,element)
	}
	if count == 0 {err = errors.New("Empty query")}
	return groupdata,err
}

func registeredStatement (tableName,attr string,condition string) string {
	return fmt.Sprintf(
		"SELECT %s FROM %s WHERE %s='%s'",
		attr,tableName,attr,condition,
	)
}

//WEB HANDLERS
//REPORTS

//Reports Backup
func reportBackup(w http.ResponseWriter, r *http.Request) {

	if r.Method == "POST" {

		jsondata := make(map[string] []report)
		buf := new(bytes.Buffer)

		buf.ReadFrom(r.Body)

		plaintext,err := decipher(buf.Bytes())
		if err != nil { return }

		json.Unmarshal(plaintext,&jsondata)
		fmt.Println(jsondata)
		return

		//If expression is faulty
		re := regexp.MustCompile(`^[a-f0-9]{16}$`)

		for _,table := range jsondata {
			for _,element := range table {

				if ! re.Match([]byte(element.Uuid)) {

					w.Header().Set("Content-Type", "text/html; charset=utf-8")
					fmt.Fprintf(w, "ERR-ID")

					return
				}				
			}
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		fmt.Fprintf(w, "DATA-OK")

		//Database open
		database,_ := sql.Open ("mysql","root:rootpass@/huachicol")
		defer database.Close()

		//for _,table := range jsondata {
		for _,element := range jsondata["usuario"] {

			err := createReportStatement(element)			
			if err != nil { fmt.Println(err) }
			
		}
		//}		
	}
}

//Reports Update
func reportUpdate(w http.ResponseWriter, r *http.Request) {

	if r.Method == "POST" {

		var jsondata insertRepObj
		buf := new(bytes.Buffer)
		
		buf.ReadFrom(r.Body)
		
		plaintext,err := decipher(buf.Bytes())
		if err != nil { return }

		json.Unmarshal(plaintext,&jsondata)
		fmt.Println("JSON: ", jsondata)

		//If expression is faulty
		re := regexp.MustCompile(`^[a-f0-9]{16}$`)

		if ! re.Match([]byte(jsondata.Device_uuid)) {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			fmt.Fprintf(w, "ERR-ID")
			fmt.Println(jsondata)
			return
		}				
		w.Header().Set("Content-Type", "text/html; charset=utf-8")

		nonce := makeNonce()
		data := aes_gcm.Seal(nonce,nonce,[]byte("DATA-OK"),nil)
		
		w.Write(data)

		fmt.Println("OK-RESPONDED")
		err = createReportStatement(jsondata.makeRep())
		if err != nil { fmt.Println(err) }
		
	}
}

//MANAGER
//Manager petition
func managerPet(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		
		var jsondata insertRepObj
		buf := new(bytes.Buffer)
		
		buf.ReadFrom(r.Body)

		plaintext,err := decipher(buf.Bytes())
		if err != nil { return }
		
		json.Unmarshal(plaintext,&jsondata)
		fmt.Println(jsondata)

		database,_ := sql.Open ("mysql","root:rootpass@/huachicol")
		defer database.Close()

		sentence := registeredStatement (jsondata.Table_name,"device_uuid",jsondata.Device_uuid)
		query,err := database.Prepare(sentence)
		resp,err := query.Query()

		if err != nil {
			fmt.Println("Exec: ",err)
		}
		
		defer resp.Close()

		var uuid string
		
		for resp.Next() {
			err := resp.Scan(&uuid)
			if err != nil {
				
				nonce := makeNonce()
				data := aes_gcm.Seal(nonce,nonce,[]byte("OK"),nil)
				w.Write(data)
				
				//fmt.Fprintf(w, string(data))
				
				fmt.Println("Query: ",err)
			}
			fmt.Println(uuid)
		}
		
		err = resp.Err()
		if err != nil {
			fmt.Println("Result: ",err)
			return
		}
		
		nonce := makeNonce()
		data := aes_gcm.Seal(nonce,nonce,[]byte("OK"),nil)
		
		w.Write(data)
		
	}
}

//Manager Update
func managerUserMap(w http.ResponseWriter, r *http.Request) {

	if r.Method == "POST" {
		//POST//
		
		var jsondata insertRepObj
		buf := new(bytes.Buffer)
		
		buf.ReadFrom(r.Body)

		plaintext,err := decipher(buf.Bytes())
		if err != nil { fmt.Println("<Decipher insertRepObj>\n",err) ; return }

		json.Unmarshal(plaintext,&jsondata)

		groupdata,err := createGroupPoints(jsondata.makeRep())
		if err != nil {	fmt.Println(err)	}

		//groupdata := []insertGrpObj{}

		fmt.Println(groupdata)
		
		data,err := json.Marshal(groupdata)
		if err != nil {	fmt.Println(err)	}
		
		nonce := makeNonce()
		data = aes_gcm.Seal(nonce,nonce,data,nil)

		w.Write(data)
		
	} else if r.Method == "GET" {
		//GET//
		
		key,ok := r.URL.Query()["key"]
		if !ok || len(key[0]) < 1 { return }

		buf,_ := hex.DecodeString(key[0])
		//buf.Read([]byte(key[0]))

		plaintext,err := decipher(buf)
		if err != nil { return }

		if "givememap" != string(plaintext) {
			//PONER ALGO AQUÍ
			fmt.Println("FAIL -- ",string(plaintext))
			
			nonce := makeNonce()
			data := aes_gcm.Seal(nonce,nonce,[]byte("MAP-BR"),nil)
		
			w.Write(data)
			return
		}
		
		geo := tablePointsToGeo()
		jgeo,_ := json.Marshal(geo)
		
		nonce := makeNonce()
		data := aes_gcm.Seal(nonce,nonce,jgeo,nil)
		
		w.Write(data)
	}
	
}



func main () {

	err := initAES()
	if err != nil { return }
	
	http.HandleFunc("/reportBackup",reportBackup)
	http.HandleFunc("/reportUpdate",reportUpdate)
	http.HandleFunc("/managerUserMap",managerUserMap)
	http.HandleFunc("/managerPet",managerPet)
	log.Fatal(http.ListenAndServe(":9000", nil))
}
