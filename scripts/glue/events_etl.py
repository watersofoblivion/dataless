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

mappings = [("session_id",  "string", "session_id",  "string"),
            ("context_id",  "string", "context_id",  "string"),
            ("parent_id",   "string", "parent_id",   "string"),
            ("actor_type",  "string", "actor_type",  "string"),
            ("actor_id",    "string", "actor_id",    "string"),
            ("event_type",  "string", "event_type",  "string"),
            ("event_id",    "string", "event_id",    "string"),
            ("object_type", "string", "object_type", "string"),
            ("object_id",   "string", "object_id",   "string"),
            ("occurred_at", "string", "occurred_at", "timestamp"),
            ("partition_0", "string", "year",        "int"),
            ("partition_1", "string", "month",       "int")]

ad_clicks = glueContext.create_dynamic_frame_from_catalog(database_name, raw_table_name) \
            .apply_mapping(mappings) \
            .resolveChoice(choice = "MATCH_CATALOG", database = database_name, table_name = table_name)

glueContext.write_dynamic_frame.from_catalog(ad_clicks,
                                             database_name,
                                             table_name,
                                             additional_options = {"partitionKeys": ["year", "month"]})

job.commit()
