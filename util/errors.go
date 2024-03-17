package util

import (
	"log"
	"net/http"
)

func CheckError(err error) {
	if err != nil {
		log.Panicln(err)
	}
}

func OnPanic(w http.ResponseWriter) {
	if r := recover(); r != nil {
		log.Println(r)
		http.Error(w, "Server error", 500)
	}
}
