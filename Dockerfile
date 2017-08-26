FROM alpine

# Add CA Certificates
RUN apk update && apk add ca-certificates && rm -rf /var/cache/apk/*

# Install electro
ENV ELECTRO_VERSION 0.1.0
ADD https://github.com/pnegahdar/electro/releases/download/${ELECTRO_VERSION}/linux_amd64 /usr/local/bin/electro
RUN chmod +x /usr/local/bin/electro
