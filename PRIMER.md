# Primer of workflow traits

> [!NOTE]
> If you are already familiar with Temporal, Cadence, go-workflows, or similar systems, this section likely won't contain much new information. This section explains the traits of a workflow.

> [!WARNING]
> If you are not familiar with Temporal, treat the code example here as pseudo-code as the actual package and function names likely differ.

## Programmatic workflow declarations

You may already be familiar with continuous integration configurations. They often look like this:

```yaml
jobs:
  - name: Install dependencies
    run: go mod tidy
  - name: Run tests
    run: go test ./...
```

However, when working on a fairly advanced project, you are likely to find this configuration to be slightly more complicated.

```yaml
jobs:
  - name: Install dependencies
    run: go mod tidy
  - name: Run tests
    run: go test ./...
    on_failure:
      - name: Notify Slack
        run: curl -X POST -d "message=Tests failed" https://slack.com/notify
  - name: Deploy to staging
    run: kubectl apply -f staging.yaml
    on_success:
      - name: Notify Slack
        run: curl -X POST -d "message=Deployed to staging" https://slack.com/notify
```

This snippet above looks _fine_, but any more conditionals would exponentially make this harder to maintain. Compare that configuration to the following:

```go
func ContinousIntegrationWorkflow(ctx context.Context) error {
  if err := doJobs(ctx); err != nil {
    if notifyErr := client.ExecuteActivity(ctx, jobs.NotifySlack, jobs.NotifySlackParams{Channel: "#build-failures"}); notifyErr != nil {
      return errors.Join(err, notifyErr)
    }
    return err
  }
  if err := client.ExecuteActivity(ctx, jobs.NotifySlack, jobs.NotifySlackParams{Channel: "#deploys-staging"}); err != nil {
    return err
  }
  return nil
}

func doJobs(ctx context.Context) error {
  if err := client.ExecuteActivity(ctx, jobs.InstallDependencies, jobs.InstallDependenciesOptions{Timeout: 5 * time.Minute}); err != nil {
    return err
  }
  if err := client.ExecuteActivity(ctx, jobs.RunTests, jobs.RunTestsOptions{Parallelization: 10}); err != nil {
    return err
  }
  if err := client.ExecuteActivity(ctx, jobs.DeployToStaging, jobs.DeployToStagingOptions{}); err != nil {
    return err
  }
  return nil
}
```

While more verbose, the above example allows us to have more control over how we want our workflow to proceed: do everything we want and then notify Slack at the end. The concept of workflows allow us to think about building applications similar to how we think about building CI pipelines. In context of a web application, this may look like:

```go
// Imagine this workflow is responsible for paging someone because their server is down.
func NotifyWorkflow(ctx context.Context, target db.Target) error {
  if err := client.ExecuteActivity(ctx, jobs.NotifyUser, jobs.NotifyUserOptions{Target: target}); err != nil {
    return err
  }

  renotify := time.NewTicker(5 * time.Minute)
  defer renotify.Stop()

  select {
  case <-renotify.C:
    if err := client.ExecuteActivity(ctx, jobs.NotifyUser, jobs.NotifyUserOptions{Target: target}); err != nil {
      return err
    }
  case <-ctx.Done():
    return ctx.Err()
  }
}
```

At this point, you may wonder... why the heck is `client.ExecuteActivity` called and not just the function itself? We will get to that in the next section.

## Determinism

Let's go back at our original example in YAML:

```yaml
jobs:
  - name: Install dependencies
    run: go mod tidy
  - name: Run tests
    run: go test ./...
    on_failure:
      - name: Notify Slack
        run: curl -X POST -d "message=Tests failed" https://slack.com/notify
  - name: Deploy to staging
    run: kubectl apply -f staging.yaml
    on_success:
      - name: Notify Slack
        run: curl -X POST -d "message=Deployed to staging" https://slack.com/notify
```

This configuration is _deterministic_. If we run this configuration multiple times (assuming there are no flaky tests), we will always get the same result: if current commit is broken, we will see `Tests failed` message no matter how many times we run it. This is because the configuration is _static_.

Go code, however, has the chance to _not_ be deterministic. We need to make it deterministic.

Here how determinism is valuable: imagine that a deploy of CI workers happened after running tests but before deploying to staging. How would the new worker know where to resume the current workflow from? One "solution" is to rely on determinism and replay the workflow from the beginning. The workflow execution will evaluate:
1. Does the database has result of `Install dependencies` for workflow execution #123? Yup, use that result.
1. Does the database has result of `Run tests` for workflow execution #123? Nope, the record is missing. Let's run it and restart evaluation after the job is done.

Upon restart, the workflow execution will evaluate:
1. Does the database has result of `Install dependencies` for workflow execution #123? Yup, use that result.
1. Does the database has result of `Run tests` for workflow execution #123? **Yup, use that result**.
1. Does the database has result of `Deploy to staging` for workflow execution #123? Nope, the record is missing. Let's run it and restart evaluation after the job is done.

See where this is going? We will continuously evaluate the workflow until we reach the end.

When we converted the YAML above to Go code, we need to enable the same thing. Our workers may get cycled mid-workflow and we need to ensure that the workers can resume the workflows from where it left off. This is why `client.ExecuteActivity` is used as a shim: it can perform "get result or perform action", wrapping this job execution to accommodate workflow execution.

---

# Appendix

1. You may notice that the `select` statement above is problematic in Temporal. You are correct, but it was done to illustrate the idea while minimizing verbosity.
