package rds

import (
	"context"
	"fmt"

	databasesv1 "github.com/cloud104/kube-db/api/v1"
	controllers "github.com/cloud104/kube-db/controllers"
	"k8s.io/apimachinery/pkg/types"
)

func (a *Actuator) Reconcile(instance *databasesv1.Rds, client *controllers.RdsReconciler, ctx context.Context, namespacedName types.NamespacedName) (err error, status databasesv1.RdsStatus) {
	log := a.log.WithValues("reconcile", instance.Name)
	log.Info("Starting reconcile")

	// Based in the field, it creates or restores
	if instance.Spec.DBSnapshotIdentifier != "" {
		log.Info("restoring")
		err = a.handleRestoreDatabase(instance, client, ctx, namespacedName)
	} else {
		log.Info("creating")
		err = a.handleCreateDatabase(instance, client, ctx, namespacedName)
	}

	if err != nil {
		log.Error(err, "database creation/restore failed")
		err = a.updateStatus(instance, databasesv1.RdsStatus{Message: fmt.Sprintf("%v", err), State: Failed}, client, ctx, namespacedName)
		if err != nil {
			log.Error(err, "database CRD status update failed")
		}
	}

	return
}
