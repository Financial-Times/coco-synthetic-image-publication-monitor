FROM golang

RUN go get github.com/Financial-Times/coco-synthetic-image-publication-monitor

ENV AWS_ADDRESS s3.amazonaws.com
ENV BUCKET_ADDRESS com.ft.imagepublish.int
EXPOSE 8080

CMD $GOPATH/bin/coco-synthetic-image-publication-monitor \
	--dynRouting true \
	--postHost $PROXY \
	--s3Host $BUCKET_ADDRESS.$AWS_ADDRESS

