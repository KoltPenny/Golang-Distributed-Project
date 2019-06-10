package main

import (
	"enconding/json"
	"fmt"
	"os"
)

type geometry struct{
	Type string
	Coordinates [][][2]float64
}

type properties struct{
	Layer string
	Id int
	Length_met float64
}

type feature struct{
	Type string
	Id string
	Geometry geometry
	Geometry_name string
	properties
}

type geojson struct{
	Type string
	TotalFeatures int
	Features []feature
}

func main () {

	

}
