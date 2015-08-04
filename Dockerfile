FROM golang

RUN go get github.com/Financial-Times/coco-synthetic-image-publication-monitor
RUN cp $GOPATH/bin/coco-synthetic-image-publication-monitor /synth-publication
RUN cp $GOPATH/src/github.com/Financial-Times/coco-synthetic-image-publication-monitor/attributes.template /attributes.template
RUN cp $GOPATH/src/github.com/Financial-Times/coco-synthetic-image-publication-monitor/systemAttributes.template /systemAttributes.template
RUN cp $GOPATH/src/github.com/Financial-Times/coco-synthetic-image-publication-monitor/usageTickets.template /usageTickets.template


ENV AWS_ADDRESS s3.amazonaws.com
ENV BUCKET_ADDRESS com.ft.imagepublish.int
EXPOSE 8080

CMD cd / && /synth-publication -dynRouting=$DYN_ROUTING -postHost=$POST_ADDRESS -postCredentials="$POST_CREDENTIALS" -s3Host=$BUCKET_ADDRESS.$AWS_ADDRESS -testUuid=$TEST_UUID

