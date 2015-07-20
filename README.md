#synthetic-publication

Performs test image publications and monitors for problem.

This service periodically hits the current stack's cms-notifier with a random image, which then is picked up by the kafka-bridge and forwarded to the containerised stack's cms-notifier. The image is then stored in a specific S3 bucket. The service after a given waiting time tests whether the content in S3 matches the published image.

---

As of July 17 2015 this service hits cms-notifier in the dockerized stack, hence it checks the status of image publication only in the new stack's perspective.

####Endpoints
1. /__health : ft-standard healthcheck, OK if publishing works
2. /history : displays the latest ten publications' result in custom format
3. /forcePublish : in case you want to force generating and posting the img (use this without the `tick` flag enabled, otherwise it may produce false results due to publication check timings; useful locally and for testing)

Note: the '/__health' endpoint looks for the most recent publication status.

####Access
In the dockerized world vulcand routes requests to this service, if the request has the host header set to "Host: syn-pub" (note the abbreviation!).

`curl localhost:8080/history -H "Host: syn-pub"`

The status of the /__health endpoint is also listed in the aggregated-healthcheck.

