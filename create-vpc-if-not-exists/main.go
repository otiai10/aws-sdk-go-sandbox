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
	name   string
	clean  bool
)

func init() {
	flag.StringVar(&region, "region", "ap-northeast-1", "Region")
	flag.StringVar(&name, "name", "otiai10-sdk-test", "Name of VPC")
	flag.BoolVar(&clean, "clean", false, "Clean up created VPC")
	flag.Parse()
}

func main() {
	sess := session.New(&aws.Config{
		Region: aws.String(region),
	})
	client := ec2.New(sess)

	// Get VPC if exists
	vpcsout, err := client.DescribeVpcs(&ec2.DescribeVpcsInput{
		Filters: []*ec2.Filter{
			&ec2.Filter{Name: aws.String("tag:Name"), Values: aws.StringSlice([]string{name})},
		},
	})
	if err != nil {
		log.Fatalln("01", err)
	}

	if vpcs := vpcsout.Vpcs; len(vpcs) != 0 {
		fmt.Printf("Total %d VPCs found.\n", len(vpcs))
		if !clean {
			return // If exists, do nothing
		}
		// Clean up if "clean" flag is specified.
		if err := cleanupVpcs(client, vpcs); err != nil {
			log.Fatalln("Cleaning failed:", err)
		}
	}

	// Create because it doesn't exist
	createout, err := client.CreateVpc(&ec2.CreateVpcInput{
		CidrBlock: aws.String("10.0.0.0/24"),
	})
	if err != nil {
		log.Fatalln("02", err)
	}
	vpc := createout.Vpc
	fmt.Println("VPC created:", *vpc.VpcId)
	// Name this VPC
	_, err = client.CreateTags(&ec2.CreateTagsInput{
		Tags: []*ec2.Tag{
			{Key: aws.String("Name"), Value: aws.String(name)},
		},
		Resources: []*string{vpc.VpcId},
	})
	if err != nil {
		log.Fatalln("03", err)
	}

	// Create subnet
	subnetout, err := client.CreateSubnet(&ec2.CreateSubnetInput{
		AvailabilityZone: aws.String(region + "a"), // FIXME: hard coded
		CidrBlock:        aws.String("10.0.0.0/28"),
		VpcId:            vpc.VpcId,
	})
	if err != nil {
		log.Fatalln("04", err)
	}
	subnet := subnetout.Subnet
	fmt.Println("Subnet created:", *subnet.SubnetId)
	_, err = client.CreateTags(&ec2.CreateTagsInput{
		Tags: []*ec2.Tag{
			{Key: aws.String("Name"), Value: aws.String(name + "-sn")},
		},
		Resources: []*string{subnet.SubnetId},
	})
	if err != nil {
		log.Fatalln("05", err)
	}

	// Create InternetGateway
	gatewayout, err := client.CreateInternetGateway(&ec2.CreateInternetGatewayInput{})
	if err != nil {
		log.Fatalln("06", err)
	}
	gateway := gatewayout.InternetGateway
	fmt.Println("InternetGateway created:", *gateway.InternetGatewayId)
	_, err = client.CreateTags(&ec2.CreateTagsInput{
		Tags: []*ec2.Tag{
			{Key: aws.String("Name"), Value: aws.String(name + "-ig")},
		},
		Resources: []*string{gateway.InternetGatewayId},
	})
	if err != nil {
		log.Fatalln("07", err)
	}

	// Attach this InternetGateway to VPC
	_, err = client.AttachInternetGateway(&ec2.AttachInternetGatewayInput{
		InternetGatewayId: gateway.InternetGatewayId,
		VpcId:             vpc.VpcId,
	})
	if err != nil {
		log.Fatalln("08", err)
	}

	// Create RouteTables
	routetablesout, err := client.DescribeRouteTables(&ec2.DescribeRouteTablesInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("vpc-id"), Values: []*string{vpc.VpcId}},
		},
	})
	if err != nil {
		log.Fatalln("09", err)
	}
	if len(routetablesout.RouteTables) == 0 {
		log.Fatalln("10", "No route table found on this VPC")
	}
	routetable := routetablesout.RouteTables[0]
	fmt.Println("RouteTable created:", *routetable.RouteTableId)
	_, err = client.CreateTags(&ec2.CreateTagsInput{
		Tags: []*ec2.Tag{
			{Key: aws.String("Name"), Value: aws.String(name + "-rt")},
		},
		Resources: []*string{routetable.RouteTableId},
	})
	if err != nil {
		log.Fatalln("11", err)
	}

	// Create Routing Rule on this RouteTables
	_, err = client.CreateRoute(&ec2.CreateRouteInput{
		DestinationCidrBlock: aws.String("0.0.0.0/0"),
		RouteTableId:         routetable.RouteTableId,
		GatewayId:            gateway.InternetGatewayId,
	})
	if err != nil {
		log.Fatalln("12", err)
	}
	fmt.Println("Route created")

	const policydocument = `{
	"Statement": [
			{
					"Action": "*",
					"Effect": "Allow",
					"Resource": "*",
					"Principal": "*"
			}
	]
}`
	vpceout, err := client.CreateVpcEndpoint(&ec2.CreateVpcEndpointInput{
		RouteTableIds:  []*string{routetable.RouteTableId},
		ServiceName:    aws.String(fmt.Sprintf("com.amazonaws.%s.s3", region)),
		PolicyDocument: aws.String(policydocument),
		VpcId:          vpc.VpcId,
	})
	if err != nil {
		log.Fatalln("13", err)
	}

	fmt.Printf("VPC Endpoint created: %s\n", *vpceout.VpcEndpoint.VpcEndpointId)

	fmt.Println("Congrats! Everything is up!!")
}

func cleanupVpcs(client *ec2.EC2, vpcs []*ec2.Vpc) error {
	for _, vpc := range vpcs {
		if err := cleanupVpc(client, vpc); err != nil {
			return err
		}
	}
	return nil
}
func cleanupVpc(client *ec2.EC2, vpc *ec2.Vpc) error {

	// Delete InternetGateway
	igws, err := client.DescribeInternetGateways(&ec2.DescribeInternetGatewaysInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("tag:Name"), Values: aws.StringSlice([]string{name + "-ig"})},
		},
	})
	if err != nil {
		return err
	}
	fmt.Printf("[clean] %d InternetGateways found,\n", len(igws.InternetGateways))
	for _, ig := range igws.InternetGateways {
		if _, err := client.DetachInternetGateway(&ec2.DetachInternetGatewayInput{
			InternetGatewayId: ig.InternetGatewayId,
			VpcId:             vpc.VpcId,
		}); err != nil {
			return err
		}
		if _, err := client.DeleteInternetGateway(&ec2.DeleteInternetGatewayInput{
			InternetGatewayId: ig.InternetGatewayId,
		}); err != nil {
			return err
		}
	}
	fmt.Println("[clean] and deleted.")

	// Delete Subnets
	snts, err := client.DescribeSubnets(&ec2.DescribeSubnetsInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("tag:Name"), Values: aws.StringSlice([]string{name + "-sn"})},
		},
	})
	if err != nil {
		return err
	}
	fmt.Printf("[clean] %d Subnets found,\n", len(snts.Subnets))
	for _, sn := range snts.Subnets {
		if _, err := client.DeleteSubnet(&ec2.DeleteSubnetInput{
			SubnetId: sn.SubnetId,
		}); err != nil {
			return err
		}
	}
	fmt.Println("[clean] and deleted.")

	// Delete VPC endpoints
	vpces, err := client.DescribeVpcEndpoints(&ec2.DescribeVpcEndpointsInput{
		Filters: []*ec2.Filter{
			{Name: aws.String("vpc-id"), Values: []*string{vpc.VpcId}},
		},
	})
	if err != nil {
		return err
	}
	fmt.Printf("[clean] %d VPC endpoints found,\n", len(vpces.VpcEndpoints))
	vpcepIDs := []*string{}
	for _, vpcep := range vpces.VpcEndpoints {
		vpcepIDs = append(vpcepIDs, vpcep.VpcEndpointId)
	}
	if _, err := client.DeleteVpcEndpoints(&ec2.DeleteVpcEndpointsInput{
		VpcEndpointIds: vpcepIDs,
	}); err != nil {
		return err
	}
	fmt.Println("[clean] and deleted.")

	// Delete VPC
	if _, err := client.DeleteVpc(&ec2.DeleteVpcInput{
		VpcId: vpc.VpcId,
	}); err != nil {
		return err
	}

	fmt.Printf("[clean] VPC %s deleted.\n", *vpc.VpcId)
	return nil
}
