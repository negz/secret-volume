FROM golang:1.7

ENV SECRET_VOLUME_PARENT /secrets

ENV APP /go/src/github.com/negz/secret-volume

ENV INIT_VERSION 1.1.3
ENV INIT_URL https://github.com/Yelp/dumb-init/releases/download/v${INIT_VERSION}/dumb-init_${INIT_VERSION}_amd64
RUN curl -fsSL "${INIT_URL}" -o /dumb-init && chmod +x /dumb-init

COPY . "${APP}"
WORKDIR "${APP}"

RUN ./getglide.sh
ENV GLIDE_HOME "${APP}" 
RUN glide install

RUN go install .

RUN mkdir -p "${SECRET_VOLUME_PARENT}"

ENTRYPOINT ["/dumb-init", "/go/bin/secret-volume"]
