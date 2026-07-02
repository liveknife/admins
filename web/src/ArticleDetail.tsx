import { useEffect, useMemo, useState } from "react";
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

  // 拉取同分类文章作为“相关阅读”
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

  const rendered = useMemo(() => {
    if (!article) return { html: "", headings: [] as Heading[] };
    const source = article.markdown_content?.trim()
      ? article.markdown_content
      : article.content;
    return renderMarkdown(source ?? "");
  }, [article]);

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
        <div className="articleLoading">正在加载文章...</div>
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
          <span>{formatDate(article.published_at)}</span>
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
          className="articleBody"
          // dangerouslySetInnerHTML 的输入完全来自我们的 Markdown 渲染器；渲染器已 escapeHtml
          dangerouslySetInnerHTML={{ __html: rendered.html }}
        />
        <aside className="articleAside">
          {rendered.headings.length > 0 && (
            <div className="articleToc">
              <h4>目录</h4>
              <ul>
                {rendered.headings
                  .filter(h => h.level <= 3)
                  .map(h => (
                    <li key={h.slug} className={`lvl-${h.level}`}>
                      <a href={`#${h.slug}`}>{h.text}</a>
                    </li>
                  ))}
              </ul>
            </div>
          )}
          {related.length > 0 && (
            <div className="articleRelated">
              <h4>相关阅读</h4>
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
    </section>
  );
}

const ensureMeta = (name: string) => {
  let el = document.querySelector<HTMLMetaElement>(`meta[name="${name}"]`);
  if (!el) {
    el = document.createElement("meta");
    el.setAttribute("name", name);
    document.head.appendChild(el);
  }
  return el;
};
