package window

import "testing"

func TestCodexExecutableMatching(t *testing.T) {
	cases := []struct {
		name          string
		path          string
		packageFamily string
		override      string
		want          bool
	}{
		{
			name:          "official package family ignores drive version and architecture",
			path:          `D:\Apps\arbitrary\Host.exe`,
			packageFamily: `OpenAI.Codex_26.707.3351.0_arm64__2p2nqsd0c76g0`,
			want:          true,
		},
		{
			name:          "official package family works without process path",
			packageFamily: `OpenAI.Codex_2p2nqsd0c76g0`,
			want:          true,
		},
		{
			name:          "ordinary ChatGPT package is rejected",
			path:          `C:\Program Files\OpenAI Codex\ChatGPT.exe`,
			packageFamily: `OpenAI.ChatGPT_1.0_x64__2p2nqsd0c76g0`,
			want:          false,
		},
		{
			name: "legacy official package path",
			path: `C:\Program Files\WindowsApps\OpenAI.Codex_26.707.3351.0_x64__2p2nqsd0c76g0\app\ChatGPT.exe`,
			want: true,
		},
		{
			name: "enterprise ChatGPT host",
			path: `E:\Company Apps\OpenAI Codex\ChatGPT.exe`,
			want: true,
		},
		{
			name: "Codex executable",
			path: `C:\Tools\Codex.exe`,
			want: true,
		},
		{
			name: "generic ChatGPT host",
			path: `C:\Tools\ChatGPT.exe`,
			want: false,
		},
		{
			name:     "exact override",
			path:     `C:\Enterprise\DesktopHost.exe`,
			override: `"c:\enterprise\DESKTOPHOST.exe"`,
			want:     true,
		},
		{
			name:          "exact override supersedes unrelated package family",
			path:          `C:\Enterprise\DesktopHost.exe`,
			packageFamily: `Contoso.CodexDesktop_1234567890abc`,
			override:      `C:\Enterprise\DesktopHost.exe`,
			want:          true,
		},
		{
			name:     "override is exact",
			path:     `C:\Enterprise\Other\DesktopHost.exe`,
			override: `C:\Enterprise\DesktopHost.exe`,
			want:     false,
		},
		{
			name: "unrelated executable under Codex directory",
			path: `C:\Program Files\OpenAI Codex\Other.exe`,
			want: false,
		},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			if got := isCodexProcess(item.path, item.packageFamily, item.override); got != item.want {
				t.Errorf("isCodexProcess(%q, %q, %q)=%v, want %v", item.path, item.packageFamily, item.override, got, item.want)
			}
		})
	}
}

func TestChooseCodexCandidate(t *testing.T) {
	tests := []struct {
		name       string
		candidates []windowCandidate
		want       HWND
	}{
		{
			name: "largest visible fallback",
			candidates: []windowCandidate{
				{handle: 11, area: 100},
				{handle: 22, area: 200},
			},
			want: 22,
		},
		{
			name: "zero handles ignored",
			candidates: []windowCandidate{
				{handle: 0, area: 500},
				{handle: 33, area: 100},
			},
			want: 33,
		},
		{name: "empty", want: 0},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := chooseCodexCandidate(test.candidates); got != test.want {
				t.Fatalf("chooseCodexCandidate()=%v, want %v", got, test.want)
			}
		})
	}
}
