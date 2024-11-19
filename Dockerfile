FROM golang:1.23-alpine AS build
WORKDIR /
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags='-w -s -extldflags "-static"' -o /go-axfr-backend

FROM scratch
COPY --from=build /go-axfr-backend /
USER 65534:65534
EXPOSE 8080
CMD [ "/go-axfr-backend" ]