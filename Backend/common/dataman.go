package main

import (
	"crypto/cipher"
	"crypto/rand"
	"crypto/aes"
	"net/http"
	"strings"
	"errors"
	"bytes"
	"time"
	"fmt"
	"io"
)

//SECURE
const aes_seckey = "abcdABCDabcdABCDabcdABCD"

var aes_cypher cipher.Block
var aes_gcm cipher.AEAD

func initAES() error {
	aes_cypher,err := aes.NewCipher([]byte(aes_seckey));
	if err != nil { fmt.Println(err) }
	aes_gcm, err = cipher.NewGCM(aes_cypher);
	if err != nil { fmt.Println(err) }
	return err
}

func decipher(buf []byte) ([]byte, error) {
	nonceSize := aes_gcm.NonceSize();
	if len(buf) < nonceSize {return nil,errors.New("Wrong size")}
	nonce, buf := buf[:nonceSize], buf[nonceSize:]
	return aes_gcm.Open(nil, nonce, buf, nil)
}

func makeNonce() []byte {
	nonce := make([]byte, aes_gcm.NonceSize())	
	_, err := io.ReadFull(rand.Reader, nonce)
	if err != nil { fmt.Println(err) }
	return nonce
}

//DESCRIPTION AND BEHAVIOUR OF OBJECT TO INSERT

//Insert Object structure for reports
type insertRepObj struct {
	Table_name string
	Device_uuid string
	Gps_long float64
	Gps_lat float64
	Cdate int64
	Report_type int8
}

//Insert Object structure for groups
type insertGrpObj struct {
	Table_name string
	Device_uuid string
	Group_id string
}

type group struct {
	Uuid string
	GID string
}

//Table attributes
type definition struct {
	aname string
	atype string
	constraint string
}
//Json recovered
type report struct {
	Uuid string 
	Long float64 
	Lat float64
	Cdate int64 
	Rep_t int8
}

//Table descriptor
type tableDesc struct {
	name string
	attrs []definition
	constraints []definition
}

//Insert Object Rep
func (insob *insertRepObj) Fill(name string,rep report) {
	
	insob.Table_name = name
	insob.Device_uuid = rep.Uuid
	insob.Gps_lat = rep.Lat
	insob.Gps_long = rep.Long
	insob.Cdate = time.Now().Unix()
	insob.Report_type = rep.Rep_t
}

func (insob insertRepObj) makeRep() report {
	return report {
		insob.Device_uuid,
		insob.Gps_long,
		insob.Gps_lat,
		insob.Cdate,
		insob.Report_type,
	}
	
}
//DESCRIPTION AND BEHAVIOUR OF TABLE INIT

//Table attribute to string
func (a definition) ToString() string {
	var s []string
	if(len(a.constraint) == 0) {
		s = []string{a.aname, a.atype}
	}	else {
		s = []string{a.aname, a.atype, a.constraint}
	}
	return strings.Join(s, " ")
}
//Table attribute array to string
func defToString(a []definition) string {
	l := len(a)
	var s []string
	for i := 0; i < l; i++ {
		s = append(s,a[i].ToString())
	}
	return strings.Join(s,",")
}
//Interface tableDesc Create Table
func (t tableDesc) ToString() string {
	s := []string {
		"CREATE TABLE IF NOT EXISTS",
		t.name,
		"(",
		defToString(t.attrs),
		")",
	}
	return strings.Join(s," ")
}

func (r *report) All() (string,float64,float64,int64,int8) {
	return r.Uuid,r.Lat,r.Long,r.Cdate,r.Rep_t
}

func SendToServer(data []byte, server_address string) (*http.Response,error) {
	resp, err := http.Post(server_address, "application/json", bytes.NewBuffer(data))
	//fmt.Println(resp,err)
	return resp,err
}

func CastReport (jsondata []byte, update_process string, next_process string) error {
/* SENDS REPORTS TO UPDATE PROCESS AND NEXT PROCESS */
	var err error
	var res *http.Response
	//Update database

	fmt.Println("Updating database...")

	dbchan := make(chan bool,1)
	kill := make(chan bool)
	
	go func() {
		
		for {
			select {
			case <- kill:	return //Dies if timeout occurs
			default:
				res,err = SendToServer(jsondata,update_process)
				if err == nil {
					dbchan <- true
					return
				}
				fmt.Println(err)
				time.Sleep(1 * time.Second)
			}
		}
	}()

	select {
	case <- dbchan:
		fmt.Println("DB updated")
		
	case <- time.After(5 * time.Second):
		kill <- true
		fmt.Println("Sending timed out. Aborting...")
		close(dbchan)
		close(kill)
		return err
	}

	nodechan := make(chan bool,1)
	kill = make(chan bool)
	
	go func () {
		//Notify manager
		select {
		case <- kill:	return //Dies if timeout occurs
		default:
			fmt.Println("Notifying next node...")
			for {
				res,err = SendToServer(jsondata,next_process)
				if err == nil { nodechan <- true ; break }
				fmt.Println(err)
				time.Sleep(1 * time.Second)
			}
		}
	}()

	select {
	case <- nodechan:
		fmt.Println("Node notified")
		
	case <- time.After(5 * time.Second):
		fmt.Println("Sending timed out. Aborting...")
		return err
	}

	return err
}
