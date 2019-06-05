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
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	databasesv1 "github.com/cloud104/kube-db/api/v1"
	controllers "github.com/cloud104/kube-db/controllers"
	k8srds "github.com/cloud104/kube-db/pkg/actuators/rds/client"
)

const Failed = "Failed"
const dryRunDelete = false

type Actuator struct {
	log       logr.Logger
	ec2client *ec2.Client
	rdsclient *rds.Client
	kubectl   *kubernetes.Clientset
}

func NewActuator(log logr.Logger, config *rest.Config) (*Actuator, error) {
	kubectl, err := getKubectl(config)
	if err != nil {
		return nil, err
	}

	awsConfig, err := configClient(kubectl)
	if err != nil {
		return nil, err
	}

	return &Actuator{
		log:       log,
		ec2client: ec2.New(awsConfig),
		rdsclient: rds.New(awsConfig),
		kubectl:   kubectl,
	}, nil
}

func (a *Actuator) Reconcile(instance *databasesv1.Rds, client *controllers.RdsReconciler, ctx context.Context, namespacedName types.NamespacedName) (err error) {
	log := a.log.WithValues("rds", instance.Name)
	log.Info("Starting reconcile")

	// Based in the field, it creates or restores
	if instance.Spec.DBSnapshotIdentifier != "" {
		log.Info("restoring")
		err = handleRestoreDatabase(log, instance, a.ec2client, a.rdsclient, client, a.kubectl, ctx, namespacedName)
	} else {
		log.Info("creating")
		err = handleCreateDatabase(log, instance, a.ec2client, a.rdsclient, client, a.kubectl, ctx, namespacedName)
	}

	if err != nil {
		log.Error(err, "database creation/restore failed")
		err = updateStatus(instance, databasesv1.RdsStatus{Message: fmt.Sprintf("%v", err), State: Failed}, client, ctx, namespacedName)
		if err != nil {
			log.Error(err, "database CRD status update failed")
		}
	}

	return
}

func (a *Actuator) Delete(instance *databasesv1.Rds, client *controllers.RdsReconciler, ctx context.Context, namespacedName types.NamespacedName) (err error) {
	log := a.log.WithValues("rds", instance.Name)

	log.Info("deleting database", "name", instance.Name)
	r := k8srds.AWS{RDS: a.rdsclient}
	if !dryRunDelete {
		err := r.DeleteDatabase(instance)
		if err != nil {
			return err
		}
	}

	k := Kube{Client: a.kubectl}
	err = k.DeleteService(instance.Namespace, instance.Name)
	if err != nil {
		log.Error(err, "could not delete service")
		return err
	}

	log.Info("Deletion of database done", "name", instance.Name)

	return nil
}

func getKubectl(config *rest.Config) (*kubernetes.Clientset, error) {
	kubectl, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return kubectl, nil
}

func configClient(kubectl *kubernetes.Clientset) (aws.Config, error) {
	nodes, err := kubectl.CoreV1().Nodes().List(metav1.ListOptions{})
	if err != nil {
		return aws.Config{}, errors.Wrap(err, "unable to get nodes")
	}
	name := ""
	region := ""

	if len(nodes.Items) > 0 {
		// take the first one, we assume that all nodes are created in the same VPC
		name = nodes.Items[0].Name
		region = nodes.Items[0].Labels["failure-domain.beta.kubernetes.io/region"]
	} else {
		return aws.Config{}, fmt.Errorf("unable to find any nodes in the cluster")
	}
	// @TODO: Use log
	fmt.Printf("Found node with ID: %v in region %v", name, region)

	cfg, err := external.LoadDefaultAWSConfig()
	if err != nil {
		panic("unable to load SDK config, " + err.Error())
	}

	// Set the AWS Region that the service clients should use
	cfg.Region = region
	cfg.HTTPClient.Timeout = 5 * time.Second
	return cfg, nil
}
