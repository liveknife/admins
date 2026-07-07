import React, { useEffect, useMemo, useState } from "react";
import { createRoot } from "react-dom/client";
import * as THREE from "three";
import "./styles.css";
import {
  apiBase,
  articleRouteHash,
  assetURL,
  emptyHome,
  splitTags,
  type SiteHome,
  type SiteProject,
  type SiteResource,
  type SiteTechStack,
  type SiteTimelineEvent,
  type SiteMessage,
  type KnowledgeAnswer
} from "./shared";
import { navigate, searchHash, useHashRoute } from "./router";
import { ArticleDetail } from "./ArticleDetail";
import { SearchPage } from "./SearchPage";
import { CodeDemoPage } from "./CodeDemo";

type AnswerBlock =
  | { type: "heading"; text: string }
  | { type: "paragraph"; text: string }
  | { type: "list"; items: string[] }
  | { type: "code"; language: string; code: string };

function answerTitle(question: string, answer?: string) {
  const firstHeading = answer?.split(/\r?\n/).find(line => /^#{1,4}\s+/.test(line.trim()));
  if (firstHeading) return firstHeading.replace(/^#{1,4}\s+/, "").trim();
  return question.trim() ? `\u56de\u7b54\uff1a${question.trim()}` : "\u77e5\u8bc6\u5e93\u56de\u7b54";
}

function answerBlocks(markdown?: string): AnswerBlock[] {
  const text = (markdown || "").trim();
  if (!text) return [];
  const blocks: AnswerBlock[] = [];
  const fencePattern = /```([a-zA-Z0-9_-]*)\s*([\s\S]*?)```/g;
  let cursor = 0;
  let match: RegExpExecArray | null;
  while ((match = fencePattern.exec(text))) {
    pushTextBlocks(text.slice(cursor, match.index), blocks);
    blocks.push({ type: "code", language: match[1] || "text", code: match[2].trim() });
    cursor = match.index + match[0].length;
  }
  pushTextBlocks(text.slice(cursor), blocks);
  return blocks;
}

function pushTextBlocks(text: string, blocks: AnswerBlock[]) {
  const lines = text.split(/\r?\n/).map(line => line.trim()).filter(Boolean);
  let listItems: string[] = [];
  const flushList = () => {
    if (listItems.length) {
      blocks.push({ type: "list", items: listItems });
      listItems = [];
    }
  };
  lines.forEach(line => {
    const heading = line.match(/^#{1,4}\s+(.+)$/);
    if (heading) {
      flushList();
      blocks.push({ type: "heading", text: heading[1].trim() });
      return;
    }
    const list = line.match(/^[-*]\s+(.+)$/);
    if (list) {
      listItems.push(list[1].trim());
      return;
    }
    flushList();
    blocks.push({ type: "paragraph", text: line });
  });
  flushList();
}

function InlineMarkdown({ text }: { text: string }) {
  const parts = text.split(/(`[^`]+`|\*\*[^*]+\*\*)/g).filter(Boolean);
  return (
    <>
      {parts.map((part, index) => {
        if (part.startsWith("`") && part.endsWith("`")) {
          return <code key={index}>{part.slice(1, -1)}</code>;
        }
        if (part.startsWith("**") && part.endsWith("**")) {
          return <strong key={index}>{part.slice(2, -2)}</strong>;
        }
        return <React.Fragment key={index}>{part}</React.Fragment>;
      })}
    </>
  );
}

function AnswerContent({ answer }: { answer?: string }) {
  const blocks = answerBlocks(answer);
  if (!blocks.length) {
    return <p>{"\u8fd9\u4e2a\u5165\u53e3\u4f1a\u57fa\u4e8e\u4f60\u5728\u540e\u53f0\u53d1\u5e03\u7684\u6587\u7ae0\u3001\u7b14\u8bb0\u548c\u9879\u76ee\u5185\u5bb9\u505a\u672c\u5730\u68c0\u7d22\u5f0f\u56de\u7b54\u3002"}</p>;
  }
  return (
    <div className="answerBody">
      {blocks.map((block, index) => {
        if (block.type === "heading") {
          return <h4 key={index}><InlineMarkdown text={block.text} /></h4>;
        }
        if (block.type === "list") {
          return <ul key={index}>{block.items.map((item, itemIndex) => <li key={itemIndex}><InlineMarkdown text={item} /></li>)}</ul>;
        }
        if (block.type === "code") {
          return (
            <figure className="answerCode" key={index}>
              <figcaption>{block.language}</figcaption>
              <pre><code>{block.code}</code></pre>
            </figure>
          );
        }
        return <p key={index}><InlineMarkdown text={block.text} /></p>;
      })}
    </div>
  );
}

function KnowledgeAsk({ resources }: { resources: SiteResource[] }) {
  const [question, setQuestion] = useState("React 项目经验怎么总结？");
  const [answer, setAnswer] = useState<KnowledgeAnswer | null>(null);
  const [asking, setAsking] = useState(false);

  const ask = async () => {
    if (!question.trim()) return;
    setAsking(true);
    try {
      const res = await fetch(`${apiBase}/api/v1/site/knowledge`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ question })
      });
      const data = await res.json();
      setAnswer(data.answer ?? null);
    } finally {
      setAsking(false);
    }
  };

  return (
    <section id="ask" className="section askSection">
      <div className="sectionHead">
        <span>Knowledge Base</span>
        <h2>问我的知识库</h2>
      </div>
      <div className="askPanel">
        <div className="askInputCard">
          <textarea
            value={question}
            onChange={event => setQuestion(event.target.value)}
            placeholder="问 React、Go、数据库、项目经验..."
          />
          <button onClick={ask} disabled={asking}>{asking ? "检索中..." : "提问"}</button>
        </div>
        <div className="answerPanel">
          <div className="answerHeader">
            <span>{resources.length} {"\u7bc7\u5df2\u53d1\u5e03\u5185\u5bb9\u53ef\u68c0\u7d22"}</span>
            <h3>{answerTitle(question, answer?.answer)}</h3>
          </div>
          <AnswerContent answer={answer?.answer} />
          {!!answer?.sources?.length && (
            <div className="answerSources">
              {answer.sources.slice(0, 4).map(item => (
                <div key={`${item.source_type}-${item.source_id}-${item.title}`}>
                  {item.url ? (
                    <a href={item.url} target={item.url.startsWith("http") ? "_blank" : undefined} rel="noreferrer">
                      {item.title}
                    </a>
                  ) : (
                    <strong>{item.title}</strong>
                  )}
                  <small>{item.source_type} #{item.source_id} · {Math.round(item.score * 100)}%</small>
                  {item.highlighted_text && (
                    <p dangerouslySetInnerHTML={{ __html: item.highlighted_text }} />
                  )}
                </div>
              ))}
            </div>
          )}
          <div className="answerLinks">
            {answer?.matches?.map(item => (
              <a
                key={item.id}
                href={articleRouteHash(item)}
                onClick={event => {
                  event.preventDefault();
                  navigate(articleRouteHash(item));
                }}
              >
                {item.title}
              </a>
            ))}
            {answer?.projects?.map(item => (
              <a key={`p-${item.id}`} href={item.demo_url || item.repo_url || "#"} target="_blank" rel="noreferrer">{item.name}</a>
            ))}
          </div>
        </div>
      </div>
    </section>
  );
}

type ChatMessage = {
  id: string;
  role: "user" | "assistant";
  content: string;
  sources?: KnowledgeAnswer["sources"];
  suggestions?: string[];
  queryLogId?: number;
  feedback?: "up" | "down";
};

function AskPage() {
  const initialQuestion = new URLSearchParams(window.location.hash.split("?")[1] || "").get("q") || "";
  const [messages, setMessages] = useState<ChatMessage[]>([
    { id: "welcome", role: "assistant", content: "你好，我可以基于已发布的文章、项目、技术栈和时间线回答问题。" }
  ]);
  const [input, setInput] = useState(initialQuestion);
  const [loading, setLoading] = useState(false);

  const submit = async (value = input) => {
    const question = value.trim();
    if (!question || loading) return;
    const userMessage: ChatMessage = { id: `u-${Date.now()}`, role: "user", content: question };
    const assistantID = `a-${Date.now()}`;
    setMessages(prev => [...prev, userMessage, { id: assistantID, role: "assistant", content: "" }]);
    setInput("");
    setLoading(true);
    try {
      const context = messages.slice(-5).map(item => ({ role: item.role, content: item.content }));
      const res = await fetch(`${apiBase}/api/v1/site/knowledge/stream`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ question, context })
      });
      if (!res.body) throw new Error("stream unavailable");
      const reader = res.body.getReader();
      const decoder = new TextDecoder();
      let buffer = "";
      const applyEvent = (raw: string) => {
        const event = raw.match(/^event:\s*(.+)$/m)?.[1]?.trim() || "message";
        const dataText = raw.match(/^data:\s*([\s\S]*)$/m)?.[1]?.trim();
        if (!dataText) return;
        const data = JSON.parse(dataText);
        if (event === "token") {
          setMessages(prev => prev.map(item => item.id === assistantID ? { ...item, content: item.content + (data.content || "") } : item));
        }
        if (event === "sources") {
          setMessages(prev => prev.map(item => item.id === assistantID ? { ...item, sources: data } : item));
        }
        if (event === "suggestions") {
          setMessages(prev => prev.map(item => item.id === assistantID ? { ...item, suggestions: data } : item));
        }
        if (event === "done" && data.answer) {
          setMessages(prev => prev.map(item => item.id === assistantID ? {
            ...item,
            content: data.answer.answer || item.content,
            sources: data.answer.sources,
            suggestions: data.answer.suggestions,
            queryLogId: data.answer.query_log_id
          } : item));
        }
      };
      for (;;) {
        const { done, value: chunk } = await reader.read();
        if (done) break;
        buffer += decoder.decode(chunk, { stream: true });
        const parts = buffer.split("\n\n");
        buffer = parts.pop() || "";
        parts.forEach(applyEvent);
      }
    } finally {
      setLoading(false);
    }
  };

  const feedback = async (message: ChatMessage, rating: "up" | "down") => {
    await fetch(`${apiBase}/api/v1/site/feedback`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ query_log_id: message.queryLogId || 0, question: message.content, rating })
    }).catch(() => undefined);
    setMessages(prev => prev.map(item => item.id === message.id ? { ...item, feedback: rating } : item));
  };

  return (
    <section className="askPage">
      <header className="askTopbar">
        <a className="brand" href="#/">Tech Lab</a>
        <div>
          <a href="#/">Home</a>
          <a href="#/search">Search</a>
          <button type="button" onClick={() => setMessages([])}>New chat</button>
        </div>
      </header>
      <div className="chatShell">
        <aside className="chatSide">
          <strong>AI Ask</strong>
          <p>Use published knowledge, project notes, timeline and stack records.</p>
          {messages.filter(item => item.role === "user").slice(-6).map(item => (
            <button key={item.id} type="button" onClick={() => submit(item.content)}>{item.content}</button>
          ))}
        </aside>
        <main className="chatMain">
          <div className="chatMessages">
            {messages.map(item => (
              <article key={item.id} className={`chatBubble ${item.role}`}>
                <div className="bubbleAvatar">{item.role === "user" ? "You" : "AI"}</div>
                <div className="bubbleBody">
                  {item.content ? <AnswerContent answer={item.content} /> : <div className="typingDots"><i /><i /><i /></div>}
                  {!!item.sources?.length && <SourceCards sources={item.sources} />}
                  {item.role === "assistant" && item.content && (
                    <div className="bubbleActions">
                      <button type="button" className={item.feedback === "up" ? "active" : ""} onClick={() => feedback(item, "up")}>Good</button>
                      <button type="button" className={item.feedback === "down" ? "active" : ""} onClick={() => feedback(item, "down")}>Bad</button>
                    </div>
                  )}
                  {!!item.suggestions?.length && (
                    <div className="suggestionRow">
                      {item.suggestions.map(suggestion => <button key={suggestion} type="button" onClick={() => submit(suggestion)}>{suggestion}</button>)}
                    </div>
                  )}
                </div>
              </article>
            ))}
          </div>
          <form className="chatComposer" onSubmit={event => { event.preventDefault(); submit(); }}>
            <textarea value={input} onChange={event => setInput(event.target.value)} placeholder="Ask about articles, projects, code or timeline..." />
            <button type="submit" disabled={loading}>{loading ? "Thinking" : "Send"}</button>
          </form>
        </main>
      </div>
    </section>
  );
}

function SourceCards({ sources }: { sources: NonNullable<KnowledgeAnswer["sources"]> }) {
  const [open, setOpen] = useState(true);
  return (
    <div className="sourceBlock">
      <button type="button" onClick={() => setOpen(value => !value)}>{open ? "Hide sources" : "Show sources"} ({sources.length})</button>
      {open && (
        <div className="answerSources">
          {sources.slice(0, 4).map(item => (
            <div key={`${item.source_type}-${item.source_id}-${item.title}`}>
              {item.url ? <a href={item.url}>{item.title}</a> : <strong>{item.title}</strong>}
              <small>{item.source_type} #{item.source_id} - {Math.round(item.score * 100)}%</small>
              {item.highlighted_text && <p dangerouslySetInnerHTML={{ __html: item.highlighted_text }} />}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}

function MessageBoard({ messages }: { messages: SiteMessage[] }) {
  const [visitorName, setVisitorName] = useState("");
  const [email, setEmail] = useState("");
  const [content, setContent] = useState("");
  const [sent, setSent] = useState(false);

  const submit = async () => {
    if (!content.trim()) return;
    await fetch(`${apiBase}/api/v1/site/messages`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ visitor_name: visitorName, email, content })
    });
    setContent("");
    setSent(true);
  };

  return (
    <section id="messages" className="section messageSection">
      <div className="sectionHead">
        <span>Question Box</span>
        <h2>留言板</h2>
      </div>
      <div className="messageLayout">
        <div className="messageForm">
          <input value={visitorName} onChange={event => setVisitorName(event.target.value)} placeholder="你的名字" />
          <input value={email} onChange={event => setEmail(event.target.value)} placeholder="邮箱，可选" />
          <textarea value={content} onChange={event => setContent(event.target.value)} placeholder="想问什么，或者留一句建议。" />
          <button onClick={submit}>提交留言</button>
          {sent && <p>已提交，审核通过后会展示。</p>}
        </div>
        <div className="messageList">
          {(messages.length ? messages : []).map(item => (
            <article key={item.id}>
              <strong>{item.visitor_name || "匿名访客"}</strong>
              <p>{item.content}</p>
              {item.reply && <blockquote>{item.reply}</blockquote>}
            </article>
          ))}
          {!messages.length && <p className="empty">还没有公开留言，来做第一个提问的人。</p>}
        </div>
      </div>
    </section>
  );
}

function TechOrbit({ stacks }: { stacks: SiteTechStack[] }) {
  const mountRef = React.useRef<HTMLDivElement | null>(null);

  useEffect(() => {
    const mount = mountRef.current;
    if (!mount) return;
    const width = mount.clientWidth;
    const height = mount.clientHeight;
    const scene = new THREE.Scene();
    const camera = new THREE.PerspectiveCamera(45, width / height, 0.1, 100);
    camera.position.set(0, 0, width < 640 ? 11 : 9);

    const renderer = new THREE.WebGLRenderer({ antialias: true, alpha: true });
    renderer.setPixelRatio(Math.min(window.devicePixelRatio, 2));
    renderer.setSize(width, height);
    mount.appendChild(renderer.domElement);

    const group = new THREE.Group();
    scene.add(group);

    const palette = ["#7dd3fc", "#f0abfc", "#a7f3d0", "#facc15", "#fb7185"];
    const orbitMaterial = new THREE.LineBasicMaterial({ color: "#6b7280", transparent: true, opacity: 0.32 });
    for (let i = 1; i <= 4; i += 1) {
      const curve = new THREE.EllipseCurve(0, 0, i * 0.82, i * 0.42, 0, Math.PI * 2);
      const points = curve.getPoints(140).map(point => new THREE.Vector3(point.x, point.y, -i * 0.05));
      const orbit = new THREE.Line(new THREE.BufferGeometry().setFromPoints(points), orbitMaterial);
      orbit.rotation.x = 0.72 + i * 0.08;
      orbit.rotation.z = i * 0.42;
      group.add(orbit);
    }

    const core = new THREE.Mesh(
      new THREE.IcosahedronGeometry(0.72, 2),
      new THREE.MeshStandardMaterial({ color: "#e0f2fe", emissive: "#2563eb", emissiveIntensity: 0.45, roughness: 0.35 })
    );
    group.add(core);

    const nodes = (stacks.length ? stacks : [
      { name: "React", level: 86 },
      { name: "Go", level: 82 },
      { name: "PostgreSQL", level: 78 },
      { name: "Three.js", level: 72 }
    ] as SiteTechStack[]).slice(0, 9).map((stack, index) => {
      const angle = (index / Math.max(stacks.length || 4, 4)) * Math.PI * 2;
      const radius = 2.25 + (index % 3) * 0.58;
      const geometry = new THREE.SphereGeometry(0.08 + stack.level / 900, 24, 24);
      const material = new THREE.MeshStandardMaterial({
        color: palette[index % palette.length],
        emissive: palette[index % palette.length],
        emissiveIntensity: 0.35,
        roughness: 0.2
      });
      const mesh = new THREE.Mesh(geometry, material);
      mesh.position.set(Math.cos(angle) * radius, Math.sin(angle * 1.4) * 0.82, Math.sin(angle) * radius * 0.22);
      group.add(mesh);
      return { mesh, angle, radius, speed: 0.003 + index * 0.00045 };
    });

    scene.add(new THREE.AmbientLight("#ffffff", 0.9));
    const light = new THREE.PointLight("#93c5fd", 2.2, 20);
    light.position.set(4, 3, 5);
    scene.add(light);

    let frame = 0;
    let raf = 0;
    const animate = () => {
      frame += 1;
      group.rotation.y += 0.003;
      group.rotation.x = Math.sin(frame * 0.006) * 0.08;
      core.rotation.x += 0.006;
      core.rotation.y += 0.008;
      nodes.forEach(node => {
        node.angle += node.speed;
        node.mesh.position.x = Math.cos(node.angle) * node.radius;
        node.mesh.position.z = Math.sin(node.angle) * node.radius * 0.28;
      });
      renderer.render(scene, camera);
      raf = requestAnimationFrame(animate);
    };
    animate();

    const resize = () => {
      const nextWidth = mount.clientWidth;
      const nextHeight = mount.clientHeight;
      camera.aspect = nextWidth / nextHeight;
      camera.position.z = nextWidth < 640 ? 11 : 9;
      camera.updateProjectionMatrix();
      renderer.setSize(nextWidth, nextHeight);
    };
    window.addEventListener("resize", resize);
    return () => {
      cancelAnimationFrame(raf);
      window.removeEventListener("resize", resize);
      renderer.dispose();
      mount.removeChild(renderer.domElement);
    };
  }, [stacks]);

  return <div className="orbit" ref={mountRef} aria-label="技术轨道动画" />;
}

function ProjectGalaxy({ stacks, projects }: { stacks: SiteTechStack[]; projects: SiteProject[] }) {
  const [active, setActive] = useState<string>(stacks[0]?.name ?? "React");
  const nodes = useMemo<SiteTechStack[]>(() => {
    if (stacks.length) return stacks.slice(0, 8);
    const names = Array.from(new Set(projects.flatMap(project => splitTags(project.stack_tags)))).slice(0, 8);
    return names.map((name, index) => ({
      id: -index - 1,
      name,
      category: "project",
      level: 70,
      icon_url: "",
      description: ""
    }));
  }, [projects, stacks]);
  const activeProjects = projects.filter(project =>
    splitTags(project.stack_tags).some(tag => tag.toLowerCase() === active.toLowerCase())
  );

  useEffect(() => {
    if (!nodes.some(node => node.name === active) && nodes[0]) {
      setActive(nodes[0].name);
    }
  }, [active, nodes]);

  return (
    <section id="galaxy" className="section galaxySection">
      <div className="sectionHead">
        <span>Project Galaxy</span>
        <h2>项目星图</h2>
      </div>
      <div className="galaxyPanel">
        <div className="galaxyMap">
          {nodes.map((stack, index) => {
            const angle = (index / Math.max(nodes.length, 1)) * Math.PI * 2;
            const radius = index % 2 === 0 ? 34 : 42;
            const x = 50 + Math.cos(angle) * radius;
            const y = 50 + Math.sin(angle) * radius * 0.7;
            const related = projects.filter(project =>
              splitTags(project.stack_tags).some(tag => tag.toLowerCase() === stack.name.toLowerCase())
            ).length;
            return (
              <button
                key={stack.id}
                className={`galaxyNode ${active === stack.name ? "active" : ""}`}
                style={{ left: `${x}%`, top: `${y}%` }}
                onClick={() => setActive(stack.name)}
              >
                <strong>{stack.name}</strong>
                <span>{related} projects</span>
              </button>
            );
          })}
          <div className="galaxyCore">LAB</div>
        </div>
        <div className="galaxyDetail">
          <span>Selected node</span>
          <h3>{active}</h3>
          <p>
            这个节点关联 {activeProjects.length} 个项目。点击不同技术，可以看到项目和知识点之间的连接关系。
          </p>
          <div className="linkedProjects">
            {(activeProjects.length ? activeProjects : projects.slice(0, 3)).map(project => (
              <a key={project.id} href={project.demo_url || project.repo_url || "#"} target="_blank" rel="noreferrer">
                {project.name}
              </a>
            ))}
            {!projects.length && <span>后台发布项目后，这里会生成关联关系。</span>}
          </div>
        </div>
      </div>
    </section>
  );
}

function formatTimelineDate(value?: string) {
  if (!value) return "Now";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return "Now";
  return `${date.getFullYear()}.${String(date.getMonth() + 1).padStart(2, "0")}`;
}

function TimelineLab({ events }: { events: SiteTimelineEvent[] }) {
  const visibleEvents = events.length ? events : [];
  return (
    <section id="timeline" className="section timelineLab">
      <div className="sectionHead">
        <span>Timeline Lab</span>
        <h2>时间轴实验室</h2>
      </div>
      <div className="timelineRail">
        {visibleEvents.map((item, index) => (
          <article className="timelineEvent" key={item.id}>
            <div className="timelineMark">
              <span>{String(index + 1).padStart(2, "0")}</span>
              <i />
            </div>
            <div className="timelineBody">
              <div className="resourceMeta">
                <span>{formatTimelineDate(item.happened_at || item.published_at)}</span>
                <span>{item.phase || item.event_type}</span>
              </div>
              <h3>{item.title}</h3>
              <p>{item.summary || item.content}</p>
              <div className="tagRow">
                {splitTags(item.tags).slice(0, 4).map(tag => <span key={tag}>{tag}</span>)}
              </div>
              {item.link_url && <a href={item.link_url} target="_blank" rel="noreferrer">查看关联内容</a>}
            </div>
          </article>
        ))}
        {!visibleEvents.length && <p className="empty">后台发布学习记录后，这里会生成滚动路线图。</p>}
      </div>
    </section>
  );
}

function RocketButton() {
  const [visible, setVisible] = useState(false);
  useEffect(() => {
    const onScroll = () => setVisible(window.scrollY > 420);
    onScroll();
    window.addEventListener("scroll", onScroll, { passive: true });
    return () => window.removeEventListener("scroll", onScroll);
  }, []);
  return (
    <button
      className={`rocketButton ${visible ? "visible" : ""}`}
      type="button"
      aria-label="回到首页"
      onClick={() => window.scrollTo({ top: 0, behavior: "smooth" })}
    >
      <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
        <path d="M12 19V5M5 12l7-7 7 7" />
      </svg>
    </button>
  );
}

/** 首页头部搜索框 —— 复用到 hero */
function HeroSearchBar() {
  const [q, setQ] = useState("");
  return (
    <form
      className="heroSearch"
      onSubmit={event => {
        event.preventDefault();
        if (!q.trim()) return;
        navigate(searchHash(q.trim()));
      }}
    >
      <input
        value={q}
        onChange={event => setQ(event.target.value)}
        placeholder="搜索文章 / 笔记 / 项目"
      />
      <button type="submit">搜索</button>
    </form>
  );
}

function HomePage({ home, loading }: { home: SiteHome; loading: boolean }) {
  const [activeBanner, setActiveBanner] = useState(0);
  const [isBannerPaused, setIsBannerPaused] = useState(false);

  const banner = home.banners[activeBanner] ?? home.banners[0];
  const featured = useMemo(
    () => home.resources.filter(item => item.is_featured).slice(0, 3),
    [home.resources]
  );

  useEffect(() => {
    if (home.banners.length <= 1 || isBannerPaused) return;
    const timer = window.setInterval(() => {
      setActiveBanner(index => (index + 1) % home.banners.length);
    }, 4200);
    return () => window.clearInterval(timer);
  }, [home.banners.length, isBannerPaused]);

  useEffect(() => {
    const reducedMotion = window.matchMedia("(prefers-reduced-motion: reduce)").matches;
    const nodes = Array.from(
      document.querySelectorAll<HTMLElement>(
        ".section, .bannerBand, .resourceCard, .demoCard, .stackItem, .timelineEvent, .messageList article"
      )
    );
    if (reducedMotion) {
      nodes.forEach(node => node.classList.add("is-visible"));
      return;
    }
    const observer = new IntersectionObserver(
      entries => {
        entries.forEach(entry => {
          if (entry.isIntersecting) {
            entry.target.classList.add("is-visible");
            observer.unobserve(entry.target);
          }
        });
      },
      { threshold: 0.16, rootMargin: "0px 0px -48px 0px" }
    );
    nodes.forEach(node => {
      node.classList.add("reveal");
      observer.observe(node);
    });
    return () => observer.disconnect();
  }, [home]);

  return (
    <>
      <section className="hero">
        <nav className="nav">
          <a className="brand" href="#/">Tech Lab</a>
          <div>
            <a href="#resources">学习</a>
            <a href="#ask">问答</a>
            <a href="#/ask">AI 问答</a>
            <a href="#stack">技术栈</a>
            <a href="#galaxy">星图</a>
            <a href="#demos">Demo</a>
            <a href="#messages">留言</a>
            <a href="#timeline">时间轴</a>
            <a href={searchHash("")}>搜索</a>
          </div>
        </nav>

        <div className="heroGrid">
          <div className="heroCopy">
            {home.announcements[0] && (
              <a className="notice" href={home.announcements[0].link_url || "#"}>
                {home.announcements[0].title} · {home.announcements[0].content}
              </a>
            )}
            <h1>把学习痕迹整理成一个会发光的技术档案。</h1>
            <p>
              这里放我的技术栈、源码笔记、项目复盘和正在打磨的想法。内容由 Admins 管理端发布，前台实时读取。
            </p>
            <HeroSearchBar />
            <div className="heroActions">
              <a href="#resources">查看学习资源</a>
              <a href="#ask">问我的知识库</a>
            </div>
            <div className="heroStats">
              <span>{home.analytics.visit_count} 次访问</span>
              <span>{home.analytics.article_count} 篇文章</span>
              <span>{home.analytics.message_count} 条留言</span>
            </div>
          </div>
          <TechOrbit stacks={home.tech_stacks} />
        </div>
      </section>

      <section
        className="bannerBand"
        onMouseEnter={() => setIsBannerPaused(true)}
        onMouseLeave={() => setIsBannerPaused(false)}
      >
        <div className="bannerMedia">
          {banner?.image_url && (
            <img
              key={banner.id}
              src={assetURL(banner.image_url)}
              alt={banner.title}
            />
          )}
        </div>
        <div className="bannerCopy">
          <span>后台轮播</span>
          <h2>{banner?.title ?? "官网内容中心"}</h2>
          <p>{banner?.subtitle ?? "在后台上传图片、编辑文案，前台自动展示。"}</p>
          <div className="dots">
            {home.banners.map((item, index) => (
              <button
                key={item.id}
                className={index === activeBanner ? "active" : ""}
                aria-label={`查看轮播 ${index + 1}`}
                onClick={() => setActiveBanner(index)}
              />
            ))}
          </div>
        </div>
      </section>

      <section id="resources" className="section">
        <div className="sectionHead">
          <span>Notes</span>
          <h2>文章和笔记</h2>
        </div>
        <div className="resourceGrid">
          {(featured.length ? featured : home.resources).slice(0, 6).map(item => (
            <article className="resourceCard" key={item.id}>
              {item.cover_url && <img className="resourceCover" src={assetURL(item.cover_url)} alt={item.title} />}
              <div className="resourceMeta">
                <span>{item.category}</span>
                <span>{item.view_count ?? 0} reads</span>
              </div>
              <h3>{item.title}</h3>
              <p>{item.summary}</p>
              <div className="tagRow">
                {splitTags(item.tags).slice(0, 4).map(tag => <span key={tag}>{tag}</span>)}
              </div>
              <a
                href={articleRouteHash(item)}
                onClick={event => {
                  event.preventDefault();
                  navigate(articleRouteHash(item));
                }}
              >
                阅读文章
              </a>
            </article>
          ))}
          {!loading && home.resources.length === 0 && <p className="empty">后台发布资源后会显示在这里。</p>}
        </div>
      </section>

      <KnowledgeAsk resources={home.resources} />

      <ProjectGalaxy stacks={home.tech_stacks} projects={home.projects} />

      <section id="demos" className="section demosSection">
        <div className="sectionHead">
          <span>Demo Room</span>
          <h2>在线 Demo 展厅</h2>
        </div>
        <div className="demoGrid">
          {home.projects.slice(0, 6).map(project => (
            <article className="demoCard" key={project.id}>
              <div className="demoCover">
                {project.cover_url ? (
                  <img src={assetURL(project.cover_url)} alt={project.name} />
                ) : (
                  <span>{project.name.slice(0, 2).toUpperCase()}</span>
                )}
              </div>
              <div className="demoBody">
                <div className="resourceMeta">
                  <span>{project.is_featured ? "featured" : "project"}</span>
                  <span>{project.stack_tags}</span>
                </div>
                <h3>{project.name}</h3>
                <p>{project.summary}</p>
                <div className="demoActions">
                  {project.demo_url && <a href={project.demo_url} target="_blank" rel="noreferrer">在线预览</a>}
                  {project.repo_url && <a href={project.repo_url} target="_blank" rel="noreferrer">查看源码</a>}
                </div>
              </div>
            </article>
          ))}
          {!loading && home.projects.length === 0 && <p className="empty">后台发布项目后会显示在这里。</p>}
        </div>
      </section>

      <MessageBoard messages={home.messages} />

      <section id="stack" className="section stackSection">
        <div className="sectionHead">
          <span>Stack Map</span>
          <h2>技术栈</h2>
        </div>
        <div className="stackGrid">
          {home.tech_stacks.map(item => (
            <div className="stackItem" key={item.id}>
              <div>
                <strong>{item.name}</strong>
                <span>{item.category}</span>
              </div>
              <p>{item.description}</p>
              <div className="meter"><i style={{ width: `${item.level}%` }} /></div>
            </div>
          ))}
        </div>
      </section>

      <TimelineLab events={home.timeline} />
    </>
  );
}

function App() {
  const [home, setHome] = useState<SiteHome>(emptyHome);
  const [loading, setLoading] = useState(true);
  const route = useHashRoute();

  useEffect(() => {
    fetch(`${apiBase}/api/v1/site/home`)
      .then(res => res.json())
      .then(res => setHome(res.home ?? emptyHome))
      .catch(() => setHome(emptyHome))
      .finally(() => setLoading(false));
  }, []);

  // 访问统计上报（首次进站）
  useEffect(() => {
    const payload = JSON.stringify({
      path: window.location.pathname + window.location.hash,
      referrer: document.referrer,
      device: window.innerWidth < 768 ? "mobile" : "desktop"
    });
    const url = `${apiBase}/api/v1/site/visit`;
    if (navigator.sendBeacon) {
      navigator.sendBeacon(url, new Blob([payload], { type: "application/json" }));
      return;
    }
    fetch(url, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: payload,
      keepalive: true
    }).catch(() => undefined);
  }, []);

  // 详情页 / 搜索页顶部导航
  const showTopBar = route.name !== "home";

  return (
    <main>
      {showTopBar && (
        <nav className="innerNav">
          <a className="brand" href="#/">Tech Lab</a>
          <a href="#/ask">AI 问答</a>
          <a href={searchHash("")}>搜索</a>
        </nav>
      )}
      {route.name === "home" && <HomePage home={home} loading={loading} />}
      {route.name === "article" && <ArticleDetail slug={route.slug} />}
      {route.name === "demo" && <CodeDemoPage />}
      {route.name === "ask" && <AskPage />}
      {route.name === "search" && (
        <SearchPage
          query={route.q}
          category={route.category}
          tag={route.tag}
          page={route.page}
        />
      )}
      <RocketButton />
    </main>
  );
}

createRoot(document.getElementById("root")!).render(<App />);
