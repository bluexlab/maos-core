// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.27.0

package dbsqlc

import (
	"database/sql/driver"
	"fmt"
)

type ActorRole string

const (
	ActorRoleAgent   ActorRole = "agent"
	ActorRoleService ActorRole = "service"
	ActorRolePortal  ActorRole = "portal"
	ActorRoleUser    ActorRole = "user"
	ActorRoleOther   ActorRole = "other"
)

func (e *ActorRole) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = ActorRole(s)
	case string:
		*e = ActorRole(s)
	default:
		return fmt.Errorf("unsupported scan type for ActorRole: %T", src)
	}
	return nil
}

type NullActorRole struct {
	ActorRole ActorRole
	Valid     bool // Valid is true if ActorRole is not NULL
}

// Scan implements the Scanner interface.
func (ns *NullActorRole) Scan(value interface{}) error {
	if value == nil {
		ns.ActorRole, ns.Valid = "", false
		return nil
	}
	ns.Valid = true
	return ns.ActorRole.Scan(value)
}

// Value implements the driver Valuer interface.
func (ns NullActorRole) Value() (driver.Value, error) {
	if !ns.Valid {
		return nil, nil
	}
	return string(ns.ActorRole), nil
}

type DeploymentStatus string

const (
	DeploymentStatusDraft     DeploymentStatus = "draft"
	DeploymentStatusReviewing DeploymentStatus = "reviewing"
	DeploymentStatusApproved  DeploymentStatus = "approved"
	DeploymentStatusDeploying DeploymentStatus = "deploying"
	DeploymentStatusDeployed  DeploymentStatus = "deployed"
	DeploymentStatusFailed    DeploymentStatus = "failed"
	DeploymentStatusRejected  DeploymentStatus = "rejected"
	DeploymentStatusRetired   DeploymentStatus = "retired"
	DeploymentStatusCancelled DeploymentStatus = "cancelled"
)

func (e *DeploymentStatus) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = DeploymentStatus(s)
	case string:
		*e = DeploymentStatus(s)
	default:
		return fmt.Errorf("unsupported scan type for DeploymentStatus: %T", src)
	}
	return nil
}

type NullDeploymentStatus struct {
	DeploymentStatus DeploymentStatus
	Valid            bool // Valid is true if DeploymentStatus is not NULL
}

// Scan implements the Scanner interface.
func (ns *NullDeploymentStatus) Scan(value interface{}) error {
	if value == nil {
		ns.DeploymentStatus, ns.Valid = "", false
		return nil
	}
	ns.Valid = true
	return ns.DeploymentStatus.Scan(value)
}

// Value implements the driver Valuer interface.
func (ns NullDeploymentStatus) Value() (driver.Value, error) {
	if !ns.Valid {
		return nil, nil
	}
	return string(ns.DeploymentStatus), nil
}

type InvocationState string

const (
	InvocationStateAvailable InvocationState = "available"
	InvocationStateCancelled InvocationState = "cancelled"
	InvocationStateCompleted InvocationState = "completed"
	InvocationStateDiscarded InvocationState = "discarded"
	InvocationStateRunning   InvocationState = "running"
)

func (e *InvocationState) Scan(src interface{}) error {
	switch s := src.(type) {
	case []byte:
		*e = InvocationState(s)
	case string:
		*e = InvocationState(s)
	default:
		return fmt.Errorf("unsupported scan type for InvocationState: %T", src)
	}
	return nil
}

type NullInvocationState struct {
	InvocationState InvocationState
	Valid           bool // Valid is true if InvocationState is not NULL
}

// Scan implements the Scanner interface.
func (ns *NullInvocationState) Scan(value interface{}) error {
	if value == nil {
		ns.InvocationState, ns.Valid = "", false
		return nil
	}
	ns.Valid = true
	return ns.InvocationState.Scan(value)
}

// Value implements the driver Valuer interface.
func (ns NullInvocationState) Value() (driver.Value, error) {
	if !ns.Valid {
		return nil, nil
	}
	return string(ns.InvocationState), nil
}

type Actor struct {
	ID           int64
	Name         string
	QueueID      int64
	CreatedAt    int64
	Metadata     []byte
	UpdatedAt    *int64
	Enabled      bool
	Deployable   bool
	Configurable bool
	Role         ActorRole
	Migratable   bool
}

type ApiToken struct {
	ID          string
	ActorId     int64
	ExpireAt    int64
	CreatedBy   string
	CreatedAt   int64
	Permissions []string
}

type Config struct {
	ID              int64
	ActorId         int64
	ConfigSuiteID   *int64
	Content         []byte
	MinActorVersion []int32
	CreatedBy       string
	CreatedAt       int64
	UpdatedBy       *string
	UpdatedAt       *int64
}

type ConfigSuite struct {
	ID         int64
	Active     bool
	CreatedBy  string
	CreatedAt  int64
	UpdatedBy  *string
	UpdatedAt  *int64
	DeployedAt *int64
}

type Deployment struct {
	ID            int64
	Name          string
	Status        DeploymentStatus
	Reviewers     []string
	ConfigSuiteID *int64
	Notes         []byte
	CreatedBy     string
	CreatedAt     int64
	ApprovedBy    *string
	ApprovedAt    *int64
	FinishedBy    *string
	FinishedAt    *int64
	MigrationLogs []byte
	LastError     *string
	DeployingAt   *int64
	DeployedAt    *int64
}

type Invocation struct {
	ID          int64
	State       InvocationState
	QueueID     int64
	AttemptedAt *int64
	CreatedAt   int64
	FinalizedAt *int64
	Priority    int16
	Payload     []byte
	Errors      []byte
	Result      []byte
	Metadata    []byte
	Tags        []string
	AttemptedBy []int64
}

type Migration struct {
	ID        int64
	CreatedAt int64
	Version   int64
}

type Queue struct {
	ID        int64
	Name      string
	CreatedAt int64
	Metadata  []byte
	PausedAt  *int64
	UpdatedAt *int64
}

type ReferenceConfigSuites struct {
	ID          int64
	Name        string
	ConfigSuite []byte
	CreatedAt   int64
	UpdatedAt   *int64
}

type Settings struct {
	Key   string
	Value []byte
}
