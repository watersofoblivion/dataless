import sys
from awsglue.transforms import *
from awsglue.utils import getResolvedOptions
from pyspark.context import SparkContext
from awsglue.context import GlueContext
from awsglue.job import Job

sc = SparkContext()
glueContext = GlueContext(sc)

job = Job(glueContext)
args = getResolvedOptions(sys.argv, ['JOB_NAME', "database_name", "raw_table_name", "table_name"])
job.init(args['JOB_NAME'], args)

database_name = args["database_name"]
raw_table_name = args["raw_table_name"]
table_name = args["table_name"]

mappings = [("impression_id", "string", "impression_id", "string"),
            ("ad_id",         "string", "ad_id",         "string"),
            ("occurred_at",   "string", "occurred_at",   "timestamp"),
            ("partition_0",   "string", "year",          "int"),
            ("partition_1",   "string", "month",         "int")]

events = glueContext.create_dynamic_frame_from_catalog(database_name, raw_table_name, transformation_ctx = "events-etl") \
                    .apply_mapping(mappings) \
                    .resolveChoice(choice = "MATCH_CATALOG", database = database_name, table_name = table_name)

glueContext.write_dynamic_frame.from_catalog(events,
                                             database_name,
                                             table_name,
                                             additional_options = {"partitionKeys": ["year", "month"]})

job.commit()
