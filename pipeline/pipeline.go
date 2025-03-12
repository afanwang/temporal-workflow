package pipeline

import (
	"fmt"
	"reflect"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type PipelineParams struct {
	GitURL        string   `json:"git_url" yaml:"git_url"`
	TestFlags     []string `json:"test_flags" yaml:"test_flags"`
	BuildFlags    []string `json:"build_flags" yaml:"build_flags"`
	GenerateFlags []string `json:"generate_flags" yaml:"generate_flags"`
}

func (pp *PipelineParams) Validate() error {
	if pp.GitURL == "" {
		return fmt.Errorf("GitURL is required")
	}
	return nil
}

type PipelineResult struct {
	Failures []PipelineFailure `json:"failures"`
}

type PipelineFailure struct {
	Activity string `json:"activity"`
	Details  any    `json:"details"`
}

var pa = PipelineActivity{}

func PipelineWorkflow(ctx workflow.Context, params PipelineParams) (*PipelineResult, error) {
	result := &PipelineResult{Failures: []PipelineFailure{}}

	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 3,
		},
	})

	fClone := workflow.ExecuteActivity(ctx, pa.GitClone, GitCloneParams{
		Remote: params.GitURL,
	})
	rClone := &GitCloneResult{}
	if err := fClone.Get(ctx, rClone); err != nil {
		return nil, fmt.Errorf("GitClone activity: %w", err)
	}

	metadata := rClone.Metadata

	// Define activities to run in parallel
	activities := []struct {
		name   string
		future workflow.Future
	}{
		{"GoTest", workflow.ExecuteActivity(ctx, pa.GoTest, GoTestParams{Metadata: metadata, Flags: params.TestFlags})},
		{"GoFmt", workflow.ExecuteActivity(ctx, pa.GoFmt, GoFmtParams{Metadata: metadata})},
		{"GoModTidy", workflow.ExecuteActivity(ctx, pa.GoModTidy, GoModTidyParams{Metadata: metadata})},
		{"GoBuild", workflow.ExecuteActivity(ctx, pa.GoBuild, GoBuildParams{Metadata: metadata, Flags: params.BuildFlags})},
		{"GoGenerate", workflow.ExecuteActivity(ctx, pa.GoGenerate, GoGenerateParams{Metadata: metadata, Flags: params.GenerateFlags})},
		{"GolangCILint", workflow.ExecuteActivity(ctx, pa.GolangCILint, GolangCILintParams{Metadata: metadata})},
	}

	// Create a selector to wait for all activities
	selector := workflow.NewSelector(ctx)
	for i := range activities {
		activity := activities[i]
		selector.AddFuture(activity.future, func(f workflow.Future) {
			// This function will be called when the future is ready
		})
	}

	// Wait for all activities to complete
	for i := 0; i < len(activities); i++ {
		selector.Select(ctx)
	}

	// Process results
	for _, activity := range activities {
		var err error
		switch activity.name {
		case "GoTest":
			var rTest GoTestResult
			err = activity.future.Get(ctx, &rTest)
			if err == nil && len(rTest.FailedTests) > 0 {
				result.Failures = append(result.Failures, PipelineFailure{Activity: activity.name, Details: rTest.FailedTests})
			}
		case "GoFmt":
			var rFmt GoFmtResult
			err = activity.future.Get(ctx, &rFmt)
			if err == nil && len(rFmt.FailedFiles) > 0 {
				result.Failures = append(result.Failures, PipelineFailure{Activity: activity.name, Details: rFmt.FailedFiles})
			}
		case "GoModTidy":
			var rModTidy GoModTidyResult
			err = activity.future.Get(ctx, &rModTidy)
			if err == nil && len(rModTidy.FailedFiles) > 0 {
				result.Failures = append(result.Failures, PipelineFailure{Activity: activity.name, Details: rModTidy.FailedFiles})
			}
		case "GoBuild":
			var rBuild GoBuildResult
			err = activity.future.Get(ctx, &rBuild)
			if err == nil && len(rBuild.FailedFiles) > 0 {
				result.Failures = append(result.Failures, PipelineFailure{Activity: activity.name, Details: rBuild.FailedFiles})
			}
		case "GoGenerate":
			var rGenerate GoGenerateResult
			err = activity.future.Get(ctx, &rGenerate)
			if err == nil && len(rGenerate.FailedFiles) > 0 {
				result.Failures = append(result.Failures, PipelineFailure{Activity: activity.name, Details: rGenerate.FailedFiles})
			}
		case "GolangCILint":
			var rLint GolangCILintResult
			err = activity.future.Get(ctx, &rLint)
			if err == nil && len(rLint.Issues) > 0 {
				result.Failures = append(result.Failures, PipelineFailure{Activity: activity.name, Details: rLint.Issues})
			}
		}
		if err != nil {
			result.Failures = append(result.Failures, PipelineFailure{Activity: activity.name, Details: err.Error()})
		}
	}

	// If all checks pass, execute deploy
	if !hasErrors(result) {
		fDeploy := workflow.ExecuteActivity(ctx, pa.GoDeploy, GoDeployParams{Metadata: metadata})
		rDeploy := &GoDeployResult{}
		if err := fDeploy.Get(ctx, rDeploy); err != nil {
			return nil, fmt.Errorf("deploy activity: %w", err)
		}
		if rDeploy.Error != nil {
			result.Failures = append(result.Failures, PipelineFailure{
				Activity: "Deploy",
				Details:  rDeploy.Error,
			})
		}
	}

	// Finally, workflow finished successfully. Clean up the directory.
	fCleanup := workflow.ExecuteActivity(ctx, pa.DeleteWorkdir, DeleteWorkdirParams{
		Metadata: metadata,
	})
	if err := fCleanup.Get(ctx, nil); err != nil {
		return nil, fmt.Errorf("deleteWorkdir activity: %w", err)
	}

	fmt.Printf("==debug: result=%v", result)

	return result, nil
}

func hasErrors(result *PipelineResult) bool {
	for _, failure := range result.Failures {
		if !isEmptyOrNil(failure.Details) {
			return true
		}
	}
	return false
}

func isEmptyOrNil(value any) bool {
	if value == nil {
		return true
	}

	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.String:
		return v.Len() == 0
	case reflect.Slice:
		return v.Len() == 0 || v.IsNil()
	default:
		return false
	}
}
