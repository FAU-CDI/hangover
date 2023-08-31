FROM docker.io/library/alpine:3.14 as os

# install ca-certificates
RUN apk add --update --no-cache ca-certificates

# create www-data
RUN set -x ; \
  addgroup -g 82 -S www-data ; \
  adduser -u 82 -D -S -G www-data www-data && exit 0 ; exit 1

# build the backend
FROM docker.io/library/golang:1.21-bookworm as builder

# install dependencies
RUN curl -sS https://dl.yarnpkg.com/debian/pubkey.gpg | apt-key add -
RUN echo "deb https://dl.yarnpkg.com/debian/ stable main" | tee /etc/apt/sources.list.d/yarn.list
RUN apt-get update -qq && apt-get install -y make nodejs yarn

ADD . /app/
WORKDIR /app/
RUN make deps generate
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o hangover ./cmd/hangover

# add it into a scratch image
FROM scratch

# add the user
COPY --from=os /etc/passwd /etc/passwd
COPY --from=os /etc/group /etc/group

# grab ssl certs
COPY --from=os /etc/ssl/certs /etc/ssl/certs

# add the app
COPY --from=builder /app/hangover /hangover

# and set the entry command
EXPOSE 3000
USER www-data:www-data
ENTRYPOINT ["/hangover", "-addr", "0.0.0.0:3000"]