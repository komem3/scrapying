package main

import (
	"context"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"cloud.google.com/go/compute/metadata"
	"cloud.google.com/go/datastore"
	"github.com/PuerkitoBio/goquery"
	"github.com/go-chi/chi/v5"
	"github.com/hashicorp/logutils"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
	"golang.org/x/xerrors"
)

var (
	discordURL string
	mediumURL  string
	level      string
	port       string
	projectID  string
)

func init() {
	pflag.StringP("medium", "m", "", "target meduim url")
	pflag.StringP("discord", "d", "", "discord webhook url")
	pflag.StringP("level", "v", "INFO", "output verbose level")
	pflag.StringP("port", "P", "8080", "bind port")
	pflag.String("project", "test-project", "project id")

	pflag.Parse()
	viper.BindPFlags(pflag.CommandLine)

	viper.BindEnv("discord")
	viper.BindEnv("medium")
	viper.BindEnv("level")
	viper.BindEnv("port")

	if metadata.OnGCE() {
		p, err := metadata.ProjectID()
		if err != nil {
			log.Fatalf("[ERROR] project id is missing: %v", err)
		}
		projectID = p
	}

	discordURL = viper.GetString("discord")
	if discordURL == "" {
		pflag.PrintDefaults()
		log.Fatal("[ERROR] discord is required.")
	}
	mediumURL = viper.GetString("medium")
	if mediumURL == "" {
		pflag.PrintDefaults()
		log.Fatal("[ERROR] meduim is required.")
	}
	level = viper.GetString("level")
	port = viper.GetString("port")
	filter := &logutils.LevelFilter{
		Levels:   []logutils.LogLevel{"DEBUG", "INFO", "WARN", "ERROR"},
		MinLevel: logutils.LogLevel(level),
		Writer:   os.Stderr,
	}
	log.SetOutput(filter)
}

func main() {
	log.Printf("[DEBUG] discordURL is '%s'", discordURL)
	log.Printf("[DEBUG] mediumURL is '%s'", mediumURL)
	ctx := context.Background()

	if !metadata.OnGCE() {
		end, err := NewLocalDatastore()
		if err != nil {
			log.Fatal(err)
		}
		defer end()
	}

	client, err := datastore.NewClient(ctx, projectID)
	if err != nil {
		log.Printf("[ERROR] %v", err)
		return
	}
	dsClient = client

	r := chi.NewRouter()
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		if err := scrayping(r.Context()); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	log.Printf("bind port: %s", port)
	http.ListenAndServe(":"+port, r)
}

func getImage(doc *goquery.Document) string {
	var image string
	doc.Find(".paragraph-image").Each(func(i int, s *goquery.Selection) {
		if i != 0 {
			return
		}
		atag := s.Find("img")
		src, _ := atag.Attr("src")
		log.Printf("[DEBUG] Link %d: %s\n", i, src)
		image = src
	})
	return image
}

func getLinks(doc *goquery.Document) []string {
	var links []string
	doc.Find(".postItem").Each(func(i int, s *goquery.Selection) {
		// For each item found, get the band and title
		atag := s.Find("a")
		title := atag.Text()
		link, _ := atag.Attr("href")
		log.Printf("[DEBUG] Link %d: %s - %s\n", i, link, title)

		links = append(links, link)
	})
	return links
}

func fetchHTML(ctx context.Context, link string) (*goquery.Document, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, link, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		bytes, err := io.ReadAll(resp.Body)
		defer resp.Body.Close()
		if err != nil {
			return nil, err
		}
		return nil, xerrors.Errorf("response error(status: %d, body: %s)", resp.StatusCode, bytes)
	}
	return goquery.NewDocumentFromReader(resp.Body)
}
