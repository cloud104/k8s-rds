package main

import (
	"fmt"
	"os"

	databasesv1 "github.com/cloud104/kube-db/api/v1"
	"github.com/cloud104/kube-db/controllers"
	"github.com/cloud104/kube-db/pkg/actuators/rds"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	// +kubebuilder:scaffold:imports
)

var (
	scheme   = runtime.NewScheme()
	setupLog = ctrl.Log.WithName("setup")
)

func init() {
	databasesv1.AddToScheme(scheme)
	// +kubebuilder:scaffold:scheme
}

func commandServe(c *Config) *cobra.Command {
	return &cobra.Command{
		Use:     "server",
		Short:   "...",
		Long:    ``,
		Example: "kube-db server",
		Run: func(cmd *cobra.Command, args []string) {
			if c.Provider != "aws" && c.Provider != "gcloud" {
				fmt.Fprintln(os.Stderr, fmt.Errorf("invalid provider: %s", c.Provider))
				os.Exit(2)
			}
			if err := serve(c); err != nil {
				fmt.Fprintln(os.Stderr, err)
				os.Exit(2)
			}
		},
	}
}

func serve(c *Config) (err error) {
	ctrl.SetLogger(zap.Logger(true))

	mgr, err := ctrl.NewManager(ctrl.GetConfigOrDie(), ctrl.Options{Scheme: scheme, MetricsBindAddress: c.MetricsAddr})
	if err != nil {
		setupLog.Error(err, "unable to start manager")
		return err
	}

	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		setupLog.Error(err, "unable to set up client config")
		return err
	}

	if c.Provider == "aws" {
		// Initialize Actuator
		actuator, err := rds.NewActuator(
			ctrl.Log.WithName("controllers").WithName("databases").WithName("rds").WithName("actuator"),
			cfg,
		)
		if err != nil {
			setupLog.Error(err, "unable to start actuator")
			return err
		}

		err = (&controllers.RdsReconciler{
			Client:   mgr.GetClient(),
			Log:      ctrl.Log.WithName("controllers").WithName("databases").WithName("rds").WithName("reconciler"),
			Actuator: actuator,
		}).SetupWithManager(mgr)
		if err != nil {
			setupLog.Error(err, "unable to create controller", "controller", "Rds")
			return err
		}
	}

	if c.Provider == "gcloud" {
	}

	// +kubebuilder:scaffold:builder

	setupLog.Info("starting manager")
	if err := mgr.Start(ctrl.SetupSignalHandler()); err != nil {
		setupLog.Error(err, "problem running manager")
		return err
	}

	return nil
}
