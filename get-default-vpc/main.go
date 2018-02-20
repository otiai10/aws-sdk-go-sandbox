package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

var (
	region string
)

func init() {
	flag.StringVar(&region, "region", "ap-northeast-1", "Region")
	flag.Parse()
}

func main() {
	sess := session.New(&aws.Config{
		Region: aws.String(region),
	})
	client := ec2.New(sess)
	out, err := client.DescribeAccountAttributes(&ec2.DescribeAccountAttributesInput{
		AttributeNames: []*string{aws.String("default-vpc")},
	})
	if err != nil {
		log.Fatalln(err)
	}
	for _, attr := range out.AccountAttributes {
		if *attr.AttributeName == "default-vpc" {
			fmt.Println("VPC ID:", attr.String())
			return
		}
	}
	log.Fatalln("no vpc id found")
}
