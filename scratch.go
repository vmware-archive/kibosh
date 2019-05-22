package main

import (
	"fmt"
	"github.com/cf-platform-eng/kibosh/pkg/httphelpers"
	"io"
	"net/http"
	"os"
)

func UHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var err error
		switch r.Method {
		case "POST":
			err = uploadHandler(w, r)
			break

		default:
			w.WriteHeader(405)
			w.Header().Set("Allow", "POST")
		}

		if err != nil {
			fmt.Fprintf(w, "Error writing response ")
		}
	})
}

func uploadHandler(w http.ResponseWriter, r *http.Request) error {

	err := r.ParseMultipartForm(200000) // grab the multipart form
	if err != nil {
		fmt.Fprintln(w, err)
		return nil
	}

	formdata := r.MultipartForm // ok, no problem so far, read the Form data

	//get the *fileheaders
	files := formdata.File["multiplefiles"] // grab the filenames

	for i, _ := range files { // loop through the files one by one
		file, err := files[i].Open()
		defer file.Close()
		if err != nil {
			fmt.Fprintln(w, err)
			return nil
		}

		out, err := os.Create("/tmp/aam/" + files[i].Filename)

		defer out.Close()
		if err != nil {
			fmt.Fprintf(w, "Unable to create the file for writing. Check your write access privilege")
			return nil
		}

		_, err = io.Copy(out, file) // file not files[i] !

		if err != nil {
			fmt.Fprintln(w, err)
			return nil
		}

		fmt.Fprintf(w, "Files uploaded successfully : ")
		fmt.Fprintf(w, files[i].Filename+"\n")

	}
	return nil
}

func main() {
	authFilter := httphelpers.NewAuthFilter("admin", "nopassword")
	http.Handle("/charts", authFilter.Filter(UHandler(),
		))
	http.ListenAndServe(":8080", nil)
}