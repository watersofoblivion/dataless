. + {
  "Parameters": (.Parameters + {
    "CodeBucket": "${CODE_BUCKET}",
    "CodePrefix": "build/${CODEBUILD_BUILD_ID}",
    "CodePipeline": "${CODEBUILD_INITIATOR}",
    "BuildRegion": "${AWS_REGION}",
    "Build": "${CODEBUILD_BUILD_ID}",
    "SourceVersion": "${CODEBUILD_SOURCE_VERSION}"
  })
}
