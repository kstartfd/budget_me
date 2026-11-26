import { useMemo, useState } from "react";
import { Bookmark, BookOpen, Check, ChevronDown, ChevronRight, Clock3, Headphones, MessageSquare, MoreHorizontal, Play, RefreshCw, Search, Settings2, Sparkles, X } from "lucide-react";
export type Page = "Overview" | "Learning Path" | "Knowledge Base" | "Practice" | "Assessments" | "Progress" | "AI Tutor" | "Settings";
export type Preview = "concept" | "new" | "done" | "current" | "unfinished" | "due" | "insufficient" | "missing-ai" | "unavailable";
export type Language = "English" | "Spanish" | "German" | "";
const card = "rounded-xl border border-[#e6e6e3] bg-white";
const muted = "text-[#777773]";
const button = "inline-flex min-h-9 items-center justify-center gap-2 rounded-md bg-[#292929] px-3.5 py-2 text-[12px] font-semibold text-white hover:bg-black focus-visible:outline-2 focus-visible:outline-offset-2 disabled:opacity-40";
const secondary = "inline-flex min-h-9 items-center justify-center gap-2 rounded-md border border-[#e6e6e3] bg-white px-3.5 py-2 text-[12px] font-medium hover:bg-[#f5f5f3] focus-visible:outline-2 focus-visible:outline-offset-2";
const skills = ["Grammar", "Vocabulary", "Reading", "Listening", "Speaking", "Writing", "Pronunciation"];
const modules = [{
  title: "Foundations",
  objective: "Use everyday sentence patterns with confidence.",
  completed: 4,
  total: 4,
  skills: "Grammar · Vocabulary",
  state: "Completed",
  activities: ["Articles: a, an, the", "Everyday expressions"]
}, {
  title: "Talking about past experiences",
  objective: "Tell stories and compare past events with their present result.",
  completed: 1,
  total: 4,
  skills: "Grammar · Listening · Speaking",
  state: "In progress",
  activities: ["Review irregular verbs", "Learn Present Perfect", "Listening comprehension", "Speaking practice"]
}, {
  title: "Travel and plans",
  objective: "Understand travel stories and plan a journey.",
  completed: 0,
  total: 3,
  skills: "Vocabulary · Reading",
  state: "Next",
  activities: ["Travel vocabulary", "Read a travel story", "Use words in context"]
}];
function Tag({
  children,
  tone = "neutral"
}: {
  children: React.ReactNode;
  tone?: "neutral" | "blue" | "green" | "amber";
}) {
  const colors = {
    neutral: "bg-[#f3f3f3] text-[#777]",
    blue: "bg-[#eef1ff] text-[#4e69eb]",
    green: "bg-[#eaf7ec] text-[#24833b]",
    amber: "bg-[#fff8e9] text-[#b56417]"
  };
  return <span className={`rounded-full px-2 py-0.5 text-[11px] ${colors[tone]}`}>{children}</span>;
}
function SectionHead({
  title,
  note,
  action
}: {
  title: string;
  note?: string;
  action?: React.ReactNode;
}) {
  return <div className="flex flex-wrap items-start justify-between gap-3 border-b border-[#e8e8e8] px-5 py-4"><div><h2 className="text-[15px] font-semibold">{title}</h2>{note && <p className={`mt-0.5 text-[11px] ${muted}`}>{note}</p>}</div>{action}</div>;
}
export function LearningPathPage({
  preview,
  go,
  startNewPlan
}: {
  preview: Preview;
  go: (page: Page) => void;
  startNewPlan: () => void;
}) {
  const [filter, setFilter] = useState("All skills");
  const [expanded, setExpanded] = useState(1);
  const [panel, setPanel] = useState<"goals" | "history" | "activity" | null>(null);
  const [selected, setSelected] = useState("Learn Present Perfect");
  const [goal, setGoal] = useState("Everyday and professional communication");
  const [weekly, setWeekly] = useState("5 hours");
  const [session, setSession] = useState("30 minutes");
  const fresh = preview === "new";
  const done = preview === "done";
  if (fresh) return <div className="grid gap-6 lg:grid-cols-[minmax(0,1.4fr)_minmax(290px,.7fr)]"><section className={`${card} p-6`}><Tag>PROFILE READY</Tag><h2 className="mt-4 text-[22px] font-semibold">Set up your learning profile</h2><p className={`mt-2 text-[13px] ${muted}`}>Your goals and starting point shape the first learning path. You can begin from scratch or take a short assessment.</p><div className="mt-5 flex flex-wrap gap-2"><button className={button} onClick={() => go("Assessments")}>Choose a starting point</button><button className={secondary} onClick={() => go("Overview")}>Back to Overview</button></div></section><section className={`${card} p-5`}><h2 className="font-semibold">Profile</h2><p className={`mt-2 text-[12px] ${muted}`}>AI Teacher · settings saved</p><p className="mt-4 text-[13px]">No modules, estimates or progress are shown until a learning path exists.</p></section></div>;
  return <div className="grid items-start gap-6 lg:grid-cols-[minmax(0,1.5fr)_minmax(300px,.8fr)]">
    <div className="space-y-6">
      <section className={`${card} p-6`}><div className="flex flex-wrap items-center justify-between gap-2"><Tag tone="blue">{done ? "Plan completed" : "Active learning path"}</Tag><span className={`text-[11px] ${muted}`}>Sample plan · design preview</span></div><h2 className="mt-4 text-[22px] font-semibold">{done ? "All planned activities completed" : "Talking about past experiences"}</h2><p className={`mt-2 text-[13px] ${muted}`}>{done ? "All modules in this path are complete. Completion does not certify a CEFR level; keep reviewing what you learned." : "Practise the difference between Past Simple and Present Perfect and tell a short story about a past experience."}</p><div className="mt-5 grid grid-cols-2 gap-3 sm:grid-cols-4">{[["Current focus", done ? "Review" : "Present Perfect"], ["Weekly target", weekly], ["Preferred session", session], ["Remaining", done ? "0 activities" : "6 activities"]].map(([label, value]) => <div key={label} className="rounded-lg bg-[#f5f5f4] p-3"><p className={`text-[11px] ${muted}`}>{label}</p><b className="mt-1 block text-[12px]">{value}</b></div>)}</div><div className="mt-5 flex flex-wrap gap-2"><button className={button} onClick={() => go("Practice")}>{done ? "Continue reviews" : "Continue learning"}</button><button className={secondary} onClick={() => setPanel("goals")}>Edit learning goals</button><button className={secondary} onClick={() => setPanel("history")}>View plan history</button><button className={secondary} onClick={startNewPlan}>Start a new plan</button></div><p className={`mt-3 text-[11px] ${muted}`}>Your previous plan, assessment results and learning history remain available. Deleting history is a separate action.</p></section>
      {done && <section className={`${card} p-5`}><h2 className="font-semibold">Suggested next objectives</h2><p className={`mt-1 text-[12px] ${muted}`}>Illustrative options for extending a completed path.</p><div className="mt-4 flex flex-wrap gap-2">{["Tell longer stories", "Improve listening", "Use language at work"].map(item => <button key={item} className={secondary} onClick={() => {
            setSelected(item);
            setPanel("activity");
          }}>{item} · Add to path</button>)}</div></section>}
      <section className={card}><SectionHead title="Your modules" note="Activities can be filtered without changing your learning history." action={<label className="text-[12px]">Skill <select aria-label="Filter modules by skill" value={filter} onChange={e => setFilter(e.target.value)} className="ml-2 rounded-md border border-[#e6e6e3] px-2 py-1.5">{["All skills", ...skills].map(s => <option key={s}>{s}</option>)}</select></label>} /><div className="space-y-2 p-3">{modules.filter(m => filter === "All skills" || m.skills.includes(filter)).map((m, i) => <div key={m.title} className="rounded-lg border border-[#e8e8e8]"><button className="flex w-full items-center gap-3 p-4 text-left" onClick={() => setExpanded(expanded === i ? -1 : i)} aria-expanded={expanded === i}><span className="flex size-8 items-center justify-center rounded-md bg-[#f2f2f2] text-[12px]">{i + 1}</span><span className="min-w-0 flex-1"><b className="text-[14px]">{m.title}</b><span className={`block text-[11px] ${muted}`}>{m.objective}</span></span><Tag tone={m.state === "Completed" || done ? "green" : m.state === "In progress" ? "blue" : "neutral"}>{done ? "Completed" : m.state}</Tag><ChevronDown size={16} /></button>{expanded === i && <div className="border-t border-[#eee] px-4 py-3"><p className={`mb-3 text-[11px] ${muted}`}>{done ? m.total : m.completed} of {m.total} activities completed · {m.skills}</p>{m.activities.map((a, j) => <div key={a} className="flex flex-wrap items-center justify-between gap-2 border-t border-[#eee] py-3"><div><b className="text-[12px]">{a}</b><p className={`text-[11px] ${muted}`}>{j === 0 ? "Review" : "Learn or practise"} · 5–10 min · Knowledge linked</p></div><button className={secondary} onClick={() => {
                  setSelected(a);
                  setPanel("activity");
                }}>Details</button></div>)}</div>}</div>)}</div></section>
    </div>
    <div className="space-y-6"><section className={`${card} p-5`}><h2 className="font-semibold">Learning goals</h2><p className={`mt-2 text-[12px] ${muted}`}>{goal}</p><button className="mt-3 text-[12px] underline" onClick={() => setPanel("goals")}>Edit goals</button></section><section className={`${card} p-5`}><div className="flex items-center justify-between"><h2 className="font-semibold">Review queue</h2><button className={secondary} onClick={() => go("Practice")}>Start review</button></div><p className={`mt-3 text-[12px] ${muted}`}>Past Simple: irregular verbs · overdue · ~5 min</p><p className={`mt-2 text-[12px] ${muted}`}>Articles: a, an, the · due today · ~4 min</p></section><section className={`${card} p-5`}><h2 className="font-semibold">Skill coverage</h2><p className={`text-[11px] ${muted}`}>Activities in this path, not proficiency.</p>{[["Grammar", 5], ["Vocabulary", 2], ["Reading", 1], ["Listening", 1], ["Speaking", 1]].map(([s, n]) => <div key={s} className="flex justify-between border-b border-[#eee] py-2 text-[12px]"><span>{s}</span><span>{n} activities</span></div>)}</section><section className={`${card} p-5`}><h2 className="font-semibold">Recent plan changes</h2><p className={`mt-3 text-[12px] ${muted}`}>Review of irregular verbs added after repeated mistakes.</p><button className="mt-2 text-[12px] underline" onClick={() => setPanel("history")}>See history</button></section></div>
    {panel && <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/30 p-4" role="presentation" onMouseDown={() => setPanel(null)}><section role="dialog" aria-modal="true" aria-label={panel === "goals" ? "Edit learning goals" : panel === "history" ? "Plan history" : "Activity details"} className={`${card} w-full max-w-[560px] p-6 shadow-xl`} onMouseDown={e => e.stopPropagation()}><div className="flex items-start justify-between gap-3"><h2 className="text-[20px] font-semibold">{panel === "goals" ? "Edit learning goals" : panel === "history" ? "Plan history" : selected}</h2><button aria-label="Close" onClick={() => setPanel(null)}><X size={18} /></button></div>{panel === "goals" ? <div className="mt-5 space-y-4 text-[13px]"><label className="block">Learning purpose<select className="mt-1 block w-full rounded-md border p-2" value={goal} onChange={e => setGoal(e.target.value)}>{["Everyday and professional communication", "Travel", "Study", "Work"].map(x => <option key={x}>{x}</option>)}</select></label><label className="block">Weekly available time<select className="mt-1 block w-full rounded-md border p-2" value={weekly} onChange={e => setWeekly(e.target.value)}>{["3 hours", "5 hours", "8 hours"].map(x => <option key={x}>{x}</option>)}</select></label><label className="block">Preferred session duration<select className="mt-1 block w-full rounded-md border p-2" value={session} onChange={e => setSession(e.target.value)}>{["15 minutes", "30 minutes", "45 minutes", "60 minutes"].map(x => <option key={x}>{x}</option>)}</select></label><p className={muted}>Future activities may change; completed work stays in history.</p></div> : panel === "history" ? <div className="mt-5 space-y-3 text-[12px]"><p><b>Today · Review added</b><br />Repeated irregular-verb mistakes made this item due.</p><p><b>Yesterday · Session resumed</b><br />Present Perfect remains the next learning topic.</p><p className={muted}>Completed activities are kept when the plan changes.</p></div> : <div className="mt-5 space-y-3 text-[13px]"><p>Objective: practise {selected.toLowerCase()} in context.</p><p className={muted}>Why recommended: linked to your current learning objective and review needs.</p><p>Knowledge: <button className="underline" onClick={() => {
              setPanel(null);
              go("Knowledge Base");
            }}>Open related rule</button></p></div>}<div className="mt-6 flex justify-end gap-2"><button className={secondary} onClick={() => setPanel(null)}>Close</button>{panel === "goals" && <button className={button} onClick={() => setPanel(null)}>Save changes</button>}{panel === "activity" && <button className={button} onClick={() => {
            setPanel(null);
            go("Practice");
          }}>Open activity</button>}</div></section></div>}
  </div>;
}
type Domain = "Grammar" | "Vocabulary" | "Pronunciation" | "Functional Language" | "Communication";
type Mastery = "Not started" | "Learning" | "Practiced" | "Mastered";
type Review = "" | "Due" | "Overdue" | "Scheduled";
type Path = "In path" | "Recommended" | "Optional" | "Not planned";
type Rule = { title: string; category: string; description: string; cefr: string; domain: Domain; mastery: Mastery; review: Review; path: Path; keywords?: string; example?: string; mistake?: string };
type Category = { id: string; title: string; domain: Domain; parent?: string };

const categories: Category[] = [
  { id: "sent", title: "Sentence Structure", domain: "Grammar" }, { id: "noun", title: "Nouns and Articles", domain: "Grammar" },
  { id: "pronoun", title: "Pronouns and Determiners", domain: "Grammar" }, { id: "vform", title: "Verb Forms", domain: "Grammar" },
  { id: "tense", title: "Verb Tenses and Aspect", domain: "Grammar" },
  { id: "pres", title: "Present Forms", domain: "Grammar", parent: "tense" }, { id: "past", title: "Past Forms", domain: "Grammar", parent: "tense" },
  { id: "perf", title: "Perfect Forms", domain: "Grammar", parent: "tense" }, { id: "fut", title: "Future Forms", domain: "Grammar", parent: "tense" },
  { id: "modal", title: "Modal Verbs", domain: "Grammar" }, { id: "quest", title: "Questions and Negatives", domain: "Grammar" },
  { id: "cond", title: "Conditionals", domain: "Grammar" }, { id: "pass", title: "Passive Voice", domain: "Grammar" },
  { id: "rep", title: "Reported Speech", domain: "Grammar" }, { id: "adj", title: "Adjectives and Adverbs", domain: "Grammar" },
  { id: "prep", title: "Prepositions", domain: "Grammar" }, { id: "conj", title: "Conjunctions and Linking", domain: "Grammar" },
  { id: "relc", title: "Relative Clauses", domain: "Grammar" }, { id: "vpat", title: "Verb Patterns", domain: "Grammar" },
  { id: "adv", title: "Advanced Structures", domain: "Grammar" },
  { id: "pv", title: "Phrasal verbs", domain: "Vocabulary" }, { id: "col", title: "Collocations", domain: "Vocabulary" }, { id: "idi", title: "Idioms", domain: "Vocabulary" },
  { id: "req", title: "Requests and offers", domain: "Functional Language" }, { id: "opin", title: "Giving opinions", domain: "Functional Language" }
];

const rule = (title: string, category: string, description: string, cefr: string, domain: Domain, mastery: Mastery, review: Review, path: Path, keywords = "", example = "", mistake = ""): Rule => ({ title, category, description, cefr, domain, mastery, review, path, keywords, example, mistake });
// Demonstration catalogue transcribed from the original Knowledge Base design archive.
const englishRules: Rule[] = [
  rule("Word order in statements", "sent", "Subject, verb and object order in basic sentences.", "A1–B1", "Grammar", "Mastered", "Scheduled", "In path", "svo"),
  rule("Subject–verb agreement", "sent", "Matching verb forms to singular and plural subjects.", "A1–B2", "Grammar", "Practiced", "", "Optional", "agreement"),
  rule("Articles: a, an, the", "noun", "Choosing between a, an, the and no article.", "A1–B2", "Grammar", "Learning", "Due", "In path", "articles definite indefinite", "I saw a dog. The dog was big.", "I am teacher → I am a teacher."),
  rule("Countable and uncountable nouns", "noun", "Nouns that can and cannot be counted, with much, many, some.", "A1–B1", "Grammar", "Not started", "", "Recommended", "much many"),
  rule("Present Simple", "pres", "Habits, facts and routines.", "A1–B1", "Grammar", "Mastered", "Scheduled", "In path", "present tense third person", "She works in a bank.", "She work → She works."),
  rule("Present Continuous", "pres", "Actions happening now and temporary situations.", "A1–B1", "Grammar", "Practiced", "", "In path", "present tense -ing"),
  rule("Present Simple vs Present Continuous", "pres", "Choosing between routine and temporary actions.", "A2–B1", "Grammar", "Not started", "", "Optional", "present tense contrast"),
  rule("Past Simple", "past", "Finished actions at a definite time in the past.", "A1–B1", "Grammar", "Practiced", "", "In path", "past tense did you went yesterday", "We visited Lisbon last year.", "Did you went to work? → Did you go to work?"),
  rule("Past Simple: irregular verbs", "past", "Irregular past forms such as go–went and buy–bought.", "A1–B1", "Grammar", "Learning", "Overdue", "In path", "past tense irregular", "He bought a car.", "I buyed → I bought."),
  rule("Past Continuous", "past", "Actions in progress at a moment in the past.", "A2–B1", "Grammar", "Not started", "", "Recommended", "past tense was were -ing"),
  rule("Present Perfect", "perf", "Connects past events or states with the present.", "A2–B2", "Grammar", "Not started", "", "Recommended", "have been since and for has gone experience", "She has lived here since 2019.", "I have seen him yesterday → I saw him yesterday."),
  rule("Present Perfect Continuous", "perf", "Ongoing actions with a visible result or duration.", "B1–B2", "Grammar", "Not started", "", "Optional", "have been doing since for"),
  rule("Past Perfect", "perf", "An earlier past action before another past action.", "B1–B2", "Grammar", "Not started", "", "Not planned", "had done past tense"),
  rule("Present Perfect vs Past Simple", "perf", "Choosing between the two tenses to talk about the past.", "A2–B2", "Grammar", "Not started", "", "In path", "since and for past tense contrast", "", "I have lived here for 2019 → since 2019."),
  rule("Will and going to", "fut", "Predictions, plans and decisions about the future.", "A2–B1", "Grammar", "Not started", "", "In path", "future"),
  rule("Future Continuous", "fut", "Actions in progress at a future time.", "B1–B2", "Grammar", "Not started", "", "Not planned", "will be doing"),
  rule("Can, could, be able to", "modal", "Ability, possibility and permission.", "A1–B1", "Grammar", "Practiced", "Scheduled", "Optional", "modal ability"),
  rule("Must, have to, should", "modal", "Obligation, necessity and advice.", "A2–B1", "Grammar", "Learning", "", "Optional", "modal obligation"),
  rule("Questions with did", "quest", "Forming questions and negatives in the past simple.", "A1–B1", "Grammar", "Not started", "", "In path", "did you went questions negatives past", "", "Did you went? → Did you go?"),
  rule("Question tags", "quest", "Short questions added to statements.", "B1–B2", "Grammar", "Not started", "", "Not planned", "isn’t it tag"),
  rule("Zero and first conditional", "cond", "Real and likely situations with if.", "A2–B1", "Grammar", "Not started", "", "Optional", "if"),
  rule("Second conditional", "cond", "Unreal or unlikely situations.", "B1–B2", "Grammar", "Not started", "", "Not planned", "if I were"),
  rule("Passive voice: present and past", "pass", "Focusing on the action instead of the doer.", "B1–B2", "Grammar", "Not started", "", "Not planned", "is made was built"),
  rule("Prepositions of time: in, on, at", "prep", "Choosing in, on or at with times and dates.", "A1–B1", "Grammar", "Practiced", "Due", "Optional", "preposition time"),
  rule("Phrasal verbs: separable and inseparable", "pv", "How phrasal verbs take objects.", "B1–B2", "Vocabulary", "Not started", "", "Optional", "phrasal verb"),
  rule("Common phrasal verbs: daily routines", "pv", "Everyday phrasal verbs such as get up and wake up.", "A2–B1", "Vocabulary", "Learning", "", "In path", "phrasal verb routines"),
  rule("Phrasal verbs with get", "pv", "Meanings of get up, get on, get over and more.", "B1–B2", "Vocabulary", "Not started", "", "Not planned", "phrasal verb get"),
  rule("Polite requests", "req", "Asking for things and help politely.", "A1–B1", "Functional Language", "Practiced", "", "Optional", "could you would you"),
  rule("Offers and suggestions", "req", "Offering help and suggesting actions.", "A2–B1", "Functional Language", "Not started", "", "Optional", "shall we why don’t we")
];

const spanishRules: Rule[] = [
  rule("Gender of nouns", "noun", "Masculine and feminine nouns and their endings.", "A1–A2", "Grammar", "Learning", "Due", "In path", "gender agreement"),
  rule("Articles: el, la, un, una", "noun", "Definite and indefinite articles.", "A1", "Grammar", "Practiced", "", "In path", "articles"),
  rule("Present tense: -ar verbs", "pres", "Regular -ar verb endings.", "A1", "Grammar", "Practiced", "", "In path", "present hablar"),
  rule("Ser and estar", "pres", "Two verbs for to be and when to use each.", "A1–A2", "Grammar", "Learning", "Overdue", "In path", "ser estar to be"),
  rule("Polite requests with querer", "req", "Asking for things using quiero and quisiera.", "A1", "Functional Language", "Not started", "", "In path", "querer quisiera")
];
const domains: Domain[] = ["Grammar", "Vocabulary", "Pronunciation", "Functional Language", "Communication"];
const description: Record<Domain, string> = {
  Grammar: "Language structures, grammatical rules, comparisons, and usage patterns.",
  Vocabulary: "Words, phrases, collocations, phrasal verbs, idioms, and word formation.",
  Pronunciation: "Sounds, stress, intonation, rhythm, and connected speech.",
  "Functional Language": "Requests, opinions, questions, explanations, instructions, and other language functions.",
  Communication: "Practical communication scenarios, social interaction, professional communication, and discourse strategies."
};
const surface = "rounded-xl border border-[#e4e4e7] bg-white";
const smallButton = "inline-flex min-h-8 items-center justify-center rounded-lg border border-[#e4e4e7] bg-white px-3 text-[12px] hover:bg-[#f4f4f5] focus-visible:outline-2 focus-visible:outline-offset-2";

export function KnowledgeBasePage({ preview, language, go }: { preview: Preview; language: Language; go: (page: Page) => void }) {
  const [domain, setDomain] = useState<Domain>("Grammar");
  const [category, setCategory] = useState<string | null>(null);
  const [treeOpen, setTreeOpen] = useState(true);
  const [tenseOpen, setTenseOpen] = useState(true);
  const [query, setQuery] = useState("");
  const [preset, setPreset] = useState("All knowledge");
  const [filtersOpen, setFiltersOpen] = useState(false);
  const [level, setLevel] = useState("Any");
  const [masteryFilter, setMasteryFilter] = useState("Any");
  const [sort, setSort] = useState("Taxonomy order");
  const [view, setView] = useState("Browse");
  const [selected, setSelected] = useState<Rule | null>(null);
  const [menuFor, setMenuFor] = useState<string | null>(null);
  const [bookmarks, setBookmarks] = useState<string[]>(["Present Perfect vs Past Simple", "Articles: a, an, the", "Past Simple: irregular verbs"]);
  const [recent, setRecent] = useState<string[]>([]);
  const [kbState, setKbState] = useState("Active learner");
  const rules = language === "Spanish" ? spanishRules : language === "English" ? englishRules : [];
  const fresh = preview === "new" || kbState === "New learner" || kbState === "No assessment evidence";
  const masteryOf = (r: Rule) => fresh ? "Not assessed" : r.mastery;
  const reviewOf = (r: Rule) => fresh ? "" : r.review;
  const pathOf = (r: Rule) => kbState === "New learner" ? "Not planned" : r.path;
  const childrenOf = (id: string) => categories.filter(c => c.parent === id).map(c => c.id);
  const belongsTo = (r: Rule, id: string): boolean => r.category === id || childrenOf(id).some(child => belongsTo(r, child));
  const countOf = (id: string) => rules.filter(r => belongsTo(r, id)).length;
  const domainCount = (d: Domain) => rules.filter(r => r.domain === d).length;
  const categoryName = categories.find(c => c.id === category)?.title;
  const categoryPath = (id: string): string[] => { const c = categories.find(x => x.id === id); return c ? [...(c.parent ? categoryPath(c.parent) : []), c.title] : []; };
  const visible = useMemo(() => {
    let rows = rules.filter(r => {
      const text = `${r.title} ${r.description} ${r.keywords || ""} ${r.example || ""} ${r.mistake || ""}`.toLowerCase();
      if (view === "Bookmarks" && !bookmarks.includes(r.title)) return false;
      if (view === "Recently viewed" && !recent.includes(r.title)) return false;
      if (view === "Browse" && !query.trim() && (r.domain !== domain || (category && !belongsTo(r, category)))) return false;
      if (query.trim() && !text.includes(query.trim().toLowerCase())) return false;
      if (preset === "Currently learning" && masteryOf(r) !== "Learning") return false;
      if (preset === "Needs review" && !["Due", "Overdue"].includes(reviewOf(r))) return false;
      if (preset === "Mastered" && masteryOf(r) !== "Mastered") return false;
      if (preset === "Not assessed" && masteryOf(r) !== "Not assessed") return false;
      if (preset === "In my learning path" && pathOf(r) !== "In path") return false;
      if (preset === "Bookmarked" && !bookmarks.includes(r.title)) return false;
      if (masteryFilter !== "Any" && masteryOf(r) !== masteryFilter) return false;
      if (level !== "Any" && !r.cefr.includes(level)) return false;
      return true;
    });
    if (sort === "Alphabetical") rows = rows.sort((a, b) => a.title.localeCompare(b.title));
    if (sort === "Review priority") rows = rows.sort((a, b) => ({ Overdue: 0, Due: 1, Scheduled: 2, "": 3 }[reviewOf(a)] - { Overdue: 0, Due: 1, Scheduled: 2, "": 3 }[reviewOf(b)]));
    if (view === "Recently viewed") rows = rows.sort((a, b) => recent.indexOf(a.title) - recent.indexOf(b.title));
    return rows;
  }, [rules, view, bookmarks, recent, query, domain, category, preset, masteryFilter, level, sort, fresh, kbState]);
  const openPreview = (r: Rule) => { setSelected(r); setRecent(old => [r.title, ...old.filter(x => x !== r.title)]); setMenuFor(null); };
  const toggleBookmark = (title: string) => setBookmarks(old => old.includes(title) ? old.filter(x => x !== title) : [...old, title]);
  const reset = () => { setQuery(""); setPreset("All knowledge"); setLevel("Any"); setMasteryFilter("Any"); setCategory(null); setView("Browse"); };
  const categoryRows = categories.filter(c => c.domain === domain && (!c.parent || tenseOpen));
  const categoryItems = rules.filter(r => r.domain === domain && (!category || belongsTo(r, category)));
  const masteryCount = categoryItems.filter(r => masteryOf(r) === "Mastered").length;
  const learningCount = categoryItems.filter(r => masteryOf(r) === "Learning").length;
  const dueCount = categoryItems.filter(r => ["Due", "Overdue"].includes(reviewOf(r))).length;
  return <div className="space-y-4 font-[Inter,Geist,system-ui,sans-serif] text-[#18181b]">
    <header className="flex flex-wrap items-start justify-between gap-4"><div><p className="text-[12px] text-[#71717a]">learning / Knowledge Base</p><h1 className="mt-1 text-[24px] font-semibold tracking-tight">Knowledge Base</h1><p className="text-[14px] text-[#71717a]">Explore, understand and practise the language.</p><div className="mt-2 flex flex-wrap gap-2 text-[11px]"><span className="rounded-full border border-[#e4e4e7] bg-white px-2 py-0.5">{language || "No language"}</span><span className="rounded-full border border-[#e4e4e7] bg-white px-2 py-0.5">{domains.length} domains · {domains.filter(d => domainCount(d)).length} with published content</span><span className="rounded-full border border-[#e4e4e7] bg-white px-2 py-0.5">Explanations: English</span></div></div><div className="flex flex-wrap items-center gap-2"><label className="text-[12px] text-[#71717a]">Preview state <select className="ml-1 rounded-lg border border-[#e4e4e7] bg-white px-2 py-1.5 text-[12px] text-[#18181b]" value={kbState} onChange={e => setKbState(e.target.value)}>{["Active learner", "New learner", "No assessment evidence"].map(x => <option key={x}>{x}</option>)}</select></label>{["Bookmarks", "Recently viewed", "Reference Library"].map(x => <button key={x} className={smallButton + (view === x ? " bg-[#f4f4f5] font-semibold" : "")} onClick={() => { setView(view === x ? "Browse" : x); setQuery(""); }}>{x}{x === "Bookmarks" ? ` (${bookmarks.length})` : ""}</button>)}</div></header>
    {view === "Browse" && <div><div className="flex h-11 items-center gap-2 rounded-[10px] border border-[#e4e4e7] bg-white px-3"><Search size={17} className="text-[#71717a]" /><input type="search" aria-label="Search the knowledge base" placeholder="Search rules, concepts, examples..." value={query} onChange={e => setQuery(e.target.value)} className="min-w-0 flex-1 bg-transparent text-[14px] outline-none" />{query && <button aria-label="Clear search" onClick={() => setQuery("")}><X size={15} /></button>}</div>{!query && <p className="mt-1.5 text-[11px] text-[#71717a]">Searches titles, alternative names, keywords, examples and common mistakes. Try: {['Present Perfect', 'Did you went', 'Since and for', 'Articles'].map(term => <button key={term} className="mr-2 text-[#18181b] underline" onClick={() => setQuery(term)}>{term}</button>)}</p>}</div>}
    <div className="grid items-start gap-4 lg:grid-cols-[280px_minmax(0,1fr)]">
      {view === "Browse" && !query && treeOpen && <nav aria-label="Knowledge categories" className={surface + " p-2"}><div className="flex items-center justify-between px-2 py-1.5 text-[11px] font-medium uppercase tracking-wide text-[#71717a]"><span>{domain}</span><button className="normal-case tracking-normal" onClick={() => setTreeOpen(false)}>Hide</button></div><button className={"flex w-full justify-between rounded-md px-2 py-1.5 text-left text-[13px] " + (!category ? "bg-[#f4f4f5] font-semibold" : "hover:bg-[#f4f4f5]")} onClick={() => setCategory(null)}><span>All {domain}</span><span className="text-[#71717a]">{domainCount(domain)}</span></button>{categoryRows.map(c => <div key={c.id} className={"flex items-center " + (c.parent ? "pl-4" : "")}><button aria-label={c.id === "tense" ? (tenseOpen ? "Collapse Verb Tenses and Aspect" : "Expand Verb Tenses and Aspect") : undefined} onClick={() => c.id === "tense" && setTenseOpen(v => !v)} className="flex h-7 w-5 shrink-0 items-center justify-center text-[#71717a]">{c.id === "tense" ? tenseOpen ? <ChevronDown size={11} /> : <ChevronRight size={11} /> : null}</button><button className={"flex min-w-0 flex-1 justify-between gap-2 rounded-md px-2 py-1 text-left text-[12px] " + (category === c.id ? "bg-[#f4f4f5] font-semibold" : "hover:bg-[#f4f4f5]") + (!countOf(c.id) ? " text-[#8b8b94]" : "")} onClick={() => setCategory(c.id)}><span className="truncate">{c.title}</span><span className="text-[#71717a]">{countOf(c.id)}</span></button></div>)}</nav>}
      <div className={(view === "Browse" && !query && treeOpen ? "" : "lg:col-span-2 ") + "min-w-0 space-y-3"}>
        {view === "Browse" && <div className="flex flex-wrap items-center gap-2">{!treeOpen && !query && <button className={smallButton} onClick={() => setTreeOpen(true)}>Show categories</button>}<div role="tablist" aria-label="Knowledge domains" className="flex flex-wrap gap-1 rounded-lg bg-[#f4f4f5] p-1">{domains.map(d => <button key={d} role="tab" aria-selected={domain === d} className={"rounded-md px-3 py-1.5 text-[12px] " + (domain === d ? "bg-white font-semibold shadow-sm" : "text-[#71717a]")} onClick={() => { setDomain(d); setCategory(null); setQuery(""); }}><span>{d}</span> <span className="text-[#71717a]">{domainCount(d) || "not published"}</span></button>)}</div></div>}
        {view === "Reference Library" ? <section className={surface + " p-5"}><h2 className="text-[19px] font-semibold">Reference Library</h2><p className="mt-1 text-[12px] text-[#71717a]">These are reference candidates from the original design. Verified source links are not connected to individual rules yet.</p><div className="mt-4 divide-y divide-[#e4e4e7] text-[13px]">{language === "English" ? ["Essential Grammar in Use · Raymond Murphy · Cambridge University Press", "English Grammar in Use · Raymond Murphy · Cambridge University Press", "Advanced Grammar in Use · Martin Hewings · Cambridge University Press"].map(s => <p key={s} className="py-3">{s}</p>) : <p className="py-3">Reference sources pending verification.</p>}</div></section> : <>
          {view === "Browse" && !query && domainCount(domain) > 0 && <section className={surface + " space-y-2 px-5 py-4"}><p className="text-[11px] text-[#71717a]">{domain}{category ? " › " + categoryPath(category).join(" › ") : ""}</p><h2 className="text-[20px] font-semibold">{categoryName || domain}</h2><p className="text-[13px] text-[#71717a]">{category === "tense" ? "Understand how English expresses time, duration, completed actions, ongoing situations, and relationships between events." : category ? "Published knowledge items in this category." : description[domain]}</p><p className="text-[12px] text-[#71717a]"><b className="font-medium text-[#18181b]">{categoryItems.length}</b> published items {fresh ? "· Personal status unknown until you start learning" : `· ${masteryCount} mastered · ${learningCount} learning · ${dueCount} due for review`}</p><div className="flex flex-wrap items-center gap-2"><span className="text-[11px] text-[#71717a]">Subcategories</span>{categories.filter(c => c.domain === domain && c.parent === (category || undefined)).map(c => <button key={c.id} className="rounded-full border border-[#e4e4e7] px-2.5 py-1 text-[11px] hover:bg-[#f4f4f5]" onClick={() => { setCategory(c.id); if (c.id === "tense") setTenseOpen(true); }}>{c.title} · {countOf(c.id)}</button>)}</div></section>}
          {(query || view !== "Browse") && <div><h2 className="text-[18px] font-semibold">{query ? `Results for “${query}”` : view}</h2><p className="text-[12px] text-[#71717a]">{query ? `${visible.length} matching items across all domains` : view === "Bookmarks" ? "Personal bookmarks. They do not influence mastery." : "Items you opened recently."}</p></div>}
          {view === "Browse" && domainCount(domain) > 0 && <><div className="flex flex-wrap items-center gap-2"><div className="flex flex-1 flex-wrap gap-1.5">{["All knowledge", "Currently learning", "Needs review", "Mastered", "Not assessed", "In my learning path", "Bookmarked"].map(p => <button key={p} aria-pressed={preset === p} className={"rounded-full border px-2.5 py-1 text-[12px] " + (preset === p ? "border-[#18181b] bg-[#18181b] text-white" : "border-[#e4e4e7] bg-white hover:bg-[#f4f4f5]")} onClick={() => setPreset(p)}>{p}</button>)}</div><button className={smallButton} aria-expanded={filtersOpen} onClick={() => setFiltersOpen(x => !x)}>Filters{level !== "Any" || masteryFilter !== "Any" ? " · active" : ""}</button><label className="flex items-center gap-1 text-[11px] text-[#71717a]">Sort <select className="rounded-lg border border-[#e4e4e7] bg-white px-2 py-1.5 text-[12px] text-[#18181b]" value={sort} onChange={e => setSort(e.target.value)}>{["Taxonomy order", "Alphabetical", "Review priority"].map(x => <option key={x}>{x}</option>)}</select></label></div>{filtersOpen && <div className={surface + " flex flex-wrap items-center gap-3 p-3 text-[12px]"}><label>Mastery <select className="ml-1 rounded-md border p-1" value={masteryFilter} onChange={e => setMasteryFilter(e.target.value)}>{["Any", "Not assessed", "Not started", "Learning", "Practiced", "Mastered"].map(x => <option key={x}>{x}</option>)}</select></label><label>CEFR metadata <select className="ml-1 rounded-md border p-1" value={level} onChange={e => setLevel(e.target.value)}>{["Any", "A1", "A2", "B1", "B2"].map(x => <option key={x}>{x}</option>)}</select></label><button className="underline" onClick={reset}>Reset filters</button></div>}</>}
          <section className={surface + " overflow-visible"}><p className="px-4 py-2.5 text-[11px] text-[#71717a]">{visible.length} {visible.length === 1 ? "item" : "items"}{view === "Browse" && sort === "Taxonomy order" ? " · taxonomy order" : ""}</p>{visible.length ? <ul className="divide-y divide-[#e4e4e7]">{visible.map(r => <li key={r.title} className="relative flex flex-wrap items-center gap-2 px-3 py-2.5"><button aria-label={bookmarks.includes(r.title) ? `Remove bookmark: ${r.title}` : `Bookmark: ${r.title}`} aria-pressed={bookmarks.includes(r.title)} onClick={() => toggleBookmark(r.title)} className="flex size-8 shrink-0 items-center justify-center rounded-md hover:bg-[#f4f4f5]"><Bookmark size={16} fill={bookmarks.includes(r.title) ? "currentColor" : "none"} /></button><div className="min-w-[190px] flex-[1_1_260px]"><div className="flex flex-wrap items-baseline gap-2"><button className="text-left text-[13px] font-medium hover:underline" onClick={() => openPreview(r)}>{r.title}</button><span className="text-[11px] text-[#71717a]">{r.cefr}</span></div><p className="text-[12px] text-[#71717a]">{r.description}</p><p className="text-[11px] text-[#71717a]">{r.domain} › {categoryPath(r.category).join(" › ")}</p></div><div className="flex flex-wrap items-center gap-1.5 text-[11px]"><span className={"rounded-full px-2 py-0.5 " + (masteryOf(r) === "Mastered" ? "bg-[#f0fdf4] text-[#15803d]" : masteryOf(r) === "Learning" ? "bg-[#eff6ff] text-[#2563eb]" : "bg-[#f4f4f5] text-[#52525b]")}>{masteryOf(r) === "Mastered" ? "✓ " : masteryOf(r) === "Not started" || masteryOf(r) === "Not assessed" ? "○ " : "◐ "}{masteryOf(r)}</span>{["Due", "Overdue"].includes(reviewOf(r)) && <span className="rounded-full bg-[#fffbeb] px-2 py-0.5 text-[#b45309]">◷ Review {reviewOf(r).toLowerCase()}</span>}{["In path", "Recommended"].includes(pathOf(r)) && <span className="whitespace-nowrap text-[#2563eb]">◆ {pathOf(r) === "In path" ? "In your path" : "Recommended next"}</span>}</div><div className="flex items-center gap-1"><button className={smallButton} onClick={() => openPreview(r)}>Preview</button><button className={smallButton} onClick={() => go("Practice")}>Practise</button><button aria-label={`More actions for ${r.title}`} className="flex size-8 items-center justify-center rounded-md hover:bg-[#f4f4f5]" onClick={() => setMenuFor(menuFor === r.title ? null : r.title)}><MoreHorizontal size={16} /></button></div>{menuFor === r.title && <div className="absolute right-3 top-11 z-10 w-48 rounded-lg border border-[#e4e4e7] bg-white p-1 shadow-lg"><button className="block w-full rounded-md px-3 py-2 text-left text-[12px] hover:bg-[#f4f4f5]" onClick={() => { openPreview(r); setMenuFor(null); }}>Open knowledge item</button><button className="block w-full rounded-md px-3 py-2 text-left text-[12px] hover:bg-[#f4f4f5]" onClick={() => { toggleBookmark(r.title); setMenuFor(null); }}>{bookmarks.includes(r.title) ? "Remove bookmark" : "Bookmark"}</button><button className="block w-full rounded-md px-3 py-2 text-left text-[12px] hover:bg-[#f4f4f5]" onClick={() => go("Practice")}>Open practice</button></div>}</li>)}</ul> : <div className="p-8 text-center"><h3 className="text-[14px] font-semibold">{query ? `No results for “${query}”` : view === "Bookmarks" ? "No bookmarks yet" : view === "Recently viewed" ? "Nothing viewed yet" : domainCount(domain) === 0 ? "No published content in this domain yet" : "No items match these filters"}</h3><p className="mt-1 text-[12px] text-[#71717a]">{domainCount(domain) === 0 && !query ? "Other domains remain available." : "Try another category, search term or filter."}</p><button className="mt-3 text-[12px] underline" onClick={reset}>Show all knowledge</button></div>}</section>
        </>}</div>
    </div>
    {selected && <div className="fixed inset-0 z-50 bg-black/30" onMouseDown={() => setSelected(null)}><section role="dialog" aria-modal="true" aria-label="Knowledge item preview" className="ml-auto flex h-full w-full max-w-[480px] flex-col bg-white shadow-xl" onMouseDown={e => e.stopPropagation()}><div className="flex items-start justify-between border-b border-[#e4e4e7] p-5"><div><p className="text-[11px] text-[#71717a]">{selected.domain} › {categoryPath(selected.category).join(" › ")}</p><h2 className="mt-2 text-[21px] font-semibold">{selected.title}</h2><p className="mt-1 text-[11px] text-[#71717a]">{selected.cefr} · {selected.domain}</p></div><button aria-label="Close preview" onClick={() => setSelected(null)}><X size={18} /></button></div><div className="flex-1 space-y-5 overflow-y-auto p-5 text-[13px]"><p>{selected.description}</p><div><p className="mb-2 text-[11px] font-semibold text-[#71717a]">Your status (independent dimensions)</p><dl className="grid grid-cols-[100px_1fr] gap-y-1"><dt className="text-[#71717a]">Mastery</dt><dd>{masteryOf(selected)}</dd><dt className="text-[#71717a]">Review</dt><dd>{reviewOf(selected) || "Not scheduled"}</dd><dt className="text-[#71717a]">Learning Path</dt><dd>{pathOf(selected)}</dd></dl></div>{selected.example && <div><p className="text-[11px] font-semibold text-[#71717a]">Example · demonstration content</p><p>{selected.example}</p></div>}{selected.mistake && <div><p className="text-[11px] font-semibold text-[#71717a]">Common mistake</p><p>{selected.mistake}</p></div>}<div><p className="text-[11px] font-semibold text-[#71717a]">Relationships</p><p>Part of {categoryPath(selected.category).join(" › ")}</p><p className="mt-1 text-[#71717a]">Missing prerequisites never block reading.</p></div><div><p className="text-[11px] font-semibold text-[#71717a]">Reference sources</p><p className="text-[#71717a]">No verified source references linked yet.</p></div><p className="rounded-lg bg-[#f4f4f5] p-3 text-[11px]">Opening or reading an item does not change mastery. Tracked practice creates learning evidence.</p></div><div className="flex flex-wrap gap-2 border-t border-[#e4e4e7] p-5"><button className="rounded-lg bg-[#18181b] px-4 py-2 text-[12px] text-white" onClick={() => { setSelected(null); go("Practice"); }}>Practise this item</button><button className={smallButton} onClick={() => toggleBookmark(selected.title)}>{bookmarks.includes(selected.title) ? "Remove bookmark" : "Bookmark"}</button></div></section></div>}
  </div>;
}

export function PracticePage({
  go
}: {
  go: (page: Page) => void;
}) {
  const [kind, setKind] = useState<"review" | "practice" | "session">("review");
  const [started, setStarted] = useState(false);
  const [answer, setAnswer] = useState("");
  const [checked, setChecked] = useState(false);
  const [completed, setCompleted] = useState(0);
  const prompt = kind === "review" ? "Yesterday I ___ to school." : kind === "practice" ? "I have ___ London twice." : "Choose the best sentence for a completed event yesterday.";
  const options = kind === "session" ? ["I have gone yesterday.", "I went yesterday.", "I go yesterday."] : [];
  const correct = kind === "review" ? answer.trim().toLowerCase() === "went" : kind === "practice" ? answer.trim().toLowerCase() === "visited" : answer === "I went yesterday.";
  return <div className="grid items-start gap-6 lg:grid-cols-[minmax(0,1.4fr)_300px]"><div className="space-y-6"><section className={`${card} p-6`}><Tag tone="amber">LEARNING PRACTICE</Tag><h2 className="mt-4 text-[22px] font-semibold">Choose an activity</h2><p className={`mt-1 text-[13px] ${muted}`}>Practice and reviews both create learning evidence. A correct answer here is not mastery by itself.</p><div className="mt-5 grid gap-3 sm:grid-cols-3">{[["review", "Due reviews", "Return to knowledge that needs recall", RefreshCw], ["practice", "Rule practice", "Use a rule in a new context", BookOpen], ["session", "Learning session", "Work through the next activity", Play]].map(([id, title, note, Icon]) => <button key={id as string} className={`rounded-lg border p-4 text-left ${kind === id ? "border-[#333] bg-[#f5f5f3]" : "border-[#e6e6e3]"}`} onClick={() => {
            setKind(id as typeof kind);
            setStarted(false);
            setChecked(false);
            setAnswer("");
          }}><Icon size={18} /><b className="mt-2 block text-[13px]">{title as string}</b><span className={`mt-1 block text-[11px] ${muted}`}>{note as string}</span></button>)}</div></section>{!started ? <section className={`${card} p-6`}><h2 className="text-[18px] font-semibold">{kind === "review" ? "Past Simple: irregular verbs" : kind === "practice" ? "Present Perfect" : "Talking about past experiences"}</h2><p className={`mt-2 text-[12px] ${muted}`}>One sample activity · about 5 minutes · linked to the Knowledge Base.</p><button className={`${button} mt-5`} onClick={() => setStarted(true)}>Start activity</button></section> : <section className={`${card} p-6`}><div className="flex justify-between"><Tag tone="blue">{kind === "review" ? "Review" : kind === "practice" ? "Practice" : "Session"}</Tag><button className="text-[12px] underline" onClick={() => setStarted(false)}>Save and exit</button></div><h2 className="mt-5 text-[20px] font-semibold">{prompt}</h2><p className={`mt-2 text-[12px] ${muted}`}>{kind === "review" ? "Answer from memory." : "Use the knowledge item in context."}</p>{options.length ? <fieldset className="mt-5 space-y-2"><legend className="sr-only">Choose an answer</legend>{options.map(v => <label key={v} className="flex gap-2 rounded-md border p-3 text-[13px]"><input type="radio" checked={answer === v} onChange={() => {
              setAnswer(v);
              setChecked(false);
            }} />{v}</label>)}</fieldset> : <label className="mt-5 block text-[12px]">Your answer<input value={answer} onChange={e => {
            setAnswer(e.target.value);
            setChecked(false);
          }} className="mt-1 block w-full rounded-md border border-[#ddd] p-3 text-[13px]" placeholder="Type your answer" /></label>}{checked && <div role="status" className={`mt-4 rounded-md p-3 text-[12px] ${correct ? "bg-[#eaf7ec] text-[#24833b]" : "bg-[#fff8e9] text-[#9a5d16]"}`}>{correct ? "Correct. This answer adds practice evidence." : "Needs another attempt. Review the rule and try again."}</div>}<div className="mt-5 flex flex-wrap gap-2"><button className={button} disabled={!answer} onClick={() => {
            setChecked(true);
            if (correct && !checked) setCompleted(v => v + 1);
          }}>Check answer</button><button className={secondary} onClick={() => {
            setStarted(false);
            setChecked(false);
            setAnswer("");
          }}>Back to activities</button></div></section>}</div><aside className="space-y-6"><section className={`${card} p-5`}><h2 className="font-semibold">Review queue</h2><p className={`mt-2 text-[12px] ${muted}`}>Past Simple: irregular verbs · overdue</p><p className={`mt-2 text-[12px] ${muted}`}>Articles: a, an, the · due today</p></section><section className={`${card} p-5`}><h2 className="font-semibold">This preview</h2><p className={`mt-2 text-[12px] ${muted}`}>{completed} sample answers completed in this view. Real evidence lives in the application.</p><button className="mt-3 text-[12px] underline" onClick={() => go("Knowledge Base")}>Browse linked knowledge</button></section></aside></div>;
}
export function ProgressPage({
  preview,
  go
}: {
  preview: Preview;
  go: (page: Page) => void;
}) {
  const fresh = preview === "new";
  const [tab, setTab] = useState<"Skills" | "Evidence" | "Mistakes">("Skills");
  return <div className="grid items-start gap-6 lg:grid-cols-[minmax(0,1.4fr)_300px]"><div className="space-y-6"><section className={`${card} p-6`}><h2 className="text-[22px] font-semibold">Your learning progress</h2><p className={`mt-1 text-[13px] ${muted}`}>Study time, assessment estimates and knowledge mastery are separate measures.</p><div className="mt-5 grid grid-cols-2 gap-3 sm:grid-cols-3">{[["Study time", fresh ? "—" : "2h 15m"], ["Knowledge items", fresh ? "—" : "24 mastered"], ["Review due", fresh ? "—" : "8 items"]].map(([k, v]) => <div key={k} className="rounded-lg bg-[#f5f5f4] p-4"><p className={`text-[11px] ${muted}`}>{k}</p><b className="mt-1 block text-[19px]">{v}</b></div>)}</div></section><section className={card}><div className="flex gap-4 border-b border-[#e8e8e8] px-5">{(["Skills", "Evidence", "Mistakes"] as const).map(x => <button key={x} className={`py-3 text-[13px] ${tab === x ? "border-b-2 border-[#222] font-semibold" : muted}`} onClick={() => setTab(x)}>{x}</button>)}</div>{tab === "Skills" ? <div className="p-5"><p className={`mb-3 text-[11px] ${muted}`}>Estimates require assessment evidence. Unassessed skills have no level.</p>{skills.map((s, i) => <div key={s} className="flex items-center justify-between border-t border-[#eee] py-3 text-[13px]"><span>{s}</span><span className="flex gap-3"><span className="text-[#289145]" aria-hidden="true">{!fresh && i < 4 ? "━━ ━" : "— — —"}</span><span className={fresh || i > 3 ? muted : "font-medium"}>{fresh || i > 3 ? "Not assessed" : ["B1", "B1", "B2", "A2"][i]}</span></span></div>)}</div> : tab === "Evidence" ? <div className="space-y-3 p-5 text-[12px]">{fresh ? <p className={muted}>No assessment or practice evidence yet.</p> : <><p>✓ Grammar assessment · 12 answers · preliminary estimate</p><p>✓ Present Perfect practice · 3 attempts</p><p>✓ Reading comprehension · 4 texts</p><p className={muted}>Opening reference content is not evidence of mastery.</p></>}</div> : <div className="space-y-3 p-5 text-[12px]">{fresh ? <p className={muted}>Mistakes appear after practice.</p> : <><p><b>Irregular verb forms</b> · repeated 3 times this week</p><p><b>Articles</b> · due for review</p><button className="underline" onClick={() => go("Practice")}>Practise these items</button></>}</div>}</section></div><aside className="space-y-6"><section className={`${card} p-5`}><h2 className="font-semibold">Assessment history</h2>{fresh ? <p className={`mt-2 text-[12px] ${muted}`}>No assessments completed yet.</p> : <><p className="mt-3 text-[12px]">Quick assessment · Grammar · completed today</p><p className={`text-[11px] ${muted}`}>Preliminary result; not a certificate.</p></>}<button className={`${secondary} mt-4`} onClick={() => go("Assessments")}>Open assessments</button></section><section className={`${card} p-5`}><h2 className="font-semibold">What affects recommendations?</h2><p className={`mt-2 text-[12px] ${muted}`}>Completed assessments, independent recall, repeated practice and due reviews can change what appears next.</p><button className="mt-3 text-[12px] underline" onClick={() => go("Learning Path")}>View learning path</button></section></aside></div>;
}
export function TutorPage({
  go
}: {
  go: (page: Page) => void;
}) {
  const [messages, setMessages] = useState<{
    from: "tutor" | "learner";
    text: string;
  }[]>([{
    from: "tutor",
    text: "Hi! We can practise talking about past experiences. What did you do last weekend?"
  }]);
  const [draft, setDraft] = useState("");
  const [topic, setTopic] = useState("Past experiences");
  const send = () => {
    if (!draft.trim()) return;
    setMessages(v => [...v, {
      from: "learner",
      text: draft.trim()
    }, {
      from: "tutor",
      text: "Thanks for sharing. Try adding one detail about when it happened, then compare it with something you have done recently."
    }]);
    setDraft("");
  };
  return <div className="grid items-start gap-6 lg:grid-cols-[minmax(0,1.4fr)_300px]"><section className={`${card} flex min-h-[620px] flex-col`}><SectionHead title="AI Tutor" note="A design conversation connected to your learning objective." action={<Tag tone="blue">Preview conversation</Tag>} /><div className="flex-1 space-y-4 p-5">{messages.map((m, i) => <div key={i} className={`max-w-[80%] rounded-lg p-3 text-[13px] ${m.from === "learner" ? "ml-auto bg-[#eef1ff]" : "bg-[#f3f3f3]"}`}><p className={`mb-1 text-[10px] font-semibold ${muted}`}>{m.from === "learner" ? "You" : "AI Tutor"}</p>{m.text}</div>)}</div><form className="flex gap-2 border-t border-[#eee] p-4" onSubmit={e => {
        e.preventDefault();
        send();
      }}><input aria-label="Message AI Tutor" placeholder="Write a message…" value={draft} onChange={e => setDraft(e.target.value)} className="min-w-0 flex-1 rounded-md border border-[#ddd] px-3 text-[13px]" /><button className={button} disabled={!draft.trim()}>Send</button></form></section><aside className="space-y-6"><section className={`${card} p-5`}><h2 className="font-semibold">Conversation focus</h2><select aria-label="Conversation focus" value={topic} onChange={e => setTopic(e.target.value)} className="mt-3 w-full rounded-md border p-2 text-[12px]">{["Past experiences", "Travel vocabulary", "Present Perfect"].map(t => <option key={t}>{t}</option>)}</select><p className={`mt-3 text-[12px] ${muted}`}>Current topic: {topic}. Tutor guidance and evidence extraction are product goals; this chat is illustrative.</p><button className="mt-3 text-[12px] underline" onClick={() => go("Knowledge Base")}>Open related knowledge</button></section><section className={`${card} p-5`}><Sparkles size={17} /><h2 className="mt-2 font-semibold">Learning context</h2><p className={`mt-2 text-[12px] ${muted}`}>The future Tutor will use the learner profile, current path, knowledge items and recent evidence.</p></section></aside></div>;
}
export function SettingsPage({
  language,
  setLanguage,
  go,
  onReset
}: {
  language: Language;
  setLanguage: (value: Language) => void;
  go: (page: Page) => void;
  onReset: () => void;
}) {
  const [explain, setExplain] = useState("English");
  const [weekly, setWeekly] = useState("5 hours");
  const [theme, setTheme] = useState("System");
  const [saved, setSaved] = useState(false);
  const [ai, setAi] = useState("Not connected");
  const [confirmReset, setConfirmReset] = useState(false);
  return <div className="grid items-start gap-6 lg:grid-cols-[minmax(0,1.2fr)_300px]"><div className="space-y-6"><section className={card}><SectionHead title="Language settings" note="Each language has its own AI Teacher, plan, assessments and history." /><div className="space-y-4 p-5 text-[13px]"><label className="block">Learning language<select className="mt-1 block w-full rounded-md border p-2" value={language} onChange={e => setLanguage(e.target.value as Language)}>{["English", "Spanish", "German"].map(x => <option key={x}>{x}</option>)}</select></label><label className="block">Explanation language<select className="mt-1 block w-full rounded-md border p-2" value={explain} onChange={e => setExplain(e.target.value)}>{["English", "Russian", "Spanish", "German"].map(x => <option key={x}>{x}</option>)}</select></label><label className="block">Weekly study target<select className="mt-1 block w-full rounded-md border p-2" value={weekly} onChange={e => setWeekly(e.target.value)}>{["3 hours", "5 hours", "8 hours"].map(x => <option key={x}>{x}</option>)}</select></label><button className={button} onClick={() => setSaved(true)}>Save preferences</button>{saved && <p role="status" className="text-[#24833b]">Preferences saved in this preview.</p>}</div></section><section className={card}><SectionHead title="AI provider" note="Connection status and generation settings." /><div className="p-5"><p className="text-[13px]">Status: <b>{ai}</b></p><p className={`mt-2 text-[12px] ${muted}`}>The prototype never asks for an API key. Provider credentials belong only in the real app.</p><button className={`${secondary} mt-4`} onClick={() => setAi(ai === "Not connected" ? "Demo connected" : "Not connected")}>{ai === "Not connected" ? "Preview connected state" : "Preview disconnected state"}</button></div></section><section className={card}><SectionHead title="Appearance" /><div className="p-5"><label className="text-[13px]">Theme<select className="ml-3 rounded-md border p-2" value={theme} onChange={e => setTheme(e.target.value)}>{["System", "Light", "Dark"].map(x => <option key={x}>{x}</option>)}</select></label><p className={`mt-2 text-[11px] ${muted}`}>Theme selection is represented in the design; this preview uses the original light visual system.</p></div></section></div><aside className="space-y-6"><section className={`${card} p-5`}><Settings2 size={18} /><h2 className="mt-2 font-semibold">Account</h2><p className={`mt-2 text-[12px] ${muted}`}>Demo learner · design preview. No personal account data is included.</p><button className={`${secondary} mt-4`} onClick={() => setConfirmReset(true)}>Start over in this preview</button>{confirmReset && <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/30 p-4" role="presentation"><section role="dialog" aria-modal="true" aria-label="Reset demo learner" className={`${card} w-full max-w-[420px] p-6 shadow-xl`}><h2 className="text-[20px] font-semibold">Start from zero?</h2><p className={`mt-2 text-[13px] ${muted}`}>This clears only this browser's sample assessment progress and returns to the onboarding design. It does not change your real learning account.</p><div className="mt-5 flex gap-2"><button className={button} onClick={() => { setConfirmReset(false); onReset(); }}>Reset demo</button><button className={secondary} onClick={() => setConfirmReset(false)}>Cancel</button></div></section></div>}</section><section className={`${card} p-5`}><h2 className="font-semibold">Language profiles</h2><p className={`mt-2 text-[12px] ${muted}`}>Switching the language changes the preview context.</p><button className="mt-3 text-[12px] underline" onClick={() => go("Overview")}>Back to Overview</button></section></aside></div>;
}
