import { createContext, useContext, useEffect, useState, type ReactNode } from "react";

export type Language = "en" | "zh";
export type Theme = "dark" | "light";

const messages = {
  en: {
    brandEyebrow: "SELF-HOSTED OBSERVABILITY",
    traceExplorer: "Trace Explorer",
    connectProject: "Connect project",
    connected: "Connected",
    enterApiKey: "Enter project API key",
    apiKeyActive: "API key active",
    pasteApiKey: "Paste API key",
    projectApiKey: "Project API key",
    connect: "Connect",
    reconnect: "Reconnect",
    operations: "OPERATIONS / LAST 24 HOURS",
    traceHealth: "Trace health at a glance",
    inspectHealth: "Inspect throughput, failures and latency before drilling into a run.",
    liveStore: "LIVE LOCAL STORE",
    requests: "Requests",
    tracesObserved: "traces observed",
    errorRate: "Error rate",
    failedTraces: "failed traces",
    needsAttention: "needs attention",
    healthy: "healthy",
    p95Latency: "P95 latency",
    endToEnd: "end-to-end trace duration",
    tailPerformance: "tail performance",
    tokenVolume: "Token volume",
    acrossTraces: "across all traces",
    projectDefault: "PROJECT / DEFAULT",
    recentTraces: "Recent traces",
    refresh: "↻ Refresh",
    filterByTraceID: "Filter by trace ID",
    searchTraceID: "Search trace ID…",
    allStatus: "All status",
    healthyStatus: "Healthy",
    errors: "Errors",
    allKinds: "All kinds",
    tracesInView: "traces in view",
    filteredResults: "Filtered results",
    newestFirst: "Newest first",
    noTracesMatch: "No traces match",
    clearFiltersHint: "Try clearing the filters or send a span to the ingest endpoint.",
    clearFilters: "Clear filters",
    loadingTraces: "Loading traces…",
    connectHint: "Enter an API key above to inspect traces.",
    openTrace: "Open trace",
    loadOlder: "Load older traces",
    traceDetail: "TRACE DETAIL",
    spans: "spans",
    feedback: "FEEDBACK",
    annotations: "Annotations",
    annotationKey: "key",
    score: "score",
    label: "label",
    comment: "comment",
    add: "Add",
    selectTrace: "Select a trace",
    selectTraceHint:
      "Choose a trace from the list to inspect its span tree, timing, inputs and outputs.",
    attributes: "Attributes",
    input: "INPUT",
    output: "OUTPUT",
    localMode: "TRACY / LOCAL MODE",
    sqliteStore: "SQLite trace store · API v1",
    language: "中文",
    theme: "Light",
    darkTheme: "Dark",
  },
  zh: {
    brandEyebrow: "自托管可观测性",
    traceExplorer: "Trace 探索器",
    connectProject: "连接项目",
    connected: "已连接",
    enterApiKey: "输入项目 API Key",
    apiKeyActive: "API Key 已生效",
    pasteApiKey: "粘贴 API Key",
    projectApiKey: "项目 API Key",
    connect: "连接",
    reconnect: "重新连接",
    operations: "运营概览 / 最近 24 小时",
    traceHealth: "Trace 健康概览",
    inspectHealth: "先查看吞吐、错误和延迟，再深入分析具体运行记录。",
    liveStore: "本地存储运行中",
    requests: "请求数",
    tracesObserved: "条 Trace",
    errorRate: "错误率",
    failedTraces: "条失败 Trace",
    needsAttention: "需要关注",
    healthy: "运行健康",
    p95Latency: "P95 延迟",
    endToEnd: "端到端 Trace 耗时",
    tailPerformance: "尾部性能",
    tokenVolume: "Token 用量",
    acrossTraces: "所有 Trace 合计",
    projectDefault: "项目 / 默认",
    recentTraces: "最近 Trace",
    refresh: "↻ 刷新",
    filterByTraceID: "按 Trace ID 筛选",
    searchTraceID: "搜索 Trace ID…",
    allStatus: "全部状态",
    healthyStatus: "健康",
    errors: "错误",
    allKinds: "全部类型",
    tracesInView: "条结果",
    filteredResults: "已筛选",
    newestFirst: "最新优先",
    noTracesMatch: "没有匹配的 Trace",
    clearFiltersHint: "可以清除筛选条件，或向 ingest 接口发送一条 Span。",
    clearFilters: "清除筛选",
    loadingTraces: "正在加载 Trace…",
    connectHint: "请在上方输入 API Key 以查看 Trace。",
    openTrace: "打开 Trace",
    loadOlder: "加载更早的 Trace",
    traceDetail: "TRACE 详情",
    spans: "个 Span",
    feedback: "反馈",
    annotations: "标注",
    annotationKey: "键",
    score: "评分",
    label: "标签",
    comment: "备注",
    add: "添加",
    selectTrace: "选择一个 Trace",
    selectTraceHint: "从左侧选择 Trace，查看 Span 树、耗时、输入和输出。",
    attributes: "属性",
    input: "输入",
    output: "输出",
    localMode: "TRACY / 本地模式",
    sqliteStore: "SQLite Trace 存储 · API v1",
    language: "EN",
    theme: "亮色",
    darkTheme: "暗色",
  },
} as const;

type TranslationKey = keyof typeof messages.en;

type Preferences = {
  language: Language;
  theme: Theme;
  toggleLanguage: () => void;
  toggleTheme: () => void;
  t: (key: TranslationKey) => string;
};

const PreferencesContext = createContext<Preferences | null>(null);

export function PreferencesProvider({ children }: { children: ReactNode }) {
  const [language, setLanguage] = useState<Language>(() =>
    localStorage.getItem("tracy.language") === "zh" ? "zh" : "en",
  );
  const [theme, setTheme] = useState<Theme>(() =>
    localStorage.getItem("tracy.theme") === "light" ? "light" : "dark",
  );

  useEffect(() => {
    localStorage.setItem("tracy.language", language);
    document.documentElement.lang = language === "zh" ? "zh-CN" : "en";
  }, [language]);
  useEffect(() => {
    localStorage.setItem("tracy.theme", theme);
    document.documentElement.dataset.theme = theme;
  }, [theme]);

  const value: Preferences = {
    language,
    theme,
    toggleLanguage: () => setLanguage((current) => (current === "en" ? "zh" : "en")),
    toggleTheme: () => setTheme((current) => (current === "dark" ? "light" : "dark")),
    t: (key) => messages[language][key],
  };

  return <PreferencesContext.Provider value={value}>{children}</PreferencesContext.Provider>;
}

export function usePreferences() {
  const preferences = useContext(PreferencesContext);
  if (!preferences) throw new Error("usePreferences must be used inside PreferencesProvider");
  return preferences;
}
