# Summary

<!--
Insert your work summary here :) we'd like to see:
- What did you learn in this exercise?
- Your experience on breaking workflow determinism and thoughts around it, i.e. if you were to explain what you learned to someone else, what would you say?
- Any trade-offs you made in completing this project?

Remember that we like to see thoughtful work, so elaborating your thought process helps us understand your workstyle even better.
-->

## Checklist

- [ ] Running `go run . worker` and `go run . pipeline` still works, just like it does in the boilerplate project.
- [ ] Given a YAML file as input for `PipelineParams` in [_examples/simple.yaml](./_examples/simple.yaml), the program should be able to start a pipeline.
- [ ] The pipeline should exercise fan-out and fan-in concurrency patterns (e.g. tests don't necessarily depend on linters to finish and vice versa, but deploy should wait on both).
- [ ] Write tests on what you think is worth testing. Remember to consider the failure modes within the workflow (e.g. if a step fails, what should happen to the rest?).
