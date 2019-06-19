package client

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/rds"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/pkg/errors"

	databasesv1 "github.com/cloud104/kube-db/api/v1"
)

// AWS ...
type AWS struct {
	RDS            *rds.Client
	EC2            *ec2.Client
	Subnets        []string
	SecurityGroups []string
}

// CreateDatabase ...
func (a *AWS) CreateDatabase(db *databasesv1.Rds, password string) error {
	ctx := context.Background()
	log.Println("Trying to find the correct subnets")
	subnetName, err := a.ensureSubnets(db)
	if err != nil {
		return err
	}

	input := convertSpecToInputCreate(db, subnetName, a.SecurityGroups, password)

	// search for the instance
	log.Printf("Trying to find db instance %v\n", db.Spec.DBName)
	k := &rds.DescribeDBInstancesInput{DBInstanceIdentifier: input.DBInstanceIdentifier}
	res := a.RDS.DescribeDBInstancesRequest(k)
	_, err = res.Send(ctx)
	if err != nil && err.Error() != rds.ErrCodeDBInstanceNotFoundFault {
		log.Printf("DB instance %v not found trying to create it\n", db.Spec.DBName)
		// seems like we didn't find a database with this name, let's create on
		res := a.RDS.CreateDBInstanceRequest(input)
		_, err = res.Send(ctx)
		if err != nil {
			return errors.Wrap(err, "CreateDBInstance")
		}
	} else if err != nil {
		return errors.Wrap(err, fmt.Sprintf("wasn't able to describe the db instance with id %v", input.DBInstanceIdentifier))
	}

	return nil
}

// RestoreDatabase ...
func (a *AWS) RestoreDatabase(db *databasesv1.Rds) error {
	ctx := context.Background()
	log.Println("Trying to find the correct subnets")
	subnetName, err := a.ensureSubnets(db)
	if err != nil {
		return err
	}

	var securityGroups []string
	if len(db.Spec.VpcSecurityGroupIds) > 0 {
		securityGroups = append(securityGroups, db.Spec.VpcSecurityGroupIds)
	} else {
		securityGroups = a.SecurityGroups
	}

	input := convertSpecToInputRestore(db, subnetName, securityGroups)

	fmt.Printf("%v\n", subnetName)
	fmt.Printf("%v\n", a.SecurityGroups)
	fmt.Printf("%v\n", input)
	//panic("Nope")

	// search for the instance
	log.Printf("Trying to find db instance %v\n", db.Spec.DBName)
	k := &rds.DescribeDBInstancesInput{DBInstanceIdentifier: input.DBInstanceIdentifier}
	res := a.RDS.DescribeDBInstancesRequest(k)
	_, err = res.Send(ctx)
	if err != nil && err.Error() != rds.ErrCodeDBInstanceNotFoundFault {
		log.Printf("DB instance %v not found trying to create it\n", db.Spec.DBName)
		// seems like we didn't find a database with this name, let's create on
		res := a.RDS.RestoreDBInstanceFromDBSnapshotRequest(input)
		_, err = res.Send(ctx)
		if err != nil {
			return errors.Wrap(err, "RestoreDBInstance")
		}

	} else if err != nil {
		return errors.Wrap(err, fmt.Sprintf("wasn't able to describe the db instance with id %v", input.DBInstanceIdentifier))
	}

	return nil
}

// Get Endpoint
func (a *AWS) GetEndpoint(db *databasesv1.Rds) (string, error) {
	log.Println("Trying to find the correct subnets")
	subnetName, err := a.ensureSubnets(db)
	if err != nil {
		return "", err
	}

	var securityGroups []string
	if len(db.Spec.VpcSecurityGroupIds) > 0 {
		securityGroups = append(securityGroups, db.Spec.VpcSecurityGroupIds)
	} else {
		securityGroups = a.SecurityGroups
	}

	input := convertSpecToInputRestore(db, subnetName, securityGroups)

	// Get the newly created database so we can get the endpoint
	dbHostname, err := getEndpoint(input.DBInstanceIdentifier, a.RDS)
	if err != nil {
		return "", err
	}

	return dbHostname, nil
}

// https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/Overview.DBInstance.Status.html
// GetStatus
// available
// creating
// deleting
// rebooting
func (a *AWS) GetStatus(db *databasesv1.Rds) (string, error) {
	instance, err := a.getInstance(db)
	if err != nil {
		if err.Error() == "DBInstanceNotFound" {
			return "pending", nil
		}
		return "error", err
	}
	return *instance.DBInstanceStatus, nil
}

func (a *AWS) PendingReboot(db *databasesv1.Rds) (bool, error) {
	_, err := a.getInstance(db)
	if err != nil {
		if err.Error() == "DBInstanceNotFound" {
			return false, nil
		}
		// If error, send rebooted and error
		return false, err
	}

	// Get status and return if rebooted
	return false, nil
}

func (a *AWS) getInstance(db *databasesv1.Rds) (*rds.DBInstance, error) {
	subnetName, err := a.ensureSubnets(db)
	if err != nil {
		return nil, err
	}

	var securityGroups []string
	if len(db.Spec.VpcSecurityGroupIds) > 0 {
		securityGroups = append(securityGroups, db.Spec.VpcSecurityGroupIds)
	} else {
		securityGroups = a.SecurityGroups
	}

	input := convertSpecToInputRestore(db, subnetName, securityGroups)
	instance, err := a.RDS.DescribeDBInstancesRequest(&rds.DescribeDBInstancesInput{DBInstanceIdentifier: input.DBInstanceIdentifier}).Send(context.Background())
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			return nil, fmt.Errorf(awsErr.Code())
		}
		return nil, err
	}
	if len(instance.DescribeDBInstancesOutput.DBInstances) <= 0 {
		return nil, fmt.Errorf("InstancesOutputEmpty")
	}
	return &instance.DescribeDBInstancesOutput.DBInstances[0], nil
}

// RebootDatabase
func (a *AWS) RebootDatabase(db *databasesv1.Rds) error {
	ctx := context.Background()
	log.Println("Trying to find the correct subnets")
	subnetName, err := a.ensureSubnets(db)
	if err != nil {
		return err
	}

	var securityGroups []string
	if len(db.Spec.VpcSecurityGroupIds) > 0 {
		securityGroups = append(securityGroups, db.Spec.VpcSecurityGroupIds)
	} else {
		securityGroups = a.SecurityGroups
	}

	input := convertSpecToInputRestore(db, subnetName, securityGroups)

	log.Printf("Reboot instance after restoring %v to apply params\n", *input.DBInstanceIdentifier)
	r := &rds.RebootDBInstanceInput{DBInstanceIdentifier: input.DBInstanceIdentifier}
	_, err = a.RDS.RebootDBInstanceRequest(r).Send(ctx)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("something went wrong in RebootDBInstanceRequest for db instance %v", input.DBInstanceIdentifier))
	}

	return nil
}

// DeleteDatabase ...
func (a *AWS) DeleteDatabase(db *databasesv1.Rds) error {
	ctx := context.Background()
	// delete the database instance
	svc := a.RDS
	dbName := db.Name
	t := time.Now()
	finalSnapshotIdentifier := fmt.Sprintf("kube-db-%v-%v", dbName, t.Format("20060102150405"))

	log.Printf("DB: %v to be deleted, with finalSnapshot: %v\n", dbName, finalSnapshotIdentifier)
	res := svc.DeleteDBInstanceRequest(&rds.DeleteDBInstanceInput{
		DBInstanceIdentifier:      aws.String(dbName),
		FinalDBSnapshotIdentifier: aws.String(finalSnapshotIdentifier),
	})
	_, err := res.Send(ctx)
	if err != nil {
		if awsErr, ok := err.(awserr.Error); ok {
			if awsErr.Code() == "DBInstanceNotFound" {
				return nil
			}
		}

		log.Println(errors.Wrap(err, fmt.Sprintf("unable to delete database %v", dbName)))
		return err
	}

	// delete subnetgroup only for creation process
	//if db.Spec.DBSnapshotIdentifier == "" {
	//	log.Printf("SubnetGroup %v to be deleted\n", db.Spec.DBSubnetGroupName)
	//	a.deleteSubnetGroup(db)
	//}

	return nil
}

// deleteSubnetGroup ...
func (a *AWS) deleteSubnetGroup(db *databasesv1.Rds) {
	ctx := context.Background()
	svc := a.RDS
	// delete the subnet group attached to the instance
	subnetName := db.Spec.DBSubnetGroupName
	dres := svc.DeleteDBSubnetGroupRequest(&rds.DeleteDBSubnetGroupInput{DBSubnetGroupName: aws.String(subnetName)})
	_, err := dres.Send(ctx)
	if err != nil {
		log.Println(errors.Wrap(err, fmt.Sprintf("unable to delete subnet %v", subnetName)))
	} else {
		log.Println("Deleted DBSubnet group: ", subnetName)
	}
}

func (a *AWS) ensureSubnets(db *databasesv1.Rds) (string, error) {
	ctx := context.Background()
	if len(a.Subnets) == 0 {
		log.Println("No subnets passed, will try to find a default")
	}
	subnetDescription := "subnet kube-db"
	subnetName := db.Spec.DBSubnetGroupName

	svc := a.RDS

	sf := &rds.DescribeDBSubnetGroupsInput{DBSubnetGroupName: aws.String(subnetName)}
	res := svc.DescribeDBSubnetGroupsRequest(sf)
	_, err := res.Send(ctx)
	log.Println("Subnets:", a.Subnets)
	if err != nil {
		// assume we didn't find it..
		subnet := &rds.CreateDBSubnetGroupInput{
			DBSubnetGroupDescription: aws.String(subnetDescription),
			DBSubnetGroupName:        aws.String(subnetName),
			SubnetIds:                a.Subnets,
			Tags:                     []rds.Tag{{Key: aws.String("DBName"), Value: aws.String(db.Spec.DBName)}},
		}
		res := svc.CreateDBSubnetGroupRequest(subnet)
		_, err := res.Send(ctx)
		if err != nil {
			return "", errors.Wrap(err, "CreateDBSubnetGroup")
		}
	} else {
		log.Printf("Moving on seems like %v exists", subnetName)
	}
	return subnetName, nil
}

func getEndpoint(dbName *string, svc *rds.Client) (string, error) {
	instance, err := svc.
		DescribeDBInstancesRequest(&rds.DescribeDBInstancesInput{DBInstanceIdentifier: dbName}).
		Send(context.Background())

	if err != nil || len(instance.DBInstances) == 0 {
		return "", fmt.Errorf("wasn't able to describe the db instance with id %v", dbName)
	}

	rdsdb := instance.DBInstances[0]

	if rdsdb.Endpoint == nil {
		return "", fmt.Errorf("endpoint is not available yet")
	}

	dbHostname := *rdsdb.Endpoint.Address
	return dbHostname, nil
}

func convertSpecToInputRestore(v *databasesv1.Rds, subnetName string, securityGroups []string) *rds.RestoreDBInstanceFromDBSnapshotInput {
	return &rds.RestoreDBInstanceFromDBSnapshotInput{
		AvailabilityZone:     aws.String(v.Spec.AvailabilityZone),
		CopyTagsToSnapshot:   aws.Bool(v.Spec.CopyTagsToSnapshot),
		DBInstanceClass:      aws.String(v.Spec.Class),
		DBInstanceIdentifier: aws.String(v.Name),
		DBName:               aws.String(v.Spec.DBName),
		DBParameterGroupName: aws.String(v.Spec.DBParameterGroupName),
		DBSnapshotIdentifier: aws.String(v.Spec.DBSnapshotIdentifier),
		DBSubnetGroupName:    aws.String(subnetName),
		Engine:               aws.String(v.Spec.Engine),
		LicenseModel:         aws.String("license-included"),
		MultiAZ:              aws.Bool(v.Spec.MultiAZ),
		PubliclyAccessible:   aws.Bool(v.Spec.PubliclyAccessible),
		StorageType:          aws.String(v.Spec.StorageType),
		Tags:                 createTags(v.Spec.Tags),
		VpcSecurityGroupIds:  securityGroups,
	}
}

func convertSpecToInputCreate(v *databasesv1.Rds, subnetName string, securityGroups []string, password string) *rds.CreateDBInstanceInput {
	input := &rds.CreateDBInstanceInput{
		AllocatedStorage:      aws.Int64(v.Spec.Size),
		AvailabilityZone:      aws.String(v.Spec.AvailabilityZone),
		BackupRetentionPeriod: aws.Int64(v.Spec.BackupRetentionPeriod),
		DBInstanceClass:       aws.String(v.Spec.Class),
		DBInstanceIdentifier:  aws.String(v.Name),
		DBName:                aws.String(v.Spec.DBName),
		DBParameterGroupName:  aws.String(v.Spec.DBParameterGroupName),
		DBSubnetGroupName:     aws.String(subnetName),
		Engine:                aws.String(v.Spec.Engine),
		EngineVersion:         aws.String(v.Spec.EngineVersion),
		MasterUserPassword:    aws.String(password),
		MasterUsername:        aws.String(v.Spec.Username),
		MultiAZ:               aws.Bool(v.Spec.MultiAZ),
		PubliclyAccessible:    aws.Bool(v.Spec.PubliclyAccessible),
		StorageEncrypted:      aws.Bool(v.Spec.StorageEncrypted),
		Tags:                  createTags(v.Spec.Tags),
		VpcSecurityGroupIds:   securityGroups,
	}
	if v.Spec.StorageType != "" {
		input.StorageType = aws.String(v.Spec.StorageType)
	}
	if v.Spec.Iops > 0 {
		input.Iops = aws.Int64(v.Spec.Iops)
	}
	return input
}

func createTags(t map[string]string) []rds.Tag {
	var tags []rds.Tag

	for k, v := range t {
		tags = append(tags, rds.Tag{Key: aws.String(k), Value: aws.String(v)})
	}

	return tags
}
