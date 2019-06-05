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
	log := r.Log.WithValues("rds", req.NamespacedName)
	instance := databasesv1.Rds{}

	if err := r.Get(ctx, req.NamespacedName, &instance); err != nil {
		log.Error(err, "Unable to fetch rds")
		if apierrs.IsNotFound(err) {
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	log.Info("Running reconcile rds", "name", instance.Name)

	// If object hasn't been deleted and doesn't have a finalizer, add one
	// Add a finalizer to newly created objects.
	if instance.ObjectMeta.DeletionTimestamp.IsZero() && !util.Contains(instance.ObjectMeta.Finalizers, databasesv1.RdsFinalizer) {
		instance.Finalizers = append(instance.Finalizers, databasesv1.RdsFinalizer)
		if err := r.Update(context.Background(), &instance); err != nil {
			log.Info("failed to add finalizer to rds", "name", instance.Name, "err", err)
			return ctrl.Result{}, err
		}

		// Since adding the finalizer updates the object return to avoid later update issues
		return ctrl.Result{}, nil
	}

	if !instance.ObjectMeta.DeletionTimestamp.IsZero() {
		// no-op if finalizer has been removed.
		if !util.Contains(instance.ObjectMeta.Finalizers, databasesv1.RdsFinalizer) {
			log.Info("reconciling rds object causes a no-op as there is no finalizer", "name", instance.Name)
			return ctrl.Result{}, nil
		}

		log.Info("reconciling rds object triggers delete", "name", instance.Name)
		if err := r.Actuator.Delete(&instance, r, ctx, req.NamespacedName); err != nil {
			log.Error(err, "Error deleting rds object", "name", instance.Name)
			return ctrl.Result{}, err
		}
		// Remove finalizer on successful deletion.
		log.Info("rds object deletion successful, removing finalizer", "name", instance.Name)
		instance.ObjectMeta.Finalizers = util.Filter(instance.ObjectMeta.Finalizers, databasesv1.RdsFinalizer)
		if err := r.Client.Update(context.Background(), &instance); err != nil {
			log.Error(err, "Error removing finalizer from rds object", "name", instance.Name)
			return ctrl.Result{}, err
		}
		return ctrl.Result{}, nil
	}

	log.Info("reconciling rds object triggers idempotent reconcile", "name", instance.Name)
	if err := r.Actuator.Reconcile(&instance, r, ctx, req.NamespacedName); err != nil {
		if requeueErr, ok := err.(*controllerError.RequeueAfterError); ok {
			log.Info("Actuator returned requeue after error", "requeueErr", requeueErr)
			return ctrl.Result{Requeue: true, RequeueAfter: requeueErr.RequeueAfter}, nil
		}
		log.Error(err, "Error reconciling rds object", "name", instance.Name)
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *RdsReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&databasesv1.Rds{}).Complete(r)
}
