package main

import (
	"encoding/json"
	"fmt"
	"github.com/gen2brain/go-fitz"
	"github.com/otiai10/gosseract"
	"image/png"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	setupRoutes()
}

type Page struct {
	Page int    `json:"page"`
	Text string `json:"text"`
}

func setupRoutes() {
	http.HandleFunc("/", uploadHandler)
	http.ListenAndServe(":3000", nil)
}

func uploadHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Println("File Upload endpoint hit")

	w.Header().Set("Access-Control-Allow-Origin", "*")
	if (r.Method == "OPTIONS") {
		fmt.Println("CORS HIT")
		allowedHeaders := "Content-Type"
		w.Header().Set("Access-Control-Allow-Methods", "POST")
		w.Header().Set("Access-Control-Allow-Headers", allowedHeaders)
		return
	}

	r.ParseMultipartForm(10 << 20)
	file, handler, err := r.FormFile("pdf")
	if err != nil {
		fmt.Println("Error retrieving the file")
		fmt.Println(err)
		return
	}
	defer file.Close()
	fmt.Printf("Uploaded File: %+v\n", handler.Filename)
	fmt.Printf("File Size: %+v\n", handler.Size)
	fmt.Printf("MIME Header: %+v\n", handler.Header)

	// Create a temporary file within our temp-images directory that follows
	// a particular naming pattern
	tempFile, err := ioutil.TempFile("files", "upload-*.pdf")
	if err != nil {
		fmt.Println(err)
	}
	defer tempFile.Close()

	// read all of the contents of our uploaded file into a
	// byte array
	fileBytes, err := ioutil.ReadAll(file)
	if err != nil {
		fmt.Println(err)
	}

	// write this byte array to our temporary file
	tempFile.Write(fileBytes)
	// return that we have successfully uploaded our file!

	doc, err := fitz.New(tempFile.Name())
	if err != nil {
		panic(err)
	}

	defer doc.Close()

	filePrefix := strings.TrimSuffix(tempFile.Name(), filepath.Ext(tempFile.Name()))
	filePrefix = filepath.Base(filePrefix)
	// Extract pages as images
	for n := 0; n < doc.NumPage(); n++ {
		img, err := doc.Image(n)
		if err != nil {
			fmt.Println("erro extract1")
			panic(err)
		}

		f, err := os.Create(filepath.Join("files", fmt.Sprintf(filePrefix+"-%03d.png", n)))
		if err != nil {
			fmt.Println("erro extract2")
			panic(err)
		}

		err = png.Encode(f, img)
		if err != nil {
			fmt.Println("erro extract3")
			panic(err)
		}

		f.Close()
	}

	files, err := filepath.Glob(filepath.Join("files", filePrefix+"-*"))
	if err != nil {
		fmt.Println("erro parse files")
		panic(err)
	}

	tesseractClient := gosseract.NewClient()
	tesseractClient.SetLanguage("por")
	defer tesseractClient.Close()

	var paginas []Page
	for i, v := range files {
		err = tesseractClient.SetImage(v)
		if err != nil {
			fmt.Println("Erro set image")
			panic(err)
		}

		text, err := tesseractClient.Text()

		if err != nil {
			fmt.Println("Erro get text")
			panic(err)
		}

		paginas = append(paginas, Page{
			Page: i,
			Text: text,
		})
	}

	jsonData, _ := json.Marshal(paginas)
	fmt.Fprintf(w, string(jsonData))
	//fmt.Fprintf(w, "time: %s\n", time.Since(start))
}
