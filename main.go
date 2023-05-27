package main

import "fmt"

func main() {
	//haha 为int型map，参数或者说k是string类型
	haha := make(map[string]int, 9)
	haha["jing"] = 20
	haha["lv"] = 89
	//v是20  haha["jing"]代表这个map
	v, ok := haha["jing"]
	if ok {
		fmt.Println(v)
	} else {
		fmt.Println("没这人")
	}
}
