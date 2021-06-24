package main

import (
	"context"
	"log"
	"net/http"

	"cloud.google.com/go/datastore"
	"golang.org/x/sync/errgroup"
	"golang.org/x/xerrors"
)

func scrayping(ctx context.Context) error {
	doc, err := fetchHTML(ctx, mediumURL)
	if err != nil {
		return err
	}
	log.Printf("[DEBUG] document: %#v\n", doc)

	discordClient := &DiscordClient{Client: &http.Client{}, webhookURL: discordURL}

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
				if err := discordClient.Comment(ctx, img); err != nil {
					return err
				}
			}
			if err != nil {
				return err
			}
			return nil
		})
	}

	return eg.Wait()
}
