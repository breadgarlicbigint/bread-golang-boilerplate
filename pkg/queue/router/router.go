// Package router lets the API publish onto more than one queue.Publisher at
// once and pick which backend a given task type goes to — the mechanism
// behind QUEUE_TRANSACTIONAL_DRIVER / QUEUE_PROMOTIONAL_DRIVER in
// shared/config. A Router itself satisfies queue.Publisher, so callers (and
// apps/api/app/app.go's jobQueue field) don't need to know routing is
// happening at all.
//
// Typical shape: a "default" Publisher backs every task type, and specific
// task types (queue.TaskSendEmail, queue.TaskSendPromotionalEmail, …) are
// pinned to a different Publisher via Route. Consumption is unaffected —
// every worker process registers handlers per task type exactly as before;
// it has no idea (and doesn't need to know) which broker a job arrived on
// was chosen by a Router upstream. What matters is that a worker process is
// actually running for every backend a Router might route to.
package router

import (
	"github.com/breadgarlicbigint/bread-golang-boilerplate/pkg/queue"
)

// Router dispatches Enqueue/EnqueueEmail/EnqueuePromotionalEmail calls to
// different underlying queue.Publisher backends based on task type. Safe for
// concurrent use iff the underlying Publishers are (true for all of
// pkg/worker, pkg/queue/rabbitmq, pkg/queue/kafka).
type Router struct {
	def     queue.Publisher
	routes  map[string]queue.Publisher // taskType -> publisher
	closers []queue.Publisher          // unique publishers, for Close
}

// New creates a Router backed by def for any task type without a more
// specific Route.
func New(def queue.Publisher) *Router {
	return &Router{
		def:     def,
		routes:  make(map[string]queue.Publisher),
		closers: []queue.Publisher{def},
	}
}

// Route sends taskType through pub instead of the default publisher.
// Registering the same pub for multiple task types (or passing the same
// value already used elsewhere) is safe — Close still closes each unique
// underlying connection exactly once.
func (r *Router) Route(taskType string, pub queue.Publisher) {
	r.routes[taskType] = pub
	for _, c := range r.closers {
		if c == pub {
			return
		}
	}
	r.closers = append(r.closers, pub)
}

func (r *Router) publisherFor(taskType string) queue.Publisher {
	if p, ok := r.routes[taskType]; ok {
		return p
	}
	return r.def
}

// Enqueue routes by taskType to whichever Publisher owns it (or the default).
func (r *Router) Enqueue(taskType string, payload any) error {
	return r.publisherFor(taskType).Enqueue(taskType, payload)
}

// EnqueueEmail routes queue.TaskSendEmail (transactional email).
func (r *Router) EnqueueEmail(to, subject, html, text string) error {
	return r.publisherFor(queue.TaskSendEmail).EnqueueEmail(to, subject, html, text)
}

// EnqueuePromotionalEmail routes queue.TaskSendPromotionalEmail (bulk/marketing email).
func (r *Router) EnqueuePromotionalEmail(to, subject, html, text string) error {
	return r.publisherFor(queue.TaskSendPromotionalEmail).EnqueuePromotionalEmail(to, subject, html, text)
}

// Close closes every unique underlying Publisher once, returning the first error.
func (r *Router) Close() error {
	var firstErr error
	for _, c := range r.closers {
		if err := c.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// Compile-time check that Router satisfies the same contract every backend does.
var _ queue.Publisher = (*Router)(nil)
