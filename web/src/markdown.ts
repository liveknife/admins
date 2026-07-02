/**
 * 极简 Markdown → HTML 渲染器。
 * 覆盖官网文章展示需要的语法：标题、列表、有序列表、引用、代码块（带语言）、
 * 行内代码、粗体、斜体、删除线、链接、图片、水平线、段落、内嵌 HTML 里的换行。
 * 有意保持无依赖：无 remark/marked/highlight.js。
 * 对不认识的语法直接以段落文本落回，保证不会崩溃。
 */

const escapeMap: Record<string, string> = {
  "&": "&amp;",
  "<": "&lt;",
  ">": "&gt;",
  "\"": "&quot;",
  "'": "&#39;"
};
const escapeHtml = (text: string) =>
  text.replace(/[&<>"']/g, ch => escapeMap[ch] ?? ch);

// 行内替换：先处理代码，避免代码里的星号/下划线被误当作强调
const applyInline = (raw: string) => {
  const codeSlots: string[] = [];
  let escaped = escapeHtml(raw).replace(/`([^`]+)`/g, (_m, code) => {
    codeSlots.push(`<code>${code}</code>`);
    return `${codeSlots.length - 1}`;
  });

  // 图片（放在链接之前）
  escaped = escaped.replace(
    /!\[([^\]]*)\]\(([^)\s]+)(?:\s+"([^"]*)")?\)/g,
    (_m, alt, url, title) =>
      `<img src="${url}" alt="${alt}"${title ? ` title="${title}"` : ""} loading="lazy" />`
  );
  // 链接
  escaped = escaped.replace(
    /\[([^\]]+)\]\(([^)\s]+)(?:\s+"([^"]*)")?\)/g,
    (_m, label, url, title) =>
      `<a href="${url}" target="_blank" rel="noreferrer noopener"${title ? ` title="${title}"` : ""}>${label}</a>`
  );

  // 强调
  escaped = escaped
    .replace(/\*\*([^*]+)\*\*/g, "<strong>$1</strong>")
    .replace(/__([^_]+)__/g, "<strong>$1</strong>")
    .replace(/\*([^*]+)\*/g, "<em>$1</em>")
    .replace(/_([^_]+)_/g, "<em>$1</em>")
    .replace(/~~([^~]+)~~/g, "<del>$1</del>");

  // 恢复代码占位
  return escaped.replace(/(\d+)/g, (_m, i) => codeSlots[Number(i)]);
};

type Token =
  | { type: "heading"; level: number; text: string; slug: string }
  | { type: "code"; lang: string; body: string }
  | { type: "ul"; items: string[] }
  | { type: "ol"; items: string[] }
  | { type: "quote"; text: string }
  | { type: "hr" }
  | { type: "paragraph"; text: string };

const slugify = (text: string) =>
  text
    .toLowerCase()
    .trim()
    .replace(/[^\p{L}\p{N}\s-]/gu, "")
    .replace(/\s+/g, "-");

const tokenize = (source: string): Token[] => {
  const lines = source.replace(/\r\n/g, "\n").split("\n");
  const tokens: Token[] = [];
  let i = 0;

  while (i < lines.length) {
    const line = lines[i];

    if (/^\s*```/.test(line)) {
      const lang = line.trim().slice(3).trim();
      const bodyLines: string[] = [];
      i += 1;
      while (i < lines.length && !/^\s*```/.test(lines[i])) {
        bodyLines.push(lines[i]);
        i += 1;
      }
      tokens.push({ type: "code", lang, body: bodyLines.join("\n") });
      i += 1;
      continue;
    }

    const heading = /^(#{1,6})\s+(.+)$/.exec(line);
    if (heading) {
      const level = heading[1].length;
      const text = heading[2].trim();
      tokens.push({ type: "heading", level, text, slug: slugify(text) });
      i += 1;
      continue;
    }

    if (/^\s*[-*_]{3,}\s*$/.test(line)) {
      tokens.push({ type: "hr" });
      i += 1;
      continue;
    }

    if (/^\s*>\s?/.test(line)) {
      const quoteLines: string[] = [];
      while (i < lines.length && /^\s*>\s?/.test(lines[i])) {
        quoteLines.push(lines[i].replace(/^\s*>\s?/, ""));
        i += 1;
      }
      tokens.push({ type: "quote", text: quoteLines.join(" ") });
      continue;
    }

    if (/^\s*[-*]\s+/.test(line)) {
      const items: string[] = [];
      while (i < lines.length && /^\s*[-*]\s+/.test(lines[i])) {
        items.push(lines[i].replace(/^\s*[-*]\s+/, ""));
        i += 1;
      }
      tokens.push({ type: "ul", items });
      continue;
    }

    if (/^\s*\d+\.\s+/.test(line)) {
      const items: string[] = [];
      while (i < lines.length && /^\s*\d+\.\s+/.test(lines[i])) {
        items.push(lines[i].replace(/^\s*\d+\.\s+/, ""));
        i += 1;
      }
      tokens.push({ type: "ol", items });
      continue;
    }

    if (line.trim() === "") {
      i += 1;
      continue;
    }

    // 段落：合并连续非空行
    const paraLines: string[] = [line];
    i += 1;
    while (
      i < lines.length &&
      lines[i].trim() !== "" &&
      !/^(#{1,6}\s|```|\s*[-*_]{3,}\s*$|\s*>\s?|\s*[-*]\s+|\s*\d+\.\s+)/.test(lines[i])
    ) {
      paraLines.push(lines[i]);
      i += 1;
    }
    tokens.push({ type: "paragraph", text: paraLines.join(" ") });
  }

  return tokens;
};

const renderTokens = (tokens: Token[]) =>
  tokens
    .map(token => {
      switch (token.type) {
        case "heading":
          return `<h${token.level} id="${token.slug}">${applyInline(token.text)}</h${token.level}>`;
        case "code":
          return `<pre class="md-code" data-lang="${escapeHtml(token.lang || "text")}"><code>${escapeHtml(token.body)}</code></pre>`;
        case "ul":
          return `<ul>${token.items.map(item => `<li>${applyInline(item)}</li>`).join("")}</ul>`;
        case "ol":
          return `<ol>${token.items.map(item => `<li>${applyInline(item)}</li>`).join("")}</ol>`;
        case "quote":
          return `<blockquote>${applyInline(token.text)}</blockquote>`;
        case "hr":
          return "<hr />";
        case "paragraph":
        default:
          return `<p>${applyInline(token.text)}</p>`;
      }
    })
    .join("\n");

export type Heading = { level: number; text: string; slug: string };

export const renderMarkdown = (source: string): { html: string; headings: Heading[] } => {
  const tokens = tokenize(source ?? "");
  const headings: Heading[] = tokens
    .filter((t): t is Extract<Token, { type: "heading" }> => t.type === "heading")
    .map(({ level, text, slug }) => ({ level, text, slug }));
  return { html: renderTokens(tokens), headings };
};
