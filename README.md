Dataless
===

Dataless is an toy data product in [AWS::Serverless](https://docs.aws.amazon.com/serverless-application-model/latest/developerguide/).

If you want to deploy and use Dataless, start with the Quickstart.  Otherwise,
jump ahead to the Overview.

* [Quickstart](#quickstart)
* [Overview](#overview)
* [Self-Guided Demo](#self-guided-demo)
* [Advanced](#advanced)
* [Configuration](#configuration)

Quickstart
===

This will step you through the initial deploy of the warehouse.  The deploy is
fully automated, but can take up to 30 minutes depending on configuration.
While it is deploying, you can read an [overview](#overview) of Dataless.

**Important:** This template *must* be deployed in `us-east-1` if DNS is
enabled.  Be sure to set the appropriate environment variables in your terminal:

```bash
export AWS_REGION="us-east-1"
export AWS_DEFAULT_REGION="us-east-1"
```

* [Fork and Clone](#fork-and-clone)
* [Configure](#configure)
* [Deploy CI/CD Pipeline](#deploy-ci-cd-pipeline)
* [Push](#push)

## Fork and Clone

Fork and clone the repo.

```bash
git clone https://github.com/<my-username>/dataless
cd dataless
```

## Configure

The app is configured via a `config/<branch-name>.json` file.  By default, no
configuration options are required.  Ensure your `config/master.json` looks
like:

```json
{
  "Parameters": {}
}
```

**Recommended**: [Set Deployment Preference](#safe-lambda-deploys-and-api-gateway-stage), [Enable Route53](#route53)

## Deploy CI/CD Pipeline

Deploy the build pipeline CloudFormation template using the AWS CLI.  Wait for
the template to completely deploy before continuing.

```bash
# A unique prefix for created stacks.
STACK_PREFIX="dataless"

# Create the stack
aws cloudformation create-stack \
  --stack-name ${STACK_PREFIX} \
  --template-body "$(cat build.yaml)" \
  --capabilities CAPABILITY_IAM CAPABILITY_NAMED_IAM CAPABILITY_AUTO_EXPAND
```

**Recommended**: [Enable GitHub](#github)

## Push

Push code to your repo to deploy.

If using GitHub, the initial pipeline run will trigger automatically.  If using
CodeCommit, you must perform the initial push.  The repo URL is in the template
outputs and can be fetched as below.  

```bash
# Push this repository to the created CodeCommit repo.  It may take a minute or
# two for CodePipeline to see the push.
#
# Note: SSH keys for CodeCommit repos on OSX can be janky.
# https://docs.aws.amazon.com/codecommit/latest/userguide/setting-up-ssh-unixes.html
REPO_URL=$(aws cloudformation describe-stacks --stack-name ${STACK_PREFIX} --query "Stacks[0].Outputs[0].OutputValue" --output text)
git remote add origin ${REPO_URL}
git push origin master
```

Overview
===

Dataless is a toy data product in AWS::Serverless.  It captures real-time user
tracking data and uses it to serve both the end customer that generated the data
and the business offering the service.

It is designed to be "data in, information out": raw data is written to the
warehouse and derived information is served from it.  It is implemented with a
"buy, don't build" approach, using strictly AWS services configured with
CloudFormation.

The example product used is a simplistic online advertising service: ad
impressions and clicks are captured and clickthrough information is served.
Dataless is intended as a reference implementation focusing on infrastructure
and as a template for further development, so the product has been kept
intentionally simple.

Architecturally, there are three main components to the warehouse: the capture
component, the batch component, and the real-time component.  

* [Capture](#capture)
* [Batch](#batch)
* [Real-Time](#real-time)
* [Misc.](#misc)

Capture
---

The capture component's purpose is to capture raw data, persist it to long-term
storage as quickly and reliably as possible, ETL it into the data lake, and make
it available for further processing.

Impression and click data is received by beacon endpoints in an API Gateway.
The endpoints publish the data to Kinesis Firehose, which writes it to S3 in
raw format.

A daily ETL has been created with Glue.  A crawler scans the raw data and makes
it available as Glue tables.  ETL jobs -one per datatype- rewrite the raw data
into optimized Glue tables.

Once the data is ETL-ed into Glue, it is universally available.  All data in
Glue is immediately usable in Athena, QuickSight, Redshift Spectrum, and EMR.

Batch
---

The batch component both provides internal business customers with periodic
reporting, and provides end customers with analytics about their ads.

It is built on top of the data lake using AWS Data Pipeline.  The pipeline uses
EMR (and optionally Redshift) to process data.

### EMR

The EMR portion of the pipeline primarily makes use of Hive.  First, it uses
Hive to materialize a view in Glue for business customers.

Next, it uses Hive to derive ad traffic data for end customers.  The data is
loaded into a DynamoDB table and is served to customers via API Gateway.

### Redshift

If Redshift is enabled (see [Advanced](#advanced) below,) the pipeline also
loads the data into Redshift.  It uses Redshift Spectrum to load all of the Glue
data onto the cluster, and loads the results of earlier Hive queries directly
onto the cluster from EMR.

Real-Time
---

The real-time component processes user data in real time.  It is built on the
capture infrastructure and uses Kinesis Analytics to build a CloudWatch
dashboard showing global impressions, clicks, and clickthrough rate.  

Misc.
---

An CloudWatch dashboard provides operational visibility into the running of the
warehouse.

Security has been designed into the warehouse.  All data is encrypted in flight
and at rest and IAM roles have minimal permissions.  (Notable exceptions: EMR
clusters and script instances are unencrypted, and the API is unauthenticated.)

A handful of resources are retained on template deletion, namely the buckets
containing the data lake and the source code, and the source code repository if
CodeCommit was used.

Self-Guided Demo
===

The self-guided demo will take you through the basic operations of the data
warehouse.  You will generate some load, enable real-time analytics, ETL data
into Glue, and run the batch pipeline.

In a production setup, these demo tasks would be scheduled or triggered instead
of manual.  They are not for demonstration and cost-saving purposes.  Most steps
will be about a minute of clicking in the console, followed by a (possibly long)
wait for automation to complete.

* [Generate Load](#generate-load)
* [Real-Time](#real-time-1)
* [ETL](#etl)
* [Batch Pipeline](#batch-pipeline)

Generate Load
---

### Build the Tool

To generate load, first build the ad-hoc `load-generator` tool.  It is written
in [Go](https://golang.org), so you will need the `go` tool to build it.

```bash
go build -o ./loadgen ./load-generator
```

### Configure the Tool

In `load-generator/config.yaml`, set your base URL.  If you are using DNS (see
[Configuration](#configuration) below,) the base URL will be
`https://<base-dns-name>/advertising` (for example,
`https://dataless.example.com/advertising`.)

If you are not using DNS, the config is set up to point to the API Gateway
instance of the advertising service.  There are three environment variables to
set:

* `API_ID` should be set to the `API` output of the nested `AdvertisingService` stack
* `REGION` should be the region the stack is deployed in.
* `STAGE` should be `Prod`.

All other parameters are pre-tuned to generate a steady load within the default
scaling limits.  Do not adjust them.

### Run the Tool

Finally, in a separate terminal start the load generator and pass in the
configuration file:

```bash
# With DNS
./loadgen -c load-generator/config.yaml

# Without DNS
API_ID="<api-output-from-advertising-service-stack>" REGION="us-east-1" STAGE="Prod" ./loadgen -c load-generator/config.yaml
```

This will generate a dot (`.`) for every batch of 500 records sent to the
warehouse.  Any errors will be printed to the console.

### Verify: Check CloudWatch

Once load is being generated, you should be able to see traffic coming across
the beacon endpoints using the included `OpsDashboard` in CloudWatch.  Within
15 minutes, you should see data being written to S3 via Kinesis Firehose.

Real-Time
---

To enable the real-time app, enable the two Kinesis Analytics applications in
the AWS Console.

### Enable the Kinesis Analytics Applications

In the console, navigate to Kinesis and select "Data Analytics" from the left.
For each of the "RealTimeImpressions" and "RealTimeClicks" applications, select
the applications, click "Actions", and select "Run Application".  Each
application will take 60-90 seconds to start.

**Note**: Kinesis Stream Analytics is $0.11/hour/app and does not have a free
tier.  Be sure to stop the applications when done to avoid any additional
charges.

### Verify: Check CloudWatch

Within a minute or two of the applications being started, the graph on the
`RealTimeAdvertisingDashboard` CloudWatch dashboard should be populated with
impressions, clicks, and a clickthrough rate.  If all is well, the clickthrough
rate should be 30% (the `ClickthroughProbability` value set in
`load-generator/config.yaml`.)

ETL
---

**Important**: Wait approximately 15 minutes after starting the load generator
before ETL-ing data to ensure Kinesis Firehose has flushed to S3.  You will see
metrics in the `OpsDashboard` once data has been flushed.

To ETL the data into the data lake, use Glue.  You will crawl the raw data, ETL
it into the lake with PySpark, and finally crawl the ETL-ed tables to make the
data available in Athena.

### Run the Raw Data Crawler

First, in the AWS Console, navigate to Glue and choose "Crawlers" from the left.
Select the `raw_crawler` crawler and click "Run Crawler".

### Run the ETL Jobs

Once the crawler has completed, ETL the crawled raw data into the lake using the
supplied jobs.  From the left, select "Jobs".  For each of the
`ImpressionsPythonETLJob` and `ClicksPythonETLJob` jobs, select the job, click
"Action", and select "Run Job".  On the pop-up, click "Run Job".  (Note: the
jobs will take 15-30 minutes to start while the on-demand Spark cluster spins
up the first time.)  Alternatively, run the `ImpressionsScalaETLJob` and
`ClicksScalaETLJob` jobs.

**Note**: The Python and Scala jobs are equivalent.  If both sets of jobs are
run, the data will be double ETL-ed into the lake.

### Create and Run the Lake Crawler

Finally, crawl the loaded tables to make the data available in Athena.  Since
CloudFormation does not yet support crawling existing tables, a crawler must be
manually created using the wizard:

* In Glue, select "Crawlers" from the left.  
* Click "Add Crawler", give it the name "dataless-lake-crawler", and click "Next".
* Choose "Existing catalog tables" and click "Next".
* Add the `clicks`, `impressions`, and `advertising` tables and click "Next".
* Choose "Create IAM role", give it the suffix "dataless-lake-crawler" and click "Next".
* Choose "Run on demand" and click "Next".
* Leave the default output settings and click "Next".
* Click "Finish"

Once the crawler has been created, run it when prompted (or manually the same
way as the `raw_crawler`.)

### Verify: Query Athena

Once completed, you should be able to query the `raw_impressions`, `raw_clicks`,
`impressions`, and `clicks` tables in Athena.  The data in the `raw_...` tables
should be the same as the data in the non-`raw_...` tables.

Batch Pipeline
---

### Prerequisite: Sync and ETL Existing Dataset

The batch pipeline works on "yesterday"'s data.  To avoid having to wait until
after midnight UTC, sync a pre-existing data set to the bucket created in the
nested `AdvertisingService` template.  (**Important**: The trailing slashes are
required.)

```bash
aws s3 sync s3://watersofoblivion-data/data/raw/ s3://<bucket-name>/data/raw/
```

Once the data has been synchronized, ETL it into the data lake.  In Glue, re-run
the `raw_crawler` crawler, then the `ImpressionsPythonETLJob`,
`ClicksPythonETLJob` jobs, and finally the `dataless-lake-crawler` crawler.

The data in the dataset is from 2019-06-06.

### Activate the Daily Pipeline

Activate the pipeline.  Navigate to Data Pipeline in the AWS console, and select
the "Daily Pipeline".  Click "Actions" and select "Activate".  (Note: if there
is a warning about starting in the past, it is safe to ignore for the demo.)
The pipeline will take approximately an hour to run.

### Verify: Query Athena, cURL an Endpoint, and Query Redshift

Once the pipeline has completed, the `advertising.advertising` table in Athena
should be populated with data, and the `AdTrafficTable` DynamoDB table should
be populated with ad traffic data.  You can now cURL the ad traffic endpoint
with the ID of a random ad (taken manually from DynamoDB):

```bash
# https://<base-dns-name>/advertising/traffic/{ad-id}?start=<YYYY-mm-dd>&end=<YYYY-mm-dd>
curl https://dataless.example.com/advertising/traffic/fc91623b-7c70-42a6-829b-29b0eb3d61de?start=2019-06-06&end=2019-06-06
```

The result should be something like:

```json
{
  "count": 1,
  "next": "",
  "days": [
    {
      "day": "2019-06-06",
      "impressions": 2,
      "clicks": 2
    }
  ]
}
```

If Redshift is enabled (see [Advanced](#advanced) below,) the instance should
have three tables loaded from Glue:

* `public.impressions`
* `public.clicks`
* `public.advertising`

The data in these tables should be identical to the data in the similarly named
tables in the `advertising` Athena database.

It should also have five tables loaded from spectrum:

* `public.raw_impressions_spectrum`
* `public.raw_clicks_spectrum`
* `public.impressions_spectrum`
* `public.clicks_spectrum`
* `public.advertising_spectrum`

These tables should contain the same data as the matching tables in
`advertising` Athena database without the `_spectrum` suffix.

Configuration
===

The template supports multiple configuration options.

* [Safe Lambda Deploys and API Gateway State](#safe-lambda-deploys-and-api-gateway-stage)
* [GitHub](#github)
* [Route53](#route53)

Safe Lambda Deploys and API Gateway Stage
---

The CodeDeploy deployment configuration can be set with the
`DeploymentPreference` parameter.  The default is `Canary10Percent5Minutes`.
For development, it can be set to `AllAtOnce` for faster deploys.

The `Stage` parameter sets the API Gateway stage deployed.  The default is
`Prod`.

```json
{
  "Parameters": {
    "DeploymentPreference": "AllAtOnce"
  }
}
```

GitHub
---

Have the build pipeline pull from a GitHub repository by setting configuration
parameters on the build template.  GitHub source can be toggled on or off at any
time.

The `Owner`, `Repo`, and `Branch` properties configure where the build pipeline
pulls source code from.  `AccessToken` must be set to a valid
[GitHub OAuth token with `repo` permissions](https://docs.aws.amazon.com/codepipeline/latest/userguide/GitHub-authentication.html).
The branch defaults to `master` and can be omitted.

### Example: Enable GitHub at Stack Creation Time

Replace `<my-github-username-or-org>` with your GitHub username or organization
and `<my-github-oauth-token>` with your GitHub OAuth token.

```bash
aws cloudformation create-stack \
  --stack-name ${STACK_PREFIX} \
  --template-body "$(cat build.yaml)" \
  --parameters \
      ParameterKey=Owner,ParameterValue=<my-github-username-or-org> \
      ParameterKey=Repo,ParameterValue=dataless \
      ParameterKey=Branch,ParameterValue=master \
      ParameterKey=AccessToken,ParameterValue=<my-github-oauth-token> \
  --capabilities CAPABILITY_IAM CAPABILITY_NAMED_IAM CAPABILITY_AUTO_EXPAND
```

Route53
---

Mount the warehouse behind a Route53 domain name by adding the following parameters:

* `DNSDomainName`: A domain name hosted in Route53 set up for ACM validation via either [DNS](https://docs.aws.amazon.com/acm/latest/userguide/gs-acm-validate-dns.html) or [email](https://docs.aws.amazon.com/acm/latest/userguide/gs-acm-validate-email.html)
* `BaseDNSName`: The DNS name where the warehouse will be mounted
* `HostedZoneID`: (Conditional) If provided, DNS records will be added in this hosted zone.  If not provided, a hosted zone will be created.  If the domain name already has a hosted zone attached, it must be set to that hosted zone's ID.

```json
{
  "Parameters": {
    "DNSDomainName": "example.com",
    "HostedZoneID": "Z1234567890",
    "BaseDNSName": "dataless.example.com"
  }
}
```

On the first deploy with these options set, Route53 DNS will be set up.  The
contact listed for email validation will receive an email to confirm a
certificate.  The deploy will block until the certificate is approved.

DNS settings can be toggled on or off or altered at any time with a config
change.  Changing the `BaseDNSName` will require a new certificate to be issued.

Advanced
===

The pipeline supports additional Redshift functionality and the ability to use
a long-running EMR cluster and/or script instance.  This functionality is not
enabled by default, as it incurs ongoing costs.

* [Deploy](#deploy)
* [Example: Enable Redshift](#example-enable-redshift)
* [Resource-Specific Parameters](#resource-specific-parameters)
  * [SSH Access](#ssh-access)
  * [Long-Running Script Instance](#long-running-script-instance)
  * [Long-Running EMR Cluster](#long-running-emr-cluster)
  * [Redshift](#redshift-1)

Deploy
---

Enabling advanced functionality requires a two-deploy rollout since it alters
the structure of the data pipeline.  (Note: If the pipeline has not yet been
activated, the resources can be toggled ad libitum.  Once the pipeline has been
activated, the multi-deploy rollout is required.)

To enable or disable advanced functionality:

- Disable the pipeline, enable/disable the resources, and redeploy
- Enable the pipeline and redeploy

The pipeline can be disabled by setting the `EnablePipeline` config parameter to
`""` (blank).

The various resources can be enabled by setting the following config parameters
to `yes`.  Additional parameters are available in the template to tune the
resources' parameters (instance size, count, etc.)  The resources are managed
with three main configuration parameters:

- `EnableEC2Instance`
- `EnableEMRCluster`
- `EnableRedshift`

These resources are provisioned in a VPC, which is automatically enabled when
any of the resources are enabled and disabled when all of them are disabled.
(Note: If the deploy hangs tearing down the VPC, it is safe to simply delete the
VPC from the console to un-stick the teardown.)

Example: Enable Redshift
---

To enable Redshift, first disable the pipeline and enable the Redshift cluster
by setting `EnablePipeline` to `""` (blank) and `EnableRedshift` to `yes` in
`config/master.json`:

```json
{
  "Parameters": {
    "EnablePipeline": "",
    "EnableRedshift": "yes"
  }
}
```

Then deploy:

```bash
git add config/master.json
git commit -m "Disabling pipeline and enabling Redshift"
git push origin master
```

Next, re-enable the pipeline by removing the `EnablePipeline` configuration:

```json
{
  "Parameters": {
    "EnableRedshift": "yes"
  }
}
```

Then deploy:

```bash
git add config/master.json
git commit -m "Re-enabling Pipeline"
git push origin master
```

Resource-Specific Parameters
---

### SSH Access

To be able to SSH to either the long-running EMR cluster or script instance, you
must supply the name of a valid key pair using the `KeyPair` parameter.
Otherwise, the clusters are created with no SSH access.

### Long-Running Script Instance

You must pass a valid access key ID and secret access key to the Task Runner.
These can be passed via the `AccessKeyID` and `SecretAccessKey` parameters.  Be
sure to use a dedicated, permissions limited user to minimize the security risk
of the credentials being compromised.

The instance type can be set with the `EC2InstanceSize` parameter.  The default
is `t2.nano`.

### Long-Running EMR Cluster

The size of the cluster can be configured with the
`EMR(Master|Core)Instance(Type|Count)` options.  The default is 1 m3.xlarge
instance each for master and core.

The EMR release to use can be configured with the `EMRRelease` parameter.  The
default is `emr-5.23.0`.

### Redshift

The size of the Redshift cluster can be configured using the `RedshiftNodeType`
and `RedshiftNumberOfNodes` parameters.

The default database and user credentials can be configured using the
`RedshiftDatabase`, `RedshiftUsername`, and `RedshiftPassword` parameters.
