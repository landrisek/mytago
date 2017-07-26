FROM golang:latest
ADD . /mytago
RUN go get github.com/jinzhu/configor
RUN go get github.com/BurntSushi/toml
RUN go get github.com/go-sql-driver/mysql
RUN go get gopkg.in/yaml.v1
RUN go install mytago
EXPOSE 8080