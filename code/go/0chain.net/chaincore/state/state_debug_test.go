package state

import "testing"

func TestDebug(t *testing.T) {
	tests := []struct {
		name       string
		debugState int
		want       bool
	}{
		{
			name:       "TRUE",
			debugState: DebugLevelChain,
			want:       true,
		},
		{
			name:       "FALSE",
			debugState: DebugLevelNone,
			want:       false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetDebugLevel(tt.debugState)
			if got := Debug(); got != tt.want {
				t.Errorf("Debug() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDebugChain(t *testing.T) {
	tests := []struct {
		name       string
		debugState int
		want       bool
	}{
		{
			name:       "TRUE",
			debugState: DebugLevelChain,
			want:       true,
		},
		{
			name:       "FALSE",
			debugState: DebugLevelNone,
			want:       false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetDebugLevel(tt.debugState)
			if got := DebugChain(); got != tt.want {
				t.Errorf("DebugChain() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDebugBlock(t *testing.T) {
	tests := []struct {
		name       string
		debugState int
		want       bool
	}{
		{
			name:       "TRUE",
			debugState: DebugLevelBlock,
			want:       true,
		},
		{
			name:       "FALSE",
			debugState: DebugLevelChain,
			want:       false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetDebugLevel(tt.debugState)
			if got := DebugBlock(); got != tt.want {
				t.Errorf("DebugBlock() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDebugTxn(t *testing.T) {
	tests := []struct {
		name       string
		debugState int
		want       bool
	}{
		{
			name:       "TRUE",
			debugState: DebugLevelTxn,
			want:       true,
		},
		{
			name:       "FALSE",
			debugState: DebugLevelBlock,
			want:       false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetDebugLevel(tt.debugState)
			if got := DebugTxn(); got != tt.want {
				t.Errorf("DebugTxn() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDebugNode(t *testing.T) {
	tests := []struct {
		name       string
		debugState int
		want       bool
	}{
		{
			name:       "TRUE",
			debugState: DebugLevelNode,
			want:       true,
		},
		{
			name:       "FALSE",
			debugState: DebugLevelChain,
			want:       false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			SetDebugLevel(tt.debugState)
			if got := DebugNode(); got != tt.want {
				t.Errorf("DebugNode() = %v, want %v", got, tt.want)
			}
		})
	}
}
