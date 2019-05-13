import sys
from awsglue.transforms import *
from awsglue.utils import getResolvedOptions
from pyspark.context import SparkContext
from awsglue.context import GlueContext
from awsglue.job import Job

args = getResolvedOptions(sys.argv, ['JOB_NAME', "database_name", "raw_table_name", "table_name"])

sc = SparkContext()
glueContext = GlueContext(sc)
spark = glueContext.spark_session
job = Job(glueContext)
job.init(args['JOB_NAME'], args)

database_name = args["database_name"]
raw_table_name = args["raw_table_name"]
table_name = args["table_name"]

mappings = [("at",          "string", "at",    "timestamp"),
            ("user",        "string", "user",  "string"),
            ("ad",          "string", "ad",    "string"),
            ("partition_0", "string", "year",  "int"),
            ("partition_1", "string", "month", "int")]

ad_impressions = glueContext.create_dynamic_frame_from_catalog(database_name, raw_table_name) \
                 .apply_mapping(mappings) \
                 .resolveChoice(choice = "MATCH_CATALOG", database = database_name, table_name = table_name)

glueContext.write_dynamic_frame_from_catalog(ad_impressions, database_name, table_name, additional_options = {"partitionKeys": ["year", "month"]})

job.commit()
