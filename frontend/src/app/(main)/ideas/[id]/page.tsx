"use client";

import { useParams } from "next/navigation";
import { useEffect, useRef, useState, useMemo } from "react";
import Link from "next/link";
import {
  ArrowLeft, MessageSquare, Code, FileText, Sparkles,
  Clock, Play, Loader2, Wifi, ChevronDown, ChevronUp,
  Lightbulb, Target, Swords, Users, Copy, Check,
  UserPen, X, Send, Search, Vote, CheckCircle2, CircleDot
} from "lucide-react";
import { motion } from "motion/react";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";
import { useIdea } from "@/hooks/use-ideas";
import { useDebate } from "@/hooks/use-debate";
import type { DebateEvent } from "@/hooks/use-debate";
import { useQueryClient } from "@tanstack/react-query";
import { ScoreRing } from "@/components/ideas/score-ring";
import { VerdictBadge, TagChip } from "@/components/ideas/badges";
import { ScoreSparkline } from "@/components/ideas/score-sparkline";
import { DebatePipeline } from "@/components/ideas/debate-pipeline";
import type { DebateRound } from "@/components/ideas/debate-pipeline";
import type { VoteResultData, VoteEntry } from "@/types/api";

/* ───────── Types ───────── */
interface ParsedDebateLog {
  rounds: DebateRound[];
  search_data?: string;
  summary?: string;
  judge_refined?: string;
}

/* ───────── Prose classes for markdown ───────── */
const proseClasses = `prose prose-sm dark:prose-invert max-w-none leading-relaxed
  prose-headings:font-bold prose-headings:mt-4 prose-headings:mb-2
  prose-p:my-2 prose-strong:font-semibold
  prose-li:my-1
  prose-table:text-xs prose-th:px-3 prose-th:py-2 prose-th:border-b
  prose-td:px-3 prose-td:py-2 prose-td:border-b
  prose-blockquote:border-l-2 prose-blockquote:border-primary prose-blockquote:bg-primary/5 prose-blockquote:px-4 prose-blockquote:py-1 prose-blockquote:rounded-r-lg prose-blockquote:italic
  prose-hr:my-6
  prose-code:text-primary prose-code:bg-muted prose-code:px-1.5 prose-code:py-0.5 prose-code:rounded-md prose-code:text-[13px]`;

function getScoreColor(v: number) {
  if (v >= 8) return { text: "text-green-500", bar: "bg-green-500", ring: "ring-green-500/30" };
  if (v >= 6) return { text: "text-amber-500", bar: "bg-amber-500", ring: "ring-amber-500/30" };
  if (v >= 4) return { text: "text-orange-500", bar: "bg-orange-500", ring: "ring-orange-500/30" };
  return { text: "text-red-500", bar: "bg-red-500", ring: "ring-red-500/30" };
}

/* ───────── Main Page ───────── */
export default function IdeaDetailPage() {
  const params = useParams();
  const id = Number(params.id);
  const { data: idea, isLoading } = useIdea(id);
  const debate = useDebate(id);
  const queryClient = useQueryClient();
  const streamEndRef = useRef<HTMLDivElement>(null);

  // Parse debate log JSON
  const debateLog = useMemo<ParsedDebateLog | null>(() => {
    if (!idea?.debate_log) return null;
    try {
      return JSON.parse(idea.debate_log);
    } catch {
      return null;
    }
  }, [idea?.debate_log]);

  // Parse vote result JSON
  const voteResult = useMemo<VoteResultData | null>(() => {
    if (!idea?.vote_result) return null;
    try {
      return JSON.parse(idea.vote_result);
    } catch {
      return null;
    }
  }, [idea?.vote_result]);

  // Auto-scroll to latest event
  useEffect(() => {
    if (debate.events.length > 0) {
      streamEndRef.current?.scrollIntoView({ behavior: "smooth" });
    }
  }, [debate.events.length]);

  // Refresh idea data when debate ends
  useEffect(() => {
    const last = debate.events[debate.events.length - 1];
    if (last && (last.type === "done" || last.type === "error")) {
      queryClient.invalidateQueries({ queryKey: ["idea", id] });
      queryClient.invalidateQueries({ queryKey: ["ideas"] });
    }
  }, [debate.events, id, queryClient]);

  const [showIntervene, setShowIntervene] = useState(false);
  const [interveneRounds, setInterveneRounds] = useState(3);
  const [intervenePrompt, setIntervenePrompt] = useState("");
  const [interveneLoading, setInterveneLoading] = useState(false);

  const handleStartDebate = async () => {
    await debate.startDebate();
    debate.connectSSE();
  };

  const handleIntervene = async () => {
    if (!intervenePrompt.trim()) return;
    setInterveneLoading(true);
    await debate.intervene(interveneRounds, intervenePrompt.trim());
    debate.connectSSE();
    setInterveneLoading(false);
    setShowIntervene(false);
    setIntervenePrompt("");
  };

  // Score data for sparkline
  const scoreData = useMemo(() => {
    if (!debateLog?.rounds) return [];
    return debateLog.rounds.map(r => ({ round: r.round, score: r.overall }));
  }, [debateLog]);

  if (isLoading) {
    return (
      <div className="max-w-4xl mx-auto flex flex-col items-center justify-center py-20 gap-4">
        <div className="w-12 h-12 rounded-xl bg-primary/10 border border-primary/25 flex items-center justify-center">
          <Loader2 className="w-6 h-6 text-primary animate-spin" />
        </div>
        <p className="text-muted-foreground font-semibold tracking-wider uppercase animate-pulse">加载数据中...</p>
      </div>
    );
  }

  if (!idea) {
    return (
      <div className="max-w-4xl mx-auto text-center py-20">
        <div className="w-12 h-12 rounded-full bg-red-500/10 flex items-center justify-center mx-auto mb-4">
          <Target className="w-6 h-6 text-red-500" />
        </div>
        <p className="text-red-500 font-bold mb-4">项目未找到</p>
        <Link href="/ideas" className="inline-flex items-center gap-2 px-4 py-2 rounded-full border text-sm hover:bg-accent transition-colors">
          <ArrowLeft className="w-4 h-4" /> 返回控制台
        </Link>
      </div>
    );
  }

  const showStartButton = idea.status === "pending" || idea.status === "failed";
  const isDebating = idea.status === "debating" || debate.isRunning;
  const hasScores = idea.score_overall > 0;
  const title = idea.product_name || idea.topic;

  return (
    <div className="max-w-4xl mx-auto space-y-8 pb-16">
      {/* Back */}
      <Link href="/ideas" className="inline-flex items-center gap-1 text-sm text-muted-foreground hover:text-foreground transition-colors">
        <ArrowLeft className="h-4 w-4" /> 返回控制台
      </Link>

      {/* ═══ Title Area ═══ */}
      <motion.div initial={{ opacity: 0, y: 24 }} animate={{ opacity: 1, y: 0 }} transition={{ duration: 0.7, ease: "easeOut" }} className="text-center space-y-4">
        <div className="inline-flex items-center gap-2 flex-wrap justify-center">
          {idea.tags && idea.tags.map((tag) => (
            <TagChip key={tag} tag={tag} />
          ))}
          <VerdictBadge verdict={idea.status} size="md" />
          {debate.isConnected && (
            <span className="inline-flex items-center gap-1 text-xs text-green-500 animate-pulse px-2.5 py-1 rounded-full bg-green-500/10 border border-green-500/25">
              <Wifi className="h-3 w-3" /> 实时连接
            </span>
          )}
        </div>

        <h1 className="text-4xl md:text-5xl font-extrabold tracking-tight leading-[1.15]">
          {title}
        </h1>

        {idea.one_liner && (
          <p className="text-base md:text-xl text-muted-foreground max-w-2xl mx-auto leading-relaxed font-medium">
            {idea.one_liner}
          </p>
        )}
      </motion.div>

      {/* ═══ Product Info Card ═══ */}
      <motion.div initial={{ opacity: 0, y: 20 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: 0.1, duration: 0.5 }} className="rounded-2xl border bg-card p-6 md:p-8">
        <div className="flex flex-wrap gap-6 md:gap-8 mb-6 pb-6 border-b">
          {idea.product_name && idea.topic && idea.product_name !== idea.topic && (
            <div className="flex-1 min-w-[200px]">
              <div className="flex items-center gap-2 mb-2 text-[11px] font-semibold text-muted-foreground uppercase tracking-wider">
                <Lightbulb className="w-3.5 h-3.5 text-primary" /> 原始话题
              </div>
              <div className="font-semibold">{idea.topic}</div>
            </div>
          )}
          <div className="flex-1 min-w-[150px]">
            <div className="flex items-center gap-2 mb-2 text-[11px] font-semibold text-muted-foreground uppercase tracking-wider">
              <Clock className="w-3.5 h-3.5 text-muted-foreground" /> 评估时间
            </div>
            <div className="text-muted-foreground font-medium font-mono text-sm">
              {new Date(idea.created_at).toLocaleString("zh-CN", { month: "short", day: "numeric", hour: "2-digit", minute: "2-digit" })}
            </div>
          </div>
          {idea.round_count > 0 && (
            <div className="flex-1 min-w-[100px]">
              <div className="flex items-center gap-2 mb-2 text-[11px] font-semibold text-muted-foreground uppercase tracking-wider">
                <Swords className="w-3.5 h-3.5 text-muted-foreground" /> 辩论轮次
              </div>
              <div className="text-muted-foreground font-medium font-mono text-sm">{idea.round_count} 轮</div>
            </div>
          )}
        </div>

        {debateLog?.summary && (
          <div className="bg-muted/50 rounded-xl p-5 border">
            <div className="flex gap-3">
              <MessageSquare className="w-5 h-5 text-primary shrink-0 mt-0.5" />
              <p className="text-sm md:text-base text-muted-foreground leading-relaxed font-medium">
                {debateLog.summary}
              </p>
            </div>
          </div>
        )}
      </motion.div>

      {/* ═══ Decision Timeline ═══ */}
      {!showStartButton && (
        <DecisionTimeline
          hasSearch={!!idea.search_data}
          roundCount={idea.round_count}
          hasJudge={!!idea.final_report}
          voteVerdict={voteResult?.final_verdict || null}
          finalStatus={idea.status}
          score={idea.score_overall}
        />
      )}

      {/* ═══ Score Rings ═══ */}
      {hasScores && (
        <motion.div initial={{ opacity: 0, y: 20 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: 0.2, duration: 0.5 }} className="rounded-2xl border bg-card p-6 md:p-8">
          <div className="flex items-center justify-center gap-6 md:gap-10 flex-wrap">
            <ScoreRing score={idea.score_feasibility} label="可行性" color="rgb(34,197,94)" />
            <ScoreRing score={idea.score_economics} label="经济性" color="rgb(6,182,212)" />
            <ScoreRing score={idea.score_profit} label="利润潜力" color="hsl(var(--primary))" />
            <div className="w-px h-16 bg-border hidden sm:block" />
            <div className="relative">
              <div className="absolute inset-0 bg-amber-500/10 blur-2xl rounded-full" />
              <ScoreRing score={idea.score_overall} label="综合得分" color="rgb(245,158,11)" />
            </div>
          </div>
        </motion.div>
      )}

      {/* ═══ Voting Panel ═══ */}
      {voteResult && voteResult.votes.length > 0 && (
        <VotingPanel voteResult={voteResult} />
      )}

      {/* ═══ Start Debate ═══ */}
      {showStartButton && (
        <div className="flex items-center gap-4">
          <button
            onClick={handleStartDebate}
            disabled={debate.isRunning}
            className="inline-flex items-center gap-2 rounded-xl bg-primary px-6 py-3 text-base font-semibold text-primary-foreground hover:bg-primary/90 disabled:opacity-50 disabled:cursor-not-allowed transition-all shadow-lg shadow-primary/20"
          >
            {debate.isRunning ? <Loader2 className="h-5 w-5 animate-spin" /> : <Play className="h-5 w-5" />}
            {debate.isRunning ? "辩论进行中..." : "开始 AI 辩论"}
          </button>
          {debate.error && <p className="text-sm text-destructive">{debate.error}</p>}
        </div>
      )}

      {/* ═══ Human Intervention ═══ */}
      {hasScores && !isDebating && (
        <motion.div initial={{ opacity: 0, y: 20 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: 0.15, duration: 0.5 }}>
          {!showIntervene ? (
            <button
              onClick={() => setShowIntervene(true)}
              className="inline-flex items-center gap-2 rounded-xl border border-amber-500/30 bg-amber-500/5 px-5 py-2.5 text-sm font-semibold text-amber-500 hover:bg-amber-500/10 transition-all"
            >
              <UserPen className="h-4 w-4" /> 人工介入
            </button>
          ) : (
            <div className="rounded-2xl border border-amber-500/30 bg-card p-6 space-y-4">
              <div className="flex items-center justify-between">
                <div className="flex items-center gap-2">
                  <div className="w-8 h-8 rounded-lg bg-amber-500/10 border border-amber-500/25 flex items-center justify-center">
                    <UserPen className="w-4 h-4 text-amber-500" />
                  </div>
                  <h3 className="font-bold text-lg">人工介入</h3>
                </div>
                <button onClick={() => setShowIntervene(false)} className="p-1.5 rounded-lg hover:bg-muted transition-colors">
                  <X className="w-4 h-4 text-muted-foreground" />
                </button>
              </div>

              <div className="grid grid-cols-1 sm:grid-cols-[140px_1fr] gap-4">
                <div>
                  <label className="text-xs font-semibold text-muted-foreground uppercase tracking-wider mb-1.5 block">续辩轮次</label>
                  <input
                    type="number"
                    min={1}
                    max={20}
                    value={interveneRounds}
                    onChange={(e) => setInterveneRounds(Math.max(1, Math.min(20, Number(e.target.value))))}
                    className="w-full rounded-lg border bg-background px-3 py-2 text-sm font-mono focus:outline-none focus:ring-2 focus:ring-amber-500/50"
                  />
                </div>
                <div>
                  <label className="text-xs font-semibold text-muted-foreground uppercase tracking-wider mb-1.5 block">介入指令</label>
                  <textarea
                    value={intervenePrompt}
                    onChange={(e) => setIntervenePrompt(e.target.value)}
                    placeholder="输入你对当前方案的看法、补充要求或调整方向，AI 将据此重新讨论..."
                    rows={3}
                    className="w-full rounded-lg border bg-background px-3 py-2 text-sm resize-none focus:outline-none focus:ring-2 focus:ring-amber-500/50"
                  />
                </div>
              </div>

              <div className="flex items-center justify-between pt-2">
                <p className="text-xs text-muted-foreground">
                  当前评分 <span className="font-mono font-bold text-foreground">{idea.score_overall.toFixed(1)}</span>，
                  已进行 <span className="font-mono font-bold text-foreground">{idea.round_count}</span> 轮
                </p>
                <button
                  onClick={handleIntervene}
                  disabled={interveneLoading || !intervenePrompt.trim()}
                  className="inline-flex items-center gap-2 rounded-xl bg-amber-500 px-5 py-2.5 text-sm font-semibold text-white hover:bg-amber-600 disabled:opacity-50 disabled:cursor-not-allowed transition-all shadow-lg shadow-amber-500/20"
                >
                  {interveneLoading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Send className="h-4 w-4" />}
                  开始介入辩论
                </button>
              </div>

              {debate.error && <p className="text-sm text-destructive">{debate.error}</p>}
            </div>
          )}
        </motion.div>
      )}

      {/* ═══ Live Debate Stream (SSE) ═══ */}
      {(isDebating || debate.events.length > 0) && (
        <SectionWrapper title="辩论实况" icon={<Wifi className="h-4 w-4 text-blue-500" />}>
          <div className="space-y-3 max-h-[600px] overflow-y-auto pr-2">
            {debate.events
              .filter((e) => e.type !== "connected" && e.type !== "info" && e.type !== "stream_end")
              .map((event, i) => (
                <LiveEventCard key={i} event={event} />
              ))}
            {debate.isRunning && (
              <div className="flex items-center gap-2 text-sm text-muted-foreground py-2">
                <Loader2 className="h-4 w-4 animate-spin" /> 等待下一步...
              </div>
            )}
            <div ref={streamEndRef} />
          </div>
          {debate.latestScores && (
            <div className="mt-4 pt-4 border-t grid grid-cols-2 sm:grid-cols-4 gap-3">
              {["feasibility", "economics", "profit", "overall"].map((k) => (
                <MiniScore
                  key={k}
                  label={k === "feasibility" ? "可行性" : k === "economics" ? "经济性" : k === "profit" ? "利润潜力" : "综合"}
                  value={(debate.latestScores as Record<string, number>)?.[k] ?? 0}
                  highlight={k === "overall"}
                />
              ))}
            </div>
          )}
        </SectionWrapper>
      )}

      {/* ═══ AI Dev Prompt (with copy) ═══ */}
      {idea.dev_prompt && (
        <DevPromptSection content={idea.dev_prompt} />
      )}

      {/* ═══ Final Report ═══ */}
      {idea.final_report && (
        <MarkdownSection title="深度可行性报告" icon={<FileText className="h-5 w-5 text-indigo-500" />} content={idea.final_report} defaultExpanded />
      )}

      {/* ═══ Debate Pipeline — Chat Style ═══ */}
      {debateLog && debateLog.rounds.length > 0 && (
        <motion.div initial={{ opacity: 0, y: 20 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: 0.3, duration: 0.5 }} className="space-y-4">
          <div className="flex flex-col sm:flex-row sm:items-end justify-between gap-4 px-1">
            <div>
              <div className="flex items-center gap-3 mb-2">
                <div className="w-8 h-8 rounded-lg bg-primary/10 border border-primary/25 flex items-center justify-center">
                  <Users className="w-4 h-4 text-primary" />
                </div>
                <h2 className="text-xl font-bold tracking-tight">推演全流程</h2>
              </div>
              <p className="text-sm font-medium text-muted-foreground ml-11">多 Agent {debateLog.rounds.length} 轮深度对局</p>
            </div>
            {scoreData.length >= 2 && <ScoreSparkline scores={scoreData} />}
          </div>

          <div className="rounded-2xl border bg-card p-6 md:p-8">
            <DebatePipeline
              rounds={debateLog.rounds}
              status={idea.status}
            />
          </div>
        </motion.div>
      )}

      {/* ═══ Judge Refined (standalone if no pipeline) ═══ */}
      {idea.judge_refined && (!debateLog || debateLog.rounds.length === 0) && (
        <MarkdownSection title="判官精修方案" icon={<Sparkles className="h-5 w-5 text-amber-500" />} content={idea.judge_refined} />
      )}

      {/* ═══ Tech Stack ═══ */}
      {idea.tech_stack && (
        <MarkdownSection title="推荐技术栈" icon={<Code className="h-5 w-5 text-cyan-500" />} content={idea.tech_stack} />
      )}
    </div>
  );
}

/* ───────── Dev Prompt Section (with copy button) ───────── */
function DevPromptSection({ content }: { content: string }) {
  const [copied, setCopied] = useState(false);
  const [expanded, setExpanded] = useState(false);
  const isLong = content.length > 800;

  const handleCopy = async () => {
    try {
      if (navigator.clipboard && window.isSecureContext) {
        await navigator.clipboard.writeText(content);
      } else {
        const ta = document.createElement("textarea");
        ta.value = content;
        ta.style.position = "fixed";
        ta.style.left = "-9999px";
        document.body.appendChild(ta);
        ta.select();
        document.execCommand("copy");
        document.body.removeChild(ta);
      }
      setCopied(true);
      setTimeout(() => setCopied(false), 2500);
    } catch {
      alert("复制失败，请手动选中文本复制");
    }
  };

  return (
    <motion.div initial={{ opacity: 0, y: 16 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: 0.15, duration: 0.5 }}>
      <div className="rounded-2xl border bg-card overflow-hidden">
        <div className="px-6 py-4 border-b flex items-center gap-3 bg-muted/30">
          <div className="w-8 h-8 rounded-lg bg-muted border flex items-center justify-center">
            <Code className="h-5 w-5 text-indigo-500" />
          </div>
          <h2 className="text-base font-bold tracking-tight flex-1">AI 开发 Prompt</h2>
          <button onClick={handleCopy}
            className={`flex items-center gap-1.5 px-3 py-1.5 rounded-lg text-xs font-bold font-mono transition-all ${copied
              ? "bg-green-500/10 text-green-500 border border-green-500/25"
              : "bg-muted text-muted-foreground hover:text-foreground border"
              }`}>
            {copied ? <><Check className="w-3.5 h-3.5" /> 已复制</> : <><Copy className="w-3.5 h-3.5" /> 复制 Prompt</>}
          </button>
        </div>
        <div className="p-6">
          <div className={`${proseClasses} overflow-x-auto ${!expanded && isLong ? "max-h-96 overflow-hidden relative" : ""}`}>
            <ReactMarkdown remarkPlugins={[remarkGfm]}>{content}</ReactMarkdown>
            {!expanded && isLong && (
              <div className="absolute bottom-0 left-0 right-0 h-24 bg-gradient-to-t from-card to-transparent" />
            )}
          </div>
          {isLong && (
            <button onClick={() => setExpanded(!expanded)}
              className="mt-3 text-sm text-primary hover:underline flex items-center gap-1">
              {expanded ? <><ChevronUp className="h-3.5 w-3.5" /> 收起</> : <><ChevronDown className="h-3.5 w-3.5" /> 展开全部</>}
            </button>
          )}
        </div>
      </div>
    </motion.div>
  );
}

/* ───────── Markdown Section ───────── */
function MarkdownSection({ title, icon, content, defaultExpanded = false }: { title: string; icon: React.ReactNode; content: string; defaultExpanded?: boolean }) {
  const [expanded, setExpanded] = useState(defaultExpanded);
  const isLong = content.length > 800;

  return (
    <motion.div initial={{ opacity: 0, y: 16 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: 0.15, duration: 0.5 }}>
      <div className="rounded-2xl border bg-card overflow-hidden">
        <div className="px-6 py-4 border-b flex items-center gap-3 bg-muted/30">
          <div className="w-8 h-8 rounded-lg bg-muted border flex items-center justify-center">
            {icon}
          </div>
          <h2 className="text-base font-bold tracking-tight">{title}</h2>
        </div>
        <div className="p-6">
          <div className={`${proseClasses} overflow-x-auto ${!expanded && isLong ? "max-h-96 overflow-hidden relative" : ""}`}>
            <ReactMarkdown remarkPlugins={[remarkGfm]}>{content}</ReactMarkdown>
            {!expanded && isLong && (
              <div className="absolute bottom-0 left-0 right-0 h-24 bg-gradient-to-t from-card to-transparent" />
            )}
          </div>
          {isLong && (
            <button onClick={() => setExpanded(!expanded)}
              className="mt-3 text-sm text-primary hover:underline flex items-center gap-1">
              {expanded ? <><ChevronUp className="h-3.5 w-3.5" /> 收起</> : <><ChevronDown className="h-3.5 w-3.5" /> 展开全部</>}
            </button>
          )}
        </div>
      </div>
    </motion.div>
  );
}

/* ───────── Live Event Card ───────── */
const liveEventConfig: Record<string, { color: string; label: string }> = {
  search: { color: "border-l-sky-500 bg-sky-500/5", label: "搜索" },
  proposal: { color: "border-l-emerald-500 bg-emerald-500/5", label: "创想者" },
  opponent: { color: "border-l-orange-500 bg-orange-500/5", label: "审判官" },
  eval: { color: "border-l-violet-500 bg-violet-500/5", label: "评估" },
  judge_assess: { color: "border-l-amber-500 bg-amber-500/5", label: "精修评估" },
  judge_refine: { color: "border-l-pink-500 bg-pink-500/5", label: "判官精修" },
  vote: { color: "border-l-purple-500 bg-purple-500/5", label: "投票" },
  judge: { color: "border-l-indigo-500 bg-indigo-500/5", label: "终极仲裁" },
  meta: { color: "border-l-slate-500 bg-slate-500/5", label: "元数据" },
  summary: { color: "border-l-teal-500 bg-teal-500/5", label: "摘要" },
  done: { color: "border-l-green-500 bg-green-500/5", label: "完成" },
  error: { color: "border-l-red-500 bg-red-500/5", label: "错误" },
};

function LiveEventCard({ event }: { event: DebateEvent }) {
  const cfg = liveEventConfig[event.type] || { color: "border-l-gray-500 bg-gray-500/5", label: event.type };
  const isShort = event.content && !event.content.includes("\n") && event.content.length < 100;

  if (isShort && event.type !== "done" && event.type !== "error") {
    return (
      <div className="flex items-center gap-2 text-sm text-muted-foreground py-1">
        <span>{event.agent || "🔄"}</span>
        <span>{event.content}</span>
        {event.round && event.round > 0 && (
          <span className="text-xs bg-muted px-1.5 py-0.5 rounded">R{event.round}</span>
        )}
      </div>
    );
  }

  return (
    <div className={`border-l-4 rounded-r-lg p-3 space-y-1 ${cfg.color}`}>
      <div className="flex items-center gap-2 text-xs">
        <span>{event.agent || "🤖"}</span>
        <span className="font-medium">{cfg.label}</span>
        {event.round && event.round > 0 && (
          <span className="bg-muted px-1.5 py-0.5 rounded">第 {event.round} 轮</span>
        )}
      </div>
      {event.content && (
        <div className="text-sm whitespace-pre-wrap break-words max-h-64 overflow-y-auto">
          {event.content}
        </div>
      )}
    </div>
  );
}

/* ───────── Mini Score (for live) ───────── */
function MiniScore({ label, value, highlight }: { label: string; value: number; highlight?: boolean }) {
  const sc = getScoreColor(value);
  return (
    <div className={`rounded-lg border p-2 text-center ${highlight ? `ring-2 ${sc.ring}` : ""}`}>
      <div className="text-xs text-muted-foreground">{label}</div>
      <div className={`text-lg font-bold ${sc.text}`}>{value.toFixed(1)}</div>
    </div>
  );
}

/* ───────── Decision Timeline ───────── */
function DecisionTimeline({ hasSearch, roundCount, hasJudge, voteVerdict, finalStatus, score }: {
  hasSearch: boolean; roundCount: number; hasJudge: boolean;
  voteVerdict: string | null; finalStatus: string; score: number;
}) {
  const isDebating = finalStatus === "debating";

  const steps = [
    {
      label: "选题",
      icon: <Lightbulb className="w-4 h-4" />,
      done: true,
      detail: "话题已提交",
      color: "text-sky-500",
      bg: "bg-sky-500",
    },
    {
      label: "搜索",
      icon: <Search className="w-4 h-4" />,
      done: hasSearch,
      detail: hasSearch ? "真实数据采集完成" : (isDebating ? "搜索中..." : "未搜索"),
      color: "text-cyan-500",
      bg: "bg-cyan-500",
    },
    {
      label: "辩论",
      icon: <Swords className="w-4 h-4" />,
      done: roundCount > 0,
      detail: roundCount > 0 ? `${roundCount} 轮深度对局` : (isDebating ? "辩论进行中..." : "未开始"),
      color: "text-violet-500",
      bg: "bg-violet-500",
    },
    {
      label: "仲裁",
      icon: <Target className="w-4 h-4" />,
      done: hasJudge,
      detail: hasJudge ? "可行性报告已生成" : (isDebating ? "等待仲裁..." : "未仲裁"),
      color: "text-indigo-500",
      bg: "bg-indigo-500",
    },
    {
      label: "投票",
      icon: <Vote className="w-4 h-4" />,
      done: !!voteVerdict,
      detail: voteVerdict
        ? `多模型判定: ${voteVerdict === "YES" ? "可行" : voteVerdict === "NO" ? "不可行" : "有条件可行"}`
        : (isDebating ? "等待投票..." : "未投票"),
      color: "text-purple-500",
      bg: "bg-purple-500",
    },
    {
      label: "结论",
      icon: <CheckCircle2 className="w-4 h-4" />,
      done: ["graduated", "promising", "failed"].includes(finalStatus),
      detail: finalStatus === "graduated" ? `✅ 可行 (${score.toFixed(1)}分)`
        : finalStatus === "promising" ? `🔶 有潜力 (${score.toFixed(1)}分)`
        : finalStatus === "failed" ? `❌ 不可行 (${score.toFixed(1)}分)`
        : (isDebating ? "等待结论..." : "待评估"),
      color: finalStatus === "graduated" ? "text-green-500"
        : finalStatus === "promising" ? "text-amber-500"
        : finalStatus === "failed" ? "text-red-500" : "text-muted-foreground",
      bg: finalStatus === "graduated" ? "bg-green-500"
        : finalStatus === "promising" ? "bg-amber-500"
        : finalStatus === "failed" ? "bg-red-500" : "bg-muted-foreground",
    },
  ];

  return (
    <motion.div initial={{ opacity: 0, y: 20 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: 0.15, duration: 0.5 }}>
      <div className="rounded-2xl border bg-card p-6 md:p-8">
        <div className="flex items-center gap-3 mb-6">
          <div className="w-8 h-8 rounded-lg bg-primary/10 border border-primary/25 flex items-center justify-center">
            <CircleDot className="w-4 h-4 text-primary" />
          </div>
          <h2 className="text-base font-bold tracking-tight">决策流程</h2>
        </div>

        {/* Desktop: horizontal */}
        <div className="hidden md:flex items-start gap-0">
          {steps.map((step, i) => (
            <div key={step.label} className="flex-1 flex flex-col items-center text-center relative">
              {/* Connector line */}
              {i > 0 && (
                <div className={`absolute top-4 right-1/2 w-full h-0.5 -translate-y-1/2 ${step.done ? "bg-primary/40" : "bg-border"}`} />
              )}
              {/* Circle */}
              <div className={`relative z-10 w-8 h-8 rounded-full flex items-center justify-center border-2 transition-all ${
                step.done ? `${step.bg} text-white border-transparent` : isDebating && !step.done && (i === 0 || steps[i-1].done) ? "border-primary/50 text-primary bg-primary/10 animate-pulse" : "border-border text-muted-foreground bg-card"
              }`}>
                {step.done ? <Check className="w-3.5 h-3.5" /> : step.icon}
              </div>
              {/* Label */}
              <div className={`mt-2 text-xs font-bold ${step.done ? step.color : "text-muted-foreground"}`}>
                {step.label}
              </div>
              {/* Detail */}
              <div className="mt-1 text-[10px] text-muted-foreground leading-tight max-w-[100px]">
                {step.detail}
              </div>
            </div>
          ))}
        </div>

        {/* Mobile: vertical */}
        <div className="md:hidden space-y-0">
          {steps.map((step, i) => (
            <div key={step.label} className="flex gap-3 items-start relative">
              {/* Vertical line */}
              <div className="flex flex-col items-center">
                <div className={`w-7 h-7 rounded-full flex items-center justify-center border-2 shrink-0 ${
                  step.done ? `${step.bg} text-white border-transparent` : isDebating && !step.done && (i === 0 || steps[i-1].done) ? "border-primary/50 text-primary bg-primary/10 animate-pulse" : "border-border text-muted-foreground bg-card"
                }`}>
                  {step.done ? <Check className="w-3 h-3" /> : step.icon}
                </div>
                {i < steps.length - 1 && (
                  <div className={`w-0.5 h-6 ${steps[i+1].done ? "bg-primary/40" : "bg-border"}`} />
                )}
              </div>
              {/* Text */}
              <div className="pb-4">
                <div className={`text-xs font-bold ${step.done ? step.color : "text-muted-foreground"}`}>{step.label}</div>
                <div className="text-[10px] text-muted-foreground">{step.detail}</div>
              </div>
            </div>
          ))}
        </div>
      </div>
    </motion.div>
  );
}

/* ───────── Voting Panel ───────── */
function VotingPanel({ voteResult }: { voteResult: VoteResultData }) {
  const verdictConfig: Record<string, { color: string; bg: string; label: string }> = {
    YES: { color: "text-green-600 dark:text-green-400", bg: "bg-green-500/10 border-green-500/25", label: "可行" },
    NO: { color: "text-red-600 dark:text-red-400", bg: "bg-red-500/10 border-red-500/25", label: "不可行" },
    CONDITIONAL: { color: "text-amber-600 dark:text-amber-400", bg: "bg-amber-500/10 border-amber-500/25", label: "有条件可行" },
  };

  const finalCfg = verdictConfig[voteResult.final_verdict] || verdictConfig.CONDITIONAL;

  return (
    <motion.div initial={{ opacity: 0, y: 20 }} animate={{ opacity: 1, y: 0 }} transition={{ delay: 0.25, duration: 0.5 }}>
      <div className="rounded-2xl border bg-card overflow-hidden">
        <div className="px-6 py-4 border-b flex items-center justify-between bg-muted/30">
          <div className="flex items-center gap-3">
            <div className="w-8 h-8 rounded-lg bg-purple-500/10 border border-purple-500/25 flex items-center justify-center">
              <span className="text-base">🗳️</span>
            </div>
            <h2 className="text-base font-bold tracking-tight">多模型投票</h2>
          </div>
          <div className="flex items-center gap-2">
            <span className={`inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full text-xs font-bold border ${finalCfg.bg} ${finalCfg.color}`}>
              {finalCfg.label}
            </span>
            <span className="text-xs text-muted-foreground font-mono">
              {voteResult.yes_count}Y / {voteResult.no_count}N / {voteResult.cond_count}C
            </span>
          </div>
        </div>

        <div className="p-6 grid grid-cols-1 md:grid-cols-3 gap-4">
          {voteResult.votes.map((vote, i) => {
            const cfg = verdictConfig[vote.vote] || verdictConfig.CONDITIONAL;
            return (
              <div key={i} className={`rounded-xl border p-4 space-y-3 ${cfg.bg}`}>
                <div className="flex items-center justify-between">
                  <div>
                    <div className="font-bold text-sm">{vote.voter}</div>
                    <div className="text-[10px] text-muted-foreground font-mono">{vote.model}</div>
                  </div>
                  <div className="text-right">
                    <div className={`text-lg font-black ${cfg.color}`}>{vote.vote}</div>
                    <div className="text-[10px] text-muted-foreground">置信度 {Math.round(vote.confidence * 100)}%</div>
                  </div>
                </div>

                <p className="text-sm font-medium leading-relaxed">{vote.key_reason}</p>

                {vote.strengths && vote.strengths.length > 0 && (
                  <div>
                    <div className="text-[10px] font-semibold text-green-600 dark:text-green-400 uppercase tracking-wider mb-1">优势</div>
                    <div className="flex flex-wrap gap-1">
                      {vote.strengths.map((s, j) => (
                        <span key={j} className="text-[11px] px-2 py-0.5 rounded-full bg-green-500/10 text-green-700 dark:text-green-300 border border-green-500/20">
                          {s}
                        </span>
                      ))}
                    </div>
                  </div>
                )}

                {vote.risks && vote.risks.length > 0 && (
                  <div>
                    <div className="text-[10px] font-semibold text-red-600 dark:text-red-400 uppercase tracking-wider mb-1">风险</div>
                    <div className="flex flex-wrap gap-1">
                      {vote.risks.map((r, j) => (
                        <span key={j} className="text-[11px] px-2 py-0.5 rounded-full bg-red-500/10 text-red-700 dark:text-red-300 border border-red-500/20">
                          {r}
                        </span>
                      ))}
                    </div>
                  </div>
                )}
              </div>
            );
          })}
        </div>
      </div>
    </motion.div>
  );
}

/* ───────── Section Wrapper ───────── */
function SectionWrapper({ title, icon, children }: { title: string; icon: React.ReactNode; children: React.ReactNode }) {
  return (
    <div className="space-y-3">
      <div className="rounded-2xl border bg-card overflow-hidden">
        <div className="px-6 py-4 border-b flex items-center gap-3 bg-muted/30">
          <div className="w-8 h-8 rounded-lg bg-muted border flex items-center justify-center">
            {icon}
          </div>
          <h2 className="text-base font-bold tracking-tight">{title}</h2>
        </div>
        <div className="p-6">{children}</div>
      </div>
    </div>
  );
}
