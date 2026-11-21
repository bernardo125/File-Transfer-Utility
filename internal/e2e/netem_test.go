//go:build netem

package e2e

import (
	"os"
	"os/exec"
	"testing"

	"ftu/internal/testutil"
)

// Advanced layer: real kernel-level impairment with tc/netem on the loopback device.
// Requires Linux, root (or CAP_NET_ADMIN) and the `tc` binary. It changes the qdisc of `lo`
// for the duration of the test and removes it on cleanup. Run with:
//
//	sudo -E env "PATH=$PATH" go test -tags netem -run Netem -v ./internal/e2e/
func applyNetem(t *testing.T, spec ...string) {
	t.Helper()
	if os.Geteuid() != 0 {
		t.Skip("netem tests need root (CAP_NET_ADMIN); run via sudo, see docs/TESTING.md")
	}
	if _, err := exec.LookPath("tc"); err != nil {
		t.Skip("tc binary not found (install iproute2)")
	}
	args := append([]string{"qdisc", "add", "dev", "lo", "root", "netem"}, spec...)
	if out, err := exec.Command("tc", args...).CombinedOutput(); err != nil {
		t.Fatalf("tc %v: %v\n%s", args, err, out)
	}
	t.Cleanup(func() { exec.Command("tc", "qdisc", "del", "dev", "lo", "root").Run() })
}

// Detects: transfer corruption/stalls under kernel-level loss, delay, jitter, duplication and
// reordering (as opposed to the userspace proxy used by the default suite).
func TestNetemTransferIntegrity(t *testing.T) {
	cases := map[string][]string{
		"delay_jitter": {"delay", "40ms", "15ms"},
		"loss":         {"loss", "5%"},
		"duplicate":    {"duplicate", "10%"},
		"reorder":      {"delay", "10ms", "reorder", "25%", "50%"},
		"mixed":        {"delay", "20ms", "10ms", "loss", "2%", "duplicate", "5%"},
	}
	for name, spec := range cases {
		t.Run(name, func(t *testing.T) {
			applyNetem(t, spec...)
			a, b := testutil.NewPeer(t), testutil.NewPeer(t)
			ca, cb := establish(t, a, b, b.Addr(), a.Addr())
			src, dst := t.TempDir(), t.TempDir()
			t.Chdir(dst)
			path, sum := testutil.WriteFile(t, src, "data.bin", 2<<20, 8)
			transfer(t, ca, cb, path)
			assertReceived(t, dst, "data.bin", sum)
		})
	}
}
