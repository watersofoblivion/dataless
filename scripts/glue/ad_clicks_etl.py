import sys
from awsglue.transforms import *
from awsglue.utils import getResolvedOptions
from pyspark.context import SparkContext
from awsglue.context import GlueContext
from awsglue.job import Job

args = getResolvedOptions(sys.argv,
                          ['JOB_NAME',
                           "database",
                           "raw_table_name",
                           "table_name"])

sc = SparkContext()
glueContext = GlueContext(sc)
spark = glueContext.spark_session
job = Job(glueContext)
job.init(args['JOB_NAME'], args)

database = args["database"]
raw_table_name = args["raw_table_name"]
table_name = args["table_name"]

ad_clicks = (
    glueContext
    .create_dynamic_frame.from_catalog(database = database,
                                       table_name = raw_table_name)
    .apply_mapping(mappings = [("at", "string", "at", "string"),
                               ("user", "string", "user", "string"),
                               ("ad", "string", "ad", "string"),
                               ("year", "string", "year", "string"),
                               ("month", "string", "month", "string"),
                               ("day", "string", "at", "timestamp"),
                               ("hour", "string", "hour", "string")])
    .select_fields(paths = ["ad", "user", "at", "year", "month"])
    .resolveChoice(choice = "MATCH_CATALOG",
                   database = database,
                   table_name = table_name)
    .resolveChoice(choice = "make_struct")
)

glueContext.write_dynamic_frame.from_catalog(frame = ad_clicks,
                                             database = database,
                                             table_name = table_name)

job.commit()
