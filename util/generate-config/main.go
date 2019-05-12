package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
)

func main() {

	var (
		partition                                              = os.Getenv("PARTITION")
		region                                                 = os.Getenv("AWS_REGION")
		accountID                                              = os.Getenv("ACCOUNT_ID")
		projectID                                              = os.Getenv("PROJECT_ID")
		stageName                                              string
		dnsDomainName, dnsName, validationDomain, hostedZoneID string
		enableEC2Instance, enableEMRCluster, enableRedshift    bool
		ec2InstanceSize                                        string
		emrMasterInstanceType, emrCoreInstanceType             string
		emrMasterInstanceCount, emrCoreInstanceCount           int
		redshiftNodeType                                       string
		redshiftNumberOfNodes                                  int
		output                                                 string
	)

	flag.StringVar(&partition, "partition", partition, "The AWS partition containing the CodeStar project")
	flag.StringVar(&region, "region", region, "The region containing the CodeStar project")
	flag.StringVar(&accountID, "account-id", accountID, "The AWS account containing the CodeStar project")
	flag.StringVar(&projectID, "project-id", projectID, "The CodeStar project")
	flag.StringVar(&stageName, "stage", stageName, "The name for a project pipeline stage, such as Staging or Prod, for which resources are provisioned and deployed.")
	flag.StringVar(&dnsDomainName, "dns-domain-name", dnsDomainName, "")
	flag.StringVar(&dnsName, "dns-name", dnsName, "")
	flag.StringVar(&validationDomain, "validation-domain", validationDomain, "")
	flag.StringVar(&hostedZoneID, "hosted-zone-id", hostedZoneID, "")
	flag.BoolVar(&enableEC2Instance, "ec2", enableEC2Instance, "")
	flag.StringVar(&ec2InstanceSize, "ec2-type", ec2InstanceSize, "")
	flag.BoolVar(&enableEMRCluster, "emr", enableEMRCluster, "")
	flag.StringVar(&emrMasterInstanceType, "emr-master-type", emrMasterInstanceType, "")
	flag.IntVar(&emrMasterInstanceCount, "emr-master-count", emrMasterInstanceCount, "")
	flag.StringVar(&emrCoreInstanceType, "emr-core-type", emrCoreInstanceType, "")
	flag.IntVar(&emrCoreInstanceCount, "emr-core-count", emrCoreInstanceCount, "")
	flag.BoolVar(&enableRedshift, "redshift", enableRedshift, "")
	flag.StringVar(&redshiftNodeType, "redshift-type", redshiftNodeType, "")
	flag.IntVar(&redshiftNumberOfNodes, "redshift-count", redshiftNumberOfNodes, "")
	flag.StringVar(&output, "o", output, "The file to output to.  If not given, STDOUT is used.")

	flag.Parse()

	if partition == "" {
		log.Fatal("${PARTITION} not set")
	}
	if region == "" {
		log.Fatal("${AWS_REGION} not set")
	}
	if accountID == "" {
		log.Fatal("${ACCOUNT_ID} not set")
	}
	if projectID == "" {
		log.Fatal("${PROJECT_ID} not set")
	}
	projectARN := fmt.Sprintf("arn:%s:codestar:%s:%s:project/%s", partition, region, accountID, projectID)

	parameters := make(map[string]interface{})

	// Stage
	if stageName != "" {
		parameters["Stage"] = stageName
	}

	// DNS
	if dnsDomainName != "" {
		parameters["DNSDomainName"] = dnsDomainName
	}
	if dnsName != "" {
		parameters["DNSName"] = dnsName
	}
	if validationDomain != "" {
		parameters["ValidationDomain"] = validationDomain
	}
	if hostedZoneID != "" {
		parameters["HostedZoneID"] = hostedZoneID
	}

	if enableEC2Instance {
		parameters["EnableEC2Instance"] = true
		if ec2InstanceSize != "" {
			parameters["EC2InstanceSize"] = ec2InstanceSize
		}
	}

	if enableEMRCluster {
		parameters["EnableEMRCluster"] = true

		if emrMasterInstanceType != "" {
			parameters["EMRMasterInstanceType"] = emrMasterInstanceType
		}
		if emrMasterInstanceCount != 0 {
			parameters["EMRMasterInstanceCount"] = emrMasterInstanceCount
		}
		if emrCoreInstanceType != "" {
			parameters["EMRCoreInstanceType"] = emrCoreInstanceType
		}
		if emrCoreInstanceCount != 0 {
			parameters["EMRCoreInstanceCount"] = emrCoreInstanceCount
		}
	}

	if enableRedshift {
		parameters["EnableRedshift"] = true

		if redshiftNodeType != "" {
			parameters["RedshiftNodeType"] = redshiftNodeType
		}
		if redshiftNumberOfNodes != 0 {
			parameters["RedshiftNumberOfNodes"] = redshiftNumberOfNodes
		}
	}

	config := map[string]interface{}{
		"Tags": map[string]interface{}{
			"awscodestar:projectArn": projectARN,
		},
	}

	if len(parameters) > 0 {
		config["Parameters"] = parameters
	}

	var w io.Writer = os.Stdout
	if output != "" {
		f, err := os.OpenFile(output, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()

		w = f
	}

	if err := json.NewEncoder(w).Encode(config); err != nil {
		log.Fatal(err)
	}
}
