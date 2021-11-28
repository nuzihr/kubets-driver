package main

import (
	"encoding/json"
	"github.com/pulumi/pulumi/sdk/v3/go/pulumi"
	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/ec2"
	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/eks"
	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/iam"
	"github.com/pulumi/pulumi-aws/sdk/v4/go/aws/s3"
)

func createEksClusterRole(ctx *pulumi.Context, roleName string) (*iam.Role, error) {
	assumeRolePolicy, err  :=  json.Marshal(map[string]interface{}{
		"Version": "2012-10-17",
		"Statement": []map[string]interface{}{
			{
				"Effect": "Allow",
				"Principal": map[string]interface{}{
					"Service": "eks.amazonaws.com",
				},
				"Action": "sts:AssumeRole",
			},
		},
	})
	if err != nil {
		return nil, err
	}

	role, err := iam.NewRole(ctx, roleName, &iam.RoleArgs{
		Name: pulumi.String(roleName),
		AssumeRolePolicy: pulumi.String(assumeRolePolicy),
	})
	if err != nil {
		return nil, err
	}

	_, err = iam.NewRolePolicyAttachment(ctx, "eks-cluster-policy", &iam.RolePolicyAttachmentArgs{
		PolicyArn: pulumi.String("arn:aws:iam::aws:policy/AmazonEKSClusterPolicy"),
		Role: role.Name,
	})
	if err != nil {
		return nil, err
	}

	_, err = iam.NewRolePolicyAttachment(ctx, "eks-service-policy", &iam.RolePolicyAttachmentArgs{
		PolicyArn: pulumi.String("arn:aws:iam::aws:policy/AmazonEKSServicePolicy"),
		Role: role.Name,
	})
	if err != nil {
		return nil, err
	}

	return role, nil
}


func main() {
	pulumi.Run(func(ctx *pulumi.Context) error {
		eksClusterRole, err := createEksClusterRole(ctx, "eks-cluster-role")
		if err != nil {
			return err
		}

		// Create VPC network
		vpc, err := ec2.NewVpc(ctx, "main", &ec2.VpcArgs{
			CidrBlock: pulumi.String("10.0.0.0/16"),
		})
		if err != nil {
			return err
		}

		ig, err := ec2.NewInternetGateway(ctx, "main-ig", &ec2.InternetGatewayArgs{
			VpcId: vpc.ID(),
		})
		if err != nil {
			return err
		}

		rt, err := ec2.NewRouteTable(ctx, "main-rt", &ec2.RouteTableArgs{
			VpcId: vpc.ID(),
			Routes: ec2.RouteTableRouteArray{
				&ec2.RouteTableRouteArgs{
					CidrBlock: pulumi.String("0.0.0.0/0"),
					GatewayId: ig.ID(),
				},
			},
			Tags: pulumi.StringMap{
				"Name": pulumi.String("main-rt"),
			},
		})
		if err != nil {
			return err
		}

		subnetPublicA, err := ec2.NewSubnet(ctx, "main-public-a", &ec2.SubnetArgs{
			VpcId: vpc.ID(),
			CidrBlock: pulumi.String("10.0.1.0/24"),
			AvailabilityZone: pulumi.String("ap-northeast-1a"),
			Tags: pulumi.StringMap{
				"Name": pulumi.String("Main-Public-A"),
			},
		})
		if err != nil {
			return err
		}

		_, err = ec2.NewRouteTableAssociation(ctx, "eks-rta-1", &ec2.RouteTableAssociationArgs{
			SubnetId:     subnetPublicA.ID(),
			RouteTableId: rt.ID(),
		})
		if err != nil {
			return err
		}

		subnetPublicD, err := ec2.NewSubnet(ctx, "main-public-d", &ec2.SubnetArgs{
			VpcId: vpc.ID(),
			CidrBlock: pulumi.String("10.0.2.0/24"),
			AvailabilityZone: pulumi.String("ap-northeast-1d"),
			Tags: pulumi.StringMap{
				"Name": pulumi.String("Main-Public-D"),
			},
		})
		if err != nil {
			return err
		}

		_, err = ec2.NewRouteTableAssociation(ctx, "eks-rta-2", &ec2.RouteTableAssociationArgs{
			SubnetId:     subnetPublicD.ID(),
			RouteTableId: rt.ID(),
		})
		if err != nil {
			return err
		}

		subnetPrivateA, err := ec2.NewSubnet(ctx, "main-private-A", &ec2.SubnetArgs{
			VpcId: vpc.ID(),
			CidrBlock: pulumi.String("10.0.100.0/24"),
			AvailabilityZone: pulumi.String("ap-northeast-1a"),
			Tags: pulumi.StringMap{
				"Name": pulumi.String("Main-Private"),
			},
		})
		if err != nil {
			return err
		}

		_, err = ec2.NewRouteTableAssociation(ctx, "eks-rta-3", &ec2.RouteTableAssociationArgs{
			SubnetId:     subnetPrivateA.ID(),
			RouteTableId: rt.ID(),
		})
		if err != nil {
			return err
		}

		// Create EKS cluster
		eksClusterName := "kubets-cluster"
		_, err = eks.NewCluster(ctx, eksClusterName, &eks.ClusterArgs{
			Name:    pulumi.String(eksClusterName),
			RoleArn: eksClusterRole.Arn,
			VpcConfig: &eks.ClusterVpcConfigArgs{
				SubnetIds: pulumi.StringArray{
					subnetPublicA.ID(),
					subnetPublicD.ID(),
					subnetPrivateA.ID(),
				},
			},
		})
		if err != nil {
			return err
		}

		// Create an AWS resource (S3 Bucket)
		_, err = s3.NewBucket(ctx, "my-bucket", nil)
		if err != nil {
			return err
		}

		return nil
	})
}
