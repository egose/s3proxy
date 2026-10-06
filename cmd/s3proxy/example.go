package main

import (
	"fmt"
	"io"

	"github.com/spf13/cobra"
)

func newPrintExampleConfigCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "print-example-config",
		Short: "Print a static authenticated starter config without evaluating environment variables",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			n, err := io.WriteString(cmd.OutOrStdout(), exampleConfig)
			if err == nil && n != len(exampleConfig) {
				err = io.ErrShortWrite
			}
			if err != nil {
				return fmt.Errorf("write example config: %w", err)
			}
			return nil
		},
	}
}

const exampleConfig = `listener "http" "public" {
  address = "127.0.0.1:8080"
  replay_body_max_bytes = 33554432
  replay_body_aggregate_max_bytes = 268435456

  addressing {
    path_style = true
    virtual_hosted = false
  }

  timeouts {
    read_header = "10s"
    read = "2m"
    write = "5m"
    idle = "60s"
  }
}

auth "main" {
  mode = "sigv4_static"

  client "local" {
    access_key = env("S3PROXY_CLIENT_ACCESS_KEY")
    secret_key = env("S3PROXY_CLIENT_SECRET_KEY")
    allow_routes = ["route.images_rw"]
    allow_ops = ["GetObject", "HeadObject", "PutObject", "DeleteObject", "HeadBucket", "ListObjectsV2", "ListBuckets"]
    visible_buckets = ["images"]
  }
}

credential "static" "primary" {
  access_key = env("S3PROXY_TARGET_PRIMARY_ACCESS_KEY")
  secret_key = env("S3PROXY_TARGET_PRIMARY_SECRET_KEY")
}

target "s3" "primary" {
  endpoint = env("S3PROXY_TARGET_PRIMARY_ENDPOINT")
  region = "us-east-1"
  force_path_style = true
  timeout = "2m"
  credentials = "primary"
}

parser "bucket_exact" "images" {
  bucket = "images"
}

route "images_rw" {
  parser = "images"
  operations = ["GetObject", "HeadObject", "PutObject", "DeleteObject", "HeadBucket", "ListObjectsV2"]
  destinations = ["primary"]
  dispatch = "first"
  on_match = "stop"
  read_preference = "first"

  rewrite {
    bucket = "images-store"
  }
}

bucket "images" {
  visible_name = "images"
  route = "images_rw"
}
`
