package mathrender

import "testing"

func TestBaselineAndBoundedCache(t *testing.T) {
	r, err := New()
	if err != nil {
		t.Fatal(err)
	}
	f, err := r.Render(`\frac{x_i}{2}`, 18, false)
	if err != nil {
		t.Fatal(err)
	}
	if f.Width <= 0 || f.Height <= 0 || f.Depth <= 0 {
		t.Fatalf("invalid inline metrics: %+v", f)
	}
	w, h := f.SVG.Size()
	if abs(w-f.Width) > .01 || abs(h-f.Height-f.Depth) > .01 {
		t.Fatal("SVG does not match baseline metrics")
	}
	again, err := r.Render(`\frac{x_i}{2}`, 18, false)
	if err != nil {
		t.Fatal(err)
	}
	if again.SVG != f.SVG {
		t.Fatal("cache did not reuse geometry")
	}
	r.capacity = 2
	for _, src := range []string{"a", "b", "c"} {
		if _, err = r.Render(src, 18, false); err != nil {
			t.Fatal(err)
		}
	}
	if len(r.cache) != 2 || r.order.Len() != 2 {
		t.Fatal("cache is not bounded")
	}
}

func abs(f float32) float32 {
	if f < 0 {
		return -f
	}
	return f
}
