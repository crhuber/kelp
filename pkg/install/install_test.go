package install

import "testing"

func TestCleanBinaryName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"direnv.darwin-arm64", "direnv"},
		{"direnv.darwin-amd64", "direnv"},
		{"direnv.linux-arm64", "direnv"},
		{"direnv.linux-amd64", "direnv"},
		{"tool.darwin_arm64", "tool"},
		{"tool.linux_amd64", "tool"},
		{"tool.arm64-darwin", "tool"},
		{"tool.amd64-linux", "tool"},
		{"tool.macos-arm64", "tool"},
		{"tool.windows-x86_64", "tool"},
		{"tool.linux-aarch64", "tool"},
		{"tool.darwin-x64", "tool"},
		{"talosctl-linux-amd64", "talosctl"},
		{"talosctl-linux-arm64", "talosctl"},
		{"talosctl-darwin-arm64", "talosctl"},
		{"talosctl-darwin-amd64", "talosctl"},
		{"talosctl-freebsd-amd64", "talosctl"},
		{"talosctl-linux-armv7", "talosctl"},
		{"talosctl-linux-riscv64", "talosctl"},
		{"talosctl-windows-amd64.exe", "talosctl.exe"},
		{"talosctl_linux_amd64", "talosctl"},
		{"hadolint-linux-x86_64", "hadolint"},
		{"hadolint-Linux-x86_64", "hadolint"},
		{"tool-darwin-universal", "tool"},
		{"tool_amd64_linux", "tool"},
		{"tool.linux.amd64", "tool"},
		// No matching suffix — returned as-is
		{"mybinary", "mybinary"},
		{"archive.tar.gz", "archive.tar.gz"},
		{"tool.v1.2.3", "tool.v1.2.3"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := cleanBinaryName(tt.input)
			if got != tt.want {
				t.Errorf("cleanBinaryName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
