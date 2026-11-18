package repl

import "sync"

// key is one press the menu understands.
type key struct {
	kind byte // U up, D down, E enter
}

var (
	keyOnce sync.Once
	keyCh   <-chan key
	keyStop func()
	keyOK   bool
)

// consoleKeys starts the arrow reader once. The second result is false when
// stdin is not a console, and the line reader stays in charge.
func consoleKeys() (<-chan key, bool) {
	keyOnce.Do(func() {
		var stop func()
		keyCh, stop, keyOK = openKeys()
		if stop == nil {
			stop = func() {}
		}
		keyStop = stop
	})
	return keyCh, keyOK
}

// StopKeys restores the console. Safe to call more than once.
func StopKeys() {
	if keyStop != nil {
		keyStop()
	}
}
