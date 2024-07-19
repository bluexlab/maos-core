// Code generated by sqlc. DO NOT EDIT.
// versions:
//   sqlc v1.26.0

package dbsqlc

import (
	"database/sql/driver"
	"fmt"
)

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

type Agents struct {
	ID        int64
	Name      string
	QueueID   int64
	CreatedAt int64
	Metadata  []byte
	UpdatedAt *int64
}

type ApiTokens struct {
	ID          string
	AgentID     int64
	ExpireAt    int64
	CreatedBy   string
	CreatedAt   int64
	Permissions []string
}

type Invocations struct {
	ID          int64
	State       InvocationState
	CreatedAt   int64
	FinalizedAt *int64
	Priority    int16
	Name        string
	Args        []byte
	Errors      [][]byte
	Metadata    []byte
	QueueID     int64
	Tags        []string
}

type Migration struct {
	ID        int64
	CreatedAt int64
	Version   int64
}

type Queues struct {
	ID        int64
	Name      string
	CreatedAt int64
	Metadata  []byte
	PausedAt  *int64
	UpdatedAt int64
}
