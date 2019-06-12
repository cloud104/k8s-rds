/*

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"

	"github.com/go-logr/logr"
	apierrs "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	controllerError "sigs.k8s.io/cluster-api/pkg/controller/error"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	databasesv1 "github.com/cloud104/kube-db/api/v1"
	util "github.com/cloud104/kube-db/pkg/util"
)

// RdsReconciler reconciles a Rds object
type RdsReconciler struct {
	client.Client
	Log logr.Logger
	Actuator
}

// +kubebuilder:rbac:groups=databases.tks.sh,resources=rds,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=databases.tks.sh,resources=rds/status,verbs=get;update;patch
// +kubebuilder:rbac:groups="",resources=namespaces,verbs=get;list;watch
// +kubebuilder:rbac:groups="",resources=nodes,verbs=list;watch
// +kubebuilder:rbac:groups="",resources=secrets,verbs=get
// +kubebuilder:rbac:groups="",resources=services,verbs=get;create;update;delete
func (r *RdsReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("namespacedName", req.NamespacedName)
	instance := databasesv1.Rds{}

	log.Info("Running reconcile rds")

	// Get record from kubernetes api
	if err := r.Get(ctx, req.NamespacedName, &instance); err != nil {
		if apierrs.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		log.Error(err, "No record found")
		return ctrl.Result{}, err
	}

	// If object hasn't been deleted and doesn't have a finalizer, add one
	// Add a finalizer to newly created objects.
	if instance.ObjectMeta.DeletionTimestamp.IsZero() && !util.Contains(instance.ObjectMeta.Finalizers, databasesv1.RdsFinalizer) {
		instance.Finalizers = append(instance.Finalizers, databasesv1.RdsFinalizer)
		if err := r.Update(context.Background(), &instance); err != nil {
			log.Error(err, "failed to add finalizer to rds")
			return ctrl.Result{}, err
		}

		// Since adding the finalizer updates the object return to avoid later update issues
		return ctrl.Result{}, nil
	}

	// Delete
	if !instance.ObjectMeta.DeletionTimestamp.IsZero() {
		// no-op if finalizer has been removed.
		if !util.Contains(instance.ObjectMeta.Finalizers, databasesv1.RdsFinalizer) {
			log.Info("reconciling rds object causes a no-op as there is no finalizer")
			return ctrl.Result{}, nil
		}

		// Call actuator delete
		log.Info("reconciling rds object triggers delete")
		status, err := r.Actuator.Delete(&instance, r, ctx, req.NamespacedName)
		if err != nil {
			log.Error(err, "Error deleting rds object")
			return ctrl.Result{}, err
		}

		// Update Status
		if err := r.updateStatus(&instance, status, context.Background(), req.NamespacedName); err != nil {
			log.Info("Update Status Failed", "error", err)
			return ctrl.Result{}, nil
		}

		if status.State != "DELETED" {
			log.Info("Deleting, requeueing", "status", status)
			return ctrl.Result{Requeue: true, RequeueAfter: 100}, nil
		}

		// Remove finalizer on successful deletion.
		log.Info("rds object deletion successful, removing finalizer")
		instance.ObjectMeta.Finalizers = util.Filter(instance.ObjectMeta.Finalizers, databasesv1.RdsFinalizer)
		if err := r.Client.Update(context.Background(), &instance); err != nil {
			log.Error(err, "Error removing finalizer from rds object")
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	}

	// Reconcile
	log.Info("reconciling rds object triggers idempotent reconcile")
	status, err := r.Actuator.Reconcile(&instance, r, ctx, req.NamespacedName)
	// If Error, handle error
	if err != nil {
		if requeueErr, ok := err.(*controllerError.RequeueAfterError); ok {
			log.Error(requeueErr, "Actuator returned requeue after error")
			return ctrl.Result{Requeue: true, RequeueAfter: requeueErr.RequeueAfter}, nil
		}

		log.Error(err, "Error reconciling rds object")
		return ctrl.Result{}, err
	}

	// Update Status
	if err := r.updateStatus(&instance, status, context.Background(), req.NamespacedName); err != nil {
		log.Info("Update Status Failed", "error", err)
		return ctrl.Result{}, nil
	}

	// If state is diferent (WAITING, ERROR) from CREATED requeue
	if status.State != "CREATED" {
		log.Info("Creating, requeueing", "status", status)
		return ctrl.Result{Requeue: true, RequeueAfter: 100}, nil
	}

	// If CREATED return done
	return ctrl.Result{}, nil
}

func (r *RdsReconciler) updateStatus(db *databasesv1.Rds, status databasesv1.RdsStatus, ctx context.Context, namespacedName types.NamespacedName) (err error) {
	err = r.Get(ctx, namespacedName, db)
	if err != nil {
		return
	}
	db.Status = status
	err = r.Update(ctx, db)
	return
}

func (r *RdsReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&databasesv1.Rds{}).Complete(r)
}

// func (r *RdsReconciler) addFinalizer(db *databasesv1.Rds) {
// 	finalizers := sets.NewString(db.Finalizers...)
// 	finalizers.Insert(databasesv1.RdsFinalizer)
// 	db.Finalizers = finalizers.List()
// }

// func (r *RdsReconciler) removeFinalizer(db *databasesv1.Rds) {
// 	finalizers := sets.NewString(db.Finalizers...)
// 	finalizers.Delete(databasesv1.RdsFinalizer)
// 	db.Finalizers = finalizers.List()
// }
