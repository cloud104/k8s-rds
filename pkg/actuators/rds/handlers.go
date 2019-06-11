package rds

import (
	databasesv1 "github.com/cloud104/kube-db/api/v1"
	"github.com/k0kubun/pp"
)

// @TODO: change db state check for actual cloud request to verify state
func (a *Actuator) handleCreateDatabase(db *databasesv1.Rds) (status databasesv1.RdsStatus, err error) {
	log := a.log.WithValues("createDatabase", db.Name)
	log.Info("Start creating")

	// @TODO: Maybe update later
	// If status CREATED, there's nothing to do
	if db.Is("CREATED") {
		log.Info("database already created, skipping")
		return databasesv1.NewStatus("Database Created", "CREATED"), nil
	}

	// If status WAITING:
	// Check if done, if done:
	// Create endpoint, then:
	// Return CREATED
	if db.Is("WAITING") {
		log.Info("Getting endpoint")
		hostname, err := a.k8srds.GetEndpoint(db)
		if err != nil {
			return databasesv1.NewStatus("Waiting for endpoint to be available", "WAITING"), err
		}

		log.Info("Creating service", "name", db.Name, "hostname", hostname, "namespace", db.Namespace)
		err = a.kubeClient.ReconcileService(db.Namespace, hostname, db.Name)
		if err != nil {
			return databasesv1.NewStatus("Failing Create Service", "ERROR"), err
		}

		log.Info("Rebooting database")
		err = a.k8srds.RebootDatabase(db)
		if err != nil {
			return databasesv1.NewStatus("Failed To Reboot Database", "ERROR"), err
		}

		return databasesv1.NewStatus("Database Created", "CREATED"), err
	}

	// If not CREATED and not WAITING, Create
	log.Info("getting secret: Name", "name", db.Spec.Password.Name, "key", db.Spec.Password.Key)
	pw, err := a.kubeClient.GetSecret(db.Namespace, db.Spec.Password.Name, db.Spec.Password.Key)
	if err != nil {
		return databasesv1.NewStatus("Failing Geting Secret", "ERROR"), err
	}

	log.Info("Create")
	err = a.k8srds.CreateDatabase(db, pw)
	if err != nil {
		pp.Println(err)
		return databasesv1.NewStatus("Failing Create", "ERROR"), err
	}

	log.Info("Creation of database done, will wait now")
	return databasesv1.NewStatus("Creating Database", "WAITING"), err
}

func (a *Actuator) handleRestoreDatabase(db *databasesv1.Rds) (status databasesv1.RdsStatus, err error) {
	log := a.log.WithValues("restoreDatabase", db.Name)
	log.Info("Starting restore")

	// @TODO: Maybe update if created, later
	if db.Is("CREATED") {
		log.Info("database already created, skipping")
		return databasesv1.RdsStatus{Message: "Created", State: "Created"}, nil
	}

	log.Info("Restoring Database")
	err = a.k8srds.RestoreDatabase(db)
	if err != nil {
		return databasesv1.RdsStatus{Message: "Failing Restore", State: "Failing"}, err
	}

	// log.Info("Creating service", "name", db.Name, "hostname", hostname, "namespace", db.Namespace)
	// err = a.kubeClient.ReconcileService(db.Namespace, hostname, db.Name)
	// if err != nil {
	// 	return databasesv1.RdsStatus{Message: "Failing Create Service", State: "Failing"}, err
	// }

	log.Info("Restoring database done")
	return databasesv1.RdsStatus{Message: "Created", State: "Created"}, nil
}
