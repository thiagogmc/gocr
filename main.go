package main

import (
	"encoding/json"
	"fmt"
	"github.com/gen2brain/go-fitz"
	"github.com/otiai10/gosseract"
	"image"
	"image/png"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
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
	start := time.Now()
	fmt.Println("File Upload endpoint hit")
	var wg sync.WaitGroup

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


	wg.Add(1)
	convertPdfToImage(tempFile.Name(), &wg)
	wg.Wait()
	filePrefix := strings.TrimSuffix(tempFile.Name(), filepath.Ext(tempFile.Name()))
	filePrefix = filepath.Base(filePrefix)

	files, err := filepath.Glob(filepath.Join("files", filePrefix+"-*"))
	if err != nil {
		fmt.Println("erro parse files")
		panic(err)
	}

	var wg2 sync.WaitGroup
	wg2.Add(1)
	paginas := extrairTextoDeArquivos(files, &wg2)
	wg2.Wait()

	jsonData, _ := json.Marshal(paginas)
	fmt.Println("Fim Requisicao")
	fmt.Println(time.Since(start))
	fmt.Fprintf(w, string(jsonData))
	//fmt.Fprintf(w, "time: %s\n", time.Since(start))
}

func convertPdfToImage(filename string, wg *sync.WaitGroup) {
	defer wg.Done()
	fmt.Println("Iniciando pdf to image")
	doc, err := fitz.New(filename)
	if err != nil {
		panic(err)
	}

	defer doc.Close()

	filePrefix := strings.TrimSuffix(filename, filepath.Ext(filename))
	filePrefix = filepath.Base(filePrefix)
	// Extract pages as images

	for n := 0; n < doc.NumPage(); n++ {
		fmt.Printf("Inicio PNG Pag: %d\n", n)
		img, err := doc.Image(n)
		if err != nil {
			panic(err)
		}
		wg.Add(1)
		go saveImage(img, n, filePrefix, wg)
	}
}

func saveImage(img image.Image, n int, filePrefix string, wg *sync.WaitGroup)  {
	defer wg.Done()
	f, err := os.Create(filepath.Join("files", fmt.Sprintf(filePrefix+"-%03d.png", n)))
	if err != nil {
		fmt.Println("Erro criacao de arquivo de imagem")
		panic(err)
		return
	}

	err = png.Encode(f, img)
	if err != nil {
		fmt.Println("Erro no encode png")
		panic(err)
		return
	}

	f.Close()
	fmt.Printf("Fim PNG pagina: %d\n", n)
}

func extrairTextoDeArquivos(files []string, wg *sync.WaitGroup) []Page {
	defer wg.Done()
	var paginas []Page

	for i, v := range files {
		page := Page{
			Page: i,
		}
		paginas = append(paginas, page)

		wg.Add(1)
		fmt.Println("Extraindo texto pag: %d", i)
		go extrairTexto(v, wg, &page)
	}
	return paginas
}

func extrairTexto(imgPath string, wg *sync.WaitGroup, page *Page) {
	defer wg.Done()
	tesseractClient := gosseract.NewClient()
	tesseractClient.SetLanguage("por")
	defer tesseractClient.Close()
	err := tesseractClient.SetImage(imgPath)

	if err != nil {
		fmt.Println("Erro set image")
		panic(err)
	}

	text, err := tesseractClient.Text()

	if err != nil {
		fmt.Println("Erro get text")
		panic(err)
	}

	fmt.Println("Fim extracao texto")
	page.Text = text
}