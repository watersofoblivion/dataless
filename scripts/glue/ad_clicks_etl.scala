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
    val args = GlueArgParser.getResolvedOptions(sysArgs, Seq("JOB_NAME", "database", "raw_table_name", "table_name").toArray)
    Job.init(args("JOB_NAME"), glueContext, args.asJava)

    val database = args("database")
    val rawTableName = args("raw_table_name")
    val tableName = args("table_name")

    val mappings = Seq(
      ("user", "string", "user", "string"),
      ("ad", "string", "ad", "string"),
      ("at", "timestamp", "at", "timestamp"),
      ("year", "int", "year", "int"),
      ("month", "int", "month", "int"),
      ("day", "int", "day", "int"),
      ("hour", "int", "hour", "int")
    )

    val projection = Seq("user", "ad", "at", "year", "month")

    adClicks = glueContext.getCatalogSource(database = database, tableName = rawTableName).
      getDynamicFrame().
      applyMapping(mappings = mappings).
      selectFields(paths = projection).
      resolveChoice(choiceOption = Some(ChoiceOption("MATCH_CATALOG")), database = Some(database), tableName = Some(tableName)).
      resolveChoice(choiceOption = Some(ChoiceOption("make_struct"))).

    glueContext.getCatalogSink(database = database, tableName = tableName).
      writeDynamicFrame(adClicks)

    Job.commit()
  }
}
