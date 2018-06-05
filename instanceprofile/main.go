package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
)

const (
	arnPolicyAmazonS3FullAccess = "arn:aws:iam::aws:policy/AmazonS3FullAccess"
)

var (
	region string
	name   string
)

func init() {
	flag.StringVar(&region, "region", "ap-northeast-1", "Region")
	flag.StringVar(&name, "role-name", "testtest", "Role and profile name")
	flag.Parse()
}
func main() {

	sess := session.New(&aws.Config{
		Region: aws.String(region),
	})
	client := iam.New(sess)

	assumepolicy := map[string]interface{}{
		"Version": "2012-10-17",
		"Statement": []map[string]interface{}{
			{
				"Sid":       "",
				"Effect":    "Allow",
				"Action":    "sts:AssumeRole",
				"Principal": map[string]string{"Service": "ec2.amazonaws.com"},
			},
		},
	}

	buf := bytes.NewBuffer(nil)
	if err := json.NewEncoder(buf).Encode(assumepolicy); err != nil {
		panic(err)
	}

	createRoleOutput, err := client.CreateRole(&iam.CreateRoleInput{
		Path:                     aws.String("/"),
		RoleName:                 aws.String(name),
		AssumeRolePolicyDocument: aws.String(buf.String()),
		Description:              aws.String(time.Now().Format(time.RFC3339)),
	})
	if err != nil {
		panic(err)
	}
	role := createRoleOutput.Role

	_, err = client.AttachRolePolicy(&iam.AttachRolePolicyInput{
		PolicyArn: aws.String(arnPolicyAmazonS3FullAccess),
		RoleName:  role.RoleName,
	})
	if err != nil {
		panic(err)
	}

	instanceprofileCreateOutput, err := client.CreateInstanceProfile(&iam.CreateInstanceProfileInput{
		InstanceProfileName: role.RoleName, // めんどくさいのでRole名と同じにします
	})
	if err != nil {
		panic(err)
	}
	instanceprofile := instanceprofileCreateOutput.InstanceProfile

	_, err = client.AddRoleToInstanceProfile(&iam.AddRoleToInstanceProfileInput{
		InstanceProfileName: instanceprofile.InstanceProfileName,
		RoleName:            role.RoleName,
	})
	if err != nil {
		panic(err)
	}

	fmt.Printf("%+v\n", instanceprofile)
}
