# Status
🚧 **Under Development** 🚧

This repository is a part of an ongoing project and is currently under active development. I'm continuously working on adding features, fixing bugs, and improving documentation. 
Although this is a one-man project, contributions are welcome.
Please feel free to open issues or submit pull requests.

# dev_forum-gateway
Gateway is the entrypoint for all users of the dev_forum system.
It is responsible for fetching the data from backend services required to construct responses.

## Set up
Rename `.env.example` to `.env` and fill in missing values.

### Using Go command
You need working [Go environment](https://go.dev/doc/install).
```shell
go mod tidy
go mod vendor
go build cmd/main.go
```

### On Docker
You need a working [Docker environment](https://docs.docker.com/engine).

You can use the Dockerfile located in `deployment/` to build and run the service on a docker container.

```shell
make build-image version=latest 
``` 

```shell
docker run -p 50051:50051 -p 2223:2223 krixlion/dev_forum-gateway:latest
```

### On Kubernetes (recommended)
You need a working [Kubernetes environment](https://kubernetes.io/docs/setup).

Kubernetes resources are defined in `deployment/k8s` and deployed using [Kustomize](https://kubernetes.io/docs/tasks/manage-kubernetes-objects/kustomization/).

Currently there are `stage` and `dev` overlays available and include any needed resources and configs.

Use `make` to apply manifests for either dev or stage environment.
```shell
make k8s-run overlay=<dev/stage/...>
```
```shell
# To delete
make k8s-stop overlay=<dev/stage/...>
```

## Testing
Run unit and integration tests using Go command.
Make sure to set current working directory to project root.
```shell
# Add `-short` flag to skip integration tests.
go test ./... -race
```

Generate coverage report using `go tool cover`.
```shell
go test -coverprofile  cover.out ./...
go tool cover -html cover.out -o cover.html
```

If the service is deployed on kubernetes you can use `make`.
```shell
make k8s-integration-test overlay=<dev/stage/...>
```
or
```shell
make k8s-unit-test overlay=<dev/stage/...>
```

## Documentation
For detailed documentation refer to the [Wiki](https://github.com/krixlion/dev_forum-gateway/wiki).
