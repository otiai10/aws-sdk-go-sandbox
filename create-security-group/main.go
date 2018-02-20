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
	region      string
	name        string
	description string
	clean       bool
)

func init() {
	flag.StringVar(&region, "region", "ap-northeast-1", "Region")
	flag.StringVar(&name, "name", "test-sdk-go", "Security Group Name")
	flag.StringVar(&description, "description", "Foo Bar Baz", "Description")
	flag.BoolVar(&clean, "clean", false, "DEBUG: clean up existing SG")
	flag.Parse()
}

func main() {

	sess := session.New(&aws.Config{
		Region: aws.String(region),
	})
	client := ec2.New(sess)

	// Delete existing SG if exists
	_, err := client.DeleteSecurityGroup(&ec2.DeleteSecurityGroupInput{
		GroupName: aws.String(name),
	})
	if clean {
		if err != nil {
			fmt.Println("DELETE:", err)
		}
		return
	}

	// Create new one
	group, err := client.CreateSecurityGroup(&ec2.CreateSecurityGroupInput{
		GroupName:   aws.String(name),
		Description: aws.String(description),
	})
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Printf("GROUP: %v\n", group)

	// Add rules
	_, err = client.AuthorizeSecurityGroupIngress(&ec2.AuthorizeSecurityGroupIngressInput{
		GroupId: group.GroupId,
		IpPermissions: []*ec2.IpPermission{
			&ec2.IpPermission{
				IpRanges:   []*ec2.IpRange{&ec2.IpRange{CidrIp: aws.String("0.0.0.0/0")}},
				IpProtocol: aws.String("tcp"),
				FromPort:   aws.Int64(22),
				ToPort:     aws.Int64(22),
			},
		},
	})
	if err != nil {
		log.Fatalln(err)
	}

	// Check
	out, err := client.DescribeSecurityGroups(&ec2.DescribeSecurityGroupsInput{
		GroupIds: []*string{group.GroupId},
	})
	if err != nil {
		log.Fatalln(err)
	}
	for _, g := range out.SecurityGroups {
		fmt.Printf("%+v\n", g)
	}
}
