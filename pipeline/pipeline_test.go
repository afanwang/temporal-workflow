package pipeline

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.temporal.io/sdk/testsuite"
)

const (
	gitUrl = "https://github.com/afanwang/go-sample.git"
)

func TestPipelineWorkflow(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	// Mock GitClone and DeleteWorkdir for all tests
	env.OnActivity(pa.GitClone, mock.Anything, mock.Anything).Return(&GitCloneResult{Metadata: PipelineActivityMetadata{Workdir: "/tmp/test"}}, nil)
	env.OnActivity(pa.DeleteWorkdir, mock.Anything, mock.Anything).Return(nil)

	t.Run("All steps succeed", func(t *testing.T) {
		mockAllActivitiesSuccess(env)
		env.OnActivity(pa.GoDeploy, mock.Anything, mock.Anything).Return(&GoDeployResult{}, nil)

		env.ExecuteWorkflow(PipelineWorkflow, PipelineParams{GitURL: "https://github.com/afanwang/go-sample.git"})

		assert.True(t, env.IsWorkflowCompleted())
		assert.NoError(t, env.GetWorkflowError())

		var result PipelineResult
		assert.NoError(t, env.GetWorkflowResult(&result))
		assert.Empty(t, result.Failures)
	})

	t.Run("Some failures introduced by fail flags", func(t *testing.T) {
		mockActivitiesWithFailures(env)

		env.ExecuteWorkflow(PipelineWorkflow, PipelineParams{
			GitURL:        "https://github.com/afanwang/go-sample.git",
			TestFlags:     []string{"-tags", "failtest"},
			BuildFlags:    []string{"-tags", "failbuild"},
			GenerateFlags: []string{"-tags", "failgenerate"},
		})

		assert.True(t, env.IsWorkflowCompleted())
		assert.NoError(t, env.GetWorkflowError())

		var result PipelineResult
		assert.NoError(t, env.GetWorkflowResult(&result))
		assert.Len(t, result.Failures, 3)

		// Check for GoTest failure
		foundGoTestFailure := false
		for _, failure := range result.Failures {
			if failure.Activity == "GoTest" {
				foundGoTestFailure = true
			}
		}
		assert.True(t, foundGoTestFailure)

		// Check for GoBuild failure
		foundGoBuildFailure := false
		for _, failure := range result.Failures {
			if failure.Activity == "GoBuild" {
				if details, ok := failure.Details.([]interface{}); ok {
					assert.Equal(t, []interface{}{"main.go"}, details)
					foundGoBuildFailure = true
				}
			}
		}
		assert.True(t, foundGoBuildFailure)

		// Check for GoGenerate failure
		foundGoGenerateFailure := false
		for _, failure := range result.Failures {
			if failure.Activity == "GoGenerate" {
				if details, ok := failure.Details.([]interface{}); ok {
					assert.Equal(t, []interface{}{"generated.go"}, details)
					foundGoGenerateFailure = true
				}
			}
		}
		assert.True(t, foundGoGenerateFailure)

		// Ensure GoDeploy was not called
		env.AssertNotCalled(t, "OnActivity", pa.GoDeploy, mock.Anything)
	})
}

func mockAllActivitiesSuccess(env *testsuite.TestWorkflowEnvironment) {
	// all 7 passes
	env.OnActivity(pa.GoTest, mock.Anything, mock.Anything).Return(&GoTestResult{}, nil)
	env.OnActivity(pa.GoFmt, mock.Anything, mock.Anything).Return(&GoFmtResult{}, nil)
	env.OnActivity(pa.GoModTidy, mock.Anything, mock.Anything).Return(&GoModTidyResult{}, nil)
	env.OnActivity(pa.GoBuild, mock.Anything, mock.Anything).Return(&GoBuildResult{}, nil)
	env.OnActivity(pa.GoGenerate, mock.Anything, mock.Anything).Return(&GoGenerateResult{}, nil)
	env.OnActivity(pa.GolangCILint, mock.Anything, mock.Anything).Return(&GolangCILintResult{}, nil)
	env.OnActivity(pa.GoDeploy, mock.Anything, mock.Anything).Return(&GoDeployResult{}, nil)
}

func mockActivitiesWithFailures(env *testsuite.TestWorkflowEnvironment) {
	// 3 passes
	env.OnActivity(pa.GoFmt, mock.Anything, mock.Anything).Return(&GoFmtResult{}, nil)
	env.OnActivity(pa.GoModTidy, mock.Anything, mock.Anything).Return(&GoModTidyResult{}, nil)
	env.OnActivity(pa.GolangCILint, mock.Anything, mock.Anything).Return(&GolangCILintResult{}, nil)

	// 3 failures
	env.OnActivity(pa.GoBuild, mock.Anything, mock.Anything).Return(&GoBuildResult{FailedFiles: []string{"main.go"}}, nil)
	env.OnActivity(pa.GoGenerate, mock.Anything, mock.Anything).Return(&GoGenerateResult{FailedFiles: []string{"generated.go"}}, nil)
	env.OnActivity(pa.GoTest, mock.Anything, mock.Anything).Return(&GoTestResult{FailedTests: []GoTestCLIOutput{{Test: "TestFailed"}}}, nil)
}
