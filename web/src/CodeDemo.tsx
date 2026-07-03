import { useEffect, useState } from "react";
import { renderMarkdown } from "./markdown";

const DEMO_MARKDOWN = `
# 代码高亮演示

这是一篇演示文章，展示 **Shiki** 语法高亮效果。

## JavaScript / TypeScript

\`\`\`typescript
// React 组件示例
import { useState, useCallback } from "react";

interface Props {
  initialCount?: number;
  label?: string;
}

export function Counter({ initialCount = 0, label = "计数器" }: Props) {
  const [count, setCount] = useState(initialCount);
  
  const increment = useCallback(() => {
    setCount(c => c + 1);
  }, []);

  const decrement = useCallback(() => {
    setCount(c => c - 1);
  }, []);

  return (
    <div className="counter">
      <h2>{label}</h2>
      <button onClick={decrement}>-</button>
      <span>{count}</span>
      <button onClick={increment}>+</button>
    </div>
  );
}
\`\`\`

## Go 语言示例

\`\`\`go
package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Article struct {
	ID      int    \`json:"id"\`
	Title   string \`json:"title"\`
	Content string \`json:"content"\`
	Slug    string \`json:"slug"\`
}

func handleArticles(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	articles := []Article{
		{ID: 1, Title: "入门指南", Content: "...", Slug: "getting-started"},
		{ID: 2, Title: "进阶技巧", Content: "...", Slug: "advanced-tips"},
	}
	
	json.NewEncoder(w).Encode(map[string]interface{}{
		"items": articles,
		"total": len(articles),
	})
}

func main() {
	http.HandleFunc("/api/articles", handleArticles)
	fmt.Println("Server running at :8080")
	http.ListenAndServe(":8080", nil)
}
\`\`\`

## Python 示例

\`\`\`python
from dataclasses import dataclass
from typing import List, Optional
import asyncio


@dataclass
class User:
    id: int
    name: str
    email: str
    is_active: bool = True
    
    def display(self) -> str:
        status = "active" if self.is_active else "inactive"
        return f"[{status}] {self.name} <{self.email}>"


async def fetch_users(limit: int = 10) -> List[User]:
    """异步获取用户列表"""
    # 模拟 API 调用
    await asyncio.sleep(0.1)
    
    users = [
        User(id=i, name=f"User {i}", email=f"user{i}@example.com")
        for i in range(1, limit + 1)
    ]
    return users


# 使用示例
if __name__ == "__main__":
    result = asyncio.run(fetch_users(5))
    for user in result:
        print(user.display())
\`\`\`

## SQL 查询示例

\`\`\`sql
-- 用户文章统计查询
SELECT 
    u.id AS user_id,
    u.username,
    u.email,
    COUNT(a.id) AS article_count,
    SUM(a.view_count) AS total_views,
    MAX(a.published_at) AS latest_post
FROM users u
LEFT JOIN articles a ON u.id = a.user_id
WHERE u.is_active = true
  AND a.status = 'published'
GROUP BY u.id, u.username, u.email
HAVING COUNT(a.id) > 0
ORDER BY total_views DESC
LIMIT 20;
\`\`\`

## CSS 样式示例

\`\`\`css
/* 现代卡片组件样式 */
.article-card {
  --card-radius: 12px;
  --card-shadow: 0 8px 32px rgba(0, 0, 0, 0.12);
  --card-bg: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  
  display: grid;
  gap: 1.5rem;
  padding: 2rem;
  border-radius: var(--card-radius);
  background: var(--card-bg);
  box-shadow: var(--card-shadow);
  transition: transform 0.3s cubic-bezier(0.34, 1.56, 0.64, 1);
}

.article-card:hover {
  transform: translateY(-4px) scale(1.02);
}

.article-card::before {
  content: "";
  position: absolute;
  inset: 0;
  border-radius: inherit;
  padding: 1px;
  background: linear-gradient(135deg, rgba(255,255,255,0.4), transparent);
  mask: linear-gradient(#fff 0 0) content-box, linear-gradient(#fff 0 0);
  mask-composite: exclude;
  pointer-events: none;
}

@media (prefers-reduced-motion: reduce) {
  .article-card {
    transition: none;
  }
}
\`\`\`

## Bash 脚本示例

\`\`\`bash
#!/bin/bash

set -euo pipefail

# 项目部署脚本
PROJECT_DIR="/opt/myapp"
BACKUP_DIR="/opt/backups"
TIMESTAMP=\$(date +%Y%m%d_%H%M%S)
LOG_FILE="deploy_\${TIMESTAMP}.log"

log() {
    echo "[\$(date '+%Y-%m-%d %H:%M:%S')] \$*" | tee -a "\$LOG_FILE"
}

backup_database() {
    log "Creating database backup..."
    pg_dump -U admin myapp > "\${BACKUP_DIR}/db_\${TIMESTAMP}.sql"
    log "Backup saved to \${BACKUP_DIR}/db_\${TIMESTAMP}.sql"
}

deploy_app() {
    log "Deploying new version..."
    
    # 拉取最新代码
    cd "\$PROJECT_DIR" && git pull origin main
    
    # 安装依赖
    pnpm install --frozen-lockfile
    
    # 构建
    pnpm run build
    
    # 重启服务
    systemctl restart myapp
    log "Deployment complete!"
}

main() {
    backup_database
    deploy_app
    log "All done! Check \${LOG_FILE} for details."
}

main
\`\`\`

## JSON 配置示例

\`\`\`json
{
  "name": "admins-web",
  "version": "1.0.0",
  "type": "module",
  "scripts": {
    "dev": "vite --host 0.0.0.0 --port 5174",
    "build": "tsc -b && vite build",
    "preview": "vite preview"
  },
  "dependencies": {
    "react": "^19.2.1",
    "react-dom": "^19.2.1",
    "shiki": "^4.3.0"
  },
  "devDependencies": {
    "@types/react": "^19.2.7",
    "typescript": "^5.9.3"
  }
}
\`\`\`

---

> **提示**：以上所有代码块都由 [Shiki](https://shiki.style/) 渲染，使用 **GitHub Dark** 主题。
> 支持的语言包括：JavaScript、TypeScript、Go、Python、SQL、CSS、Bash、JSON 等 20+ 种语言。
`;

export function CodeDemoPage() {
  const [html, setHtml] = useState("");
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    let cancelled = false;
    
    renderMarkdown(DEMO_MARKDOWN).then(result => {
      if (!cancelled) {
        setHtml(result.html);
        setLoading(false);
      }
    });

    return () => { cancelled = true; };
  }, []);

  return (
    <section className="articlePage">
      <header className="articleHeader">
        <span className="articleCategory">Demo</span>
        <h1>Shiki 语法高亮演示</h1>
        <p className="articleSummary">
          展示 Markdown 渲染 + 多语言代码高亮效果。使用 shiki 库，GitHub Dark 主题。
        </p>
        <div className="articleMeta">
          <span>2026-07-03</span>
          <span>Demo Page</span>
        </div>
      </header>

      <div className="articleLayout">
        <article className="articleBody">
          {loading ? (
            <div style={{ textAlign: "center", padding: "60px", color: "#94a3b8" }}>
              <div className="loadingSpinner" />
              <p style={{ marginTop: "16px" }}>正在渲染代码高亮...</p>
            </div>
          ) : (
            <div dangerouslySetInnerHTML={{ __html: html }} />
          )}
        </article>
        <aside className="articleAside">
          <nav className="articleToc" aria-label="目录">
            <h4>支持的语言</h4>
            <ul>
              <li><a href="#javascript-typescript">JavaScript / TypeScript</a></li>
              <li><a href="#go-语言示例">Go 语言</a></li>
              <li><a href="#python-示例">Python</a></li>
              <li><a href="#sql-查询示例">SQL</a></li>
              <li><a href="#css-样式示例">CSS</a></li>
              <li><a href="#bash-脚本示例">Bash</a></li>
              <li><a href="#json-配置示例">JSON</a></li>
            </ul>
          </nav>
        </aside>
      </div>
    </section>
  );
}
