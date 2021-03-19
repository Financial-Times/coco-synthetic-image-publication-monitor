<!--
    Written in the format prescribed by https://github.com/Financial-Times/runbook.md.
    Any future edits should abide by this format.
-->
# UPP - Synthetic Image Publication Monitor

This service performs test image publications and monitors for problems.

## Code

synth-image-pub-monitor

## Primary URL

https://upp-prod-delivery-glb.upp.ft.com/__coco-synthetic-image-publication-monitor/

## Service Tier

Platinum

## Lifecycle Stage

Production

## Host Platform

AWS

## Architecture

This service periodically hits the current stack's cms-notifier with a random image, which then is picked up by the kafka-bridge 
and forwarded to the containerised stack's cms-notifier. The image is then stored in a specific S3 bucket. The service after 
a given waiting time tests whether the content in S3 matches the published image. In case of "Image not found" is reported 
the service will trigger a job <https://github.com/Financial-Times/coco-image-trace> from cronjob that will trace the missing 
transaction ID thought services in kubernetes and will alert in Slack channel.

## Contains Personal Data

No

## Contains Sensitive Data

No

<!-- Placeholder - remove HTML comment markers to activate
## Can Download Personal Data
Choose Yes or No

...or delete this placeholder if not applicable to this system
-->

<!-- Placeholder - remove HTML comment markers to activate
## Can Contact Individuals
Choose Yes or No

...or delete this placeholder if not applicable to this system
-->

## Failover Architecture Type

ActiveActive

## Failover Process Type

FullyAutomated

## Failback Process Type

FullyAutomated

## Failover Details

The service is deployed in both Delivery clusters.
The failover guide for the cluster is located here:
<https://github.com/Financial-Times/upp-docs/tree/master/failover-guides/delivery-cluster>

## Data Recovery Process Type

NotApplicable

## Data Recovery Details

The service does not store data, so it does not require any data recovery steps.

## Release Process Type

PartiallyAutomated

## Rollback Process Type

Manual

## Release Details

Manual failover is needed when a new version of
the service is deployed to production.
Otherwise, an automated failover is going to take place when releasing.
For more details about the failover process please see: <https://github.com/Financial-Times/upp-docs/tree/master/failover-guides/delivery-cluster>

<!-- Placeholder - remove HTML comment markers to activate
## Heroku Pipeline Name
Enter descriptive text satisfying the following:
This is the name of the Heroku pipeline for this system. If you don't have a pipeline, this is the name of the app in Heroku. A pipeline is a group of Heroku apps that share the same codebase where each app in a pipeline represents the different stages in a continuous delivery workflow, i.e. staging, production.

...or delete this placeholder if not applicable to this system
-->

## Key Management Process Type

Manual

## Key Management Details

To access the service clients need to provide basic auth credentials.
To rotate credentials you need to login to a particular cluster and update varnish-auth secrets.

## Monitoring

Service in UPP K8S delivery clusters:

*   Delivery-Prod-EU health: <https://upp-prod-delivery-eu.ft.com/__health/__pods-health?service-name=synthetic-image-publication-monitor>
*   Delivery-Prod-US health: <https://upp-prod-delivery-us.ft.com/__health/__pods-health?service-name=synthetic-image-publication-monitor>

## First Line Troubleshooting

<https://github.com/Financial-Times/upp-docs/tree/master/guides/ops/first-line-troubleshooting>

## Second Line Troubleshooting

Please refer to the GitHub repository README for troubleshooting information.