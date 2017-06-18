package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"sort"
)

type Comic struct {
	Month      string `json:"month"`
	Num        int    `json:"num"`
	Link       string `json:"link"`
	Year       string `json:"year"`
	News       string `json:"news"`
	SafeTitle  string `json:"safe_title"`
	Transcript string `json:"transcript"`
	Alt        string `json:"alt"`
	Img        string `json:"img"`
	Title      string `json:"title"`
	Day        string `json:"day"`
	ImgName    string
}

var Metadatas = make([]Comic, 0)

func main() {
	downloadComics()
	buildIndex()
}

func downloadComics() {

	data, err := fetchMeta("https://xkcd.com/info.0.json")
	if err != nil {
		panic(err)
	}

	jobs := make(chan int, 2000)
	results := make(chan int, 2000)

	for w := 1; w <= 10; w++ {
		go fetcher(w, jobs, results)
	}

	for j := 1; j <= data.Num; j++ {
		jobs <- j
	}
	close(jobs)

	for a := 1; a <= (data.Num - 3); a++ {
		<-results
	}
}

func fetcher(id int, jobs <-chan int, results chan<- int) {

	postTemplate, err := template.ParseFiles("post.template")
	if err != nil {
		panic(err)
	}

	for j := range jobs {

		if j == 404 || j == 1608 || j == 1663 {
			// These don't actually exist in image format.
			continue
		}

		fmt.Println("Fetcher", id, "started downloading metadata for:", j)

		url := fmt.Sprintf("https://xkcd.com/%d/info.0.json", j)

		data, err := fetchMeta(url)
		if err != nil {
			fmt.Println(err)
		}

		Metadatas = append(Metadatas, *data)

		err = writePost(postTemplate, data)
		if err != nil {
			fmt.Println(err)
		}
		fmt.Println("Fetcher", id, "finished downloading metadata for:", j)

		fmt.Println("Fetcher", id, "started downloading comic no:", j)

		err = downloadImage(data)
		if err != nil {
			fmt.Println(err)
		}

		fmt.Println("Fetcher", id, "completed downloading comic no:", j)

		results <- j
	}
}

func buildIndex() error {

	t, err := template.ParseFiles("index.template")
	if err != nil {
		panic(err)
	}

	filename := "./data/index.html"

	output, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer output.Close()

	sort.Slice(Metadatas, func(i, j int) bool {
		return Metadatas[i].Num < Metadatas[j].Num
	})

	t.Execute(output, Metadatas)

	return nil
}

func fetchMeta(url string) (*Comic, error) {
	resp, err := http.Get(url)

	if err != nil {
		panic(err)
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		panic(err)
	}

	data := Comic{}
	err = json.Unmarshal(body, &data)
	if err != nil {
		fmt.Println(url)
		panic(err)
	}

	data.ImgName = fmt.Sprintf("%d%s", data.Num, filepath.Ext(data.Img))

	return &data, err
}

func writePost(t *template.Template, data *Comic) error {
	filename := fmt.Sprintf("./data/%d.html", data.Num)

	output, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer output.Close()

	t.Execute(output, data)

	return nil
}

func downloadImage(data *Comic) error {
	filename := fmt.Sprintf("./data/%s", data.ImgName)
	output, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer output.Close()

	resp, err := http.Get(data.Img)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	_, err = io.Copy(output, resp.Body)
	if err != nil {
		fmt.Println("Download failed:", data.Img, data.Num)
	}

	return nil
}
