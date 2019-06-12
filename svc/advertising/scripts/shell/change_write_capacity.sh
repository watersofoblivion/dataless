#!/bin/bash

# Changes the provisioned write capacity units on a DynamoDB table, leaving the
# provisioned read capacity unchanged.
#
# Takes two arguments: the table to change, and the delta to change the write
# capacity by.

TABLE_NAME="${1}"
WRITE_CAPACITY_CHANGE="${2}"

# Get current capacity
READ_CAPACITY_UNITS=$(aws dynamodb describe-table --table-name "${TABLE_NAME}" --query "Table.ProvisionedThroughput.ReadCapacityUnits" --output text)
WRITE_CAPACITY_UNITS=$(aws dynamodb describe-table --table-name "${TABLE_NAME}" --query "Table.ProvisionedThroughput.WriteCapacityUnits" --output text)

# Compute new write capacity.  Ensure we don't go below 1.
WRITE_CAPACITY_UNITS=$((${WRITE_CAPACITY_UNITS} + ${WRITE_CAPACITY_CHANGE}))
if [[ ${WRITE_CAPACITY_UNITS} < 1 ]]; then
  WRITE_CAPACITY_UNITS=1
fi

# Update write capacity
aws dynamodb update-table --table-name "${TABLE_NAME}" --provisioned-throughput ReadCapacityUnits=${READ_CAPACITY_UNITS},WriteCapacityUnits=${WRITE_CAPACITY_UNITS}

# Wait for update to complete
while true; do
  echo -n "Polling table ${TABLE_NAME}: "
  TABLE_STATUS=$(aws dynamodb describe-table --table-name "${TABLE_NAME}" --query "Table.TableStatus" --output text)
  echo "${TABLE_STATUS}"

  case "${TABLE_STATUS}" in
    "UPDATING")
      sleep 1
      ;;

    "ACTIVE")
      echo "Table updated"
      exit 0
      ;;

    *)
      echo "Unexpected table status: ${TABLE_STATUS}"
      exit 1
      ;;
  esac
done
