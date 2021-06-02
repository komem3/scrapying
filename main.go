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
	"github.com/hashicorp/logutils"
	"golang.org/x/sync/errgroup"
	"golang.org/x/xerrors"
)

func main() {
	filter := &logutils.LevelFilter{
		Levels:   []logutils.LogLevel{"DEBUG", "INFO", "WARN", "ERROR"},
		MinLevel: logutils.LogLevel("INFO"),
		Writer:   os.Stderr,
	}
	log.SetOutput(filter)

	ctx := context.Background()

	if !metadata.OnGCE() {
		end, err := NewLocalDatastore()
		if err != nil {
			log.Fatal(err)
		}
		defer end()
	}

	client, err := datastore.NewClient(ctx, "")
	if err != nil {
		log.Printf("[ERROR] %v", err)
		return
	}
	dsClient = client

	doc, err := fetchHTML(ctx, os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("[DEBUG] document: %#v\n", doc)

	links := getLinks(doc)
	eg, ctx := errgroup.WithContext(ctx)
	for _, link := range links {
		link := link
		eg.Go(func() error {
			imageDoc, err := fetchHTML(ctx, link)
			if err != nil {
				return err
			}
			img := getImage(imageDoc)
			log.Printf("[INFO] image: %s\n", img)

			comic := NewCommic(img)
			_, err = GetComic(ctx, comic.ID)
			if xerrors.Is(err, datastore.ErrNoSuchEntity) {
				err = SaveComic(ctx, comic)
				if err != nil {
					return err
				}
				// TODO: discord に web hook してあげる
			}
			if err != nil {
				return err
			}
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		log.Fatal(err)
	}
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
