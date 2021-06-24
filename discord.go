package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/bwmarrin/discordgo"
	"golang.org/x/xerrors"
)

type DiscordClient struct {
	*http.Client
	webhookURL string
}

func (d *DiscordClient) Comment(ctx context.Context, contents string) error {
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()

	b := &bytes.Buffer{}
	webhook := discordgo.WebhookParams{
		Content: contents,
	}
	if err := json.NewEncoder(b).Encode(&webhook); err != nil {
		return xerrors.Errorf("encode webhook params contents(%s): %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, d.webhookURL, b)
	if err != nil {
		return xerrors.Errorf("create new request(method: post, url: %s) %+v: %w", d.webhookURL, webhook, err)
	}
	req.Header.Add("Content-Type", "application/json")

	resp, err := d.Do(req)
	if err != nil {
		return xerrors.Errorf("request %v: %w", req, err)
	}
	if resp.StatusCode == http.StatusNoContent {
		return nil
	}

	if resp.StatusCode != http.StatusOK {
		respByte, err := io.ReadAll(resp.Body)
		if err != nil {
			return xerrors.Errorf("status %d, read all: %w", resp.StatusCode, err)
		}
		return xerrors.Errorf("request %+v, response status %d, response %s", webhook, resp.StatusCode, respByte)
	}
	respByte, err := io.ReadAll(resp.Body)
	if err != nil {
		return xerrors.Errorf("status %d: %w", resp.Status, err)
	}
	log.Printf("[DEBUG] response: %s", respByte)
	return nil
}
