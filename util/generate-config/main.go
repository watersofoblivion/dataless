package main

import (
	"encoding/json"
	"flag"
	"io"
	"log"
	"os"
)

func main() {
	var (
		stageName                                                      string
		codeBucket, codePrefix                                         string
		dnsDomainName, dnsName, validationDomain, hostedZoneID         string
		enableVPC, enableEC2Instance, enableEMRCluster, enableRedshift string
		ec2InstanceSize                                                string
		emrMasterInstanceType, emrCoreInstanceType                     string
		emrMasterInstanceCount, emrCoreInstanceCount                   int
		redshiftNodeType                                               string
		redshiftNumberOfNodes                                          int
		apiName                                                        string
		output                                                         string
	)

	flag.StringVar(&codeBucket, "code-bucket", codeBucket, "The bucket containing user code to run")
	flag.StringVar(&codePrefix, "code-prefix", codePrefix, "The prefix into the bucket containing user code to run")
	flag.StringVar(&stageName, "stage", stageName, "The name for a project pipeline stage, such as Staging or Prod, for which resources are provisioned and deployed.")
	flag.StringVar(&dnsDomainName, "dns-domain-name", dnsDomainName, "")
	flag.StringVar(&dnsName, "dns-name", dnsName, "")
	flag.StringVar(&validationDomain, "validation-domain", validationDomain, "")
	flag.StringVar(&hostedZoneID, "hosted-zone-id", hostedZoneID, "")
	flag.StringVar(&enableVPC, "vpc", enableVPC, "")
	flag.StringVar(&enableEC2Instance, "ec2", enableEC2Instance, "")
	flag.StringVar(&ec2InstanceSize, "ec2-type", ec2InstanceSize, "")
	flag.StringVar(&enableEMRCluster, "emr", enableEMRCluster, "")
	flag.StringVar(&emrMasterInstanceType, "emr-master-type", emrMasterInstanceType, "")
	flag.IntVar(&emrMasterInstanceCount, "emr-master-count", emrMasterInstanceCount, "")
	flag.StringVar(&emrCoreInstanceType, "emr-core-type", emrCoreInstanceType, "")
	flag.IntVar(&emrCoreInstanceCount, "emr-core-count", emrCoreInstanceCount, "")
	flag.StringVar(&enableRedshift, "redshift", enableRedshift, "")
	flag.StringVar(&redshiftNodeType, "redshift-type", redshiftNodeType, "")
	flag.IntVar(&redshiftNumberOfNodes, "redshift-count", redshiftNumberOfNodes, "")
	flag.StringVar(&apiName, "api-name", apiName, "")
	flag.StringVar(&output, "o", output, "The file to output to.  If not given, STDOUT is used.")

	flag.Parse()

	if codeBucket == "" {
		log.Fatal("code bucket not set")
	}
	if codePrefix == "" {
		log.Fatal("code prefix not set")
	}
	if apiName == "" {
		log.Fatal("${API_NAME} not set")
	}

	parameters := map[string]interface{}{
		"CodeBucket": codeBucket,
		"CodePrefix": codePrefix,
		"APIName":    apiName,
	}

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

	if enableVPC != "" {
		parameters["EnableVPC"] = "true"
	}

	if enableEC2Instance != "" {
		parameters["EnableEC2Instance"] = "true"
		if ec2InstanceSize != "" {
			parameters["EC2InstanceSize"] = ec2InstanceSize
		}
	}

	if enableEMRCluster != "" {
		parameters["EnableEMRCluster"] = "true"

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

	if enableRedshift != "" {
		parameters["EnableRedshift"] = "true"

		if redshiftNodeType != "" {
			parameters["RedshiftNodeType"] = redshiftNodeType
		}
		if redshiftNumberOfNodes != 0 {
			parameters["RedshiftNumberOfNodes"] = redshiftNumberOfNodes
		}
	}

	config := make(map[string]interface{})
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
