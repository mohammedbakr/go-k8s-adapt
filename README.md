<h1 align="center">go-k8s-adapt</h1>

<p align="center">
    <a href="https://github.com/k8-proxy/go-k8s-adapt/actions/workflows/build.yml">
        <img src="https://github.com/k8-proxy/go-k8s-adapt/actions/workflows/build.yml/badge.svg"/>
    </a>
    <a href="https://codecov.io/gh/k8-proxy/go-k8s-adapt">
        <img src="https://codecov.io/gh/k8-proxy/go-k8s-adapt/branch/main/graph/badge.svg"/>
    </a>	    
    <a href="https://goreportcard.com/report/github.com/k8-proxy/go-k8s-adapt">
      <img src="https://goreportcard.com/badge/k8-proxy/go-k8s-adapt" alt="Go Report Card">
    </a>
	<a href="https://github.com/k8-proxy/go-k8s-adapt/pulls">
        <img src="https://img.shields.io/badge/contributions-welcome-brightgreen.svg?style=flat" alt="Contributions welcome">
    </a>
    <a href="https://opensource.org/licenses/Apache-2.0">
        <img src="https://img.shields.io/badge/License-Apache%202.0-blue.svg" alt="Apache License, Version 2.0">
    </a>
    <a href="https://github.com/k8-proxy/go-k8s-adapt/releases/latest">
        <img src="https://img.shields.io/github/release/k8-proxy/go-k8s-adapt.svg?style=flat"/>
    </a>
</p>


# go-k8s-adapt

- Adaptation service integrating with processing. This service will listen on new messages and triger rebuild

### Steps of processing

- A file a sent to queue by the api
- That file gets downloaded on minio
- A rebuild is trigerred
- Once rebuild is completed the report is uploaded again to minio and published to the reply queue

## Info 
- <Placeholder, fill in at the next PR>


## Build

- Follow the steps bellow to build the code
```
cd cmd
go build .
```


### Docker build
- To perform a docker build, run
```
docker build -t go-k8s-adapt .
```

## Test

For testing, run `go test ./...`


## Video Demo

< Add video demo link here >
