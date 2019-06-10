package main

import (
	_"github.com/go-sql-driver/mysql"
	crand "crypto/rand"
	"encoding/json"
	"database/sql"
	"math/rand"
	"net/http"
	"bytes"
	"time"
	"fmt"
	"io"
)

type user struct {
	Uuid string
	Lat float64
	Long float64
	Cdate int64
	Rep int8
}

func genRUD() (string,float64,float64,int64,int8) {
//func generateRandUserData() []interface{} {
	nonce := make([]byte,8)	
	_, err := io.ReadFull(crand.Reader, nonce)
	if err != nil { fmt.Println(err) }
	
	d:= fmt.Sprintf("%x",nonce)
	x:= (rand.Float64() * 2) -102
	y:= (rand.Float64() * 2) +20
	t:= time.Now().Unix()
	r:= int8(rand.Intn(2))

	return d,x,y,t,r
}

func wrap(vs ...interface{}) []interface{} {
    return vs
}

func generateJson() ([]byte,error) {
	
	d,x,y,t,r := genRUD()
	
	cosa:=user{d,x,y,t,r}
	jdata,err:=json.Marshal(cosa)
	
	return jdata,err
}

func postUser(server_address string) (*http.Response, error) {

	jsondata,err := generateJson()
	res,err :=
		http.Post(
		server_address,
		"application/json",
		bytes.NewBuffer(jsondata)	)
	if err != nil {}

	return res,err
}

func generateDBUsers(quantity int) {
	database,_ := sql.Open ("mysql","root:rootpass@/huachicol")
	defer database.Close()
	
	for i:= 0; i < quantity; i++ {
		query:=
			fmt.Sprintf(
			"call insertUsuario('%s',%f,%f,FROM_UNIXTIME(%d),%d)",
			wrap(genRUD())...)

		println(i)
		q,err := database.Prepare(query)
		_,err = q.Exec()
		if err != nil {fmt.Println("Err: ", err)}
	}
}

func main() {
	//generateDBUsers(10)
	for {
		println(":)")
		res,err:=postUser("http://localhost:8080/report")
		if err != nil {
			fmt.Println("Post: ", err)
		} else {
			fmt.Println(res.Status)
		}
		time.Sleep(5 * time.Second)
	}
}
