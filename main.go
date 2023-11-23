package main

import (
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type EC2Config struct {
	ImageId         string
	InstanceType    string
	KeyName         string
	SecurityGroupID string
	SubnetID        string
	TagValue        string
	VolumeSize      int64
}

func configEC2Instances(batch string) []EC2Config {
	return []EC2Config{
		{
			ImageId:      "ami-0e8849aa060c28662",
			InstanceType: "t3.small",
			//KeyName:         "ec2-user",
			//SecurityGroupID: "sg-033a6552e3ffe1a48",
			//SubnetID:        "subnet-0a7e140afbc1f8f9b",
			TagValue:   batch + "MyFirstInstanceTest1",
			VolumeSize: 10,
		},
	}
}

func main() {
	batch := "b3-"
	configs := configEC2Instances(batch)

	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("ap-east-1"), // 替换为您的AWS区域
	})
	if err != nil {
		fmt.Println("创建会话失败:", err)
		return
	}

	svc := ec2.New(sess)

	for _, config := range configs {
		runResult, err := svc.RunInstances(&ec2.RunInstancesInput{
			ImageId:      aws.String(config.ImageId),
			InstanceType: aws.String(config.InstanceType),
			KeyName:      aws.String("ec2-user"),
			MinCount:     aws.Int64(1),
			MaxCount:     aws.Int64(1),
			SecurityGroupIds: []*string{
				aws.String("sg-033a6552e3ffe1a48"),
			},
			SubnetId: aws.String("subnet-0a7e140afbc1f8f9b"),
			BlockDeviceMappings: []*ec2.BlockDeviceMapping{
				{
					DeviceName: aws.String("/dev/sdh"),
					Ebs: &ec2.EbsBlockDevice{
						VolumeSize: aws.Int64(config.VolumeSize),
						VolumeType: aws.String("gp2"),
					},
				},
			},
			TagSpecifications: []*ec2.TagSpecification{
				{
					ResourceType: aws.String("instance"),
					Tags: []*ec2.Tag{
						{
							Key:   aws.String("Name"),
							Value: aws.String(config.TagValue),
						},
					},
				},
			},
		})
		if err != nil {
			fmt.Println("无法创建实例:", err)
			return
		}

		instanceId := runResult.Instances[0].InstanceId

		// 等待实例变为running状态
		fmt.Println("等待实例启动...")
		for {
			descInstances, err := svc.DescribeInstances(&ec2.DescribeInstancesInput{
				InstanceIds: []*string{instanceId},
			})
			if err != nil {
				fmt.Println("无法获取实例状态:", err)
				return
			}

			state := descInstances.Reservations[0].Instances[0].State.Name
			if *state == "running" {
				break
			}

			time.Sleep(10 * time.Second)
		}
		fmt.Println("实例已启动,正在分配弹性IP...")

		// 申请弹性IP
		allocRes, err := svc.AllocateAddress(&ec2.AllocateAddressInput{
			Domain: aws.String("vpc"), // VPC网络
		})
		if err != nil {
			fmt.Println("无法分配弹性IP:", err)
			return
		}

		// 关联弹性IP到实例
		_, err = svc.AssociateAddress(&ec2.AssociateAddressInput{
			InstanceId:   instanceId,
			AllocationId: allocRes.AllocationId,
		})
		if err != nil {
			fmt.Println("无法关联弹性IP:", err)
			return
		}
		fmt.Println("弹性IP已成功关联到实例:", *instanceId)
	}
}
