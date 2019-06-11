#!/bin/bash

# Changes the provisioned write capacity units on a DynamoDB table, leaving the
# provisioned read capacity unchanged.

TABLE_NAME=""
WRITE_CAPACITY_UNITS=""

# Get current read capacity
READ_CAPACITY_UNITS=$(aws dynamodb describe-table --table-name "${TABLE_NAME}" --query "Table.ProvisionedThroughput.ReadCapacityUnits")

# Update write capacity
aws dynamodb update-table --table-name "${TABLE_NAME}" --provisioned-throughput ReadCapacityUnits=${READ_CAPACITY_UNITS},WriteCapacityUnits=${WRITE_CAPACITY_UNITS}

# Wait for update to complete
for /bin/true; do
  echo -n "Polling table ${TABLE_NAME}: "
  TABLE_STATUS=$(aws dynamodb describe-table --table-name "${TABLE_NAME}" --query "Table.TableStatus")
  echo "${TABLE_STATUS}"
  
  case "${TABLE_STATUS}" in
    "UPDATING")
      sleep 1
      ;;

    "ACTIVE")
      break
      ;;

    *)
      echo "Unexpected table status: ${TABLE_STATUS}"
      exit 1
      ;;
  esac
done
