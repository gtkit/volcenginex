package volcenginex

import "testing"

func BenchmarkJitterDuration(b *testing.B) {
	b.ReportAllocs()
	for b.Loop() {
		_ = jitterDuration(3, 0.2)
	}
}
