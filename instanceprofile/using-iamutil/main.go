package main

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/otiai10/iamutil"
)

func main() {

	sess := session.New(&aws.Config{
		Region: aws.String("ap-northeast-1"),
	})

	name := "otiai10-test"

	if found, err := iamutil.FindInstanceProfile(sess, name); err != nil {
		ae, ok := err.(awserr.Error)
		if !ok {
			fmt.Println("Unexpected error on finding existing instance profile:", err)
			return
		}
		if ae.Code() != iam.ErrCodeNoSuchEntityException {
			fmt.Println(err)
			return
		}
	} else {
		if err := found.Delete(sess); err != nil {
			fmt.Println("Failed to delete existing instance profile", err)
			return
		}
	}

	profile := &iamutil.InstanceProfile{
		Role: &iamutil.Role{
			Description: "Test Role by iamutil",
			PolicyArns: []string{
				"arn:aws:iam::aws:policy/AmazonS3FullAccess",
			},
		},
		Name: name,
	}

	if err := profile.Create(sess); err != nil {
		fmt.Printf("ERROR: %v\n", err)
		return
	}

	fmt.Printf("DELETE: %v\n", profile.Delete(sess))
}
