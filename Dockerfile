FROM golang:1.19-alpine as build
WORKDIR /
COPY *.go ./
COPY *.mod ./
COPY *.sum ./
RUN go build -o /go-axfr-backend

FROM alpine:latest
COPY --from=build /go-axfr-backend /
EXPOSE 8080
CMD [ "/go-axfr-backend" ]