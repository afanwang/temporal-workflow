package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"

	"temporal-workflow/pipeline"

	"github.com/gosimple/slug"
	"github.com/kelseyhightower/envconfig"
	tclient "go.temporal.io/sdk/client"
	"gopkg.in/yaml.v3"
)

type WorkflowOptions struct {
	Input string `required:"true"`
}

func RunPipeline(pctx context.Context) error {
	ctx, cancel := signal.NotifyContext(pctx, os.Interrupt, os.Kill)
	defer cancel()

	var opts WorkflowOptions
	if err := envconfig.Process("workflow", &opts); err != nil {
		return fmt.Errorf("failed to process environment variables: %w", err)
	}

	var tOpts TemporalOptions
	if err := envconfig.Process("temporal", &tOpts); err != nil {
		return fmt.Errorf("failed to process Temporal environment variables: %w", err)
	}

	tc, err := NewTemporalClient(ctx, tOpts)
	if err != nil {
		return fmt.Errorf("failed to connect to Temporal server %q: %w", tOpts.HostPort, err)
	}
	defer tc.Close()

	params := pipeline.PipelineParams{}
	f, err := os.ReadFile(opts.Input)
	if err != nil {
		return fmt.Errorf("failed to read input file %q: %w", opts.Input, err)
	}
	if err := yaml.Unmarshal(f, &params); err != nil {
		return fmt.Errorf("failed to unmarshal input file %q: %w", opts.Input, err)
	}
	if err := params.Validate(); err != nil {
		return fmt.Errorf("invalid input file %q: %w", opts.Input, err)
	}

	fWorkflow, err := tc.ExecuteWorkflow(ctx, tclient.StartWorkflowOptions{
		ID:        fmt.Sprintf("PipelineWorkflow-%s", slug.Make(params.GitURL)),
		TaskQueue: tOpts.Queue,
	}, "PipelineWorkflow", params)
	if err != nil {
		return fmt.Errorf("failed to execute workflow: %w", err)
	}
	slog.Info("Started PipelineWorkflow", "WorkflowID", fWorkflow.GetID(), "RunID", fWorkflow.GetRunID())
	if err := fWorkflow.Get(ctx, nil); err != nil {
		return fmt.Errorf("failed to get workflow result: %w", err)
	}
	return nil
}
