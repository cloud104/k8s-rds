package rds

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/aws/external"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	k8srds "github.com/cloud104/kube-db/pkg/actuators/rds/client"
)

const Failed = "Failed"
const dryRun = true

func NewActuator(log logr.Logger, config *rest.Config) (a *Actuator, err error) {
	kubectl, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}

	awsConfig, err := configClient(kubectl)
	if err != nil {
		return nil, err
	}

	ec2client := ec2.New(awsConfig)
	rdsclient := rds.New(awsConfig)

	// securityGroups := []string{}
	securityGroups, err := getSecurityGroups(ec2client, kubectl)
	if err != nil {
		return nil, err
	}

	// subnets := []string{}
	subnets, err := getSubnets(ec2client, false, kubectl)
	if err != nil {
		return nil, err
	}

	return &Actuator{
		log:        log,
		ec2client:  ec2client,
		rdsclient:  rdsclient,
		kubeClient: &Kube{Client: kubectl},
		k8srds: &k8srds.AWS{
			RDS:            rdsclient,
			EC2:            ec2client,
			Subnets:        subnets,
			SecurityGroups: securityGroups,
		},
	}, nil
}

func configClient(kubectl *kubernetes.Clientset) (aws.Config, error) {
	nodes, err := kubectl.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return aws.Config{}, errors.Wrap(err, "unable to get nodes")
	}
	region := ""

	if len(nodes.Items) > 0 {
		// take the first one, we assume that all nodes are created in the same VPC
		region = nodes.Items[0].Labels["failure-domain.beta.kubernetes.io/region"]
	} else {
		return aws.Config{}, fmt.Errorf("unable to find any nodes in the cluster")
	}

	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		return aws.Config{}, err
	}

	// Set the AWS Region that the service clients should use
	cfg.Region = region
	cfg.HTTPClient.Timeout = 5 * time.Second
	return cfg, nil
}

func getSecurityGroups(svc *ec2.Client, kubectl *kubernetes.Clientset) ([]string, error) {
	ctx := context.Background()
	nodes, err := kubectl.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "unable to get nodes")
	}

	name := ""
	if len(nodes.Items) > 0 {
		// take the first one, we assume that all nodes are created in the same VPC
		name = nodes.Items[0].Name
	} else {
		return nil, fmt.Errorf("unable to find any nodes in the cluster")
	}

	params := &ec2.DescribeInstancesInput{
		Filters: []ec2.Filter{
			{
				Name:   aws.String("private-dns-name"),
				Values: []string{name},
			},
		},
	}

	req := svc.DescribeInstancesRequest(params)
	res, err := req.Send(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "unable to describe AWS instance")
	}

	var result []string
	if len(res.Reservations) >= 1 {
		for _, v := range res.Reservations[0].Instances[0].SecurityGroups {
			result = append(result, *v.GroupId)
		}
	}

	return result, nil
}

func getSubnets(svc *ec2.Client, public bool, kubectl *kubernetes.Clientset) ([]string, error) {
	ctx := context.Background()
	nodes, err := kubectl.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return nil, errors.Wrap(err, "unable to get nodes")
	}
	name := ""

	if len(nodes.Items) > 0 {
		// take the first one, we assume that all nodes are created in the same VPC
		name = nodes.Items[0].Name
	} else {
		return nil, fmt.Errorf("unable to find any nodes in the cluster")
	}

	params := &ec2.DescribeInstancesInput{
		Filters: []ec2.Filter{
			{
				Name: aws.String("private-dns-name"),
				Values: []string{
					name,
				},
			},
		},
	}

	req := svc.DescribeInstancesRequest(params)
	res, err := req.Send(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "unable to describe AWS instance")
	}

	var result []string
	if len(res.Reservations) >= 1 {
		vpcID := res.Reservations[0].Instances[0].VpcId

		res := svc.DescribeSubnetsRequest(&ec2.DescribeSubnetsInput{Filters: []ec2.Filter{{Name: aws.String("vpc-id"), Values: []string{*vpcID}}}})
		subnets, err := res.Send(ctx)

		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("unable to describe subnet in VPC %v", *vpcID))
		}
		for _, sn := range subnets.Subnets {
			if *sn.MapPublicIpOnLaunch == public {
				result = append(result, *sn.SubnetId)
			}
		}

	}

	return result, nil
}
