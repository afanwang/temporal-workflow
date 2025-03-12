package main

import (
	"context"
	"fmt"
	"log/slog"

	"temporal-workflow/pipeline"

	"github.com/kelseyhightower/envconfig"
	tworker "go.temporal.io/sdk/worker"
)

func RunWorker(ctx context.Context) error {
	var tOpts TemporalOptions
	if err := envconfig.Process("temporal", &tOpts); err != nil {
		return fmt.Errorf("failed to process environment variables: %w", err)
	}
	var wOpts tworker.Options
	if err := envconfig.Process("temporal", &wOpts); err != nil {
		return fmt.Errorf("failed to process Temporal environment variables: %w", err)
	}

	slog.Info(
		"Temporal worker options",
		"server", fmt.Sprintf("%+v", tOpts),
		"worker", fmt.Sprintf("%+v", wOpts),
	)

	tc, err := NewTemporalClient(ctx, tOpts)
	if err != nil {
		return fmt.Errorf("failed to connect to Temporal server %q: %w", tOpts.HostPort, err)
	}
	defer tc.Close()

	worker := tworker.New(tc, tOpts.Queue, wOpts)

	worker.RegisterWorkflow(pipeline.PipelineWorkflow)

	pa := pipeline.PipelineActivity{}
	worker.RegisterActivity(pa.GitClone)
	worker.RegisterActivity(pa.GoTest)
	worker.RegisterActivity(pa.GoFmt)
	worker.RegisterActivity(pa.GoGenerate)
	worker.RegisterActivity(pa.GoModTidy)
	worker.RegisterActivity(pa.GolangCILint)
	worker.RegisterActivity(pa.GoBuild)
	worker.RegisterActivity(pa.GoDeploy)
	worker.RegisterActivity(pa.DeleteWorkdir)

	return worker.Run(tworker.InterruptCh())
}
