# A Ponte num contêiner: um binário estático (sem cgo) e os certificados.
FROM golang:1.27.3 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /bridge . && mkdir /data

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /bridge /bridge
# A pasta de dados pertence ao usuário sem privilégios da imagem.
COPY --from=build --chown=65532:65532 /data /data
ENV HOST=0.0.0.0 PORT=8080 DATA_DIR=/data
VOLUME /data
EXPOSE 8080
ENTRYPOINT ["/bridge"]
