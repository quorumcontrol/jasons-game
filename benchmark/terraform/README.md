This directory contains very rudimentary support for benchmarking on lambda

First, from the root of the repo, run:
`xgo -targets='linux/amd64' -pkg benchmark/cmd/lambda -dest benchmark/terraform/package ./`

That ensures the binary for lambda is up to date

Then cd into `benchmark/terraform` and run:

```
aws-vault exec YOUR_JASONS_GAME_PROFILE_NAME -- terraform init`
aws-vault exec YOUR_JASONS_GAME_PROFILE_NAME -- terraform apply
```

Once all setup, you can modify `benchmark/terraform/benchmark-with-lambda.sh` to whatever scenario you want to test.

Then to actually run the benchmarks, run:
`aws-vault exec YOUR_JASONS_GAME_PROFILE_NAME -- ./benchmark-with-lambda.sh`

This will print out each lambda tasks response, which if successful, will include the response times.

Once you are finished, run:
`aws-vault exec YOUR_JASONS_GAME_PROFILE_NAME -- terraform destroy`