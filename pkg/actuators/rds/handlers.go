package rds

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/types"

	databasesv1 "github.com/cloud104/kube-db/api/v1"
	controllers "github.com/cloud104/kube-db/controllers"
)

func (a *Actuator) handleCreateDatabase(db *databasesv1.Rds, crdclient *controllers.RdsReconciler, ctx context.Context, namespacedName types.NamespacedName) error {
	if db.Status.State == "Created" {
		a.log.Info("database already created, skipping", "name", db.Name)
		return nil
	}

	// validate dbname is only alpha numeric
	err := a.updateStatus(db, databasesv1.RdsStatus{Message: "Creating", State: "Creating"}, crdclient, ctx, namespacedName)
	if err != nil {
		return fmt.Errorf("database CRD status update failed: %v", err)
	}

	a.log.Info("getting secret: Name", "name", db.Spec.Password.Name, "key", db.Spec.Password.Key)
	pw, err := a.kubeClient.GetSecret(db.Namespace, db.Spec.Password.Name, db.Spec.Password.Key)
	if err != nil {
		return err
	}

	hostname, err := a.k8srds.CreateDatabase(db, pw)
	if err != nil {
		return err
	}

	a.log.Info("Creating service", "name", db.Name, "hostname", hostname, "namespace", db.Namespace)
	err = a.kubeClient.CreateService(db.Namespace, hostname, db.Name)
	if err != nil {
		return err
	}

	err = a.updateStatus(db, databasesv1.RdsStatus{Message: "Created", State: "Created"}, crdclient, ctx, namespacedName)
	if err != nil {
		return err
	}

	a.log.Info("Creation of database", "name", db.Name)
	return nil
}

func (a *Actuator) handleRestoreDatabase(db *databasesv1.Rds, crdclient *controllers.RdsReconciler, ctx context.Context, namespacedName types.NamespacedName) error {
	if db.Status.State == "Created" {
		a.log.Info("database already created, skipping", "name", db.Name)
		return nil
	}

	// validate dbname is only alpha numeric
	err := a.updateStatus(db, databasesv1.RdsStatus{Message: "Creating", State: "Creating"}, crdclient, ctx, namespacedName)
	if err != nil {
		return fmt.Errorf("database CRD status update failed: %v", err)
	}

	a.log.Info("Restore Database")
	hostname, err := a.k8srds.RestoreDatabase(db)
	if err != nil {
		return err
	}

	a.log.Info("Creating service", "name", db.Name, "hostname", hostname, "namespace", db.Namespace)
	err = a.kubeClient.CreateService(db.Namespace, hostname, db.Name)
	if err != nil {
		return err
	}

	err = a.updateStatus(db, databasesv1.RdsStatus{Message: "Created", State: "Created"}, crdclient, ctx, namespacedName)
	if err != nil {
		return err
	}

	a.log.Info("Creation of database done", "name", db.Name)
	return nil
}

func (a *Actuator) updateStatus(db *databasesv1.Rds, status databasesv1.RdsStatus, crdclient *controllers.RdsReconciler, ctx context.Context, namespacedName types.NamespacedName) (err error) {
	err = crdclient.Get(ctx, namespacedName, db)
	if err != nil {
		return
	}

	db.Status = status
	err = crdclient.Update(ctx, db)

	return
}
