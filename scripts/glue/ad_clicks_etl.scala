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
      ("user", "string", "user", "string"),
      ("ad", "string", "ad", "string"),
      ("at", "timestamp", "at", "timestamp"),
      ("partition_0", "int", "year", "int"),
      ("partition_1", "int", "month", "int")
    )

    val projection = Seq("user", "ad", "at", "year", "month")

    val adClicks = glueContext.getCatalogSource(database = databaseName, tableName = rawTableName).
      getDynamicFrame().
      applyMapping(mappings = mappings).
      selectFields(paths = projection).
      resolveChoice(choiceOption = Some(ChoiceOption("MATCH_CATALOG")), database = Some(databaseName), tableName = Some(tableName)).
      resolveChoice(choiceOption = Some(ChoiceOption("make_struct")))

    glueContext.getCatalogSink(database = databaseName, tableName = tableName).
      writeDynamicFrame(adClicks)

    Job.commit()
  }
}
