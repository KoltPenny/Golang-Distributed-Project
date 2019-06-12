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
	"os"
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
	fmt.Println(resp,err)
	return resp,err
}

func CastReport (jsondata []byte, update_process string, next_process string) error {
/* SENDS REPORTS TO UPDATE PROCESS AND NEXT PROCESS */

	//Update database

	req,_ := http.NewRequest("POST",update_process,bytes.NewBuffer(jsondata))
	tr := &http.Transport{}
	client := &http.Client{Transport: tr}

	fmt.Println("Updating database...")
	
	done := make(chan error,1)

	go func() {
		resp,err := client.Do(req)
		if err != nil {	done <- err	}
		
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		plaintext,err := decipher(buf.Bytes())
		os.Stdout.Write(plaintext)
		
		if err != nil {
			done <- err
		}

		done <- nil

	}()

	timeout := make(chan bool,1)
	
	go func(){
		time.Sleep(2 * time.Second)
		tr.CancelRequest(req)
		timeout <- true
	}()

	select {
	case <- timeout:
		fmt.Println("Request Cancelled")
		return errors.New("TIMEOUT")
	case err := <- done:
		fmt.Println("Request Fulfilled")
		if err != nil {
			fmt.Println("Request ERROR: ",err)
		}
	}

	//Notify Manager

	req2,_ := http.NewRequest("POST",next_process,bytes.NewBuffer(jsondata))

	fmt.Println("Notifying manager...")
	
	done = make(chan error,1)

	go func() {
		resp,err := client.Do(req2)
		if err != nil {	done <- err	}
		
		buf := new(bytes.Buffer)
		buf.ReadFrom(resp.Body)
		plaintext,err := decipher(buf.Bytes())
		os.Stdout.Write(plaintext)
		
		if err != nil {	done <- err	}

		done <- nil

	}()

	timeout = make(chan bool,1)
	
	go func(){
		time.Sleep(2 * time.Second)
		tr.CancelRequest(req)
		timeout <- true
	}()

	select {
	case <- timeout:
		fmt.Println("Request Cancelled 2")
		return errors.New("TIMEOUT")
	case err := <- done:
		if err != nil {
			fmt.Println("Request ERROR: ",err)
		}
	}

	return nil
}
