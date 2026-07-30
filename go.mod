module github.com/your-org/go-selfupdate-template

go 1.23

require github.com/minio/selfupdate v0.6.0

replace github.com/minio/selfupdate => ./third_party/minio-selfupdate
