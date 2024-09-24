package listener

import (
	"context"
	"errors"
	"strings"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Listener interface {
	Close(ctx context.Context) error
	Connect(ctx context.Context) error
	Listen(ctx context.Context, topic string) error
	Ping(ctx context.Context) error
	Unlisten(ctx context.Context, topic string) error
	WaitForNotification(ctx context.Context) (*Notification, error)
}

type PgListener struct {
	conn   *pgxpool.Conn
	dbPool *pgxpool.Pool
	prefix string
	mu     sync.Mutex
}

type Notification struct {
	Payload string
	Topic   string
}

func NewPgListener(dbPool *pgxpool.Pool) *PgListener {
	return &PgListener{
		dbPool: dbPool,
	}
}

func (l *PgListener) Close(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.conn == nil {
		return nil
	}

	// Release below would take care of cleanup and potentially put the
	// connection back into rotation, but in case a Listen was invoked without a
	// subsequent Unlisten on the same tpic, close the connection explicitly to
	// guarantee no other caller will receive a partially tainted connection.
	err := l.conn.Conn().Close(ctx)

	// Even in the event of an error, make sure conn is set back to nil so that
	// the listener can be reused.
	l.conn.Release()
	l.conn = nil

	return err
}

func (l *PgListener) Connect(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.conn != nil {
		return errors.New("connection already established")
	}

	conn, err := l.dbPool.Acquire(ctx)
	if err != nil {
		return err
	}

	var schema string
	if err := conn.QueryRow(ctx, "SELECT current_schema();").Scan(&schema); err != nil {
		conn.Release()
		return err
	}

	l.prefix = schema + "."
	l.conn = conn
	return nil
}

func (l *PgListener) Listen(ctx context.Context, topic string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	_, err := l.conn.Exec(ctx, "LISTEN \""+l.prefix+topic+"\"")
	return err
}

func (l *PgListener) Ping(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.conn.Ping(ctx)
}

func (l *PgListener) Unlisten(ctx context.Context, topic string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	_, err := l.conn.Exec(ctx, "UNLISTEN \""+l.prefix+topic+"\"")
	return err
}

func (l *PgListener) WaitForNotification(ctx context.Context) (*Notification, error) {
	l.mu.Lock()
	defer l.mu.Unlock()

	notification, err := l.conn.Conn().WaitForNotification(ctx)
	if err != nil {
		return nil, err
	}

	return &Notification{
		Topic:   strings.TrimPrefix(notification.Channel, l.prefix),
		Payload: notification.Payload,
	}, nil
}
