package main

import (
	"context"
	"log"
	"os"

	"cloud.google.com/go/datastore"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
)

const (
	comicKind = "Comic"
)

var dsClient *datastore.Client

func SaveComic(ctx context.Context, comic *Comic) error {
	key := datastore.NameKey(comicKind, comic.ID, nil)
	log.Printf("[DEBUG] save comic %s", comic.ID)
	_, err := dsClient.Put(ctx, key, comic)
	return err
}

func GetComic(ctx context.Context, id string) (*Comic, error) {
	key := datastore.NameKey(comicKind, id, nil)
	var comic Comic
	if err := dsClient.Get(ctx, key, &comic); err != nil {
		return nil, err
	}
	log.Printf("[DEBUG] get comic %s", comic.ID)
	return &comic, nil
}

func NewLocalDatastore() (func() error, error) {
	const projectID = "test-project"
	// uses a sensible default on windows (tcp/http) and linux/osx (socket)
	pool, err := dockertest.NewPool("")
	if err != nil {
		log.Fatalf("Could not connect to docker: %s", err)
	}

	// pulls an image, creates a container based on it and runs it
	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: "singularities/datastore-emulator",
		Env: []string{
			"DATASTORE_LISTEN_ADDRESS=0.0.0.0:8081",
			"DATASTORE_PROJECT_ID=" + projectID,
		},
	}, func(config *docker.HostConfig) {
		// set AutoRemove to true so that stopped container goes away by itself
		config.AutoRemove = true
		config.RestartPolicy = docker.RestartPolicy{
			Name: "no",
		}
	})
	if err != nil {
		log.Fatalf("Could not start resource: %s", err)
	}

	os.Setenv("DATASTORE_PROJECT_ID", projectID)
	os.Setenv("DATASTORE_EMULATOR_HOST", "localhost:"+resource.GetPort("8081/tcp"))

	return func() error {
		return pool.Purge(resource)
	}, nil
}
