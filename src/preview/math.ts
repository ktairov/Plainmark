function isEscaped(source: string, index: number) {
  let backslashes = 0;
  for (let previous = index - 1; previous >= 0 && source[previous] === "\\"; previous -= 1) backslashes += 1;
  return backslashes % 2 === 1;
}

function mathCurrency(source: string) {
  let result = "";
  for (let index = 0; index < source.length; index += 1) {
    // Keep an escaped dollar inside TeX without leaving a dollar token that
    // micromark could mistake for another Markdown math delimiter.
    if (source[index] === "\\" && source[index + 1] === "$" && !isEscaped(source, index)) {
      result += "\\text{\\textdollar}";
      index += 1;
    } else result += source[index];
  }
  return result;
}

function dollarClose(source: string, start: number, width: number) {
  for (let index = start + width; index < source.length;) {
    if (source[index] !== "$" || isEscaped(source, index)) { index += 1; continue; }
    let end = index + 1;
    while (source[end] === "$") end += 1;
    if (end - index === width) return index;
    index = end;
  }
  return -1;
}

function isCurrencyDollar(source: string, index: number) {
  if (isEscaped(source, index) || source[index - 1] === "$") return false;
  const amount = /^\$\d[\d,.]*/.exec(source.slice(index));
  if (!amount) return false;
  const remainder = source.slice(index + amount[0].length);
  if (remainder.startsWith("$") || /^[a-zA-Z]+\$/.test(remainder)) return false;
  const next = remainder.trimStart();
  if (next.startsWith("(")) {
    const parenthetical = /^\(([^)\n]*)\)([^\n]*)/.exec(next);
    if (parenthetical && /\$/.test(parenthetical[2]) &&
      (/^\s*\$/.test(parenthetical[2]) || /[+*/=^_\\]/.test(parenthetical[1]))) return false;
  } else if (/^[+\-*/=^_{}\\[%]/.test(next)) return false;
  if (/^[<>]/.test(next) && !/^<br\s*\/?\s*>/i.test(next)) return false;
  return true;
}

function normalizeMathSegment(source: string) {
  let result = "";
  let codeTicks = 0;
  for (let index = 0; index < source.length;) {
    if (source[index] === "`") {
      let end = index + 1;
      while (source[end] === "`") end += 1;
      const count = end - index;
      if (codeTicks === 0) codeTicks = count;
      else if (count === codeTicks) codeTicks = 0;
      result += source.slice(index, end);
      index = end;
      continue;
    }
    if (!codeTicks && source[index] === "$" && isCurrencyDollar(source, index)) {
      result += "\\$";
      index += 1;
      continue;
    }
    if (!codeTicks && source[index] === "$" && !isEscaped(source, index)) {
      let width = 1;
      while (source[index + width] === "$") width += 1;
      const end = dollarClose(source, index, width);
      if (end !== -1) {
        const delimiter = "$".repeat(width);
        result += delimiter + mathCurrency(source.slice(index + width, end)) + delimiter;
        index = end + width;
        continue;
      }
    }
    const opener = source.slice(index, index + 2);
    if (!codeTicks && (opener === "\\(" || opener === "\\[") && source[index - 1] !== "\\") {
      const closer = opener === "\\(" ? "\\)" : "\\]";
      const end = source.indexOf(closer, index + 2);
      if (end !== -1) {
        const formula = mathCurrency(source.slice(index + 2, end).trim());
        if (formula) {
          const before = source.slice(source.lastIndexOf("\n", index - 1) + 1, index);
          const nextLine = source.indexOf("\n", end + 2);
          const after = source.slice(end + 2, nextLine === -1 ? undefined : nextLine);
          const display = opener === "\\[" && !before.trim() && !after.trim();
          result += display ? `$$\n${before}${formula}\n${before}$$` : `$${formula}$`;
          index = end + 2;
          continue;
        }
      }
    }
    result += source[index];
    index += 1;
  }
  return result;
}

/** Normalizes common model-generated math without touching inline or fenced code. */
export function normalizePreviewMath(markdown: string) {
  let result = "";
  let plain = "";
  let fence: { marker: string; length: number } | null = null;
  for (const line of markdown.match(/[^\n]*(?:\n|$)/g) || []) {
    if (!line) continue;
    const marker = /^ {0,3}(`{3,}|~{3,})/.exec(line)?.[1];
    if (!fence && marker) {
      result += normalizeMathSegment(plain) + line;
      plain = "";
      fence = { marker: marker[0], length: marker.length };
    } else if (fence) {
      result += line;
      const closing = line.trim();
      if (closing.length >= fence.length && [...closing].every(character => character === fence?.marker)) fence = null;
    } else plain += line;
  }
  return result + normalizeMathSegment(plain);
}
