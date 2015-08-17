package turnpike

// A Dealer routes and manages RPC calls to callees.
type Dealer interface {
	// Register a procedure on an endpoint
	Register(Sender, *Register)
	// Unregister a procedure on an endpoint
	Unregister(Sender, *Unregister)
	// Unregister all registered procedures on an endpoint
	Disconnect(Sender)
	// Call a procedure on an endpoint
	Call(Sender, *Call)
	// Return the result of a procedure call
	Yield(Sender, *Yield)
}

type RemoteProcedure struct {
	Endpoint  Sender
	Procedure URI
}

type defaultDealer struct {
	// map registration IDs to procedures
	procedures map[ID]RemoteProcedure
	// map procedure URIs to registration IDs
	// TODO: this will eventually need to be `map[URI][]ID` to support
	// multiple callees for the same procedure
	registrations map[URI]ID
	// keep track of call IDs so we can send the response to the caller
	calls map[ID]Sender
	// link the invocation ID to the call ID
	invocations map[ID]ID
}

func NewDefaultDealer() Dealer {
	return &defaultDealer{
		procedures:    make(map[ID]RemoteProcedure),
		registrations: make(map[URI]ID),
		calls:         make(map[ID]Sender),
		invocations:   make(map[ID]ID),
	}
}

func (d *defaultDealer) Register(callee Sender, msg *Register) {
	if id, ok := d.registrations[msg.Procedure]; ok {
		log.Println("error: procedure already exists:", msg.Procedure, id)
		callee.Send(&Error{
			Type:    msg.MessageType(),
			Request: msg.Request,
			Details: make(map[string]interface{}),
			Error:   WAMP_ERROR_PROCEDURE_ALREADY_EXISTS,
		})
		return
	}
	reg := NewID()
	d.procedures[reg] = RemoteProcedure{callee, msg.Procedure}
	d.registrations[msg.Procedure] = reg
	callee.Send(&Registered{
		Request:      msg.Request,
		Registration: reg,
	})
}

func (d *defaultDealer) Unregister(callee Sender, msg *Unregister) {
	if procedure, ok := d.procedures[msg.Registration]; !ok {
		// the registration doesn't exist
		log.Println("error: no such registration:", msg.Registration)
		callee.Send(&Error{
			Type:    msg.MessageType(),
			Request: msg.Request,
			Details: make(map[string]interface{}),
			Error:   WAMP_ERROR_NO_SUCH_REGISTRATION,
		})
	} else {
		delete(d.registrations, procedure.Procedure)
		delete(d.procedures, msg.Registration)
		callee.Send(&Unregistered{
			Request: msg.Request,
		})
	}
}

func (d *defaultDealer) Disconnect(callee Sender) {
	proceduresToDelete := make([]RemoteProcedure, 1)
	registrationsToDelete := make([]ID, 1)
	// Search d.procedures for callee
	for reg, rp := range d.procedures {
		if rp.Endpoint == callee {
			log.Info("Unregistering procedure", rp.Procedure)
			proceduresToDelete = append(proceduresToDelete, rp)
			registrationsToDelete = append(registrationsToDelete, reg)
		}
	}
	// Delete procedures and registrations
	for _, p := range proceduresToDelete {
		delete(d.registrations, p.Procedure)
	}
	for _, r := range registrationsToDelete {
		delete(d.procedures, r)
	}
}

func (d *defaultDealer) Call(caller Sender, msg *Call) {
	if reg, ok := d.registrations[msg.Procedure]; !ok {
		caller.Send(&Error{
			Type:    msg.MessageType(),
			Request: msg.Request,
			Details: make(map[string]interface{}),
			Error:   WAMP_ERROR_NO_SUCH_PROCEDURE,
		})
	} else {
		if rproc, ok := d.procedures[reg]; !ok {
			// found a registration id, but doesn't match any remote procedure
			caller.Send(&Error{
				Type:    msg.MessageType(),
				Request: msg.Request,
				Details: make(map[string]interface{}),
				// TODO: what should this error be?
				Error: URI("wamp.error.internal_error"),
			})
		} else {
			// everything checks out, make the invocation request
			// TODO: make the Request ID specific to the caller
			d.calls[msg.Request] = caller
			invocationID := NewID()
			d.invocations[invocationID] = msg.Request
			rproc.Endpoint.Send(&Invocation{
				Request:      invocationID,
				Registration: reg,
				Details:      map[string]interface{}{},
				Arguments:    msg.Arguments,
				ArgumentsKw:  msg.ArgumentsKw,
			})
			log.Printf("dispatched CALL %v to callee as INVOCATION %v", msg.Request, invocationID)
		}
	}
}

func (d *defaultDealer) Yield(callee Sender, msg *Yield) {
	if callID, ok := d.invocations[msg.Request]; !ok {
		// WAMP spec doesn't allow sending an error in response to a YIELD message
		log.Println("received YIELD message with invalid invocation request ID:", msg.Request)
	} else {
		delete(d.invocations, msg.Request)
		if caller, ok := d.calls[callID]; !ok {
			// found the invocation id, but doesn't match any call id
			// WAMP spec doesn't allow sending an error in response to a YIELD message
			log.Printf("received YIELD message, but unable to match it (%v) to a CALL ID", msg.Request)
		} else {
			delete(d.calls, callID)
			// return the result to the caller
			caller.Send(&Result{
				Request:     callID,
				Details:     map[string]interface{}{},
				Arguments:   msg.Arguments,
				ArgumentsKw: msg.ArgumentsKw,
			})
			log.Printf("returned YIELD %v to caller as RESULT %v", msg.Request, callID)
		}
	}
}
