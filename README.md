# coco-synthetic-image-publication-monitor

Performs test image publications and monitors for problem.
This service hits the current stack's cms-notifier with a random image, which then is picked up by the kafka-bridge and forwarded to the containerised stack's cms-notifier. The image is then stored in a specific S3 bucket. The service after a given waiting time tests whether the content in S3 matches the published image. 
