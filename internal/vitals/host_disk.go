package vitals

import "golang.org/x/sys/unix"

// hostRootDisk returns used and total bytes for the controller's root filesystem.
// On the appliance this is the same disk k3s and PVC data share.
func hostRootDisk() (used, total int64, ok bool) {
	var st unix.Statfs_t
	if err := unix.Statfs("/", &st); err != nil {
		return 0, 0, false
	}
	total = int64(st.Blocks) * int64(st.Bsize)
	free := int64(st.Bfree) * int64(st.Bsize)
	used = total - free
	if total <= 0 {
		return 0, 0, false
	}
	return used, total, true
}