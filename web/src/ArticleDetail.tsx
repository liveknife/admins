import { useCallback, useEffect, useRef, useState } from "react";
import { renderMarkdown, type Heading } from "./markdown";
import {
  apiBase,
  articleRouteHash,
  splitTags,
  type SiteResource
} from "./shared";
import { navigate, searchHash } from "./router";

const formatDate = (value?: string) => {
  if (!value) return "";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "";
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, "0")}-${String(date.getDate()).padStart(2, "0")}`;
};

type Props = {
  slug: string;
};

export function ArticleDetail({ slug }: Props) {
  const [article, setArticle] = useState<SiteResource | null>(null);
  const [status, setStatus] = useState<"loading" | "ok" | "notfound" | "error">("loading");
  const [related, setRelated] = useState<SiteResource[]>([]);
  const [activeHeadingId, setActiveHeadingId] = useState<string>("");
  const [codeExplain, setCodeExplain] = useState<{ open: boolean; loading: boolean; answer: string; code: string; language: string }>({
    open: false,
    loading: false,
    answer: "",
    code: "",
    language: ""
  });
  const bodyRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    let cancelled = false;
    setStatus("loading");
    setArticle(null);
    fetch(`${apiBase}/api/v1/site/resources/${encodeURIComponent(slug)}`)
      .then(async res => {
        if (cancelled) return;
        if (res.status === 404) {
          setStatus("notfound");
          return;
        }
        if (!res.ok) throw new Error(String(res.status));
        const data = await res.json();
        setArticle(data.resource ?? null);
        setStatus("ok");
      })
      .catch(() => {
        if (!cancelled) setStatus("error");
      });
    return () => {
      cancelled = true;
    };
  }, [slug]);

  // 拉取同分类文章作为"相关阅读"
  useEffect(() => {
    if (!article?.category) {
      setRelated([]);
      return;
    }
    const url = `${apiBase}/api/v1/site/search?q=${encodeURIComponent(article.category)}&category=${encodeURIComponent(article.category)}&page_size=4`;
    fetch(url)
      .then(res => res.json())
      .then(data => {
        const items = (data?.items ?? []).filter(
          (item: SiteResource) => item.id !== article.id
        );
        setRelated(items.slice(0, 3));
      })
      .catch(() => setRelated([]));
  }, [article?.id, article?.category]);

  // 异步渲染 Markdown（含 Shiki 高亮）
  const [rendered, setRendered] = useState<{ html: string; headings: Heading[] }>({
    html: "",
    headings: []
  });

  useEffect(() => {
    if (!article) {
      setRendered({ html: "", headings: [] });
      return;
    }

    let cancelled = false;
    const source = article.markdown_content?.trim()
      ? article.markdown_content
      : article.content;

    renderMarkdown(source ?? "").then(result => {
      if (!cancelled) {
        setRendered(result);
      }
    });

    return () => { cancelled = true; };
  }, [article]);

  // ─── 目录滚动高亮 ──────────────────────────────
  const observeHeadings = useCallback(() => {
    if (bodyRef.current === null || rendered.headings.length === 0) return;

    const observer = new IntersectionObserver(
      entries => {
        // 找到当前可见且最上方的标题
        const visible = entries.filter(e => e.isIntersecting);
        if (visible.length > 0) {
          // 按位置排序，取最靠前的
          visible.sort((a, b) => a.boundingClientRect.top - b.boundingClientRect.top);
          setActiveHeadingId(visible[0].target.id);
        }
      },
      {
        rootMargin: "-80px 0px -60% 0px",
        threshold: 0
      }
    );

    // 观察所有 h2/h3 标题
    rendered.headings
      .filter(h => h.level <= 3)
      .forEach(h => {
        const el = bodyRef.current?.querySelector(`#${CSS.escape(h.slug)}`);
        if (el) observer.observe(el);
      });

    return () => observer.disconnect();
  }, [rendered.headings]);

  // DOM 更新后重新绑定观察器
  useEffect(() => {
    const cleanup = observeHeadings();
    return cleanup;
  }, [observeHeadings, rendered.html]);

  useEffect(() => {
    const root = bodyRef.current;
    if (!root) return;
    root.querySelectorAll(".codeExplainButton").forEach(node => node.remove());
    root.querySelectorAll("pre").forEach(pre => {
      const codeEl = pre.querySelector("code");
      const code = codeEl?.textContent || pre.textContent || "";
      if (!code.trim()) return;
      pre.classList.add("codeBlockWithAI");
      const button = document.createElement("button");
      button.type = "button";
      button.className = "codeExplainButton";
      button.textContent = "AI 解释";
      button.addEventListener("click", () => explainCode(code, codeLanguage(codeEl)));
      pre.appendChild(button);
    });
  }, [rendered.html]);

  const explainCode = async (code: string, language: string) => {
    setCodeExplain({ open: true, loading: true, answer: "", code, language });
    try {
      const res = await fetch(`${apiBase}/api/v1/site/code-explain`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ code, language })
      });
      const data = await res.json();
      setCodeExplain({ open: true, loading: false, answer: data.answer || "", code, language });
    } catch {
      setCodeExplain({ open: true, loading: false, answer: "解释失败，请稍后重试。", code, language });
    }
  };

  // 同步 <title> 与 <meta>，便于收藏和分享
  useEffect(() => {
    if (!article) return;
    const originalTitle = document.title;
    document.title = article.seo_title || article.title;
    const metaDesc = ensureMeta("description");
    const metaKw = ensureMeta("keywords");
    metaDesc.setAttribute("content", article.seo_description || article.summary || "");
    metaKw.setAttribute("content", article.seo_keywords || article.tags || "");
    return () => {
      document.title = originalTitle;
    };
  }, [article]);

  if (status === "loading") {
    return (
      <section className="articlePage">
        <div className="articleLoading">
          <div className="loadingSpinner" />
          <p>正在加载文章...</p>
        </div>
      </section>
    );
  }
  if (status === "notfound") {
    return (
      <section className="articlePage">
        <div className="articleEmpty">
          <h1>找不到这篇文章</h1>
          <p>它可能被删除或还没有发布。</p>
          <a href="#/">回到首页</a>
        </div>
      </section>
    );
  }
  if (status === "error" || !article) {
    return (
      <section className="articlePage">
        <div className="articleEmpty">
          <h1>加载失败</h1>
          <a href="#/">回到首页</a>
        </div>
      </section>
    );
  }

  const tags = splitTags(article.tags);

  return (
    <section className="articlePage">
      <nav className="articleCrumbs">
        <a href="#/">首页</a>
        <span> / </span>
        <a href={searchHash("", { category: article.category })}>
          {article.category || "文章"}
        </a>
        <span> / </span>
        <span>{article.title}</span>
      </nav>

      <header className="articleHeader">
        <span className="articleCategory">{article.category}</span>
        <h1>{article.title}</h1>
        {article.summary && <p className="articleSummary">{article.summary}</p>}
        <div className="articleMeta">
          <time>{formatDate(article.published_at)}</time>
          <span>{article.view_count} 次阅读</span>
        </div>
        {tags.length > 0 && (
          <div className="tagRow">
            {tags.map(tag => (
              <a key={tag} href={searchHash("", { tag })}>
                #{tag}
              </a>
            ))}
          </div>
        )}
      </header>

      <div className="articleLayout">
        <article
          ref={bodyRef}
          className="articleBody"
          dangerouslySetInnerHTML={{ __html: rendered.html }}
        />
        <aside className="articleAside">
          {/* 目录（带滚动高亮） */}
          {rendered.headings.length > 0 && (
            <nav className="articleToc" aria-label="文章目录">
              <h4>
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <line x1="8" y1="6" x2="21" y2="6" /><line x1="8" y1="12" x2="21" y2="12" />
                  <line x1="8" y1="18" x2="21" y2="18" /><line x1="3" y1="6" x2="3.01" y2="6" />
                  <line x1="3" y1="12" x2="3.01" y2="12" /><line x1="3" y1="18" x2="3.01" y2="18" />
                </svg>
                目录
              </h4>
              <ul>
                {rendered.headings
                  .filter(h => h.level <= 3)
                  .map(h => (
                    <li
                      key={h.slug}
                      className={`lvl-${h.level}${h.slug === activeHeadingId ? " active" : ""}`}
                    >
                      <a href={`#${h.slug}`}>{h.text}</a>
                    </li>
                  ))}
              </ul>
            </nav>
          )}

          {/* 相关阅读 */}
          {related.length > 0 && (
            <div className="articleRelated">
              <h4>
                <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round">
                  <path d="M4 19.5A2.5 2.5 0 0 1 6.5 17H20" /><path d="M6.5 2H20v20H6.5A2.5 2.5 0 0 1 4 19.5v-15A2.5 2.5 0 0 1 6.5 2z" />
                </svg>
                相关阅读
              </h4>
              <ul>
                {related.map(item => (
                  <li key={item.id}>
                    <a
                      href={articleRouteHash(item)}
                      onClick={event => {
                        event.preventDefault();
                        navigate(articleRouteHash(item));
                      }}
                    >
                      {item.title}
                    </a>
                  </li>
                ))}
              </ul>
            </div>
          )}
        </aside>
      </div>
      {codeExplain.open && (
        <div className="codeExplainDrawer">
          <div>
            <strong>AI 代码解释</strong>
            <button type="button" onClick={() => setCodeExplain(prev => ({ ...prev, open: false }))}>关闭</button>
          </div>
          <small>{codeExplain.language || "code"} · {codeExplain.code.split("\n").length} lines</small>
          {codeExplain.loading ? <p>正在分析...</p> : <p>{codeExplain.answer}</p>}
        </div>
      )}
    </section>
  );
}

const codeLanguage = (codeEl: Element | null) => {
  const className = codeEl?.className || "";
  const match = String(className).match(/language-([a-z0-9_-]+)/i);
  return match?.[1] || "text";
};

const ensureMeta = (name: string) => {
  let el = document.querySelector<HTMLMetaElement>(`meta[name="${name}"]`);
  if (!el) {
    el = document.createElement("meta");
    el.setAttribute("name", name);
    document.head.appendChild(el);
  }
  return el;
};
