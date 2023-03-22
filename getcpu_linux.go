//go:build !amd64
// +build !amd64

package numa

// GetCPUAndNode returns the node id and cpu id which current caller running on.
// https://man7.org/linux/man-pages/man2/getcpu.2.html
func GetCPUAndNode() (cpu int, node int) {
	return 0, 0
}
