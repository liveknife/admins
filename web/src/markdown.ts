/**
 * Markdown → HTML 渲染器（带 Shiki 语法高亮）。
 * 覆盖官网文章展示需要的语法：标题、列表、有序列表、引用、代码块（带语言 + 高亮）、
 * 行内代码、粗体、斜体、删除线、链接、图片、水平线、段落、内嵌 HTML 里的换行。
 */

import { createHighlighter, type Highlighter } from "shiki";

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

// ─── Shiki 高亮器单例 ──────────────────────────────

let highlighterPromise: Promise<Highlighter> | null = null;

function getHighlighter(): Promise<Highlighter> {
  if (!highlighterPromise) {
    highlighterPromise = createHighlighter({
      themes: ["github-dark"],
      langs: [
        "javascript",
        "typescript",
        "jsx",
        "tsx",
        "html",
        "css",
        "json",
        "markdown",
        "bash",
        "python",
        "go",
        "rust",
        "java",
        "sql",
        "yaml",
        "xml",
        "c",
        "cpp",
        "shell"
      ]
    });
  }
  return highlighterPromise;
}

async function highlightCode(code: string, lang: string): Promise<string> {
  try {
    const hl = await getHighlighter();
    const resolvedLang = hl.getLoadedLanguages().includes(lang)
      ? lang
      : "text";
    return hl.codeToHtml(code, {
      lang: resolvedLang,
      theme: "github-dark"
    });
  } catch {
    // 降级：无高亮纯文本
    return `<pre class="md-code" data-lang="${escapeHtml(lang || "text")}"><code>${escapeHtml(code)}</code></pre>`;
  }
}

// ─── 渲染 Token 列表为 HTML（异步） ─────────────────

const renderTokensAsync = async (tokens: Token[]): Promise<string> => {
  const parts: string[] = [];

  for (const token of tokens) {
    switch (token.type) {
      case "heading":
        parts.push(`<h${token.level} id="${token.slug}">${applyInline(token.text)}</h${token.level}>`);
        break;

      case "code": {
        const highlighted = await highlightCode(token.body, token.lang);
        parts.push(highlighted);
        break;
      }

      case "ul":
        parts.push(`<ul>${token.items.map(item => `<li>${applyInline(item)}</li>`).join("")}</ul>`);
        break;

      case "ol":
        parts.push(`<ol>${token.items.map(item => `<li>${applyInline(item)}</li>`).join("")}</ol>`);
        break;

      case "quote":
        parts.push(`<blockquote>${applyInline(token.text)}</blockquote>`);
        break;

      case "hr":
        parts.push("<hr />");
        break;

      case "paragraph":
      default:
        parts.push(`<p>${applyInline(token.text)}</p>`);
        break;
    }
  }

  return parts.join("\n");
};

export type Heading = { level: number; text: string; slug: string };

export const renderMarkdown = async (
  source: string
): Promise<{ html: string; headings: Heading[] }> => {
  const tokens = tokenize(source ?? "");
  const headings: Heading[] = tokens
    .filter((t): t is Extract<Token, { type: "heading" }> => t.type === "heading")
    .map(({ level, text, slug }) => ({ level, text, slug }));
  const html = await renderTokensAsync(tokens);
  return { html, headings };
};
