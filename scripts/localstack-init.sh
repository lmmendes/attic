#!/bin/bash
# Create S3 bucket for attachments
awslocal s3 mb s3://attic-attachments
echo "Created S3 bucket: attic-attachments"
