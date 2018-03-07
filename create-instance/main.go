package main

import (
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/otiai10/stackerr"
)

var (
	region        string
	name          string
	instancetype  string
	securitygroup string
	keyname       string
	description   string
	clean         bool
)

func init() {
	flag.StringVar(&region, "region", "ap-northeast-1", "Region")
	flag.StringVar(&name, "name", "Test-Instance-SDK-Go", "Instance Name")
	flag.StringVar(&instancetype, "instancetype", "t2.micro", "Instance Type")
	flag.StringVar(&securitygroup, "sg", "", "SecurityGroup Name")
	flag.StringVar(&keyname, "keyname", "", "Key Pair Name")
	flag.BoolVar(&clean, "clean", false, "DEBUG: clean up existing SG")
	flag.Parse()
}

func validate() error {
	err := stackerr.New()
	if securitygroup == "" {
		err.Pushf("SecurityGroup is required: Use `-sg YourSecurityGroupName`")
	}
	if keyname == "" {
		err.Pushf("Key Pair Name is required: Use `-keyname YourKeyPairName`")
	}
	return err.Err()
}

func main() {

	if err := validate(); err != nil {
		log.Fatalf("ValidationError:\n%v", err)
	}

	sess := session.New(&aws.Config{
		Region: aws.String(region),
	})
	client := ec2.New(sess)

	out, err := client.RunInstances(&ec2.RunInstancesInput{
		InstanceType:   aws.String(instancetype),
		SecurityGroups: []*string{&securitygroup},
		ImageId:        aws.String("ami-c2680fa4"),
		KeyName:        aws.String(keyname),
		MaxCount:       aws.Int64(1),
		MinCount:       aws.Int64(1),
		TagSpecifications: []*ec2.TagSpecification{
			{
				ResourceType: aws.String("instance"),
				Tags: []*ec2.Tag{
					{Key: aws.String("Name"), Value: &name},
				},
			},
		},
	})
	if err != nil {
		log.Fatalln("RunInstances Error:\n", err)
	}
	if len(out.Instances) == 0 {
		log.Fatalln("No instances created")
	}
	instance := out.Instances[0]
	fmt.Printf("Successfully created instance: %s\n", *instance.InstanceId)

	// Defer cleanup
	if clean {
		defer cleanup(client, instance.InstanceId)
	}

	instance, err = ensure(client, instance.InstanceId, 2)
	if err != nil {
		log.Println("DescribeInstances Error:\n", err)
		return
	}

	fmt.Printf(`Next:
    SSH:        ssh -i file/path/to/%s.pem ec2-user@%s
    Terminate:  aws ec2 terminate-instances --instance-id %s

`, *instance.KeyName, *instance.PublicIpAddress, *instance.InstanceId)
}

func cleanup(client *ec2.EC2, id *string) {
	out, err := client.TerminateInstances(&ec2.TerminateInstancesInput{
		InstanceIds: []*string{id},
	})
	fmt.Printf("Terminated by -clean option: ID:%+v Error:%v\n", *out.TerminatingInstances[0].InstanceId, err)
}

func ensure(client *ec2.EC2, id *string, waitsecond int) (*ec2.Instance, error) {
	if waitsecond > 60 {
		return nil, fmt.Errorf("Max retry for DescribeInstances exeeded: %d seconds", waitsecond)
	}
	fmt.Printf("Trying to fetch Public IP Address by DescribeInstances... (Retry after %d sec)\n", waitsecond)
	res, err := client.DescribeInstances(&ec2.DescribeInstancesInput{InstanceIds: []*string{id}})
	if err != nil {
		return nil, err
	}
	// TODO: check length
	instance := res.Reservations[0].Instances[0]
	if instance.PublicIpAddress == nil {
		time.Sleep(time.Duration(waitsecond) * time.Second)
		return ensure(client, id, waitsecond+waitsecond)
	}
	return instance, nil
}
