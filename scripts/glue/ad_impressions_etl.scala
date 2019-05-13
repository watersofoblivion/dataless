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
  def main(sysArgs: Array[String]) {
    val spark: SparkContext = new SparkContext()
    val glueContext: GlueContext = new GlueContext(spark)
    val args = GlueArgParser.getResolvedOptions(sysArgs, Seq("JOB_NAME", "database-name", "raw-table-name", "table-name").toArray)
    Job.init(args("JOB_NAME"), glueContext, args.asJava)

    val databaseName = args("database-name")
    val rawTableName = args("raw-table-name")
    val tableName = args("table-name")

    val mappings = Seq(
      MappingSpec("user", "string", "user", "string"),
      MappingSpec("ad", "string", "ad", "string"),
      MappingSpec("at", "timestamp", "at", "timestamp"),
      MappingSpec("partition_0", "int", "year", "int"),
      MappingSpec("partition_1", "int", "month", "int")
    )

    val adImpressions = glueContext.getCatalogSource(databaseName, rawTableName).
      getDynamicFrame().
      applyMapping(mappings, false).
      resolveChoice(Seq.empty[ResolveSpec], Some(ChoiceOption("MATCH_CATALOG")), Some(databaseName), Some(tableName))

    glueContext.getCatalogSink(databaseName, tableName).writeDynamicFrame(adImpressions)

    Job.commit()
  }
}
