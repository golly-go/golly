package golly

import "testing"

func BenchmarkMakeRequest(b *testing.B) {

	b.Run("MakeRequest", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = makeRequestID()
		}
	})

}
