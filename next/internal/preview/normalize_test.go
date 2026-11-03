package preview

import "testing"

func TestMathCurrencyAndCode(t *testing.T) {
	for _, tc := range []struct{ in, want string }{
		{`Compute $5 + 2 = 7$, $5x$, $5 (x+1)$.`, `Compute \(5 + 2 = 7\), \(5x\), \(5 (x+1)\).`},
		{`The price is $5 and the budget is $10.`, `The price is \$5 and the budget is \$10.`},
		{`$500\text{ GB} \times \$0.015 = \$7.50$`, `\(500\text{ GB} \times \$0.015 = \$7.50\)`},
		{"`$x$`\n\n```text\n$x$\n```", "`$x$`\n\n```text\n$x$\n```"},
		{`\(x_i\) and $$x^2$$`, `\(x_i\) and $$x^2$$`},
	} {
		if got := Normalize(tc.in); got != tc.want {
			t.Errorf("%q: got %q want %q", tc.in, got, tc.want)
		}
	}
}
