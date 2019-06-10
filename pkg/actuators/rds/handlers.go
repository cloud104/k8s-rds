package rds

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"

	databasesv1 "github.com/cloud104/kube-db/api/v1"
	controllers "github.com/cloud104/kube-db/controllers"
	k8srds "github.com/cloud104/kube-db/pkg/actuators/rds/client"
)

func handleCreateDatabase(log logr.Logger, db *databasesv1.Rds, ec2client *ec2.Client, rdsclient *rds.Client, crdclient *controllers.RdsReconciler, kubectl *kubernetes.Clientset, ctx context.Context, namespacedName types.NamespacedName) error {
	if db.Status.State == "Created" {
		log.Info("database already created, skipping", "name", db.Name)
		return nil
	}
	// validate dbname is only alpha numeric
	err := updateStatus(db, databasesv1.RdsStatus{Message: "Creating", State: "Creating"}, crdclient, ctx, namespacedName)
	if err != nil {
		return fmt.Errorf("database CRD status update failed: %v", err)
	}
	log.Info("trying to get subnets")
	subnets, err := getSubnets(log, ec2client, db.Spec.PubliclyAccessible, kubectl)
	if err != nil {
		return fmt.Errorf("unable to get subnets from instance: %v", err)

	}
	log.Info("trying to get security groups")
	sgs, err := getSGS(ec2client, kubectl)
	if err != nil {
		return fmt.Errorf("unable to get security groups from instance: %v", err)

	}

	k := Kube{Client: kubectl}
	log.Info("getting secret: Name", "name", db.Spec.Password.Name, "key", db.Spec.Password.Key)
	pw, err := k.GetSecret(db.Namespace, db.Spec.Password.Name, db.Spec.Password.Key)
	if err != nil {
		return err
	}

	r := k8srds.AWS{RDS: rdsclient, EC2: ec2client, Subnets: subnets, SecurityGroups: sgs}
	hostname, err := r.CreateDatabase(db, pw)
	if err != nil {
		return err
	}
	log.Info("Creating service", "name", db.Name, "hostname", hostname, "namespace", db.Namespace)
	err = k.CreateService(db.Namespace, hostname, db.Name)
	if err != nil {
		return err
	}

	err = updateStatus(db, databasesv1.RdsStatus{Message: "Created", State: "Created"}, crdclient, ctx, namespacedName)
	if err != nil {
		return err
	}
	log.Info("Creation of database", "name", db.Name)
	return nil
}

func handleRestoreDatabase(log logr.Logger, db *databasesv1.Rds, ec2client *ec2.Client, rdsclient *rds.Client, crdclient *controllers.RdsReconciler, kubectl *kubernetes.Clientset, ctx context.Context, namespacedName types.NamespacedName) error {
	if db.Status.State == "Created" {
		log.Info("database already created, skipping", "name", db.Name)
		return nil
	}
	// validate dbname is only alpha numeric
	err := updateStatus(db, databasesv1.RdsStatus{Message: "Creating", State: "Creating"}, crdclient, ctx, namespacedName)
	if err != nil {
		return fmt.Errorf("database CRD status update failed: %v", err)
	}
	log.Info("trying to get subnets")
	subnets, err := getSubnets(log, ec2client, db.Spec.PubliclyAccessible, kubectl)
	if err != nil {
		return fmt.Errorf("unable to get subnets from instance: %v", err)

	}
	log.Info("trying to get security groups")
	sgs, err := getSGS(ec2client, kubectl)
	if err != nil {
		return fmt.Errorf("unable to get security groups from instance: %v", err)
	}

	r := k8srds.AWS{RDS: rdsclient, EC2: ec2client, Subnets: subnets, SecurityGroups: sgs}
	hostname, err := r.RestoreDatabase(db)
	if err != nil {
		return err
	}

	k := Kube{Client: kubectl}
	log.Info("Creating service", "name", db.Name, "hostname", hostname, "namespace", db.Namespace)
	err = k.CreateService(db.Namespace, hostname, db.Name)
	if err != nil {
		return err
	}

	err = updateStatus(db, databasesv1.RdsStatus{Message: "Created", State: "Created"}, crdclient, ctx, namespacedName)
	if err != nil {
		return err
	}
	log.Info("Creation of database done", "name", db.Name)
	return nil
}

func updateStatus(db *databasesv1.Rds, status databasesv1.RdsStatus, crdclient *controllers.RdsReconciler, ctx context.Context, namespacedName types.NamespacedName) (err error) {
	err = crdclient.Get(ctx, namespacedName, db)
	if err != nil {
		return
	}

	db.Status = status
	err = crdclient.Update(ctx, db)

	return
}

func getSubnets(log logr.Logger, svc *ec2.Client, public bool, kubectl *kubernetes.Clientset) ([]string, error) {
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
	log.Info("Taking subnets from node", "name", name)

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
	log.Info("trying to describe instance")
	req := svc.DescribeInstancesRequest(params)
	res, err := req.Send(ctx)
	if err != nil {
		log.Error(err, "unable to describe AWS instance")
		return nil, errors.Wrap(err, "unable to describe AWS instance")
	}
	log.Info("got instance response")

	var result []string
	if len(res.Reservations) >= 1 {
		vpcID := res.Reservations[0].Instances[0].VpcId
		for _, v := range res.Reservations[0].Instances[0].SecurityGroups {
			log.Info("Security groupid", "groupid", *v.GroupId)
		}
		log.Info("Found VPC will search for subnet in that VPC", "vpc", *vpcID)

		res := svc.DescribeSubnetsRequest(&ec2.DescribeSubnetsInput{Filters: []ec2.Filter{{Name: aws.String("vpc-id"), Values: []string{*vpcID}}}})
		subnets, err := res.Send(ctx)

		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("unable to describe subnet in VPC %v", *vpcID))
		}
		for _, sn := range subnets.Subnets {
			if *sn.MapPublicIpOnLaunch == public {
				result = append(result, *sn.SubnetId)
			} else {
				log.Info("Skipping subnet because of it's public state", "subnetId", *sn.SubnetId, "actual", *sn.MapPublicIpOnLaunch, "expecting", public)
			}
		}

	}
	log.Info("Found the follwing subnets")
	for _, v := range result {
		log.Info("Found subnet", "subnet", v)
	}
	return result, nil
}

func getSGS(svc *ec2.Client, kubectl *kubernetes.Clientset) ([]string, error) {
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
	log.Printf("Taking security groups from node %v", name)

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
	log.Println("trying to describe instance")
	req := svc.DescribeInstancesRequest(params)
	res, err := req.Send(ctx)
	if err != nil {
		log.Println(err)
		return nil, errors.Wrap(err, "unable to describe AWS instance")
	}
	log.Println("got instance response")

	var result []string
	if len(res.Reservations) >= 1 {
		for _, v := range res.Reservations[0].Instances[0].SecurityGroups {
			fmt.Println("Security groupid: ", *v.GroupId)
			result = append(result, *v.GroupId)
		}
	}

	log.Printf("Found the follwing security groups: ")
	for _, v := range result {
		log.Printf(v + " ")
	}
	return result, nil
}
