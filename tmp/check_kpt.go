package main

import (
	"fmt"
	"reflect"
	"github.com/kptdev/kpt/pkg/api/kptfile/v1"
)

func main() {
	f := v1.Function{}
	t := reflect.TypeOf(f)
	for i := 0; i < t.NumField(); i++ {
		fmt.Println(t.Field(i).Name)
	}
}
