Dataless
===

An example hybrid batch/real-time AWS::Serverless data warehouse.

Quickstart
===

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

Deploy the build pipeline CloudFormation template.  Wait for the template to completely deploy before continuing.

```bash
STACK_NAME="dataless"

aws cloudformation create-stack \
  --stack-name ${STACK_NAME} \
  --template-body "$(cat build.yaml)" \
  --capabilities CAPABILITY_IAM CAPABILITY_NAMED_IAM CAPABILITY_AUTO_EXPAND
```

**Recommended**: To pull from a GitHub repository, set some additional parameters.

```bash
aws cloudformation create-stack \
  --stack-name ${STACK_NAME} \
  --template-body "$(cat build.yaml)" \
  --parameters \
      ParameterKey=Owner,ParameterValue=<my-username> \
      ParameterKey=Repo,ParameterValue=dataless \
      ParameterKey=Branch,ParameterValue=master \
      ParameterKey=AccessToken,ParameterValue=<github-oauth-token> \
  --capabilities CAPABILITY_IAM CAPABILITY_NAMED_IAM CAPABILITY_AUTO_EXPAND
```

The `Owner`, `Repo`, and `Branch` properties configure where the build pipeline pulls source code from.  `AccessToken` must be set to a valid [GitHub OAuth token with `repo` permissions](https://docs.aws.amazon.com/codepipeline/latest/userguide/GitHub-authentication.html).  The branch defaults to `master` and can be omitted.


## 3. Config Warehouse

The app is configured via the `config/master.json` file.  By default, no
configuration options are required.

```json
{
  "Parameters": {}
}
```

### Recommended: DNS

Optionally, set up DNS by adding the following parameters:

* `DNSDomainName`: A domain name hosted in Route53
* `ValidationDomain`: A domain set up for ACM validation via either [DNS](https://docs.aws.amazon.com/acm/latest/userguide/gs-acm-validate-dns.html) or [email](https://docs.aws.amazon.com/acm/latest/userguide/gs-acm-validate-email.html)
* `BaseDNSName`: The DNS name where the warehouse will be mounted
* `HostedZoneID`: (Conditional) If provided, DNS records will be added in this hosted zone.  If not provided, a hosted zone will be created.  If the domain name already has a hosted zone attached, this must be set to that hosted zone ID.

```json
{
  "Parameters": {
    "DNSDomainName": "example.com",
    "ValidationDomain": "example.com",
    "HostedZoneID": "Z1234567890",
    "BaseDNSName": "dataless.example.com"
  }
}
```

On the first deploy with these options set, Route53 DNS will be set up.  The
contact listed for email validation will receive an email to confirm a
certificate.  The deploy will block until the certificate is approved.

#### Without DNS

If these parameters are not included, Route53 will not be set up.  The base URL
for the Advertising service is exposed via the `BaseURL` output of the nested
`AdvertisingService` stack, and paths do not have the leading `/advertising`
prefix.

For example, `https://dataless.example.com/advertising` turns into
`https://a1b2c3d4e5f6.execute-api.us-east-1.amazonaws.com/Prod`.

DNS settings can be toggled on or off or altered at any time with a config
change.  Changing the `BaseDNSName` will require a new certificate to be issued.

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
git remote add aws ${REPO_URL}
git push aws master
```

Architecture
===

The warehouse is designed to capture real-time tracking data and use it to serve
three customers: the business as a whole, engineering specifically, and the end
customer that generated the data.  It has additional hook points for scheduled
data ingestion.

There are three main components to the warehouse: the main pipe, the batch
component, and the real-time component.

The example dataset is a toy online-advertising dataset of ad impressions and
clicks.  Data can be generated by an ad-hoc tool.

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
available via the Glue Catalog.  (Note: duplicate jobs have been provided in
Python and Scala.  Running both will double ETL.)

Once the data is in the Glue Catalog, it is automatically available in Athena
and QuickSight to serve business customers, and throughout the AWS data tools.

Batch
---

The batch component sits on top of the data lake and primarily serves the
business and end customers.

A Data Pipeline batch pipeline has been built to do this.

### Hive

Via Hive, the pipeline does two things.  First, it materializes an advertising
view managed by Glue.  This table is available to business customers.  Second,
it derives per-ad traffic data by day and populates a DynamoDB table. An
endpoint has been set up for services to query this data to serve end customers.

### Redshift

**Note**: To get this functionality, you must enable Redshift.  See "Advanced"
below.

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

Safe Lambda deploys are enabled with a 5 minute canary by default.  Set
`DeploymentPreference` to `AllAtOnce` in the config for faster deploys during
development.

A handful of resources are retained on template deletion, namely the bucket
containing the data lake, and the source code repository if CodeCommit was used.

Advanced
---

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
any of the resources are enabled and disabled when all of them are disabled.  (Note: If the deploy hangs tearing down the VPC, it is safe to simply delete the
VPC from the console to un-stick the teardown.)

### Example: Enable Redshift

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
