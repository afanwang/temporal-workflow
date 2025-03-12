package main

import (
	"context"

	tclient "go.temporal.io/sdk/client"
)

type TemporalOptions struct {
	HostPort  string `required:"true"`
	Namespace string `required:"true"`
	Queue     string `required:"true"`
}

func NewTemporalClient(ctx context.Context, opts TemporalOptions) (tclient.Client, error) {
	return tclient.DialContext(ctx, tclient.Options{
		HostPort:  opts.HostPort,
		Namespace: opts.Namespace,
	})
}
