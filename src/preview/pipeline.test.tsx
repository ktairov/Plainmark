import { renderToStaticMarkup } from "react-dom/server";
import ReactMarkdown from "react-markdown";
import rehypeKatex from "rehype-katex";
import remarkFrontmatter from "remark-frontmatter";
import remarkGfm from "remark-gfm";
import remarkMath from "remark-math";
import { describe, expect, it } from "vitest";
import { normalizePreviewMath } from "./math";
import { remarkHidePreviewMetadata } from "./metadata";
import { remarkTableBreaks } from "./tables";

const render = (markdown: string) => renderToStaticMarkup(
  <ReactMarkdown
    remarkPlugins={[remarkGfm, remarkFrontmatter, remarkMath, remarkTableBreaks, remarkHidePreviewMetadata]}
    rehypePlugins={[[rehypeKatex, { trust: false, strict: "ignore" }]]}
    components={{ table: ({ node: _node, ...props }) => <div className="markdown-table-scroll"><table {...props}/></div> }}
  >
    {normalizePreviewMath(markdown)}
  </ReactMarkdown>,
);

describe("Preview Markdown pipeline", () => {
  it("hides YAML front matter and HTML comments but keeps document text", () => {
    const html = render("---\ntitle: Hidden metadata\n---\n<!-- private comment -->\nVisible text");
    expect(html).toContain("Visible text");
    expect(html).not.toContain("Hidden metadata");
    expect(html).not.toContain("private comment");
  });

  it("renders inline integrals with KaTeX", () => {
    const html = render("Euler wrote $\\int_0^1 x^2 \\, dx = \\frac{1}{3}$.");
    expect(html).toContain("class=\"katex\"");
    expect(html).toContain("∫");
    expect(html).toContain("mfrac");
  });

  it("renders display integrals with limits", () => {
    const html = render("$$\n\\int_{-\\infty}^{\\infty} e^{-x^2} \\, dx = \\sqrt{\\pi}\n$$");
    expect(html).toContain("class=\"katex-display\"");
    expect(html).toContain("∫");
    expect(html).toContain("\\infty");
  });

  it("renders LaTeX parenthesis and bracket delimiters used by generated notes", () => {
    const html = render("Compute \\(\\int_0^2 (x^2-1)\\,dx\\).\n\n\\[ \\int_{-\\infty}^{\\infty} e^{-x^2}\\,dx = \\sqrt{\\pi} \\]");
    expect(html).toContain("class=\"katex\"");
    expect(html).toContain("class=\"katex-display\"");
    expect(html.match(/∫/g)).toHaveLength(4); // MathML and visual HTML each contain both integrals.
  });

  it("does not interpret math delimiters inside inline or fenced code", () => {
    const markdown = "`\\(not math\\)`\n\n```text\n\\[not math\\]\n```";
    expect(normalizePreviewMath(markdown)).toBe(markdown);
  });

  it("preserves emphasis, code, and formulas in short pipe-table cells", () => {
    const html = render("| Type | State | Expression |\n| --- | --- | --- |\n| **Yes** | *No* | $x^2 + 1$ |\n| ***Both*** | `code` | \\(\\frac{1}{2}\\) |");
    expect(html).toContain('<div class="markdown-table-scroll"><table>');
    expect(html).toContain("<td><strong>Yes</strong></td>");
    expect(html).toContain("<td><em>No</em></td>");
    expect(html).toContain("<td><em><strong>Both</strong></em></td>");
    expect(html).toContain("<td><code>code</code></td>");
    expect(html.match(/class="katex"/g)).toHaveLength(2);
    expect(html).not.toContain("katex-error");
  });

  it("renders safe line breaks and list markers inside table cells", () => {
    const html = render("| Notes |\n| --- |\n| First<br>Second<br />Third |\n| <ul><li>One</li><li>**Two**</li></ul> |");
    expect(html).toMatch(/First<br\/>\s*Second<br\/>\s*Third/);
    expect(html).toMatch(/• One<br\/>\s*• <strong>Two<\/strong>/);
  });

  it("keeps currency in tables from swallowing breaks as math", () => {
    const html = render("| Questions |\n| --- |\n| **Market cap > $5 bn?** <br> **Cash flow > $10 bn?** |");
    expect(html).toContain("Market cap &gt; $5 bn?");
    expect(html).toContain("Cash flow &gt; $10 bn?");
    expect(html).toContain("<br/>");
    expect(html).not.toContain('class="katex"');
  });

  it("still renders dollar-delimited numeric formulas", () => {
    const html = render("Calculate $5 + 2 = 7$, $5x$, and $5 (x+1)$.");
    expect(html.match(/class="katex"/g)).toHaveLength(3);
    expect(html).not.toContain("katex-error");
  });

  it("keeps ordinary currency and code samples readable", () => {
    const html = render("The price is $5 and the budget is $10. Code: `$5`.");
    expect(html).toContain("$5 and the budget is $10");
    expect(html).toContain("<code>$5</code>");
    expect(html).not.toContain('class="katex"');
  });

  it("renders escaped dollar signs inside table formulas", () => {
    const html = render(String.raw`| Item | Cost |
| --- | --- |
| Storage | **$500\text{ GB} \times \$0.015 = \$7.50$** |`);
    expect(html).toContain("<strong>");
    expect(html).toContain('class="katex"');
    expect(html).toContain("$");
    expect(html).not.toContain("katex-error");
  });

  it("does not render arbitrary HTML in table cells", () => {
    const html = render("| Text |\n| --- |\n| Safe<br><img src=x onerror=alert(1)> |");
    expect(html).toContain("Safe<br/>");
    expect(html).toContain("&lt;img src=x onerror=alert(1)&gt;");
    expect(html).not.toContain("<img");
  });
});
