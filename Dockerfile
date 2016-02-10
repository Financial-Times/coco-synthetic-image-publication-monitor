FROM gliderlabs/alpine:3.2

ADD . /synthetic-image-publication-monitor
RUN apk --update add go git\
  && export GOPATH=/.gopath \
  && go get github.com/Financial-Times/coco-synthetic-image-publication-monitor \
  && cd synthetic-image-publication-monitor \
  && go build \
  && mv synthetic-image-publication-monitor /synth-publication \
  && mv attributes.template /attributes.template \
  && mv systemAttributes.template /systemAttributes.template \
  && mv usageTickets.template /usageTickets.template \
  && apk del go git \
  && rm -rf $GOPATH /var/cache/apk/*

ENV AWS_ADDRESS s3.amazonaws.com
ENV BUCKET_ADDRESS com.ft.imagepublish.int
EXPOSE 8080

CMD exec /synth-publication -dynRouting=$DYN_ROUTING -postHost=$POST_ADDRESS -postCredentials="$POST_CREDENTIALS" -s3Host=$BUCKET_ADDRESS.$AWS_ADDRESS -testUuid=$TEST_UUID

