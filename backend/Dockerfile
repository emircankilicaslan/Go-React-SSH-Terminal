FROM golang:1.25 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/ssh-terminal . && mkdir /out/data

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/ssh-terminal /ssh-terminal
# nonroot (65532) kullanıcısının yazabileceği veri dizini
COPY --from=build --chown=65532:65532 /out/data /data
ENV DB_PATH=/data/fullstackx.db
EXPOSE 8080
VOLUME /data
ENTRYPOINT ["/ssh-terminal"]