package controllers

import (
	"context"

	databasesv1 "github.com/cloud104/kube-db/api/v1"
	"k8s.io/apimachinery/pkg/types"
)

//go:generate mockgen -package=mocks -destination=mocks/actuator_mock.go -source=actuator.go Actuator
type Actuator interface {
	//
	Reconcile(*databasesv1.Rds, *RdsReconciler, context.Context, types.NamespacedName) error
	//
	Delete(*databasesv1.Rds, *RdsReconciler, context.Context, types.NamespacedName) error
}
