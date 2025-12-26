package repository

import "context"

// Transaction represents an atomic unit of work.
// All operations within a transaction must succeed or all will be rolled back.
type Transaction interface {
	// Commit commits the transaction.
	// Returns an error if the transaction has already been committed or rolled back.
	Commit() error

	// Rollback rolls back the transaction.
	// Returns an error if the transaction has already been committed or rolled back.
	Rollback() error
}

// TransactionManager provides transaction lifecycle management.
type TransactionManager interface {
	// BeginTx starts a new transaction.
	// The context should be the same context used for the operations within the transaction.
	BeginTx(ctx context.Context) (Transaction, error)

	// WithTransaction executes a function within a transaction.
	// If the function returns an error, the transaction is rolled back.
	// Otherwise, the transaction is committed.
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

// ContextKey for storing transaction in context
type txKey struct{}

// NewContextWithTx creates a new context with the transaction attached.
func NewContextWithTx(ctx context.Context, tx Transaction) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}

// TxFromContext retrieves the transaction from the context.
// Returns nil if no transaction is found.
func TxFromContext(ctx context.Context) Transaction {
	tx, _ := ctx.Value(txKey{}).(Transaction)
	return tx
}
