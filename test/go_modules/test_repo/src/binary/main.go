package main

import (
	"github.com/golang/snappy" // includes asm sources
)

func main(){
	_ = snappy.MaxEncodedLen(1234)
}