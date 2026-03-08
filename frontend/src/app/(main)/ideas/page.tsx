"use client";

import { useState, useMemo, useEffect, useRef, useCallback } from "react";
import { Search, Loader2 } from "lucide-react";
import { motion, AnimatePresence } from "motion/react";
import { useInfiniteIdeas } from "@/hooks/use-ideas";
import { IdeaCard } from "@/components/ideas/idea-card";

const PAGE_SIZE = 20;

const SCORE_FILTERS = [
  { label: "全部", value: 0 },
  { label: "≥6 分", value: 6 },
  { label: "≥7 分", value: 7 },
  { label: "≥8 分", value: 8 },
];

export default function IdeasPage() {
  const [sortBy, setSortBy] = useState("score_overall");
  const [sortOrder, setSortOrder] = useState("desc");
  const [status, setStatus] = useState("");
  const [minScore, setMinScore] = useState(0);
  const [search, setSearch] = useState("");
  const [activeTag, setActiveTag] = useState<string | null>(null);

  const {
    data,
    isLoading,
    isFetchingNextPage,
    hasNextPage,
    fetchNextPage,
  } = useInfiniteIdeas({
    page_size: PAGE_SIZE,
    sort_by: sortBy,
    sort_order: sortOrder,
    status: status || undefined,
    min_score: minScore || undefined,
  });

  // Flatten all pages into a single list
  const allItems = useMemo(() => {
    return data?.pages.flatMap(p => p.items) || [];
  }, [data?.pages]);

  const total = data?.pages[0]?.total || 0;

  // Extract all tags from loaded ideas
  const allTags = useMemo(() => {
    const tags = new Set<string>();
    allItems.forEach(idea => {
      if (idea.tags) idea.tags.forEach(t => tags.add(t));
    });
    return Array.from(tags).slice(0, 15);
  }, [allItems]);

  // Client-side filtering (search + tag)
  const filteredIdeas = useMemo(() => {
    return allItems.filter(idea => {
      if (search) {
        const q = search.toLowerCase();
        if (!(idea.product_name || "").toLowerCase().includes(q) &&
          !idea.topic.toLowerCase().includes(q)) return false;
      }
      if (activeTag) {
        if (!idea.tags || !idea.tags.includes(activeTag)) return false;
      }
      return true;
    });
  }, [allItems, search, activeTag]);

  // Infinite scroll: observe sentinel element
  const sentinelRef = useRef<HTMLDivElement>(null);

  const handleIntersect = useCallback(
    (entries: IntersectionObserverEntry[]) => {
      if (entries[0]?.isIntersecting && hasNextPage && !isFetchingNextPage) {
        fetchNextPage();
      }
    },
    [hasNextPage, isFetchingNextPage, fetchNextPage]
  );

  useEffect(() => {
    const el = sentinelRef.current;
    if (!el) return;
    const observer = new IntersectionObserver(handleIntersect, {
      rootMargin: "400px",
    });
    observer.observe(el);
    return () => observer.disconnect();
  }, [handleIntersect]);

  return (
    <div className="space-y-8">
      {/* Header */}
      <motion.div initial={{ opacity: 0, y: 20 }} animate={{ opacity: 1, y: 0 }} className="space-y-2">
        <h1 className="text-4xl font-extrabold tracking-tight">
          控制台：<span className="text-primary">所有点子</span>
        </h1>
        <p className="text-muted-foreground font-medium">
          这里展示所有评估中和已完成的灵感点子。共 {total} 个。
        </p>
      </motion.div>

      {/* Filter Bar */}
      <div className="sticky top-16 z-40 bg-background/80 backdrop-blur-xl py-4 -mx-2 px-2 space-y-3">
        {/* Row 1: Tags */}
        {allTags.length > 0 && (
          <div className="flex items-center gap-2 overflow-x-auto pb-2 border-b">
            <button onClick={() => { setActiveTag(null); }}
              className={`shrink-0 px-4 py-1.5 rounded-full text-xs font-semibold tracking-wide transition-all ${!activeTag ? 'bg-primary text-primary-foreground' : 'bg-secondary text-secondary-foreground hover:bg-secondary/80'}`}>
              全部
            </button>
            {allTags.map(tag => (
              <button key={tag} onClick={() => setActiveTag(activeTag === tag ? null : tag)}
                className={`shrink-0 px-3.5 py-1.5 rounded-full text-xs font-medium transition-all ${activeTag === tag ? 'bg-primary text-primary-foreground' : 'bg-secondary text-secondary-foreground hover:bg-secondary/80'}`}>
                {tag}
              </button>
            ))}
          </div>
        )}

        {/* Row 2: Search + Sort + Status + Score Filter */}
        <div className="flex items-center gap-3 flex-wrap">
          <div className="relative flex-1 min-w-[200px] max-w-sm">
            <Search className="w-4 h-4 absolute left-3.5 top-1/2 -translate-y-1/2 text-muted-foreground" />
            <input type="text" placeholder="搜索产品名或关键词..." value={search} onChange={e => setSearch(e.target.value)}
              className="w-full rounded-xl border bg-card pl-10 pr-4 py-2.5 text-sm placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-primary/50" />
          </div>

          <div className="rounded-xl border bg-card p-1 flex items-center shrink-0">
            <button onClick={() => { setSortBy("score_overall"); setSortOrder("desc"); }}
              className={`px-4 py-1.5 rounded-lg text-xs font-semibold transition-all ${sortBy === "score_overall" ? "bg-primary text-primary-foreground shadow-sm" : "text-muted-foreground hover:text-foreground"}`}>
              最热
            </button>
            <button onClick={() => { setSortBy("created_at"); setSortOrder("desc"); }}
              className={`px-4 py-1.5 rounded-lg text-xs font-semibold transition-all ${sortBy === "created_at" ? "bg-primary text-primary-foreground shadow-sm" : "text-muted-foreground hover:text-foreground"}`}>
              最新
            </button>
          </div>

          <select
            value={status}
            onChange={(e) => setStatus(e.target.value)}
            className="rounded-xl border bg-card px-3 py-2.5 text-xs font-semibold focus:outline-none focus:ring-2 focus:ring-primary/50"
          >
            <option value="">全部状态</option>
            <option value="graduated">已毕业</option>
            <option value="promising">有潜力</option>
            <option value="debating">辩论中</option>
            <option value="pending">等待中</option>
            <option value="failed">未通过</option>
          </select>

          <div className="rounded-xl border bg-card p-1 flex items-center shrink-0">
            {SCORE_FILTERS.map(f => (
              <button key={f.value} onClick={() => setMinScore(f.value)}
                className={`px-3 py-1.5 rounded-lg text-xs font-semibold transition-all ${minScore === f.value ? "bg-primary text-primary-foreground shadow-sm" : "text-muted-foreground hover:text-foreground"}`}>
                {f.label}
              </button>
            ))}
          </div>
        </div>
      </div>

      {/* Grid */}
      {isLoading ? (
        <div className="flex flex-col items-center justify-center py-20">
          <Loader2 className="w-8 h-8 animate-spin text-primary mb-4" />
          <span className="text-sm font-semibold text-muted-foreground tracking-wider uppercase animate-pulse">加载中...</span>
        </div>
      ) : filteredIdeas.length === 0 ? (
        <div className="text-center py-20">
          <div className="w-16 h-16 rounded-2xl border bg-card flex items-center justify-center mx-auto mb-4">
            <Search className="w-6 h-6 text-muted-foreground" />
          </div>
          <p className="font-semibold text-muted-foreground">没有找到相关的点子</p>
        </div>
      ) : (
        <>
          <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
            <AnimatePresence>
              {filteredIdeas.map((idea, index) => (
                <IdeaCard key={idea.id} idea={idea} index={index} />
              ))}
            </AnimatePresence>
          </div>

          {/* Infinite scroll sentinel */}
          <div ref={sentinelRef} className="py-8 flex justify-center">
            {isFetchingNextPage ? (
              <Loader2 className="w-6 h-6 animate-spin text-muted-foreground" />
            ) : hasNextPage ? (
              <span className="text-xs text-muted-foreground">向下滚动加载更多</span>
            ) : (
              <span className="text-xs text-muted-foreground">已加载全部 {allItems.length} 个点子</span>
            )}
          </div>
        </>
      )}
    </div>
  );
}
