[default]
run *FLAGS:
    go run main.go {{FLAGS}}

build:
    go build -o cuebernetes main.go

test:
    go test -v