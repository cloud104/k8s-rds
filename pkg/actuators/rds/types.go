package rds

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	databasesv1 "github.com/cloud104/kube-db/api/v1"
	controllers "github.com/cloud104/kube-db/controllers"
	k8srds "github.com/cloud104/kube-db/pkg/actuators/rds/client"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
)

type Actuator struct {
	log        logr.Logger
	ec2client  *ec2.Client
	rdsclient  *rds.Client
	kubeClient *Kube
	k8srds     *k8srds.AWS
}

type Kube struct {
	Client *kubernetes.Clientset
}

type Params struct {
	client         *controllers.RdsReconciler
	ctx            context.Context
	instance       *databasesv1.Rds
	namespacedName types.NamespacedName
}
