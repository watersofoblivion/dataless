import com.amazonaws.services.glue.ChoiceOption
import com.amazonaws.services.glue.GlueContext
import com.amazonaws.services.glue.MappingSpec
import com.amazonaws.services.glue.ResolveSpec
import com.amazonaws.services.glue.errors.CallSite
import com.amazonaws.services.glue.util.GlueArgParser
import com.amazonaws.services.glue.util.Job
import com.amazonaws.services.glue.util.JsonOptions
import org.apache.spark.SparkContext
import scala.collection.JavaConverters._

object GlueApp {
  val argNames = Seq("JOB_NAME", "database_name", "raw_table_name", "table_name")

  val mappings = Seq(
    MappingSpec("session_id",  "string", "session_id",  "string"),
    MappingSpec("context_id",  "string", "context_id",  "string"),
    MappingSpec("parent_id",   "string", "parent_id",   "string"),
    MappingSpec("actor_type",  "string", "actor_type",  "string"),
    MappingSpec("actor_id",    "string", "actor_id",    "string"),
    MappingSpec("object_type", "string", "object_type", "string"),
    MappingSpec("object_id",   "string", "object_id",   "string"),
    MappingSpec("event_type",  "string", "event_type",  "string"),
    MappingSpec("event_id",    "string", "event_id",    "string"),
    MappingSpec("occurred_at", "string", "occurred_at", "timestamp"),
    MappingSpec("partition_0", "string", "year",        "int"),
    MappingSpec("partition_1", "string", "month",       "int")
  )

  def main(sysArgs: Array[String]) {
    val spark: SparkContext = new SparkContext()
    val glueContext: GlueContext = new GlueContext(spark)

    val args = GlueArgParser.getResolvedOptions(sysArgs, argNames.toArray)
    Job.init(args("JOB_NAME"), glueContext, args.asJava)

    val databaseName = args("database_name")
    val rawTableName = args("raw_table_name")
    val tableName = args("table_name")

    val adClicks = glueContext.getCatalogSource(databaseName, rawTableName).
      getDynamicFrame().
      applyMapping(mappings, false).
      resolveChoice(Seq.empty[ResolveSpec], Some(ChoiceOption("MATCH_CATALOG")), Some(databaseName), Some(tableName))

    val partitioning = JsonOptions("""{"partitionKeys":["year","month"]}""")
    glueContext.getCatalogSink(databaseName, tableName, additionalOptions = partitioning).
      writeDynamicFrame(adClicks)

    Job.commit()
  }
}
