# K8S Databases

[![Build Status](https://travis-ci.org/cloud104/kube-db.svg?branch=master)](https://travis-ci.org/sorenmat/kube-db)
[![Go Report Card](https://goreportcard.com/badge/github.com/cloud104/kube-db)](https://goreportcard.com/report/github.com/cloud104/kube-db)

A Custom Resource Definition for provisioning AWS RDS databases.

State: BETA - use with caution

## Assumptions

The node running the pod should have an instance profile that allows creation and deletion of RDS databases and Subnets.

The codes will search for the first node, and take the subnets from that node. And depending on wether or not your DB should be public, then filter them on that. If any subnets left it will attach the DB to that.

## Building

`CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o kube-db .`

## Installing

You can start the the controller with helm
```
helm upgrade kube-db ./hack/helm \
                -f ./hack/helm/values.yaml \
                --namespace=kube-db-system \
                --set image.tag="latest" \
                --set secrets.aws_access_key_id="@TODO" \
                --set secrets.aws_secret_access_key="@TODO" \
                --set image.pullSecret="@TODO" \
                --debug \
                --install
```

## Deploying

When the controller is running in the cluster you can deploy/create a new database by running `kubectl apply` on the following
file.

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: mysecret
type: Opaque
data:
  mykey: cGFzc3dvcmRvcnNvbWV0aGluZw==
---
apiVersion: databases.tks.sh/v1
kind: Rds
metadata:
  name: pgsql
spec:
  backupRetentionPeriod: 10 # days to keep backup, 0 means diable
  class: db.t2.medium # type of the db instance
  dbname: pgsql # name of the initial created database
  encrypted: true # should the database be encrypted
  engine: postgres # what engine to use postgres, mysql, aurora-postgresql etc.
  iops: 1000 # number of iops
  multiaz: true # multi AZ support
  name: pgsql # name of the database at the provider
  size: 10 # size in BG
  storageType: gp2 # type of the underlying storage
  username: postgres # Database username
  password: # link to database secret
    key: mykey # the key in the secret
    name: mysecret # the name of the secret
```

After the deploy is done you should be able to see your database via `kubectl get rds`

```shell
NAME         AGE
test-pgsql   11h
```

And on the AWS RDS page

![subnets](docs/subnet.png "DB instance subnets")

![instances](docs/instances.png "DB instance")

## Kubebuilder init

- env GOPATH=$HOME/Workspace  GO111MODULE=on kubebuilder init --domain tks.sh
- env GOPATH=$HOME/Workspace GO111MODULE=on kubebuilder create api --group databases --version v1 --kind Rds --controller=true --resource=true

# TODO

- [X] Basic RDS support
- [] Cluster support
- [] Google Cloud SQL for PostgreSQL support
- [] Local PostgreSQL support
- [] Transform rds creation into a configurable cli
- [] Azure support
- [] Make it read from a VERSION file and log
- [] Tests

## TEST

- [] Parallel running
- [] Pass parameter group
- [] Get latest snapshot when restoring
  - [] On delete check if snapshot was done correctly
- [] Delete check snapshot
- [] Create/Restore postgres
- [] Create/Restore oracle

## References

- https://github.com/cloud104/kube-db
- https://github.com/cloud104/tks-uptimerobot-controller
- https://github.com/cloud104/tks-controller
- https://itnext.io/how-to-create-a-kubernetes-custom-controller-using-client-go-f36a7a7536cc
- https://github.com/krallistic/kafka-operator
- https://github.com/cloud104/farwell-controller
- https://github.com/hossainemruz/k8s-initializer-finalizer-practice
- https://book.kubebuilder.io/quick-start.html
- https://github.com/kubernetes-sigs/cluster-api
- https://blog.golang.org/using-go-modules
- https://github.com/morvencao/kubecronjob

### Google reference

- Create database docs: https://cloud.google.com/sdk/gcloud/reference/sql/databases/create
- Create database tutorial: https://cloud.google.com/sql/docs/mysql/create-manage-databases
- Go sdk code: https://github.com/googleapis/google-cloud-go
- Go sdk godocs: https://godoc.org/cloud.google.com/go
- Cli docs: https://cloud.google.com/sdk/gcloud/reference/sql/databases/create
