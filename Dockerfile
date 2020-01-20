FROM golang:1

ENV PROJECT=coco-synthetic-image-publication-monitor

# todo: remove
# COPY . /${PROJECT}-sources/
COPY kubeconfig-template /root/.kube/config
COPY start.sh /
#---

RUN wget "https://storage.googleapis.com/kubernetes-release/release/v1.13.7/bin/linux/amd64/kubectl" -O /usr/local/bin/kubectl \
  && chmod +x /usr/local/bin/kubectl
RUN chmod 755 /start.sh

ENV ORG_PATH="github.com/Financial-Times"
ENV SRC_FOLDER="${GOPATH}/src/${ORG_PATH}/${PROJECT}"
ENV BUILDINFO_PACKAGE="${ORG_PATH}/${PROJECT}/vendor/${ORG_PATH}/service-status-go/buildinfo."

COPY . ${SRC_FOLDER}
WORKDIR ${SRC_FOLDER}

RUN go get -u github.com/kardianos/govendor \
  && $GOPATH/bin/govendor sync 

# RUN apk --no-cache --virtual .build-dependencies add git \
#   && ORG_PATH="github.com/Financial-Times" \
#   && REPO_PATH="${ORG_PATH}/${PROJECT}" \
#   && mkdir -p $GOPATH/src/${ORG_PATH} \
#   # Linking the project sources in the GOPATH folder
#   && ln -s /${PROJECT}-sources $GOPATH/src/${REPO_PATH} \
#   && cd $GOPATH/src/${REPO_PATH} \
#   && BUILDINFO_PACKAGE="${ORG_PATH}/${PROJECT}/vendor/${ORG_PATH}/service-status-go/buildinfo." \
RUN VERSION="version=$(git describe --tag --always 2> /dev/null)" \
  && DATETIME="dateTime=$(date -u +%Y%m%d%H%M%S)" \
  && REPOSITORY="repository=$(git config --get remote.origin.url)" \
  && REVISION="revision=$(git rev-parse HEAD)" \
  && BUILDER="builder=$(go version)" \
  && LDFLAGS="-s -w -X '"${BUILDINFO_PACKAGE}$VERSION"' -X '"${BUILDINFO_PACKAGE}$DATETIME"' -X '"${BUILDINFO_PACKAGE}$REPOSITORY"' -X '"${BUILDINFO_PACKAGE}$REVISION"' -X '"${BUILDINFO_PACKAGE}$BUILDER"'" \
  && CGO_ENABLED=0 go build -a -o /artifacts/${PROJECT} -ldflags="${LDFLAGS}" \
  && echo "Fetching dependencies..." \	  && echo "Build flags: ${LDFLAGS}"


  # && echo "Build flags: $LDFLAGS" \
  # && echo "Fetching dependencies..." \
  # && go get -u github.com/kardianos/govendor \
  # && $GOPATH/bin/govendor sync \
  # && go build -ldflags="${LDFLAGS}" \
  # && mv ${PROJECT} /${PROJECT} \
  # && mv attributes.template /attributes.template \
  # && mv systemAttributes.template /systemAttributes.template \
  # && mv usageTickets.template /usageTickets.template \
  # && apk del .build-dependencies \
  # && rm -rf $GOPATH /var/cache/apk/*

# Multi-stage build - copy certs and the binary into the image
FROM alpine
WORKDIR /	WORKDIR /
COPY --from=0 /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=0 /artifacts/* /
COPY --from=0 /start.sh /
COPY --from=0 /root/.kube/config /

ENV AWS_ADDRESS s3.amazonaws.com
ENV BUCKET_ADDRESS com.ft.imagepublish.int
EXPOSE 8080

CMD exec /start.sh
