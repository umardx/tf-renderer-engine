provider "aws" {
  region = "{{ .Region }}"
}

resource "aws_s3_bucket" "this" {
  bucket = "{{ .BucketName }}"
}

resource "aws_s3_bucket_acl" "this" {
  bucket = aws_s3_bucket.this.id
  acl    = "{{ .ACL }}"
}
