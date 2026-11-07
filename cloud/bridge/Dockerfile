# A Ponte num contêiner: um binário estático (sem cgo) e os certificados.
FROM golang:1.27.3 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
# A versão vai nos erros enviados ao Blup Pulse (docker build --build-arg VERSION=...).
ARG VERSION=1.0.0
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w -X main.version=${VERSION}" -o /bridge . && mkdir -m 0700 /data

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /bridge /bridge
# A pasta de dados pertence ao usuário sem privilégios da imagem.
COPY --from=build --chown=65532:65532 /data /data
ENV HOST=0.0.0.0 PORT=8080 DATA_DIR=/data
VOLUME /data
EXPOSE 8080
# Sem shell na imagem: o próprio binário pergunta a /_blup/health.
HEALTHCHECK --interval=30s --timeout=5s --start-period=120s --retries=3 CMD ["/bridge", "health"]
ENTRYPOINT ["/bridge"]
