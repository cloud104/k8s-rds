package rds

import (
	"context"

	databasesv1 "github.com/cloud104/kube-db/api/v1"
	controllers "github.com/cloud104/kube-db/controllers"
	"github.com/k0kubun/pp"
	"k8s.io/apimachinery/pkg/types"
)

func (a *Actuator) Reconcile(db *databasesv1.Rds, client *controllers.RdsReconciler, ctx context.Context, namespacedName types.NamespacedName) (status databasesv1.RdsStatus, err error) {
	log := a.log.WithValues("reconcilingDatabase", db.Name)
	log.Info("Start reconciling")

	// Get database current status
	currentStatus, err := a.k8srds.GetStatus(db)
	if err != nil {
		return databasesv1.NewStatus(err.Error(), "error"), err
	}

	// Get database pendingReboot state
	pendingReboot, err := a.k8srds.PendingReboot(db)
	if err != nil {
		pp.Println(err)
		return databasesv1.NewStatus(err.Error(), "error"), err
	}

	// Get service current state
	hasService := a.kubeClient.HasService(db.Namespace, db.Name)

	// AVAILABLE, SKIP
	// If AVAILABLE and HAS_SERVICE: nothing to do, already Created and Reboted
	if currentStatus == "available" && hasService && pendingReboot {
		log.Info("database reconciliation done, skipping")
		return databasesv1.NewStatus("Database reconciled", currentStatus), nil
	}

	// // REBOOT
	// // If AVAILABLE and HAS_NO_SERVICE: reboot before reconciling service
	// if currentStatus == "available" && !pendingReboot {
	// 	log.Info("Rebooting database")
	// 	err = a.k8srds.RebootDatabase(db)
	// 	if err != nil {
	// 		return databasesv1.NewStatus("Failed To Reboot Database", currentStatus), err
	// 	}
	// 	return databasesv1.NewStatus("Database PendingReboot", currentStatus), err
	// }

	// SERVICE
	// If NO_SERVICE: Create service
	if currentStatus == "available" && !hasService && !pendingReboot {
		log.Info("Getting endpoint")
		hostname, err := a.k8srds.GetEndpoint(db)
		if err != nil {
			return databasesv1.NewStatus("Waiting for endpoint to be available", currentStatus), err
		}

		log.Info("Reconciling service", "name", db.Name, "hostname", hostname, "namespace", db.Namespace)
		err = a.kubeClient.ReconcileService(db.Namespace, hostname, db.Name)
		if err != nil {
			return databasesv1.NewStatus("Failing Reconciled Service", currentStatus), err
		}

		return databasesv1.NewStatus("Reconciling Database", currentStatus), err
	}

	// If went throw all validations and arrived here with status diferent  from pending, return
	if currentStatus != "pending" {
		return databasesv1.NewStatus("Database not in a reconcilable state, will wait", currentStatus), nil
	}

	// If pending and has no service, reconciliate

	// Based in the field, it creates or restores
	if db.Spec.DBSnapshotIdentifier != "" {
		log.Info("restoring")
		err = a.k8srds.RestoreDatabase(db)
	} else {
		log.Info("creating")
		log.Info("getting secret: Name", "name", db.Spec.Password.Name, "key", db.Spec.Password.Key)
		pw, err := a.kubeClient.GetSecret(db.Namespace, db.Spec.Password.Name, db.Spec.Password.Key)
		if err != nil {
			return databasesv1.NewStatus("Failing Geting Secret", currentStatus), err
		}
		err = a.k8srds.CreateDatabase(db, pw)
	}
	if err != nil {
		return databasesv1.NewStatus(err.Error(), currentStatus), err
	}

	log.Info("Reconciliation of database done, will wait now")
	return databasesv1.NewStatus("Reconciling Database", currentStatus), err
}

func (a *Actuator) Delete(db *databasesv1.Rds, client *controllers.RdsReconciler, ctx context.Context, namespacedName types.NamespacedName) (status databasesv1.RdsStatus, err error) {
	log := a.log.WithValues("delete", db.Name)

	currentStatus, err := a.k8srds.GetStatus(db)
	if err != nil {
		return databasesv1.NewStatus("Error Getting Status", currentStatus), err
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
			return databasesv1.NewStatus(err.Error(), currentStatus), err
		}

		return databasesv1.NewStatus("Deleting", currentStatus), err
	}

	// If hasService Remove it
	if hasService {
		log.Info("deleting svc")
		err = a.kubeClient.DeleteService(db.Namespace, db.Name)
		if err != nil {
			log.Error(err, "could not delete service")
			return databasesv1.NewStatus("ERROR Deleting svc", currentStatus), err
		}
		return databasesv1.NewStatus("Deleting", currentStatus), err
	}

	log.Info("Deletion of database done")
	return databasesv1.NewStatus("Deleted", currentStatus), err
}
