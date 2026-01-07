# Events

## Steps to update the app

TODO: Automate the deployment process

``` bash
# build
GOOS=linux GOARCH=amd64 go build -o main ./cmd/app/main.go
# copy templates folder
scp -r ./cmd/app/templates <username>@ssh-<username>.alwaysdata.net:./app/templates/
# copy to alwaysdata
scp ./main <username>@ssh-<username>.alwaysdata.net:./app/
# restart the app
```