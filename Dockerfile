FROM golang:1

ENV PROJECT=coco-synthetic-image-publication-monitor
ENV BUILDINFO_PACKAGE="github.com/Financial-Times/service-status-go/buildinfo."

COPY kubeconfig-template /root/.kube/config
COPY start.sh /

COPY . /${PROJECT}/
WORKDIR /${PROJECT}

RUN wget "https://storage.googleapis.com/kubernetes-release/release/v1.13.7/bin/linux/amd64/kubectl" -O /usr/local/bin/kubectl \
  && chmod +x /usr/local/bin/kubectl
RUN chmod 755 /start.sh

RUN VERSION="version=$(git describe --tag --always 2> /dev/null)" \
  && DATETIME="dateTime=$(date -u +%Y%m%d%H%M%S)" \
  && REPOSITORY="repository=$(git config --get remote.origin.url)" \
  && REVISION="revision=$(git rev-parse HEAD)" \
  && BUILDER="builder=$(go version)" \
  && LDFLAGS="-s -w -X '"${BUILDINFO_PACKAGE}$VERSION"' -X '"${BUILDINFO_PACKAGE}$DATETIME"' -X '"${BUILDINFO_PACKAGE}$REPOSITORY"' -X '"${BUILDINFO_PACKAGE}$REVISION"' -X '"${BUILDINFO_PACKAGE}$BUILDER"'" \
  && CGO_ENABLED=0 go build -mod=readonly -a -o /artifacts/${PROJECT} -ldflags="${LDFLAGS}" \
  && echo "Build flags: ${LDFLAGS}" \
  && mv attributes.template /attributes.template \
  && mv systemAttributes.template /systemAttributes.template \
  && mv usageTickets.template /usageTickets.template

FROM alpine
WORKDIR /
COPY --from=0 /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=0 /artifacts/* /
COPY --from=0 /root/.kube/config /root/.kube/config
COPY --from=0 /usr/local/bin/kubectl /usr/local/bin/kubectl
COPY --from=0 /start.sh /
COPY --from=0 /attributes.template /
COPY --from=0 /systemAttributes.template /
COPY --from=0 /usageTickets.template /

ENV AWS_ADDRESS s3.amazonaws.com
ENV BUCKET_ADDRESS com.ft.imagepublish.int
EXPOSE 8080

CMD exec /start.sh
