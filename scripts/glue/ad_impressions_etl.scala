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
    MappingSpec("user", "string", "user", "string"),
    MappingSpec("ad", "string", "ad", "string"),
    MappingSpec("at", "string", "at", "timestamp"),
    MappingSpec("partition_0", "string", "year", "int"),
    MappingSpec("partition_1", "string", "month", "int")
  )

  def main(sysArgs: Array[String]) {
    val spark: SparkContext = new SparkContext()
    val glueContext: GlueContext = new GlueContext(spark)

    val args = GlueArgParser.getResolvedOptions(sysArgs, argNames.toArray)
    Job.init(args("JOB_NAME"), glueContext, args.asJava)

    val databaseName = args("database_name")
    val rawTableName = args("raw_table_name")
    val tableName = args("table_name")

    val adImpressions = glueContext.getCatalogSource(databaseName, rawTableName).
      getDynamicFrame().
      applyMapping(mappings, false).
      resolveChoice(Seq.empty[ResolveSpec], Some(ChoiceOption("MATCH_CATALOG")), Some(databaseName), Some(tableName))

    val partitioning = JsonOptions("""{"partitionKeys":["year","month"]}""")
    glueContext.getCatalogSink(databaseName, tableName, additionalOptions = partitioning).
      writeDynamicFrame(adImpressions)

    Job.commit()
  }
}
