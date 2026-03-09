"use client";

import { useState, useEffect } from "react";
import { motion, AnimatePresence } from "motion/react";
import { TrendingUp, TrendingDown, Minus, Loader2, ChevronDown, Sparkles } from "lucide-react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";

/* ───────── Types ───────── */
export interface DebateRound {
  round: number;
  proposer: string;
  opponent: string;
  referee?: string;
  feasibility: number;
  economics: number;
  profit: number;
  overall: number;
  eval_reason: string;
  todo_snapshot?: string;
  timestamp: string;
  human_intervention?: string;
}

interface TodoItem {
  id: string;
  issue: string;
  detail: string;
  priority: string; // P0, P1, P2
  status: string;   // open, resolved, wontfix
  created_by?: string;
  created_round?: number;
  resolved_round?: number | null;
  resolution?: string | null;
}

interface ChatMessage {
  agent: "proposer" | "opponent" | "referee" | "human";
  round: number;
  phase: string;
  content: string;
  score?: number;
  prevScore?: number;
  todoSnapshot?: TodoItem[];
}

const proseClasses = `prose prose-sm dark:prose-invert max-w-none leading-relaxed
  prose-headings:font-bold prose-headings:mt-4 prose-headings:mb-2
  prose-p:my-2 prose-strong:font-semibold
  prose-li:my-1
  prose-table:text-xs prose-th:px-3 prose-th:py-2 prose-th:border-b
  prose-td:px-3 prose-td:py-2 prose-td:border-b
  prose-blockquote:border-l-2 prose-blockquote:border-primary prose-blockquote:bg-primary/5 prose-blockquote:px-4 prose-blockquote:py-1 prose-blockquote:rounded-r-lg prose-blockquote:italic
  prose-hr:my-6
  prose-code:text-primary prose-code:bg-muted prose-code:px-1.5 prose-code:py-0.5 prose-code:rounded-md prose-code:text-[13px]`;

const AGENT_CFG: Record<string, { color: string; icon: string; name: string; side: "left" | "right" | "center" }> = {
  proposer: { color: "text-indigo-400", icon: "💡", name: "创想者", side: "left" },
  opponent: { color: "text-cyan-400", icon: "🔍", name: "审判官", side: "right" },
  referee: { color: "text-amber-400", icon: "🎯", name: "裁判", side: "center" },
  human: { color: "text-amber-500", icon: "🧑‍💻", name: "人工介入", side: "center" },
};

function extractSummary(content: string, maxLen = 140): string {
  const cleaned = content.replace(/#+\s/g, "").replace(/\*\*/g, "").replace(/\n+/g, " ").trim();
  return cleaned.length > maxLen ? cleaned.slice(0, maxLen) + "…" : cleaned;
}

/* ── Referee Content Parser ── */
interface ParsedRefereeData {
  scoreDelta?: { prev: number; curr: number; delta: number; reason: string };
  evalScores?: { feasibility: number; economics: number; profit: number; overall: number; reason: string };
  remainingText: string;
}

function parseRefereeContent(content: string): ParsedRefereeData {
  let remaining = content;
  let scoreDelta: ParsedRefereeData["scoreDelta"];
  let evalScores: ParsedRefereeData["evalScores"];

  // Parse [SCORE_DELTA]
  const sdMatch = remaining.match(/\[SCORE_DELTA\]\s*上轮[=＝]([\d.]+)\s*本轮[=＝]([\d.]+)\s*变动[=＝]([+\-\d.]+)\s*原因[=＝]([\s\S]+?)(?=\n|\[EVAL\]|\[TODO|$)/);
  if (sdMatch) {
    scoreDelta = {
      prev: parseFloat(sdMatch[1]),
      curr: parseFloat(sdMatch[2]),
      delta: parseFloat(sdMatch[3]),
      reason: sdMatch[4].trim(),
    };
    remaining = remaining.replace(sdMatch[0], "");
  }

  // Parse [EVAL]
  const evalMatch = remaining.match(/\[EVAL\]\s*feasibility[=＝]([\d.]+)\s*economics[=＝]([\d.]+)\s*profit[=＝]([\d.]+)\s*overall[=＝]([\d.]+)\s*(?:reason[=＝]([\s\S]+?))?(?=\n|\[TODO|$)/);
  if (evalMatch) {
    evalScores = {
      feasibility: parseFloat(evalMatch[1]),
      economics: parseFloat(evalMatch[2]),
      profit: parseFloat(evalMatch[3]),
      overall: parseFloat(evalMatch[4]),
      reason: evalMatch[5]?.trim() || "",
    };
    remaining = remaining.replace(evalMatch[0], "");
  }

  // Remove [TODO_UPDATE], [TODO_NEW], [TODO_SNAPSHOT], [TODO_VERDICT] JSON blocks (already shown via TodoDisplay)
  remaining = remaining.replace(/\[TODO_(?:UPDATE|NEW|SNAPSHOT|VERDICT)\]\s*```(?:json)?\s*[\s\S]*?```/g, "");
  remaining = remaining.replace(/\[TODO_(?:UPDATE|NEW|SNAPSHOT|VERDICT)\]\s*\{[\s\S]*\}/g, "");
  remaining = remaining.replace(/\[TODO_(?:UPDATE|NEW|SNAPSHOT|VERDICT)\][^\n]*/g, "");
  remaining = remaining.trim();

  return { scoreDelta, evalScores, remainingText: remaining };
}

function extractRefereeSummary(content: string): string {
  const parsed = parseRefereeContent(content);
  if (parsed.scoreDelta) {
    return parsed.scoreDelta.reason;
  }
  if (parsed.evalScores?.reason) {
    return parsed.evalScores.reason;
  }
  return extractSummary(content, 100);
}

function ScoreBar({ label, value, color }: { label: string; value: number; color: string }) {
  return (
    <div className="flex items-center gap-2">
      <span className="text-xs text-muted-foreground w-14 shrink-0">{label}</span>
      <div className="flex-1 h-2 rounded-full bg-muted overflow-hidden">
        <div className={`h-full rounded-full transition-all ${color}`} style={{ width: `${Math.min(value * 10, 100)}%` }} />
      </div>
      <span className="text-xs font-bold font-mono w-8 text-right">{value.toFixed(1)}</span>
    </div>
  );
}

function RefereeContentDisplay({ content }: { content: string }) {
  const parsed = parseRefereeContent(content);

  return (
    <div className="space-y-4">
      {/* Score Delta Card */}
      {parsed.scoreDelta && (
        <div className="rounded-xl border bg-amber-500/5 border-amber-500/20 p-4">
          <div className="flex items-center gap-3 mb-2">
            <span className="text-xs font-bold text-amber-500 uppercase tracking-wider">评分变动</span>
            <span className={`text-sm font-black font-mono ${
              parsed.scoreDelta.delta > 0 ? "text-green-500" : parsed.scoreDelta.delta < 0 ? "text-red-500" : "text-muted-foreground"
            }`}>
              {parsed.scoreDelta.prev.toFixed(1)} → {parsed.scoreDelta.curr.toFixed(1)}
              ({parsed.scoreDelta.delta > 0 ? "+" : ""}{parsed.scoreDelta.delta.toFixed(1)})
            </span>
          </div>
          <p className="text-sm text-muted-foreground leading-relaxed">{parsed.scoreDelta.reason}</p>
        </div>
      )}

      {/* Eval Scores Card */}
      {parsed.evalScores && (
        <div className="rounded-xl border bg-muted/30 p-4 space-y-2.5">
          <span className="text-xs font-bold text-muted-foreground uppercase tracking-wider">分项评分</span>
          <ScoreBar label="可行性" value={parsed.evalScores.feasibility} color="bg-green-500" />
          <ScoreBar label="经济性" value={parsed.evalScores.economics} color="bg-cyan-500" />
          <ScoreBar label="利润潜力" value={parsed.evalScores.profit} color="bg-violet-500" />
          <ScoreBar label="综合" value={parsed.evalScores.overall} color="bg-amber-500" />
          {parsed.evalScores.reason && (
            <p className="text-sm text-muted-foreground leading-relaxed pt-1 border-t">{parsed.evalScores.reason}</p>
          )}
        </div>
      )}

      {/* Remaining markdown content */}
      {parsed.remainingText && (
        <div className={proseClasses}>
          <ReactMarkdown remarkPlugins={[remarkGfm]}>{parsed.remainingText}</ReactMarkdown>
        </div>
      )}
    </div>
  );
}

/* ── Score Divider ── */
function ScoreDivider({ delta, score }: { delta: number; score: number }) {
  return (
    <div className="flex items-center justify-center gap-3 py-3 my-1">
      <div className="flex-1 h-px bg-border" />
      <span className={`inline-flex items-center gap-1.5 px-3 py-1 rounded-full text-[11px] font-bold font-mono border bg-card
        ${delta > 0 ? "text-green-500 border-green-500/25" : delta < 0 ? "text-red-500 border-red-500/25" : "text-muted-foreground border-border"}`}>
        {delta > 0 ? <TrendingUp className="w-3 h-3" /> : delta < 0 ? <TrendingDown className="w-3 h-3" /> : <Minus className="w-3 h-3" />}
        评分 {score.toFixed(1)}/10 {delta !== 0 && `(${delta > 0 ? "+" : ""}${delta.toFixed(1)})`}
      </span>
      <div className="flex-1 h-px bg-border" />
    </div>
  );
}

/* ── Chat Bubble ── */
function ChatBubble({ msg, expanded, onToggle }: {
  msg: ChatMessage; expanded: boolean; onToggle: () => void;
}) {
  const c = AGENT_CFG[msg.agent] || AGENT_CFG.proposer;
  const summary = extractSummary(msg.content);
  const isLeft = c.side === "left";
  const isCenter = c.side === "center";

  // Human intervention banner — always expanded, no toggle
  if (msg.agent === "human") {
    return (
      <div className="my-4">
        <div className="rounded-2xl border-2 border-amber-500/40 bg-amber-500/5 p-4">
          <div className="flex items-center gap-2.5 mb-2">
            <span className="text-base">{c.icon}</span>
            <span className={`font-bold text-sm ${c.color}`}>{c.name}</span>
            <span className="text-muted-foreground text-[10px] font-mono">R{msg.round} 起</span>
          </div>
          <p className="text-sm leading-relaxed whitespace-pre-wrap">{msg.content}</p>
        </div>
      </div>
    );
  }

  // Center layout for referee
  if (isCenter) {
    const refSummary = msg.agent === "referee" ? extractRefereeSummary(msg.content) : summary;
    return (
      <div className="flex flex-col items-center my-2">
        <button onClick={onToggle} className="w-full max-w-[90%] text-left group">
          <div className={`rounded-2xl border-2 p-4 transition-all duration-300 ${
            expanded
              ? "border-amber-500/40 bg-amber-500/5 shadow-lg shadow-amber-500/10"
              : "border-amber-500/15 bg-card hover:border-amber-500/30 hover:bg-amber-500/5"
          }`}>
            <div className="flex items-center gap-2.5 mb-2">
              <span className="text-base">{c.icon}</span>
              <span className={`font-bold text-sm ${c.color}`}>{c.name}</span>
              <span className="text-muted-foreground text-[10px] font-mono">R{msg.round}</span>
              <div className="ml-auto flex items-center gap-1.5">
                <span className={`text-[10px] font-semibold px-2 py-0.5 rounded-full transition-colors ${
                  expanded ? "bg-amber-500/20 text-amber-500" : "bg-muted text-muted-foreground group-hover:bg-amber-500/10 group-hover:text-amber-500"
                }`}>
                  {expanded ? "收起" : "展开详情"}
                </span>
                <ChevronDown className={`w-3.5 h-3.5 text-muted-foreground transition-transform duration-300 ${expanded ? "rotate-180" : ""}`} />
              </div>
            </div>
            {!expanded && <p className="text-[13px] text-muted-foreground line-clamp-2 leading-relaxed">{refSummary}</p>}
          </div>
        </button>
        <AnimatePresence>
          {expanded && (
            <motion.div initial={{ height: 0, opacity: 0 }} animate={{ height: "auto", opacity: 1 }}
              exit={{ height: 0, opacity: 0 }} transition={{ duration: 0.3 }} className="overflow-hidden w-full max-w-[90%]">
              <div className="mt-1 rounded-2xl border border-amber-500/20 bg-card p-5 overflow-y-auto max-h-[60vh]">
                {msg.agent === "referee" ? <RefereeContentDisplay content={msg.content} /> : (
                  <div className={proseClasses}><ReactMarkdown remarkPlugins={[remarkGfm]}>{msg.content}</ReactMarkdown></div>
                )}
              </div>
            </motion.div>
          )}
        </AnimatePresence>
      </div>
    );
  }

  // Left/Right chat bubble
  return (
    <div className={`flex gap-3 my-2 ${isLeft ? "flex-row" : "flex-row-reverse"}`}>
      {/* Avatar */}
      <div className="shrink-0 pt-1">
        <div className="w-9 h-9 rounded-xl flex items-center justify-center text-[15px] bg-muted border">
          {c.icon}
        </div>
      </div>

      {/* Bubble */}
      <div className="flex-1 min-w-0 max-w-[85%]">
        <div className={`flex items-center gap-2 mb-1.5 ${isLeft ? "" : "justify-end"}`}>
          <span className={`font-bold text-sm ${c.color}`}>{c.name}</span>
          <span className="text-muted-foreground text-[10px] font-mono bg-muted px-1.5 py-0.5 rounded border">R{msg.round}</span>
          {msg.phase && <span className="text-muted-foreground text-[10px] font-mono hidden sm:inline">{msg.phase}</span>}
        </div>

        <button onClick={onToggle} className="w-full text-left group">
          <div className={`rounded-2xl border-2 p-4 transition-all duration-300 ${isLeft ? "rounded-tl-md" : "rounded-tr-md"} ${
            expanded
              ? `border-indigo-500/30 bg-indigo-500/5 shadow-lg shadow-indigo-500/10`
              : `border-border hover:border-indigo-500/20 hover:bg-muted/30 bg-card`
          }`}>
            {!expanded ? (
              <div className="flex items-start gap-2">
                <p className="text-[13px] text-muted-foreground line-clamp-3 leading-relaxed flex-1">{summary}</p>
                <span className="shrink-0 flex items-center gap-1 text-[10px] font-semibold px-2 py-0.5 rounded-full bg-muted text-muted-foreground group-hover:bg-indigo-500/10 group-hover:text-indigo-400 transition-colors">
                  展开 <ChevronDown className="w-3 h-3" />
                </span>
              </div>
            ) : (
              <div className="flex items-center gap-1.5">
                <span className="text-[10px] font-semibold px-2 py-0.5 rounded-full bg-indigo-500/15 text-indigo-400">
                  收起 <ChevronDown className="w-3 h-3 inline rotate-180" />
                </span>
              </div>
            )}
          </div>
        </button>

        <AnimatePresence>
          {expanded && (
            <motion.div initial={{ height: 0, opacity: 0 }} animate={{ height: "auto", opacity: 1 }}
              exit={{ height: 0, opacity: 0 }} transition={{ duration: 0.3 }} className="overflow-hidden">
              <div className={`mt-1 rounded-2xl border bg-card p-5 overflow-y-auto max-h-[60vh] ${isLeft ? "rounded-tl-md" : "rounded-tr-md"
                }`}>
                <div className={proseClasses}><ReactMarkdown remarkPlugins={[remarkGfm]}>{msg.content}</ReactMarkdown></div>
              </div>
            </motion.div>
          )}
        </AnimatePresence>
      </div>
    </div>
  );
}

/* ── TODO List Display ── */
function TodoDisplay({ todos }: { todos: TodoItem[] }) {
  if (!todos || todos.length === 0) return null;

  const priorityColor: Record<string, string> = {
    P0: "text-red-500 bg-red-500/10 border-red-500/25",
    P1: "text-amber-500 bg-amber-500/10 border-amber-500/25",
    P2: "text-blue-500 bg-blue-500/10 border-blue-500/25",
  };

  const statusIcon: Record<string, string> = {
    open: "⏳",
    resolved: "✅",
    wontfix: "🚫",
  };

  const openCount = todos.filter(t => t.status === "open").length;
  const resolvedCount = todos.filter(t => t.status !== "open").length;

  return (
    <div className="mt-3 rounded-xl border bg-muted/30 p-3">
      <div className="flex items-center gap-2 mb-2">
        <span className="text-xs font-bold text-muted-foreground uppercase tracking-wider">TODO LIST</span>
        <span className="text-[10px] font-mono text-muted-foreground">
          {resolvedCount}/{todos.length} 已解决
          {openCount > 0 && <span className="text-amber-500 ml-1">· {openCount} 待解决</span>}
        </span>
      </div>
      <div className="space-y-1.5">
        {todos.map((todo) => (
          <div key={todo.id} className="flex items-start gap-2 text-xs">
            <span className="shrink-0 mt-0.5">{statusIcon[todo.status] || "⏳"}</span>
            <span className={`shrink-0 px-1.5 py-0.5 rounded text-[10px] font-bold border ${priorityColor[todo.priority] || "text-muted-foreground bg-muted border-border"}`}>
              {todo.priority}
            </span>
            <div className="flex-1 min-w-0">
              <span className={`font-medium ${todo.status === "resolved" ? "line-through text-muted-foreground" : ""}`}>
                {todo.issue}
              </span>
              {todo.resolution && (
                <span className="text-green-500 ml-1">→ {todo.resolution}</span>
              )}
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

/* ── Parse TODO snapshot JSON ── */
function parseTodoSnapshot(snapshot?: string): TodoItem[] {
  if (!snapshot) return [];
  try {
    const parsed = JSON.parse(snapshot);
    return parsed.todos || [];
  } catch {
    return [];
  }
}

/* ── Main Pipeline ── */
export function DebatePipeline({ rounds, status }: {
  rounds: DebateRound[];
  judgeRefined?: string;
  status: string;
}) {
  // Convert round format to chat messages: opponent → proposer → referee per round
  const messages: ChatMessage[] = [];
  rounds.forEach((r, i) => {
    const prevScore = i > 0 ? rounds[i - 1].overall : 0;
    const todoItems = parseTodoSnapshot(r.todo_snapshot);

    // Human intervention banner (before the round's debate)
    if (r.human_intervention) {
      messages.push({
        agent: "human",
        round: r.round,
        phase: "人工介入",
        content: r.human_intervention,
      });
    }

    // Opponent message
    if (r.opponent) {
      messages.push({
        agent: "opponent",
        round: r.round,
        phase: `审查 #${r.round}`,
        content: r.opponent,
      });
    }

    // Proposer message
    if (r.proposer) {
      messages.push({
        agent: "proposer",
        round: r.round,
        phase: `回应 #${r.round}`,
        content: r.proposer,
      });
    }

    // Referee message (center, with score + TODO snapshot)
    if (r.referee || r.overall > 0) {
      messages.push({
        agent: "referee",
        round: r.round,
        phase: `裁定 #${r.round}`,
        content: r.referee || `[EVAL] 综合=${r.overall.toFixed(1)} ${r.eval_reason || ""}`,
        score: r.overall,
        prevScore,
        todoSnapshot: todoItems,
      });
    }
  });

  const isOngoing = status === "pending" || status === "debating";
  const [expandedIdx, setExpandedIdx] = useState<number | null>(isOngoing ? messages.length - 1 : null);

  useEffect(() => {
    if (isOngoing && messages.length > 0) setExpandedIdx(messages.length - 1);
  }, [messages.length, isOngoing]);

  return (
    <div className="relative pt-2">
      {messages.map((msg, i) => {
        // Show score divider after referee messages
        const showScore = msg.agent === "referee" && msg.score != null;
        const delta = showScore ? (msg.score! - (msg.prevScore || 0)) : 0;

        return (
          <motion.div key={`msg-${i}`}
            initial={{ opacity: 0, y: 16 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: Math.min(i * 0.06, 0.5), duration: 0.45 }}>
            {showScore && <ScoreDivider delta={delta} score={msg.score!} />}

            <ChatBubble msg={msg} expanded={expandedIdx === i}
              onToggle={() => setExpandedIdx(expandedIdx === i ? null : i)} />

            {/* TODO snapshot under referee bubble when expanded */}
            {expandedIdx === i && msg.todoSnapshot && msg.todoSnapshot.length > 0 && (
              <motion.div initial={{ opacity: 0, height: 0 }} animate={{ opacity: 1, height: "auto" }}
                className="px-12">
                <TodoDisplay todos={msg.todoSnapshot} />
              </motion.div>
            )}
          </motion.div>
        );
      })}

      {isOngoing && (
        <motion.div initial={{ opacity: 0 }} animate={{ opacity: 1 }}
          className="flex items-center justify-center gap-3 py-6">
          <div className="w-9 h-9 rounded-xl bg-primary/10 border border-primary/25 flex items-center justify-center">
            <Loader2 className="w-4 h-4 text-primary animate-spin" />
          </div>
          <span className="text-sm font-bold text-primary animate-pulse">
            {status === "pending" ? "排队等待引擎处理..." : "辩论进行中..."}
          </span>
        </motion.div>
      )}
    </div>
  );
}
