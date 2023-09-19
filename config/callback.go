package config

var fnCallbacks = make([]func(), 0, 10)

// RegisterCallback registers a callback function.
// Callbacks will be called in order when Parse() is called.
func RegisterCallback(fn func()) {
	fnCallbacks = append(fnCallbacks, fn)
}

func executeCallbacks() {
	for _, fn := range fnCallbacks {
		fn()
	}
}
