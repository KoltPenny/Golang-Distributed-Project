package main

import (
	_"github.com/go-sql-driver/mysql"
	//"encoding/json"
	"database/sql"
	//"io/ioutil"
	//"strings"
	"fmt"
	//"os"
)

type geometry struct{
	Type string `json:"type"`
	Coordinates interface{} `json:"coordinates"`
}

type properties struct{
	Layer string `json:"LAYER"`
	Id int `json:"ID"`
}

type feature struct{
	Type string `json:"type"`
	Id string `json:"id"`
	Geometry geometry `json:"geometry"`
	Geometry_name string`json:"geometry_name"`
	Properties properties `json:"properties"`
}

type geojson struct{
	Type string `json:"type"`
	TotalFeatures int `json:"totalFeatures"`
	Features []feature `json:"features"`
}

func tablePointsToGeo() geojson {

	database,_ := sql.Open ("mysql","root:rootpass@/huachicol")
	defer database.Close()

	rows,err := database.Query("call getNearPoints()")
	if err != nil { fmt.Println(err) }

	geo := geojson {"FeatureCollection",0,make([]feature,0)}

	for rows.Next() {

		temp_feat := feature{
			"Feature",
			"",
			geometry{ "Point" ,	make([]float64,2) },
			"usr_p",
			properties{"Points",geo.TotalFeatures},
		}
		
		rows.Scan(
			&temp_feat.Id,
			&temp_feat.Geometry.Coordinates.([]float64)[0],
			&temp_feat.Geometry.Coordinates.([]float64)[1],
		)

		geo.Features = append(geo.Features,temp_feat)
		geo.TotalFeatures += 1
		
	}

	return geo

}

/*
func main() {
	geo:=tablePointsToGeo()
	//obj,_ := json.MarshalIndent(geo,""," ")
	obj,_ := json.Marshal(geo)
	os.Stdout.Write(obj)
}

/*
func main() {

	file,_ := ioutil.ReadFile("ductos.json")
	var jsondata geojson
	json.Unmarshal(file,&jsondata)
  jsons, _ := json.MarshalIndent(jsondata, " ", " ")
	//fmt.Println(string(jsons))

	database,_ := sql.Open ("mysql","root:rootpass@/huachicol")
	defer database.Close()

	for _,element := range jsondata.Features {
		var st []string
		for _,geom := range element.Geometry.Coordinates[0] {
			
			st = append(st,fmt.Sprintf("POINT(%f,%f)",geom[0],geom[1]))
		}
		last := strings.Join(st,",")
		query := fmt.Sprintf("INSERT INTO duct_line values ('%s',LINESTRING(%s))",element.Id,last)
		fmt.Println(query)
		q,err := database.Prepare(query)
		q.Exec()
		if err != nil {println(err)}
	}
}*/

