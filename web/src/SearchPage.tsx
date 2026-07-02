import { useEffect, useMemo, useState } from "react";
import {
  apiBase,
  articleRouteHash,
  splitTags,
  type SearchResponse,
  type SiteResource
} from "./shared";
import { navigate, searchHash } from "./router";

type Props = {
  query: string;
  category: string;
  tag: string;
  page: number;
};

const PAGE_SIZE = 10;

export function SearchPage({ query, category, tag, page }: Props) {
  const [input, setInput] = useState(query);
  const [data, setData] = useState<SearchResponse | null>(null);
  const [status, setStatus] = useState<"idle" | "loading" | "ok" | "error">(
    query ? "loading" : "idle"
  );

  useEffect(() => {
    setInput(query);
  }, [query]);

  useEffect(() => {
    if (!query) {
      setData(null);
      setStatus("idle");
      return;
    }
    let cancelled = false;
    setStatus("loading");
    const params = new URLSearchParams({
      q: query,
      page: String(page),
      page_size: String(PAGE_SIZE)
    });
    if (category) params.set("category", category);
    if (tag) params.set("tag", tag);
    fetch(`${apiBase}/api/v1/site/search?${params.toString()}`)
      .then(res => (res.ok ? res.json() : Promise.reject()))
      .then((res: SearchResponse) => {
        if (cancelled) return;
        setData(res);
        setStatus("ok");
      })
      .catch(() => {
        if (!cancelled) setStatus("error");
      });
    return () => {
      cancelled = true;
    };
  }, [query, category, tag, page]);

  const totalPages = useMemo(() => {
    if (!data) return 0;
    return Math.max(1, Math.ceil(data.total / PAGE_SIZE));
  }, [data]);

  const submit = () => {
    const q = input.trim();
    navigate(searchHash(q, { category, tag }));
  };

  return (
    <section className="searchPage">
      <div className="searchHeader">
        <a href="#/" className="backLink">
          ← 返回首页
        </a>
        <h1>搜索文章</h1>
        <p>基于全部已发布内容做实时匹配。</p>
        <form
          className="searchBar"
          onSubmit={event => {
            event.preventDefault();
            submit();
          }}
        >
          <input
            value={input}
            onChange={event => setInput(event.target.value)}
            placeholder="输入关键词，比如 React、Go、状态管理..."
            autoFocus
          />
          <button type="submit">搜索</button>
        </form>
        {(category || tag) && (
          <div className="searchFilters">
            {category && (
              <span className="filterChip">
                分类：{category}
                <button
                  type="button"
                  onClick={() => navigate(searchHash(query, { tag }))}
                >
                  ×
                </button>
              </span>
            )}
            {tag && (
              <span className="filterChip">
                标签：#{tag}
                <button
                  type="button"
                  onClick={() => navigate(searchHash(query, { category }))}
                >
                  ×
                </button>
              </span>
            )}
          </div>
        )}
      </div>

      {status === "idle" && (
        <div className="searchHint">输入关键词开始搜索。</div>
      )}
      {status === "loading" && <div className="searchHint">搜索中...</div>}
      {status === "error" && <div className="searchHint">搜索失败，请重试。</div>}

      {status === "ok" && data && (
        <>
          <div className="searchSummary">
            共 <strong>{data.total}</strong> 条结果匹配 “{data.query}”
          </div>
          <div className="searchList">
            {data.items.map(item => (
              <ResultCard key={item.id} item={item} />
            ))}
            {data.items.length === 0 && (
              <p className="searchHint">
                没有找到匹配的内容。可以换个词，比如 React、Go、状态管理。
              </p>
            )}
          </div>

          {totalPages > 1 && (
            <div className="searchPager">
              <button
                type="button"
                disabled={page <= 1}
                onClick={() =>
                  navigate(
                    searchHash(query, { category, tag, page: page - 1 })
                  )
                }
              >
                上一页
              </button>
              <span>
                第 {page} / {totalPages} 页
              </span>
              <button
                type="button"
                disabled={page >= totalPages}
                onClick={() =>
                  navigate(
                    searchHash(query, { category, tag, page: page + 1 })
                  )
                }
              >
                下一页
              </button>
            </div>
          )}
        </>
      )}
    </section>
  );
}

function ResultCard({ item }: { item: SiteResource }) {
  const tags = splitTags(item.tags).slice(0, 5);
  return (
    <article className="resultCard">
      <div className="resultMeta">
        <span>{item.category}</span>
        <span>{item.view_count ?? 0} reads</span>
      </div>
      <h3>
        <a
          href={articleRouteHash(item)}
          onClick={event => {
            event.preventDefault();
            navigate(articleRouteHash(item));
          }}
        >
          {item.title}
        </a>
      </h3>
      <p>{item.summary}</p>
      <div className="tagRow">
        {tags.map(tag => (
          <a key={tag} href={searchHash("", { tag })}>
            #{tag}
          </a>
        ))}
      </div>
    </article>
  );
}
