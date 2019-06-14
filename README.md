Dataless
===

An template hybrid batch/real-time AWS::Serverless data warehouse.

* [Quickstart](#quickstart)
* [Architecture](#architecture)
* [Self-Guided Demo](#self-guided-demo)
* [Configuration](#configuration)
* [Advanced](#advanced)

Quickstart
===

This will step you through the initial deploy of the warehouse.  While it is
deploying, read about Dataless's [Architecture](#architecture).

**Important:** This template *must* be deployed in `us-east-1` so that ACM
can issue valid certificates for CloudFront.

```bash
export AWS_REGION="us-east-1"
export AWS_DEFAULT_REGION="us-east-1"
```

## 1. Clone

Fork and clone the repo.

```bash
git clone https://github.com/<my-username>/dataless
cd dataless
```

## 2. Deploy Build Pipeline

Deploy the build pipeline CloudFormation template.  Wait for the template to
completely deploy before continuing.

```bash
STACK_NAME="dataless"

aws cloudformation create-stack \
  --stack-name ${STACK_NAME} \
  --template-body "$(cat build.yaml)" \
  --capabilities CAPABILITY_IAM CAPABILITY_NAMED_IAM CAPABILITY_AUTO_EXPAND
```

**Recommended**: [Enable GitHub Integration](#github)

## 3. Config Warehouse

The app is configured via a `config/<branch-name>.json` file.  By default, no
configuration options are required.  Ensure your `config/master.json` looks
like:

```json
{
  "Parameters": {}
}
```

**Recommended**: [Enable Route53](#route53)

## 4. Push

Now push to deploy!

If using CodeCommit, perform the initial push.  The repo URL is in the template
outputs and can be fetched as below.  If using GitHub, the pipeline will run
automatically.

```bash
# Push this repository to the created CodeCommit repo.  It may take a minute or
# two for CodePipeline to see the push.
#
# Note: SSH keys for CodeCommit repos on OSX can be janky.
# https://docs.aws.amazon.com/codecommit/latest/userguide/setting-up-ssh-unixes.html
REPO_URL=$(aws cloudformation describe-stacks --stack-name ${STACK_NAME} --query "Stacks[0].Outputs[0].OutputValue" --output text)
git remote add origin ${REPO_URL}
git push origin master
```

Architecture
===

The warehouse is designed to capture real-time tracking data and use it to serve
three customers: the business as a whole, engineering specifically, and the end
customer that generated the data.  It has additional hook points for scheduled
data ingestion.

It is architected with a "data in, information out" philosophy, with
applications pushing write-only raw data into the warehouse and fetching
read-only derived information from it.  It is implemented with a strict "buy,
don't build" approach, making a conscious decision to use purely AWS services
continuously delivered with CloudFormation.

There are three main components to the warehouse: the main pipe, the batch
component, and the real-time component.  The example dataset is a toy
online-advertising dataset of ad impressions and clicks.  Data can be generated
by an ad-hoc tool.

Main Pipe
---

The main pipe's purpose is to capture raw data, persist it to long-term storage
as quickly and reliably as possible, and ETL it into the data lake for further
processing.

Impression and click data is received via a pair of beacon endpoints in an API
Gateway API.  Via a pair of Lambda functions, this is published to a Kinesis
Firehose and persisted to S3 raw as GZipped JSON.

A daily ETL has been created with Glue.  A crawler and a pair of ETL jobs have
been configured.  The crawler scans the raw data and makes it available via the
Glue Catalog.  The ETL jobs -one per datatype- pick up the raw data and write it
back down into Hive-partitioned ORC tables in S3, and make those tables
available via the Glue Catalog.

Once the data is in the Glue Catalog, it is automatically available in Athena
and QuickSight to serve business customers, and throughout the AWS data
toolchain.

Batch
---

The batch component sits on top of the data lake and primarily serves the
business and end customers.  A batch pipeline has been built with Data Pipeline
to do this.

### Hive

The Hive portion of the pipeline does two things.  First, it materializes an
advertising view managed by Glue.  This table is available to business
customers.  Second, it derives per-ad traffic data by day and populates a
DynamoDB table. An endpoint has been set up for services to query this data to
serve end customers.

### Redshift

**Note**: To get this functionality, you must enable Redshift.  See
[Advanced](#advanced) below.

The pipeline also moves advertising data into Redshift.  It first exports the
advertising table to CSV and loads it into the cluster.  Then, it loads the same
data directly from the data lake using Redshift Spectrum.

Real-Time
---

The real-time component sits on top of the real-time capture infrastructure and
serves primarily engineering.

A pair of Kinesis Analytics apps read the real-time capture firehoses.  Each app
counts events by minute and outputs them to a single Lambda function.  The
Lambda then publishes the metrics to CloudWatch.  A CloudWatch dashboard has
been built showing impressions, clicks, and clickthrough rate.  This would allow
rapid iteration on features based on user behavior.

Misc.
---

An CloudWatch dashboard provides operational visibility into the running of the
warehouse.

A handful of resources are retained on template deletion, namely the buckets
containing the data lake and the source code, and the source code repository if
CodeCommit was used.

Self-Guided Demo
===

The self-guided demo will take you through the basic operations of the data
warehouse.  We will generate some load, enable the real-time analytics, ETL the
data into the Glue catalog, and finally run the batch pipeline.

The demo should take about 2 hours to complete, mostly waiting on automation.

Generate Load
---

### Build the Tool

To generate load, first build the ad-hoc `load-generator` tool.

```bash
go build -o ./loadgen ./load-generator
```

### Configure the Tool

In `load-generator/config.yaml`, set your base URL.  If you are using DNS (see
[Advanced](#advanced) below,) the base URL will be
`https://<base-dns-name/advertising` (for example,
`https://dataless.example.com/advertising`.)

If you are not using DNS, the config is set up to point to the API Gateway
instance of the advertising service when the `API_ID`, `REGION`, and `STAGE`
environment variables are set.  `API_ID` should be set to the `API` output of
the nested `AdvertisingService` stack, `REGION` should be `us-east-1`, and
`STAGE` should be `Prod`.

All other parameters are pre-tuned to generate a steady load within the default
scaling limits.

### Run the Tool

Finally, in a separate terminal start the load generator:

```bash
# With DNS
./loadgen -c load-generator/config.yaml

# Without DNS
API_ID="<api-output-from-advertising-service-stack>" REGION="us-east-1" STAGE="Prod" ./loadgen -c load-generator/config.yaml
```

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

To ETL the data into the data lake, we will use Glue.  We will crawl the raw
data, ETL it into the lake with PySpark, and finally crawl the ETL-ed tables to
make the data available in Athena.  In a production scenario, this process would
be scheduled on a nightly cron.  For the demo, we will run the jobs by hand.

### Run the Raw Data Crawler

First, in the AWS Console, navigate to Glue and choose "Crawlers" from the left.
Select the `raw_crawler` crawler and click "Run Crawler".

### Run the ETL Jobs

Once the crawler has completed, ETL the discovered raw data into the lake using
the supplied jobs.  From the left, select "Jobs".  For each of the
`ImpressionsPythonETLJob` and `ClicksPythonETLJob` jobs, select the job, click
"Action", and select "Run Job".  On the pop-up, click "Run Job".  (Note: the
jobs will take 15-30 minutes to start while the on-demand Spark cluster spins
up the first time.)

Alternatively, run the `ImpressionsScalaETLJob` and `ClicksScalaETLJob` jobs.
The Python and Scala jobs are equivalent.  If both sets of jobs are run, the
data will be double ETL-ed into the lake.

### Create and Run the Lake Crawler

Finally, we need to crawl the loaded tables to make the data available in
Athena.  Since CloudFormation does not yet support crawling existing tables, a
crawler must be manually created using the "Create Crawler" wizard:

* In Glue, select "Crawlers" from the left.  
* Click "Add Crawler", give it the name "dataless-lake-crawler", and click "Next".
* Choose "Existing catalog tables" and click "Next".
* Add the `clicks`, `impressions`, and `advertising` tables and click "Next".
* Choose "Create IAM role", give it the suffix "dataless-lake-crawler" and click "Next".
* Choose "Run on demand" and click "Next".
* Leave the default output settings and click "Next".
* Click "Finish"

Once the crawler has been created, run it when promped (or manually the same way
as the `raw-crawler`.)

### Verify: Query Athena

Once completed, you should be able to query the `raw_impressions`, `raw_clicks`,
`impressions`, and `clicks` tables in Athena.  The data in the `raw_...` tables
should be the same as the data in the non-`raw_...` tables.

Batch Pipeline
---

### Sync Existing Dataset

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

### Activate the Daily Pipeline

Activate the pipeline.  Navigate to Data Pipeline in the AWS console, and select
the "Daily Pipeline".  Click "Actions" and select "Activate".  (Note: if there
is a warning about starting in the past, it is safe to ignore for the demo.)
The pipeline will take approximately an hour to run.

### Verify: Query Athena, cURL an Endpoint, and Query Redshift

Once the pipeline has completed, the `advertising.advertising` table in Athena
should be populated with data, and the `AdTrafficTable` DynamoDB table should
be populated with ad traffic data.  You can now cURL the ad traffic endpoint
with the ID of a random ad (taken from DynamoDB):

```bash
# https://<base-dns-name>/advertising/traffic/{ad-id}?start=<YYYY-mm-dd>&end=<YYYY-mm-dd>
curl https://dataless.example.com/advertising/traffic/fc91623b-7c70-42a6-829b-29b0eb3d61de?start=2019-06-06&end=2019-06-06
```

If Redshift is enabled (see [Advanced](#advanced) below,) the instance should
have two tables, `public.advertising` and `public.advertising_spectrum`, each
containing the same data as the `advertising.advertising` Athena table.

Configuration
===

The template supports multiple configuration options.

Safe Lambda Deploys and API Gateway
---

Configure the CodeDeploy deployment configuration with the
`DeploymentPreference` parameter.  The default is `Canary10Percent5Minutes`.
For development, it can be set to `AllAtOnce` for faster deploys.

The `Stage` parameter sets the API Gateway stage deployed.  The default is
`Prod`.

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
  --stack-name ${STACK_NAME} \
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
* `HostedZoneID`: (Conditional) If provided, DNS records will be added in this hosted zone.  If not provided, a hosted zone will be created.  If the domain name already has a hosted zone attached, this must be set to that hosted zone ID.

DNS settings can be toggled on or off or altered at any time with a config
change.  Changing the `BaseDNSName` will require a new certificate to be issued.

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

Advanced
===

The pipeline supports additional Redshift functionality and the ability to use
a long-running EMR cluster and/or script instance.  This functionality is not
enabled by default, as it incurs ongoing costs.

Enabling it requires a two-deploy rollout since it alters the structure of the
pipeline.  (Note: If the pipeline has not yet been activated, the resources can
be toggled ad libitum.  Once the pipeline has been activated, the multi-deploy
rollout is required.)

To enable or disable advanced functionality:

- Disable the pipeline, enable/disable the resources, and redeploy
- Enable the pipeline and redeploy

The pipeline can be disabled by setting the `EnablePipeline` config parameter to
`""` (blank).

The various resources can be enabled by setting the following config parameters
to `yes`.  Additional parameters are available in the template to tune the
resources' parameters (instance size, count, etc.).

- `EnableEC2Instance`
- `EnableEMRCluster`
- `EnableRedshift`

These resources are provisioned in a VPC, which is automatically enabled when
any of the resources are enabled and disabled when all of them are disabled.
(Note: If the deploy hangs tearing down the VPC, it is safe to simply delete the
VPC from the console to un-stick the teardown.)

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
instance for each master and core.

The EMR release to use can be configured with the `EMRRelease` parameter.  The
default is `emr-5.23.0`.

### Redshift

The size of the Redshift cluster can be configured using the `RedshiftNodeType`
and `RedshiftNumberOfNodes` parameters.

The default database and user credentials can be configured using the
`RedshiftDatabase`, `RedshiftUsername`, and `RedshiftPassword` parameters.

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
