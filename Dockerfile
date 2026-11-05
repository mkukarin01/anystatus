# syntax=docker/dockerfile:1
FROM golang:1.27-alpine AS build
WORKDIR /src
COPY go.mod ./
COPY *.go ./
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/anystatus .

FROM gcr.io/distroless/static-debian13:nonroot
COPY --from=build /out/anystatus /anystatus
ENV PORT=8080 LOG_FILE=/dev/stdout
EXPOSE 8080
ENTRYPOINT ["/anystatus"]
