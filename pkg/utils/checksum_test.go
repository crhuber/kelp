package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestComputeSHA256(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kelp-checksum-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testFile, []byte("hello world\n"), 0o644)
	require.NoError(t, err)

	// echo "hello world" | sha256sum -> d2a84f4b8b650937ec8f73cd8be2c74add5a911ba64df27458ed8229da804a26
	hash, err := ComputeSHA256(testFile)
	require.NoError(t, err)
	require.Equal(t, "a948904f2f0f479b8f8197694b30184b0d2ed1c1cd2a1ec0fb85d299a192a447", hash)
}

func TestVerifyFileSHA256(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "kelp-checksum-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	testFile := filepath.Join(tempDir, "test.txt")
	err = os.WriteFile(testFile, []byte("hello world\n"), 0o644)
	require.NoError(t, err)

	expectedHash := "a948904f2f0f479b8f8197694b30184b0d2ed1c1cd2a1ec0fb85d299a192a447"
	err = VerifyFileSHA256(testFile, expectedHash)
	require.NoError(t, err)

	// Uppercase should also match (case-insensitive)
	err = VerifyFileSHA256(testFile, "A948904F2F0F479B8F8197694B30184B0D2ED1C1CD2A1EC0FB85D299A192A447")
	require.NoError(t, err)

	// Mismatched hash
	err = VerifyFileSHA256(testFile, "0000000000000000000000000000000000000000000000000000000000000000")
	require.Error(t, err)
	require.Contains(t, err.Error(), "checksum mismatch")
}

func TestParseSHA256FromManifest(t *testing.T) {
	const sampleHash = "29454bc351f4433e66c00f5d37841627cbbcc02e4c70a6d796529d355237671c"

	tests := []struct {
		name           string
		content        string
		targetFilename string
		expectedHash   string
		expectedFound  bool
	}{
		{
			name:           "single hash only",
			content:        sampleHash + "\n",
			targetFilename: "kubectl",
			expectedHash:   sampleHash,
			expectedFound:  true,
		},
		{
			name: "standard sha256sum output",
			content: `# Checksums
e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  other-file.tar.gz
29454bc351f4433e66c00f5d37841627cbbcc02e4c70a6d796529d355237671c  helm-v4.0.4-linux-amd64.tar.gz
16b88acc6503d646b7537a298e7389bef469c5cc9ebadf727547abe9f6a35903  helm-v4.0.4-linux-arm64.tar.gz
`,
			targetFilename: "helm-v4.0.4-linux-amd64.tar.gz",
			expectedHash:   sampleHash,
			expectedFound:  true,
		},
		{
			name:           "sha256sum with binary asterisk",
			content:        "29454bc351f4433e66c00f5d37841627cbbcc02e4c70a6d796529d355237671c *helm-v4.0.4-linux-amd64.tar.gz\n",
			targetFilename: "helm-v4.0.4-linux-amd64.tar.gz",
			expectedHash:   sampleHash,
			expectedFound:  true,
		},
		{
			name:           "BSD format",
			content:        "SHA256 (helm-v4.0.4-linux-amd64.tar.gz) = 29454bc351f4433e66c00f5d37841627cbbcc02e4c70a6d796529d355237671c\n",
			targetFilename: "helm-v4.0.4-linux-amd64.tar.gz",
			expectedHash:   sampleHash,
			expectedFound:  true,
		},
		{
			name: "Markdown release body format",
			content: `## Download
- [Linux amd64](https://get.helm.sh/helm-v4.0.4-linux-amd64.tar.gz) ([checksum](...) / 29454bc351f4433e66c00f5d37841627cbbcc02e4c70a6d796529d355237671c)
- [Linux arm64](https://get.helm.sh/helm-v4.0.4-linux-arm64.tar.gz) ([checksum](...) / 16b88acc6503d646b7537a298e7389bef469c5cc9ebadf727547abe9f6a35903)
`,
			targetFilename: "helm-v4.0.4-linux-amd64.tar.gz",
			expectedHash:   sampleHash,
			expectedFound:  true,
		},
		{
			name: "longer file name containing the target is listed first",
			content: `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  agent-tool-linux-amd64
29454bc351f4433e66c00f5d37841627cbbcc02e4c70a6d796529d355237671c  tool-linux-amd64
`,
			targetFilename: "tool-linux-amd64",
			expectedHash:   sampleHash,
			expectedFound:  true,
		},
		{
			name: "target with a suffix is listed first",
			content: `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  tool-linux-amd64.sbom.json
29454bc351f4433e66c00f5d37841627cbbcc02e4c70a6d796529d355237671c  tool-linux-amd64
`,
			targetFilename: "tool-linux-amd64",
			expectedHash:   sampleHash,
			expectedFound:  true,
		},
		{
			name: "Markdown release body lists a longer name first",
			content: `- [agent](https://example.com/download/agent-tool-linux-amd64) e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
- [tool](https://example.com/download/tool-linux-amd64) 29454bc351f4433e66c00f5d37841627cbbcc02e4c70a6d796529d355237671c
`,
			targetFilename: "tool-linux-amd64",
			expectedHash:   sampleHash,
			expectedFound:  true,
		},
		{
			name:           "CRLF manifest with ./ prefix",
			content:        "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  ./other-file.tar.gz\r\n29454bc351f4433e66c00f5d37841627cbbcc02e4c70a6d796529d355237671c  ./helm-v4.0.4-linux-amd64.tar.gz\r\n",
			targetFilename: "helm-v4.0.4-linux-amd64.tar.gz",
			expectedHash:   sampleHash,
			expectedFound:  true,
		},
		{
			name: "only a longer file name containing the target is listed",
			content: `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  agent-tool-linux-amd64
16b88acc6503d646b7537a298e7389bef469c5cc9ebadf727547abe9f6a35903  agent-tool-linux-arm64
`,
			targetFilename: "tool-linux-amd64",
			expectedHash:   "",
			expectedFound:  false,
		},
		{
			name: "Target not in manifest",
			content: `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855  other-file.tar.gz
16b88acc6503d646b7537a298e7389bef469c5cc9ebadf727547abe9f6a35903  different-file.tar.gz
`,
			targetFilename: "helm-v4.0.4-linux-amd64.tar.gz",
			expectedHash:   "",
			expectedFound:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotHash, found := ParseSHA256FromManifest(tt.content, tt.targetFilename)
			require.Equal(t, tt.expectedFound, found)
			if tt.expectedFound {
				require.Equal(t, tt.expectedHash, gotHash)
			}
		})
	}
}
