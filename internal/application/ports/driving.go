package ports

// AccountProcessor is the primary driving-side port of the application.
// Any driving adapter (CLI, AWS Lambda, HTTP, etc.) can depend on this
// interface to trigger the main business flow without coupling to concrete
// implementations.
type AccountProcessor interface {
	// Process runs the full account processing pipeline once and returns an
	// error if the operation fails.
	Process() error
}

