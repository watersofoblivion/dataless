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
  val argNames = Seq(
    "JOB_NAME",
    "database-name",
    "raw-table-name",
    "table-name"
  )

  val mappings = Seq(
    MappingSpec("user", "string", "user", "string"),
    MappingSpec("ad", "string", "ad", "string"),
    MappingSpec("at", "timestamp", "at", "timestamp"),
    MappingSpec("partition_0", "int", "year", "int"),
    MappingSpec("partition_1", "int", "month", "int")
  )

  def main(sysArgs: Array[String]) {
    val spark: SparkContext = new SparkContext()
    val glueContext: GlueContext = new GlueContext(spark)
    val args = GlueArgParser.getResolvedOptions(sysArgs, argNames.toArray)
    Job.init(args("JOB_NAME"), glueContext, args.asJava)

    // val databaseName = args("database-name")
    // val rawTableName = args("raw-table-name")
    // val tableName = args("table-name")

    val adClicksPreCast = glueContext.getCatalogSource("warehouse", "raw_ad_clicks").
      getDynamicFrame()

    adClicksPreCast.printSchema()
    adClicksPreCast.toDF().show()

    val adClicksPreMapping = adClicksPreCast.
      resolveChoice(Seq(
        ("at", "cast:timestamp"),
        ("partition_0", "cast:int"),
        ("partition_0", "cast:int")))

    adClicksPreMapping.printSchema()
    adClicksPreMapping.toDF().show()

    val adClicksPreResolve = adClicksPreMapping.
      applyMapping(mappings, false)

    adClicksPreResolve.printSchema()
    adClicksPreResolve.toDF().show()

    val adClicks = adClicksPreResolve.
      resolveChoice(Seq.empty[ResolveSpec], Some(ChoiceOption("MATCH_CATALOG")), Some("warehouse"), Some("ad_clicks"))

    adClicks.printSchema()
    adClicks.toDF().show()

    val partitioning = JsonOptions("""{"partitionKeys":["year","month"]}""")
    glueContext.getCatalogSink("warehouse", "ad_clicks", additionalOptions = partitioning).
      writeDynamicFrame(adClicks)

    Job.commit()
  }
}
