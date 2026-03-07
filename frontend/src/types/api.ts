export interface IdeaInfo {
  id: number;
  topic: string;
  status: "pending" | "debating" | "graduated" | "failed";
  product_name: string;
  one_liner: string;
  score_feasibility: number;
  score_economics: number;
  score_profit: number;
  score_overall: number;
  round_count: number;
  tags: string[];
  created_at: string;
  updated_at: string;
}

export interface IdeaDetail extends IdeaInfo {
  proposal: string;
  debate_log: string;
  final_report: string;
  tech_stack: string;
  dev_prompt: string;
  search_data: string;
  judge_refined: string;
}

export interface ListIdeasResponse {
  items: IdeaInfo[];
  total: number;
  page: number;
  page_size: number;
}

export interface ListIdeasParams {
  page?: number;
  page_size?: number;
  status?: string;
  sort_by?: string;
  sort_order?: string;
  min_score?: number;
}

export interface LoginRequest {
  username: string;
  password: string;
}

export interface LoginResponse {
  token: string;
  user: UserInfo;
}

export interface UserInfo {
  id: number;
  username: string;
  role: string;
}

export interface SubmitTopicRequest {
  topic: string;
}

export interface QueueItem {
  id: number;
  topic: string;
  status: string;
  submitted_by: number;
  created_at: string;
}

export interface DebateStatus {
  idea_id: number;
  status: string;
  current_round: number;
  max_rounds: number;
  current_agent: string;
  latest_message: string;
  current_score: number;
}

// ---- Admin Types ----

export interface SystemSettings {
  debate_max_rounds: number;
  debate_graduation_score: number;
  debate_timeout: string;
  discovery_interval: string;
  discovery_enabled: boolean;
  topics_per_source: number;
  auto_submit_debate: boolean;
}

export interface SearchKeyword {
  id: number;
  keyword: string;
  enabled: boolean;
}

export interface CrawlerPlugin {
  id: number;
  name: string;
  label: string;
  enabled: boolean;
  config: string;
}

export interface DiscoveryTag {
  id: number;
  tag: string;
  category: string; // constraint, trend_query_cn, trend_query_en
  enabled: boolean;
}

export interface DiscoveredTopic {
  id: number;
  title: string;
  source: string;
  source_url: string;
  popularity: number;
  replies: number;
  snippet: string;
  status: string;
  recommendation: string;
  recommend_score: number;
  idea_id: number;
  discovered_at: string;
  batch_id: string;
}
