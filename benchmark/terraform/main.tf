provider "aws" {
  version = "~> 1.60"
  region  = "us-east-1"
}

data "aws_iam_policy_document" "lambda_assume_role" {
  statement {
    actions = ["sts:AssumeRole"]

    principals {
      type        = "Service"
      identifiers = ["lambda.amazonaws.com"]
    }
  }
}

resource "aws_iam_role" "iam_role_for_cluster_benchmark" {
  name = "lambda-benchmark"
  assume_role_policy = "${data.aws_iam_policy_document.lambda_assume_role.json}"
}

resource "aws_iam_role_policy_attachment" "basic_execution_for_cluster_benchmark" {
  role = "${aws_iam_role.iam_role_for_cluster_benchmark.id}"
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole"
}

data "archive_file" "benchmark_zip" {
  type        = "zip"
  source_dir  = "${path.root}/package/"
  output_path = "${path.root}/package.zip"
}

resource "aws_lambda_function" "cluster_benchmark" {
  filename = "${data.archive_file.benchmark_zip.output_path}"
  role = "${aws_iam_role.iam_role_for_cluster_benchmark.arn}"
  handler = "lambda-linux-amd64"
  function_name = "benchmark"
  runtime = "go1.x"
  source_code_hash = "${data.archive_file.benchmark_zip.output_base64sha256}"
  memory_size = "256"
  timeout = "600"
}