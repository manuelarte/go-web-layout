package tracing

//nolint:gochecknoglobals // Context key used for tracing.
var TracingContextKey = Context{}

// Context tracing value to be passed through the stack trace through [context.Context].
type Context struct{}
