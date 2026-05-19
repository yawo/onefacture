package invoice

import "errors"

// ErrInvalidTransition is returned when a state machine transition is rejected.
var ErrInvalidTransition = errors.New("invalid status transition")

var allowedTransitions = map[Status]map[Status]struct{}{
	StatusDraft:     {StatusValidated: {}, StatusCancelled: {}},
	StatusValidated: {StatusSubmitted: {}, StatusCancelled: {}, StatusRejected: {}},
	StatusSubmitted: {StatusAccepted: {}, StatusRejected: {}, StatusReceived: {}},
	StatusReceived:  {StatusAccepted: {}, StatusRejected: {}},
	StatusAccepted:  {StatusPaid: {}, StatusCancelled: {}},
	StatusRejected:  {},
	StatusPaid:      {},
	StatusCancelled: {},
}

// CanTransition reports whether moving from `from` to `to` is permitted.
func CanTransition(from, to Status) bool {
	allowed, ok := allowedTransitions[from]
	if !ok {
		return false
	}
	_, ok = allowed[to]
	return ok
}

// Transition applies a status change, returning ErrInvalidTransition if illegal.
func (inv *Invoice) Transition(to Status) error {
	if !CanTransition(inv.Status, to) {
		return ErrInvalidTransition
	}
	inv.Status = to
	return nil
}
