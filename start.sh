#!/bin/sh
sed -i "s@JENKINS_TOKEN@$JENKINS_TOKEN@g" /root/.kube/config
sed -i "s/ENVNAME/$ENVNAME/g" /root/.kube/config
/coco-synthetic-image-publication-monitor -postHost=$POST_ADDRESS -postCredentials="$POST_CREDENTIALS" -s3Host=$BUCKET_ADDRESS.$AWS_ADDRESS -testUuid=$TEST_UUID
