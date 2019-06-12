package rds

import (
	"context"

	databasesv1 "github.com/cloud104/kube-db/api/v1"
	controllers "github.com/cloud104/kube-db/controllers"
	"k8s.io/apimachinery/pkg/types"
)

func (a *Actuator) Reconcile(db *databasesv1.Rds, client *controllers.RdsReconciler, ctx context.Context, namespacedName types.NamespacedName) (status databasesv1.RdsStatus, err error) {
	log := a.log.WithValues("reconcilingDatabase", db.Name)
	log.Info("Start reconciling")

	currentStatus, err := a.k8srds.GetStatus(db)
	if err != nil {
		return databasesv1.NewStatus("Error Getting Status", "ERROR"), err
	}

	hasService := a.kubeClient.HasService(db.Namespace, db.Name)

	// If available and hasService: already Created and Reboted
	if currentStatus == "available" && hasService {
		log.Info("database reconciliation done, skipping")
		return databasesv1.NewStatus("Database reconciled", "CREATED"), nil
	}

	// If available and doesn't hasService, reboot before reconciling service
	if currentStatus == "available" && !hasService {
		log.Info("Rebooting database")
		err = a.k8srds.RebootDatabase(db)
		if err != nil {
			return databasesv1.NewStatus("Failed To Reboot Database", "ERROR"), err
		}

		return databasesv1.NewStatus("Database Rebooted", "WAITING"), err
	}

	// If available and doesn't has service, reconcile service
	if currentStatus == "available" && !hasService {
		log.Info("Getting endpoint")
		hostname, err := a.k8srds.GetEndpoint(db)
		if err != nil {
			return databasesv1.NewStatus("Waiting for endpoint to be available", "WAITING"), err
		}

		log.Info("Reconciling service", "name", db.Name, "hostname", hostname, "namespace", db.Namespace)
		err = a.kubeClient.ReconcileService(db.Namespace, hostname, db.Name)
		if err != nil {
			return databasesv1.NewStatus("Failing Reconciled Service", "ERROR"), err
		}

		return databasesv1.NewStatus("Reconciling Database", "WAITING"), err
	}

	// If not available and has no service, reconciliate

	// Based in the field, it creates or restores
	if db.Spec.DBSnapshotIdentifier != "" {
		log.Info("restoring")
		err = a.k8srds.RestoreDatabase(db)
	} else {
		log.Info("creating")
		log.Info("getting secret: Name", "name", db.Spec.Password.Name, "key", db.Spec.Password.Key)
		pw, err := a.kubeClient.GetSecret(db.Namespace, db.Spec.Password.Name, db.Spec.Password.Key)
		if err != nil {
			return databasesv1.NewStatus("Failing Geting Secret", "ERROR"), err
		}
		err = a.k8srds.CreateDatabase(db, pw)
	}
	if err != nil {
		return databasesv1.NewStatus("Failing Create", "ERROR"), err
	}

	log.Info("Reconciliation of database done, will wait now")
	return databasesv1.NewStatus("Reconciling Database", "WAITING"), err
}

func (a *Actuator) Delete(db *databasesv1.Rds, client *controllers.RdsReconciler, ctx context.Context, namespacedName types.NamespacedName) (status databasesv1.RdsStatus, err error) {
	log := a.log.WithValues("delete", db.Name)

	currentStatus, err := a.k8srds.GetStatus(db)
	if err != nil {
		return databasesv1.NewStatus("Error Getting Status", "ERROR"), err
	}
	hasService := a.kubeClient.HasService(db.Namespace, db.Name)

	if currentStatus == "rebooting" || currentStatus == "creating" || currentStatus == "deleting" {
		return databasesv1.NewStatus("Database not in a deletable state, will wait", "WAITING"), err
	}

	// If status pending, meaning that the database does not exist
	if currentStatus != "pending" {
		log.Info("deleting database")
		err := a.k8srds.DeleteDatabase(db)
		if err != nil {
			return databasesv1.NewStatus("ERROR Deleting", "ERROR"), err
		}

		return databasesv1.NewStatus("Deleting", "DELETING"), err
	}

	// If hasService Remove it
	if hasService {
		log.Info("deleting svc")
		err = a.kubeClient.DeleteService(db.Namespace, db.Name)
		if err != nil {
			log.Error(err, "could not delete service")
			return databasesv1.NewStatus("ERROR Deleting svc", "ERROR"), err
		}
		return databasesv1.NewStatus("Deleting", "DELETING"), err
	}

	log.Info("Deletion of database done")
	return databasesv1.NewStatus("Deleted", "DELETED"), err
}
