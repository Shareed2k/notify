package notify

// EventDiff describes a change to an event set - EventDiff[0] is an old state,
// while EventDiff[1] is a new state. If event set has not changed (old == new),
// functions typically return the None value.
type eventDiff [2]Event

// Event TODO
func (diff eventDiff) Event() Event {
	return diff[1] &^ diff[0]
}

// Watchpoint TODO
//
// The nil key holds total event set - logical sum for all registered events.
// It speeds up computing EventDiff for Add method.
//
// The rec key holds an event set for a watchpoints created by RecursiveWatch
// for a Watcher implementation which is not natively recursive.
type watchpoint map[chan<- EventInfo]Event

// None is an empty event diff, think null object.
var none eventDiff

// rec is just a placeholder
var rec = func() (ch chan<- EventInfo) {
	ch = make(chan<- EventInfo)
	close(ch)
	return
}()

// Diff TODO
func (wp watchpoint) Diff(e Event) eventDiff {
	e &^= recursive
	if wp[nil]&e == e {
		return none
	}
	total := wp[nil] &^ recursive
	return eventDiff{total, total | e}
}

// Add TODO
//
// Add assumes neither c nor e are nil or zero values.
func (wp watchpoint) Add(c chan<- EventInfo, e Event) (diff eventDiff) {
	wp[c] |= e
	diff[0] = wp[nil]
	diff[1] = diff[0] | e
	wp[nil] = diff[1]
	// Strip diff from recursive events.
	diff[0] &^= recursive
	diff[1] &^= recursive
	if diff[0] == diff[1] {
		return none
	}
	return
}

// Del TODO
func (wp watchpoint) Del(c chan<- EventInfo, e Event) (diff eventDiff) {
	wp[c] &^= e
	if wp[c] == 0 {
		delete(wp, c)
	}
	diff[0] = wp[nil]
	delete(wp, nil)
	if len(wp) != 0 {
		// Recalculate total event set.
		for _, e := range wp {
			diff[1] |= e
		}
		wp[nil] = diff[1]
	}
	// Strip diff from recursive events.
	diff[0] &^= recursive
	diff[1] &^= recursive
	if diff[0] == diff[1] {
		return none
	}
	return
}

// Dispatch TODO
func (wp watchpoint) Dispatch(ei EventInfo, recursiveonly bool) {
	event := ei.Event()
	if recursiveonly {
		event |= recursive
	}
	if wp[nil]&event != event {
		return
	}
	for ch, e := range wp {
		if ch != nil && ch != rec && e&event == event {
			select {
			case ch <- ei:
			default:
				// Drop event if receiver is too slow
			}
		}
	}
}

// AddRecursive TODO
func (wp watchpoint) AddRecursive(e Event) eventDiff {
	return wp.Add(rec, e|recursive)
}

// DelRecursive TODO
func (wp watchpoint) DelRecursive(e Event) eventDiff {
	// If this delete would remove all events from rec event set, ensure Recursive
	// is also gone.
	if wp[rec] == e|recursive {
		e |= recursive
	}
	return wp.Del(rec, e)
}

// Recursive TODO
func (wp watchpoint) recursive() Event {
	return wp[rec]
}

// Total TODO
func (wp watchpoint) Total() Event {
	return wp[nil] &^ recursive
}

// IsRecursive TODO
func (wp watchpoint) IsRecursive() bool {
	return wp[nil]&recursive == recursive
}
