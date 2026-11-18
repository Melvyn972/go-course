//go:build !windows && !linux

package repl

func openKeys() (<-chan key, func(), bool) {
	return nil, func() {}, false
}
