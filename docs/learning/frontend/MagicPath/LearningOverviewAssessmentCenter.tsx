import { useState } from "react";
import { AssessmentPrototype } from "./AssessmentPrototype";
import { OnboardingPrototype } from "./OnboardingPrototype";
import { LearningPathPage, KnowledgeBasePage, PracticePage, ProgressPage, TutorPage, SettingsPage } from "./WorkspacePages";
import type { Preview } from "./WorkspacePages";
import { BookOpen, CalendarDays, Check, ChevronDown, ClipboardCheck, Clock3, LayoutDashboard, Menu, MessageSquare, Play, RefreshCw, Settings, Sparkles, UserRound, X, Route, Zap, ChartNoAxesCombined, Headphones } from "lucide-react";
type Page = "Overview" | "Learning Path" | "Knowledge Base" | "Practice" | "Assessments" | "Progress" | "AI Tutor" | "Settings";
type Language = "English" | "Spanish" | "German" | "";
type Mode = "quick" | "full" | "skill" | "manual" | "scratch";
const card = "rounded-xl border border-[#e6e6e3] bg-white";
const muted = "text-[#777773]";
const btn = "inline-flex min-h-9 items-center justify-center gap-2 rounded-md bg-[#292929] px-3.5 py-2 text-[12px] font-semibold text-white hover:bg-black focus-visible:outline-2 focus-visible:outline-offset-2 disabled:opacity-40";
const outline = "inline-flex min-h-9 items-center justify-center gap-2 rounded-md border border-[#e6e6e3] bg-white px-3.5 py-2 text-[12px] font-medium hover:bg-[#f5f5f3] focus-visible:outline-2 focus-visible:outline-offset-2";
const nav: {
  page: Page;
  icon: typeof LayoutDashboard;
}[] = [{
  page: "Overview",
  icon: LayoutDashboard
}, {
  page: "Learning Path",
  icon: Route
}, {
  page: "Knowledge Base",
  icon: BookOpen
}, {
  page: "Practice",
  icon: Zap
}, {
  page: "Assessments",
  icon: ClipboardCheck
}, {
  page: "Progress",
  icon: ChartNoAxesCombined
}, {
  page: "AI Tutor",
  icon: MessageSquare
}, {
  page: "Settings",
  icon: Settings
}];
const conceptSkills = [["Grammar", "B1", 2], ["Vocabulary", "B1", 2], ["Reading", "B2", 1], ["Listening", "A2", 1], ["Speaking", "Not assessed", 0], ["Writing", "Insufficient evidence", 0], ["Pronunciation", "Not assessed", 0]] as const;
function Metric({
  icon: Icon,
  label,
  value,
  unit,
  kind,
  pct,
  green
}: {
  icon: typeof Clock3;
  label: string;
  value: string;
  unit: string;
  kind: string;
  pct: number;
  green?: boolean;
}) {
  return <div className={card + " min-h-[128px] p-4"}>
    <p className={"flex items-center gap-1.5 text-[11px] " + muted}><Icon size={13} />{label}</p>
    <div className="mt-2"><b className="text-[21px]">{value}</b><span className={"ml-1 text-[11px] " + muted}>{unit}</span></div>
    <div className="mt-2 h-[4px] overflow-hidden rounded-full bg-[#ededeb]"><div className={"h-full " + (green ? "bg-[#299141]" : "bg-[#5069ee]")} style={{
        width: String(pct) + "%"
      }} /></div>
    <p className={"mt-1.5 text-[11px] " + muted}>{kind}</p>
  </div>;
}
function Overview({
  language,
  go,
  concept,
  done
}: {
  language: Language;
  go: (page: Page) => void;
  concept: boolean;
  done: boolean;
}) {
  const [minutes, setMinutes] = useState(30);
  const [customizing, setCustomizing] = useState(false);
  const title = concept ? "Talking about past experiences" : language === "Spanish" ? "Ser and estar" : "The verb to be: am, is, are";
  const goal = concept ? "Practise the difference between Past Simple and Present Perfect, review previous mistakes, and complete a short listening activity." : language === "Spanish" ? "Say who you are and where you are from using ser and estar." : "Say who you are, where you are from and how old you are, and ask other people the same questions.";
  const plan = concept ? [{
    type: "Review · 5 min",
    title: "Review previous mistakes",
    note: "Objective: Irregular verbs, 3 errors",
    status: "Completed",
    icon: RefreshCw
  }, {
    type: "Grammar · 10 min",
    title: "Learn Present Perfect",
    note: "Objective: Past experiences",
    status: done ? "Completed" : "In progress",
    icon: BookOpen
  }, {
    type: "Listening · 10 min",
    title: "Listening comprehension",
    note: "Objective: Travel stories, A2–B1",
    status: done ? "Completed" : "Not started",
    icon: Headphones
  }, {
    type: "Speaking · 5 min",
    title: "Speaking practice",
    note: "Objective: Describe past experiences",
    status: done ? "Completed" : "Not started",
    icon: MessageSquare
  }] : [{
    type: "Grammar",
    title,
    note: "Objective: " + goal,
    status: "Not started",
    icon: BookOpen
  }];
  const skillRows: ReadonlyArray<readonly [string, string, number]> = concept ? conceptSkills : [["Grammar", "0 / 4 topics", 0], ["Vocabulary", "Outside this program", 0], ["Reading", "0 / 1 topics", 0], ["Listening", "Outside this program", 0], ["Speaking", "Outside this program", 0], ["Writing", "0 / 1 topics", 0], ["Pronunciation", "Not assessed", 0]];
  return <div className="grid items-start gap-6 lg:grid-cols-2">
    <div className="flex min-w-0 flex-col gap-6">
      <section className={card + " p-6"}>
        <div className="flex flex-wrap items-center justify-between gap-2">
          <span className={"text-[11px] font-semibold tracking-[0.04em] " + muted}>{done ? "PLAN COMPLETED" : concept ? "CONTINUE LEARNING · RESUME FROM LAST SESSION" : "CONTINUE LEARNING"}</span>
          <span className="rounded-full bg-[#eef1ff] px-2.5 py-0.5 text-[11px] font-medium text-[#4e69eb]">{concept ? "Learning · Intermediate path" : "Program · A1"}</span>
        </div>
        <h2 className="mt-5 text-[22px] font-semibold tracking-tight">{title}</h2>
        <p className={"mt-2 text-[13px] leading-[1.5] " + muted}>{goal}</p>
        <div className={"mt-4 flex flex-wrap gap-x-4 gap-y-1 text-[12px] " + muted}>
          <span><b className="text-[#333]">Next topic</b> · {concept ? "Past Simple vs Present Perfect" : "The passive: Present and Past Simple"}</span>
          <span><b className="text-[#333]">{concept ? "4" : "1"}</b> activities</span>
          <span><b className="text-[#333]">{minutes}</b> minutes</span>
        </div>
        <div className="mt-4 rounded-md bg-[#f2f2f3] px-3 py-3 text-[12px] leading-[1.45]">
          <b>{concept ? "Why this session?" : "Why this plan?"}</b>{" "}
          {concept ? "Your last session ended on Present Perfect, and 3 irregular-verb errors from this week are due for review." : "Your next topic is " + title + "."}
        </div>
        <div className="mt-4 flex flex-wrap items-center gap-2">
          <button className={btn} onClick={() => go("Practice")}><Play size={13} fill="currentColor" />{done ? "Start extra practice" : concept ? "Start learning" : "Continue learning"}</button>
          {concept && <button className={outline} onClick={() => setCustomizing(!customizing)}>Customize session</button>}
          <div className="ml-auto flex rounded-md bg-[#f3f3f3] p-1" role="group" aria-label="Session length">
            {[15, 30, 45, 60].map(value => <button key={value} onClick={() => setMinutes(value)} className={"rounded px-2.5 py-1 text-[11px] " + (minutes === value ? "bg-white font-semibold shadow-sm" : "")}>{value}m</button>)}
          </div>
        </div>
        {customizing && <p className={"mt-3 text-[11px] " + muted}>Session length: {minutes} minutes. This preview does not change your real program.</p>}
      </section>
      <section aria-label="Learning summary" className="grid grid-cols-2 gap-3 sm:grid-cols-3">
        <Metric icon={Clock3} label="Weekly study time" value={concept ? "2h 15m" : "0m"} unit={concept ? "/ 5h" : "/ 3h"} kind="Learning activity" pct={concept ? 45 : 0} />
        <Metric icon={RefreshCw} label="Review due" value={concept ? "8" : "0"} unit="topics" kind="Learning activity" pct={concept ? 39 : 0} />
        <Metric icon={Check} label="Mastered topics" value={concept ? "24" : "0"} unit="items" kind="Knowledge mastery" pct={concept ? 60 : 0} green />
        <Metric icon={CalendarDays} label="Learning consistency" value={concept ? "5" : "0"} unit="active days" kind="Learning activity" pct={concept ? 70 : 0} />
      </section>
      <section className={card}>
        <div className="flex items-center justify-between border-b border-[#e8e8e8] px-5 py-4">
          <div><h2 className="text-[15px] font-semibold">Today's learning plan</h2><p className={"text-[11px] " + muted}>{done ? "4 of 4 completed · 30 min planned" : concept ? "1 of 4 completed · 30 min planned" : "0 of 6 program topics completed"}</p></div>
          <button className="text-[11px] underline" onClick={() => go("Learning Path")}>Edit plan</button>
        </div>
        <ol className="space-y-1.5 p-2">{plan.map((row, i) => <li key={row.title} className={"flex flex-wrap items-center gap-3 rounded-md px-3 py-3 " + (concept && !done && i === 1 ? "bg-[#f3f3f3]" : "")}>
          <span className="flex h-8 w-8 items-center justify-center rounded-md bg-[#f4f4f4]"><row.icon size={15} /></span>
          <div className="min-w-[180px] flex-1"><p className={"text-[11px] " + muted}>{row.type}</p><p className={"text-[13px] font-medium " + (row.status === "Completed" ? "line-through" : "")}>{row.title}</p><p className={"text-[11px] " + muted}>{row.note}</p></div>
          <span className={"rounded-full px-2 py-0.5 text-[11px] " + (row.status === "Completed" ? "bg-[#eaf7ec] text-[#24833b]" : row.status === "In progress" ? "bg-[#edf2ff] text-[#4668ee]" : "bg-[#f3f3f3] text-[#777]")}>{row.status}</span>
          <button className={outline + " min-h-8 px-2.5"} onClick={() => go("Learning Path")}>Open</button>
        </li>)}</ol>
      </section>
    </div>
    <div className="flex min-w-0 flex-col gap-6">
      <section className={card + " px-5 py-4"}>
        <div className="flex items-start justify-between"><div><h2 className="text-[15px] font-semibold">Language skills</h2><p className={"text-[11px] " + muted}>{concept ? "Based on assessment evidence" : "Completed topics by skill"}</p></div><button className="text-[11px] underline" onClick={() => go(concept ? "Progress" : "Assessments")}>{concept ? "Skill analytics" : "Assess skills"}</button></div>
        <ul className="mt-2">{skillRows.map(([skill, result, bars]) => <li key={skill} className="flex items-center justify-between gap-3 border-t border-[#e6e6e6] py-2.5 text-[13px]">
          <b className="font-medium">{skill}</b>
          <span className="flex items-center gap-3"><span className="flex gap-0.5" aria-hidden="true">{[0, 1, 2].map(i => <i key={i} className={"block h-1 w-3 rounded-full " + (i < bars ? "bg-[#318949]" : "bg-[#dfe0e3]")} />)}</span><span className={bars ? "min-w-[90px] text-right text-[11px] font-semibold" : "min-w-[90px] text-right text-[11px] " + muted}>{result}</span></span>
        </li>)}</ul>
        <p className={"mt-2 text-[11px] " + muted}>{concept ? "Illustrative confidence bars. Unassessed skills have no estimate." : "Skill estimates require assessment or enough practice."}</p>
      </section>
      <section className={card + " px-5 py-4"}>
        <div className="flex items-center justify-between"><h2 className="text-[15px] font-semibold">Review queue</h2>{concept && <button className={outline + " min-h-8"} onClick={() => go("Practice")}>Start review</button>}</div>
        {concept ? <div className="mt-2">{[["Past Simple: irregular verbs", "3 repeated mistakes", "Last practised 6 days ago · ~5 min", "Overdue"], ["Articles: a, an, the", "Scheduled interval reached", "Last practised 9 days ago · ~4 min", "Due today"], ["Present Simple: third-person singular", "Low accuracy last attempt", "Last practised 12 days ago · ~3 min", "Due today"]].map(([title, reason, last, status]) => <div key={title} className="flex items-start justify-between gap-3 border-t border-[#e6e6e6] py-3.5"><div><p className="text-[12px] font-medium">{title}</p><p className={"text-[11px] " + muted}>{reason}</p><p className={"text-[11px] " + muted}>{last}</p></div><span className="whitespace-nowrap rounded-full bg-[#fff8e9] px-2 py-0.5 text-[10px] text-[#b56417]">{status}</span></div>)}</div> : <p className={"mt-4 text-[12px] " + muted}>You're up to date with your reviews.</p>}
      </section>
      <section className={card + " px-5 py-4"}>
        <h2 className="flex items-center gap-2 text-[15px] font-semibold"><Sparkles size={15} />Learning insights</h2>
        {concept ? <div className="mt-3 space-y-3">
          <div className="rounded-md bg-[#f3f3f3] p-3"><p className="text-[12px]">Your reading comprehension is improving, but you need more listening practice.</p><p className={"mt-1 text-[11px] " + muted}>Evidence: Reading 3 of last 4 texts above 80%; 1 listening activity this week.</p><button className="mt-1 text-[11px] underline" onClick={() => go("Learning Path")}>Add listening to plan</button></div>
          <div className="rounded-md bg-[#f3f3f3] p-3"><p className="text-[12px]">You have repeatedly made mistakes with irregular verb forms. Additional practice has been added to your learning plan.</p><p className={"mt-1 text-[11px] " + muted}>Evidence: 3 errors in the last 5 sessions.</p><button className="mt-1 text-[11px] underline" onClick={() => go("Progress")}>View mistakes</button></div>
        </div> : <p className={"mt-4 text-[12px] " + muted}>No insights yet.</p>}
      </section>
    </div>
  </div>;
}
function NewLearnerOverview({
  language,
  go,
  begin
}: {
  language: Language;
  go: (page: Page) => void;
  begin: (mode: Mode) => void;
}) {
  const starts = [{
    title: "Start from scratch",
    description: "Begin with the fundamentals of " + language + ".",
    icon: Play,
    mode: "scratch" as Mode,
    available: true
  }, {
    title: "Select a starting point",
    description: "Pick an approximate level. You can change it later.",
    icon: BookOpen,
    mode: "manual" as Mode,
    available: true
  }, {
    title: "Take adaptive assessment",
    description: "Short and optional. Stop at any time.",
    icon: ClipboardCheck,
    mode: "quick" as Mode,
    available: language === "English" || language === "Spanish"
  }];
  const skills = ["Grammar", "Vocabulary", "Reading", "Listening", "Speaking", "Writing", "Pronunciation"];
  return <div className="grid items-start gap-6 lg:grid-cols-2">
    <div className="flex min-w-0 flex-col gap-6">
      <section className={card + " p-6"}>
        <p className={"text-[11px] font-semibold uppercase tracking-[0.04em] " + muted}>Get started with {language}</p>
        <h2 className="mt-1.5 text-[20px] font-semibold">Choose where to begin</h2>
        <p className={"mt-1 max-w-[560px] text-[13px] leading-[1.5] " + muted}>You can start learning right away. Skill estimates appear after enough practice or an assessment.</p>
        <div className="mt-4 grid gap-3 sm:grid-cols-3">
          {starts.map(start => <button key={start.title} type="button" disabled={!start.available} onClick={() => begin(start.mode)} className={card + " flex min-h-[126px] flex-col items-start gap-1.5 p-4 text-left transition hover:bg-[#f5f5f3] focus-visible:outline-2 focus-visible:outline-offset-2 disabled:cursor-not-allowed disabled:opacity-50"}>
            <start.icon size={18} aria-hidden="true" />
            <b className="text-[13px] leading-[1.3]">{start.title}</b>
            <span className={"text-[12px] leading-[1.35] " + muted}>{start.available ? start.description : "Assessment is not available for this language yet."}</span>
          </button>)}
        </div>
      </section>
      <section aria-label="Learning activity summary" className="grid grid-cols-2 gap-3 sm:grid-cols-3">
        <Metric icon={Clock3} label="Weekly study time" value="–" unit="" kind="No data yet" pct={0} />
        <Metric icon={RefreshCw} label="Review due" value="–" unit="" kind="No data yet" pct={0} />
        <Metric icon={Check} label="Mastered topics" value="–" unit="" kind="No data yet" pct={0} green />
        <Metric icon={CalendarDays} label="Learning consistency" value="–" unit="" kind="No data yet" pct={0} />
      </section>
      <section className={card}>
        <div className="flex items-center justify-between border-b border-[#e8e8e8] px-5 py-4">
          <div><h2 className="text-[15px] font-semibold">Today's learning plan</h2><p className={"text-[11px] " + muted}>Not created yet</p></div>
          <button className="text-[11px] underline" onClick={() => go("Learning Path")}>Edit plan</button>
        </div>
        <div className={"px-6 py-9 text-center text-[13px] " + muted}>Your plan is created after your first session or assessment.</div>
      </section>
      <section className={card + " px-5 py-4"}>
        <div className="flex items-start justify-between gap-3">
          <div><h2 className="text-[15px] font-semibold">Weekly activity</h2><p className={"text-[11px] " + muted}>Study time, not proficiency</p></div>
          <div className="text-right"><b className="text-[18px]">0m</b><span className={"text-[12px] " + muted}> / 5h</span><p className={"text-[11px] " + muted}>No activity yet</p></div>
        </div>
        <div className="mt-5 flex h-[130px] items-end gap-3 border-b border-[#e6e6e6]" role="img" aria-label="No daily study minutes this week">{["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"].map(day => <div key={day} className="flex-1 border-b-2 border-[#e5e5e5]" />)}</div>
        <div className="mt-1.5 flex gap-3">{["Mon", "Tue", "Wed", "Thu", "Fri", "Sat", "Sun"].map(day => <span key={day} className={"flex-1 text-center text-[11px] " + muted}>{day}</span>)}</div>
      </section>
    </div>
    <div className="flex min-w-0 flex-col gap-6">
      <section className={card + " px-5 py-4"}>
        <div className="flex items-start justify-between"><div><h2 className="text-[15px] font-semibold">Language skills</h2><p className={"text-[11px] " + muted}>Based on assessment evidence</p></div><button className="text-[11px] underline" onClick={() => go("Progress")}>Skill analytics</button></div>
        <ul className="mt-2">{skills.map(skill => <li key={skill} className="flex items-center justify-between gap-3 border-t border-[#e6e6e6] py-2.5 text-[13px]">
          <b className="font-medium">{skill}</b>
          <span className="flex items-center gap-3"><span className="flex gap-0.5" aria-hidden="true">{[0, 1, 2].map(i => <i key={i} className="block h-1 w-3 rounded-full bg-[#dfe0e3]" />)}</span><span className={"min-w-[90px] text-right text-[11px] " + muted}>Not assessed</span></span>
        </li>)}</ul>
        <p className={"mt-2 text-[11px] " + muted}>Bars show assessment confidence. Unassessed skills have no estimate.</p>
      </section>
      <section className={card + " px-5 py-4"}>
        <div className="flex items-center justify-between"><h2 className="text-[15px] font-semibold">Review queue</h2><button className={outline + " min-h-8"} disabled title="Nothing to review yet">Start review</button></div>
        <p className={"mt-4 text-[12px] " + muted}>Nothing to review yet. Items appear as you practise.</p>
      </section>
      <section className={card + " px-5 py-4"}>
        <h2 className="flex items-center gap-2 text-[15px] font-semibold"><Sparkles size={15} />Learning insights</h2>
        <p className={"mt-4 text-[12px] " + muted}>Insights appear after a few completed activities.</p>
      </section>
    </div>
  </div>;
}
export const LearningOverviewAssessmentCenter = () => {
  const [page, setPage] = useState<Page>("Overview");
  const [language, setLanguage] = useState<Language>("English");
  const [menu, setMenu] = useState(false);
  const [preview, setPreview] = useState<Preview>("concept");
  const [startMode, setStartMode] = useState<Mode>("quick");
  const [assessmentLaunch, setAssessmentLaunch] = useState(0);
  const go = (next: Page) => {
    setPage(next);
    setMenu(false);
  };
  const concept = (preview === "concept" || preview === "due") && language === "English";
  const begin = (mode: Mode) => {
    setStartMode(mode);
    setAssessmentLaunch(value => value + 1);
    go("Assessments");
  };
  const startNewPlan = () => {
    setPreview("new");
    begin("quick");
  };
  const stateNotice: Partial<Record<Preview, { title: string; detail: string; action: string; destination: Page }>> = {
    unfinished: { title: "Assessment in progress", detail: "Sample state for a paused Quick grammar screening. Saved answers can be resumed in the application.", action: "Resume assessment", destination: "Assessments" },
    due: { title: "Reviews due", detail: "This preview shows a sample review queue. A real due item opens tracked review in the application.", action: "Open reviews", destination: "Practice" },
    insufficient: { title: "Not enough progress data yet", detail: "Skill levels and trends remain unknown until the application has enough evidence.", action: "See progress", destination: "Progress" },
    "missing-ai": { title: "AI key required", detail: "AI assessment and tutoring need a configured provider. Your existing learning history remains available.", action: "Open settings", destination: "Settings" },
    unavailable: { title: "Learning service unavailable", detail: "Keep the current language and activity visible, then retry when the service returns.", action: "Return to Overview", destination: "Overview" }
  };
  const notice = stateNotice[preview];
  return <div className="min-h-screen bg-[#f8f8f7] text-[#282828]" style={{
    fontFamily: "Geist, Inter, system-ui, sans-serif"
  }}>
    <div className="flex min-h-screen">
      <aside className={(menu ? "flex " : "hidden ") + "fixed inset-y-0 left-0 z-30 w-[232px] flex-col border-r border-[#e7e7e5] bg-white px-4 py-5 md:sticky md:top-0 md:flex md:h-screen"}>
        <div className="mb-5 flex items-center gap-2.5 px-2"><span className="flex h-7 w-7 items-center justify-center rounded-md bg-[#161616] text-[14px] font-bold text-white">l</span><b className="text-[14px]">learning</b><button className="ml-auto md:hidden" aria-label="Close menu" onClick={() => setMenu(false)}><X size={18} /></button></div>
        <div className="relative mb-3 rounded-lg border border-[#e7e7e5] px-2 py-2"><div className="flex items-center gap-2"><span className="flex h-6 w-6 items-center justify-center rounded-md bg-[#f3f3f1] text-[10px] font-semibold">{language ? language.slice(0, 2).toUpperCase() : "+"}</span><span className="min-w-0 flex-1"><b className="block text-[12px]">{language || "Add language"}</b><span className={"block text-[10px] " + muted}>{language ? "AI teacher" : "No profile"}</span></span><ChevronDown size={13} /></div><label className="sr-only" htmlFor="language-select">Learning language</label><select id="language-select" className="absolute inset-0 h-full w-full cursor-pointer opacity-0" value={language} onChange={event => {
            setLanguage(event.target.value as Language);
            go("Overview");
          }}><option value="">No language profile</option><option>English</option><option>Spanish</option><option>German</option></select></div>
        <nav aria-label="Learning navigation" className="space-y-0.5">{nav.map(({
            page: item,
            icon: Icon
          }) => <button key={item} onClick={() => go(item)} aria-current={page === item ? "page" : undefined} className={"flex h-9 w-full items-center gap-2.5 rounded-md px-3 text-left text-[12px] font-medium transition " + (page === item ? "border border-[#cececb] bg-[#f0f0ee]" : "text-[#686864] hover:bg-[#f5f5f3]")}><Icon size={15} />{item}</button>)}</nav>
        <div className="mt-auto border-t border-[#efefed] pt-3"><p className="text-[12px] font-medium">Demo learner</p><p className={"text-[11px] " + muted}>Design preview</p></div>
      </aside>
      {menu && <button className="fixed inset-0 z-20 bg-black/20 md:hidden" aria-label="Close navigation" onClick={() => setMenu(false)} />}
      <main className="min-w-0 flex-1"><div className="mx-auto max-w-[1490px] px-4 pb-16 pt-5 sm:px-6 md:pt-6">
        <div className="mb-4 flex items-center gap-2 md:hidden"><button className={outline + " px-2"} aria-label="Open menu" onClick={() => setMenu(true)}><Menu size={17} /></button><b className="text-[13px]">learning</b></div>
        {page !== "Knowledge Base" && <header className="mb-6 flex flex-wrap items-start justify-between gap-4"><div><p className={"text-[11px] " + muted}>learning / {page}</p><h1 className="mt-0.5 text-[23px] font-semibold tracking-tight">{page === "Overview" ? "Your learning overview" : page}</h1><p className={"mt-1 text-[13px] " + muted}>{page === "Overview" ? "Continue your personalized learning journey." : page === "Assessments" ? "Choose a starting point or assess your knowledge." : "Learning workspace"}</p><p className={"mt-1 text-[11px] " + muted}>Design preview · statistics, levels, answers and insights are sample content</p></div>{language && (page === "Overview" || page === "Learning Path") && <div className="flex flex-wrap items-center gap-2"><label className={"text-[11px] " + muted} htmlFor="preview-state">Preview state</label><select id="preview-state" className="rounded-md border border-[#e7e7e5] bg-white px-2.5 py-1.5 text-[11px]" value={preview} onChange={event => setPreview(event.target.value as Preview)}><option value="concept">Active learning path</option><option value="new">New learner / no program</option><option value="done">Plan completed</option><option value="current">Current app snapshot</option><option value="unfinished">Unfinished assessment</option><option value="due">Due review</option><option value="insufficient">Insufficient progress data</option><option value="missing-ai">Missing AI key</option><option value="unavailable">Unavailable service</option></select><div className="rounded-md border border-[#e7e7e5] bg-white px-2.5 py-1.5 text-[11px]">{language} · explanations in English <button className={"ml-1 underline " + muted} onClick={() => go("Settings")}>Language settings</button></div></div>}</header>}
        {language && notice && <section role="status" className="mb-5 flex flex-wrap items-center justify-between gap-3 rounded-xl border border-[#e6e6e3] bg-white p-4"><div><h2 className="text-[13px] font-semibold">{notice.title}</h2><p className={"mt-1 text-[12px] " + muted}>{notice.detail}</p></div><button className={outline} onClick={() => go(notice.destination)}>{notice.action}</button></section>}
        {!language ? <OnboardingPrototype complete={(chosen, mode) => {
            setLanguage(chosen);
            setPreview("new");
            begin(mode);
          }} /> : page === "Overview" ? (preview === "new" || preview === "unfinished") ? <NewLearnerOverview language={language} go={go} begin={begin} /> : <Overview language={language} go={go} concept={concept || preview === "done"} done={preview === "done"} /> : page === "Assessments" ? <AssessmentPrototype key={language + "-" + assessmentLaunch} language={language} initialMode={startMode} go={go} onPathCreated={() => setPreview("concept")} /> : page === "Learning Path" ? <LearningPathPage preview={preview} language={language} go={go} startNewPlan={startNewPlan} /> : page === "Knowledge Base" ? <KnowledgeBasePage preview={preview} language={language} go={go} /> : page === "Practice" ? <PracticePage go={go} /> : page === "Progress" ? <ProgressPage preview={preview} go={go} /> : page === "AI Tutor" ? <TutorPage go={go} /> : <SettingsPage language={language} setLanguage={setLanguage} go={go} onReset={() => { ["English", "Spanish", "German"].forEach(name => localStorage.removeItem(`magicpath.learning.assessment.${name}`)); setLanguage(""); setPreview("new"); go("Overview"); }} />}
      </div></main>
    </div>
  </div>;
};
