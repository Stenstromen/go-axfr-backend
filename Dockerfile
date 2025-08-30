FROM golang:1.25-alpine AS build
WORKDIR /app
COPY . .
RUN go mod download
RUN GOEXPERIMENT=greenteagc CGO_ENABLED=0 GOOS=linux go build -a -ldflags='-w -s -extldflags "-static"' -o /go-axfr-backend ./cmd/server

FROM scratch
COPY --from=build /go-axfr-backend /
USER 65534:65534
EXPOSE 8080
CMD [ "/go-axfr-backend" ]