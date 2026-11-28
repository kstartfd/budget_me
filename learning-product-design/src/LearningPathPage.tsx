import { useCallback, useEffect, useRef, useState } from "react";
import { Check, ChevronRight, X } from "lucide-react";
import type { Language, Page, Preview } from "./WorkspacePages";

type ActivityState = "Completed" | "In progress" | "Not started";
type Activity = {
  title: string;
  type: "Learn" | "Practice" | "Apply" | "Assess" | "Review";
  skill: string;
  minutes: number;
  knowledge: string;
  state: ActivityState;
  mastery: "Mastered" | "Practiced" | "Learning" | "Not yet learned";
  review?: "Due" | "Overdue" | "Scheduled";
  prerequisite?: [string, string];
};
type Module = { title: string; description: string; objective: string; state: "Completed" | "In progress" | "Upcoming"; activities: Activity[] };
type Change = { title: string; reason: string; when: string; item: string };
type QueueItem = { title: string; reason: string; due: string; minutes: number };
type Plan = { objective: string; target: string; session: string; goals: string[]; modules: Module[]; reviews: QueueItem[]; changes: Change[]; history: Change[] };

const card = "rounded-xl border border-[#e6e6e3] bg-white";
const muted = "text-[#777773]";
const button = "inline-flex min-h-9 items-center justify-center gap-2 rounded-md bg-[#292929] px-3.5 py-2 text-[12px] font-semibold text-white hover:bg-black focus-visible:outline-2 focus-visible:outline-offset-2 disabled:cursor-not-allowed disabled:opacity-40";
const secondary = "inline-flex min-h-9 items-center justify-center gap-2 rounded-md border border-[#e6e6e3] bg-white px-3.5 py-2 text-[12px] font-medium hover:bg-[#f5f5f3] focus-visible:outline-2 focus-visible:outline-offset-2 disabled:cursor-not-allowed disabled:opacity-40";
const skills = ["Grammar", "Vocabulary", "Reading", "Listening", "Speaking", "Writing", "Pronunciation"];
const step = (title: string, type: Activity["type"], skill: string, minutes: number, state: ActivityState, mastery: Activity["mastery"], knowledge: string, review?: Activity["review"], prerequisite?: Activity["prerequisite"]): Activity => ({ title, type, skill, minutes, state, mastery, knowledge, review, prerequisite });

// The activity names and their example states come from Learning Path.dc.html.
const englishPlan: Plan = {
  objective: "Communicate confidently in everyday and professional situations.", target: "5 hours", session: "30 minutes",
  goals: ["Improve everyday communication", "Participate in professional meetings", "Understand spoken English", "Express opinions confidently"],
  modules: [
    { title: "Sentence foundations", description: "Core sentence structure and everyday verb forms.", objective: "Build correct basic sentences about daily life.", state: "Completed", activities: [
      step("Be and have: basic forms", "Learn", "Grammar", 8, "Completed", "Mastered", "To be / have"),
      step("Everyday vocabulary: routines", "Learn", "Vocabulary", 10, "Completed", "Mastered", "Daily routine words"),
      step("Present Simple: third person", "Practice", "Grammar", 10, "Completed", "Practiced", "Present Simple", "Due"),
      step("Word order in statements", "Practice", "Grammar", 10, "Completed", "Mastered", "Sentence structure"),
      step("Module checkpoint", "Assess", "Grammar", 15, "Completed", "Mastered", "Sentence foundations")
    ] },
    { title: "Talking about the past", description: "Form and use past tenses to describe events.", objective: "Describe what happened using Past Simple.", state: "In progress", activities: [
      step("Past Simple: regular verbs", "Learn", "Grammar", 10, "Completed", "Practiced", "Past Simple"),
      step("Past Simple: irregular verbs", "Practice", "Grammar", 12, "Completed", "Learning", "Irregular verbs", "Overdue"),
      step("Questions and negatives with did", "Learn", "Grammar", 10, "Not started", "Not yet learned", "Past Simple questions"),
      step("A weekend trip", "Practice", "Listening", 10, "Not started", "Not yet learned", "Past narratives"),
      step("Describe your last weekend", "Apply", "Speaking", 10, "Not started", "Not yet learned", "Past Simple in speech", undefined, ["Past Simple: irregular verbs", "This activity requires knowledge of past verb forms."]),
      step("Module checkpoint", "Assess", "Grammar", 15, "Not started", "Not yet learned", "Talking about the past")
    ] },
    { title: "Experiences and achievements", description: "Link past events to the present.", objective: "Talk about life experience and work history.", state: "Upcoming", activities: [
      step("Present Perfect: forms", "Learn", "Grammar", 10, "Not started", "Not yet learned", "Present Perfect"),
      step("Present Perfect vs Past Simple", "Practice", "Grammar", 12, "Not started", "Not yet learned", "Tense contrast", undefined, ["Past Simple: irregular verbs", "Past participles build on past verb forms."]),
      step("Work and career vocabulary", "Learn", "Vocabulary", 10, "Not started", "Not yet learned", "Career words"),
      step("A career story", "Practice", "Reading", 12, "Not started", "Not yet learned", "Narrative reading"),
      step("Talk about your experience", "Apply", "Speaking", 10, "Not started", "Not yet learned", "Experience talk")
    ] },
    { title: "Making plans and predictions", description: "Express future plans and arrangements.", objective: "Discuss plans, schedules and predictions.", state: "Upcoming", activities: [
      step("Future: will and going to", "Learn", "Grammar", 10, "Not started", "Not yet learned", "Future forms"),
      step("Making arrangements", "Practice", "Listening", 10, "Not started", "Not yet learned", "Arrangements"),
      step("Write a short plan", "Apply", "Writing", 12, "Not started", "Not yet learned", "Plans in writing"),
      step("Weak forms in sentences", "Practice", "Pronunciation", 8, "Not started", "Not yet learned", "Sentence rhythm")
    ] },
    { title: "Expressing opinions and arguments", description: "Give, support and challenge opinions.", objective: "Take part in a short discussion.", state: "Upcoming", activities: [
      step("Opinion phrases", "Learn", "Vocabulary", 8, "Not started", "Not yet learned", "Opinion language"),
      step("Agreeing and disagreeing", "Practice", "Speaking", 10, "Not started", "Not yet learned", "Discussion"),
      step("Short opinion text", "Apply", "Writing", 15, "Not started", "Not yet learned", "Opinion writing"),
      step("Module checkpoint", "Assess", "Reading", 15, "Not started", "Not yet learned", "Opinions")
    ] }
  ],
  reviews: [
    { title: "Irregular verbs", reason: "Recurring mistakes", due: "Overdue", minutes: 5 },
    { title: "Articles", reason: "Review due", due: "Due today", minutes: 4 },
    { title: "Present Simple", reason: "Retention check", due: "Scheduled", minutes: 3 }
  ],
  changes: [
    { title: "Extra irregular verb practice added", reason: "Recurring mistakes in 3 recent sessions.", when: "21 Nov", item: "Irregular verbs" },
    { title: "Present Simple moved to review", reason: "Successful independent practice; retention check scheduled.", when: "19 Nov", item: "Present Simple" },
    { title: "Listening activity added", reason: "Improve skill coverage for the listening goal.", when: "17 Nov", item: "A weekend trip" },
    { title: "Articles scheduled for a retention check", reason: "Last practised 9 days ago.", when: "15 Nov", item: "Articles" }
  ],
  history: [
    { when: "2 Oct · Plan created", title: "Initial path created from a sample assessment", reason: "Starting point and four goals applied.", item: "" },
    { when: "9 Oct · Assessment", title: "Reading and Grammar estimates updated", reason: "New sample evidence from a checkpoint.", item: "" },
    { when: "28 Oct · Module completed", title: "Sentence foundations completed", reason: "5 of 5 activities done.", item: "" },
    { when: "3 Nov · Goal change", title: "Professional meetings goal added", reason: "Future activities updated.", item: "" },
    { when: "15 Nov · Rescheduled", title: "Articles retention check scheduled", reason: "Spaced repetition interval reached.", item: "" },
    { when: "17 Nov · Added", title: "Listening activity added to Module 2", reason: "Coverage for the listening goal.", item: "" },
    { when: "21 Nov · Added", title: "Irregular verb practice added", reason: "Recurring mistakes.", item: "" }
  ]
};

const spanishPlan: Plan = {
  objective: "Handle everyday travel and café conversations.", target: "3 hours", session: "20 minutes", goals: ["Travel conversation", "Everyday communication"],
  modules: [
    { title: "Greetings and introductions", description: "Introduce yourself and greet people.", objective: "Hold a short first conversation.", state: "Completed", activities: [
      step("Greetings", "Learn", "Vocabulary", 8, "Completed", "Mastered", "Greetings"),
      step("Ser: identity", "Learn", "Grammar", 10, "Completed", "Practiced", "ser", "Due"),
      step("Introduce yourself", "Apply", "Speaking", 8, "Completed", "Practiced", "Introductions"),
      step("Module checkpoint", "Assess", "Grammar", 10, "Completed", "Mastered", "Introductions")
    ] },
    { title: "Ordering food and drinks", description: "Vocabulary and polite requests for cafés.", objective: "Order and pay in a café.", state: "In progress", activities: [
      step("Present tense: -ar verbs", "Learn", "Grammar", 10, "Completed", "Practiced", "-ar verbs"),
      step("Food and drink words", "Learn", "Vocabulary", 10, "In progress", "Learning", "Food vocabulary"),
      step("Polite requests with querer", "Learn", "Grammar", 10, "Not started", "Not yet learned", "querer"),
      step("Café dialogue", "Practice", "Listening", 10, "Not started", "Not yet learned", "Café dialogue", undefined, ["Food and drink words", "The dialogue uses menu vocabulary."]),
      step("Order a coffee", "Apply", "Speaking", 8, "Not started", "Not yet learned", "Ordering")
    ] },
    { title: "Directions and places", description: "Ask for and give directions.", objective: "Navigate a city in conversation.", state: "Upcoming", activities: [
      step("Places in town", "Learn", "Vocabulary", 8, "Not started", "Not yet learned", "Places"),
      step("Estar: location", "Learn", "Grammar", 10, "Not started", "Not yet learned", "estar"),
      step("Asking directions", "Apply", "Speaking", 10, "Not started", "Not yet learned", "Directions")
    ] }
  ],
  reviews: [
    { title: "Ser vs estar", reason: "Mixed up in 2 exercises", due: "Overdue", minutes: 5 },
    { title: "Gender agreement", reason: "Low accuracy last attempt", due: "Due today", minutes: 4 }
  ],
  changes: [
    { title: "Gender agreement drills added", reason: "4 errors in the last 3 sessions.", when: "20 Nov", item: "Gender agreement" },
    { title: "Café dialogue added", reason: "Matches the travel goal.", when: "18 Nov", item: "Café dialogue" }
  ],
  history: [
    { when: "8 Nov · Plan created", title: "Path created from a manually selected starting point", reason: "Two goals applied.", item: "" },
    { when: "14 Nov · Module completed", title: "Greetings and introductions completed", reason: "4 of 4 activities done.", item: "" },
    { when: "20 Nov · Added", title: "Gender agreement drills added", reason: "Recurring mistakes.", item: "" }
  ]
};

function starterPlan(language: Language): Plan {
  const title = language === "Spanish" ? "Ser and estar" : language === "German" ? "The verb sein" : "The verb to be: am, is, are";
  return { objective: "Introduce yourself and ask simple questions.", target: "3 hours", session: "20 minutes", goals: ["Everyday communication"], reviews: [], changes: [], history: [], modules: [{ title: "First conversations", description: "Introduce yourself and ask simple questions.", objective: "Hold a short first conversation.", state: "In progress", activities: [step(title, "Learn", "Grammar", 10, "Not started", "Not yet learned", title), step("Introduce yourself", "Apply", "Speaking", 10, "Not started", "Not yet learned", "Introductions"), step("Ask simple questions", "Practice", "Speaking", 10, "Not started", "Not yet learned", "Questions")] }] };
}

function Badge({ children, tone = "neutral" }: { children: React.ReactNode; tone?: "neutral" | "blue" | "green" | "amber" }) {
  const colors = { neutral: "bg-[#f3f3f3] text-[#777]", blue: "bg-[#eef1ff] text-[#4e69eb]", green: "bg-[#eaf7ec] text-[#24833b]", amber: "bg-[#fff8e9] text-[#b56417]" };
  return <span className={`inline-flex rounded-full px-2 py-0.5 text-[11px] font-medium ${colors[tone]}`}>{children}</span>;
}

function Drawer({ title, kicker, close, children, footer }: { title: string; kicker?: string; close: () => void; children: React.ReactNode; footer?: React.ReactNode }) {
  const closeRef = useRef<HTMLButtonElement>(null);
  useEffect(() => {
    const previous = document.activeElement as HTMLElement | null;
    closeRef.current?.focus();
    const onKey = (event: KeyboardEvent) => { if (event.key === "Escape") close(); };
    window.addEventListener("keydown", onKey);
    return () => { window.removeEventListener("keydown", onKey); previous?.focus(); };
  }, [close]);
  return <div className="fixed inset-0 z-50 flex justify-end" role="presentation"><div className="absolute inset-0 bg-black/30" onClick={close} /><aside role="dialog" aria-modal="true" aria-label={title} className="relative flex h-full w-full max-w-[460px] flex-col border-l border-[#e6e6e3] bg-white shadow-xl"><div className="flex items-start justify-between gap-3 border-b border-[#eee] p-5"><div>{kicker && <p className={`text-[11px] ${muted}`}>{kicker}</p>}<h2 className="text-[18px] font-semibold">{title}</h2></div><button ref={closeRef} className={secondary + " !min-h-8 !px-2"} aria-label="Close" onClick={close}><X size={16} /></button></div><div className="flex-1 space-y-5 overflow-auto p-5 text-[13px]">{children}</div>{footer && <div className="flex flex-wrap gap-2 border-t border-[#eee] p-5">{footer}</div>}</aside></div>;
}

export function LearningPathPage({ preview, setPreview, language, go, startNewPlan, begin, openActivity, currentActivity, archived }: {
  preview: Preview; setPreview: (value: Preview) => void; language: Language; go: (page: Page) => void; startNewPlan: () => void; begin: (mode: "quick" | "manual" | "scratch") => void; openActivity: (title: string) => void; currentActivity?: string; archived?: Preview;
}) {
  const [tab, setTab] = useState<"My path" | "Review" | "Completed">("My path");
  const [skill, setSkill] = useState("All skills");
  const [kind, setKind] = useState("All types");
  const [opened, setOpened] = useState<string[]>([]);
  const [closed, setClosed] = useState<string[]>([]);
  const [panel, setPanel] = useState<"activity" | "goals" | "history" | null>(null);
  const [selected, setSelected] = useState<{ activity: Activity; module: Module } | null>(null);
  const [purpose, setPurpose] = useState("Everyday and professional communication");
  const [target, setTarget] = useState("");
  const [duration, setDuration] = useState("");
  const [explanation, setExplanation] = useState("English");
  const [targetLevel, setTargetLevel] = useState("No target");
  const [priorities, setPriorities] = useState<string[]>(["Grammar", "Speaking"]);
  const [savedPreferences, setSavedPreferences] = useState(false);
  const [reviewAdded, setReviewAdded] = useState<string[]>([]);
  const completedPlan = preview === "done";
  const noProgram = preview === "new" || preview === "unfinished" || (language === "German" && preview === "concept");
  const plan = preview === "current" || preview === "insufficient" || language === "German" ? starterPlan(language) : language === "Spanish" ? spanishPlan : englishPlan;
  const savedPlan = archived === "current" || archived === "insufficient" || language === "German" ? starterPlan(language) : language === "Spanish" ? spanishPlan : englishPlan;
  const all = plan.modules.flatMap(module => module.activities.map(activity => ({ activity, module })));
  const resumed = currentActivity ? all.find(({ activity }) => activity.title === currentActivity && activity.state !== "Completed") : undefined;
  const currentModule = resumed?.module || plan.modules.find(module => module.state === "In progress");
  const next = completedPlan ? undefined : resumed?.activity || currentModule?.activities.find(activity => activity.state !== "Completed");
  const filtered = plan.modules.map((module, index) => ({ module, index, activities: module.activities.filter(activity => {
    const state = completedPlan ? "Completed" : activity.state;
    return (skill === "All skills" || activity.skill === skill) && (kind === "All types" || activity.type === kind) && (tab === "My path" || (tab === "Review" ? Boolean(activity.review) || reviewAdded.includes(activity.title) : state === "Completed"));
  }) })).filter(item => item.activities.length > 0);
  const reviewItems = [...plan.reviews, ...all.filter(({ activity }) => reviewAdded.includes(activity.title)).map(({ activity }) => ({ title: activity.title, reason: "Added to review", due: "Added", minutes: activity.minutes }))];
  const remaining = completedPlan ? 0 : all.filter(({ activity }) => activity.state !== "Completed").length;
  const finishedCount = completedPlan ? all.length : all.filter(({ activity }) => activity.state === "Completed").length;
  const masteredCount = all.filter(({ activity }) => activity.mastery === "Mastered").length;
  const dueCount = reviewItems.filter(item => item.due !== "Scheduled").length;
  const blocked = preview === "unavailable";
  const aiBlocked = blocked || preview === "missing-ai";
  const openDetails = (module: Module, activity: Activity) => { setSelected({ module, activity }); setPanel("activity"); };
  const closePanel = useCallback(() => setPanel(null), []);
  const start = (activity: Activity) => { setPanel(null); openActivity(activity.title); };

  const historyDrawer = panel === "history" && <Drawer title="Plan history" kicker="Sample learning history" close={closePanel} footer={<button className={secondary} onClick={closePanel}>Close</button>}><p className={muted}>{archived ? "The previous sample plan, results and learning history are kept when you start a new plan." : "Plans, assessment results and completed activities remain in history."}</p>{archived && <section className="rounded-lg border border-[#e6e6e3] p-4"><h3 className="font-semibold">Previous sample plan</h3><p className={`mt-1 text-[12px] ${muted}`}>{savedPlan.objective}</p><details className="mt-3 text-[12px]"><summary className="cursor-pointer underline">View saved modules and activities</summary><ol className="mt-3 space-y-3">{savedPlan.modules.map(module => <li key={module.title}><b>{module.title}</b><ul className={`mt-1 space-y-1 ${muted}`}>{module.activities.map(activity => <li key={activity.title}>{activity.title} · {archived === "done" ? "Completed" : activity.state}</li>)}</ul></li>)}</ol></details></section>}{(archived ? savedPlan : plan).history.length ? <ol className="space-y-4 border-l-2 border-[#eee] pl-4">{(archived ? savedPlan : plan).history.map((entry, index) => <li key={index}><p className={`text-[11px] ${muted}`}>{entry.when}</p><b>{entry.title}</b><p className={muted}>{entry.reason}</p></li>)}</ol> : <p className={muted}>{archived ? "Previous sample plan saved in history." : "No history is shown for this sample."}</p>}<p className={`rounded-lg bg-[#f5f5f4] p-3 text-[12px] ${muted}`}>Deleting history and progress is a separate Settings action. Its scope is still to be decided.</p></Drawer>;

  if (preview === "ready" || preview === "generating" || preview === "genfail") return <div className="grid gap-6 lg:grid-cols-[minmax(0,1.4fr)_minmax(290px,.7fr)]"><section className={`${card} p-6`}><Badge tone={preview === "genfail" ? "amber" : "blue"}>{preview === "ready" ? "Ready to create path" : preview === "generating" ? "Generating" : "Generation failed"}</Badge><h2 className="mt-4 text-[22px] font-semibold">{preview === "ready" ? `Create your ${language} learning path` : preview === "generating" ? "Building your learning path" : "The learning path could not be created"}</h2><p className={`mt-2 text-[13px] ${muted}`}>{preview === "ready" ? "Your starting point, goals and study routine are kept, so you do not need to repeat setup." : preview === "generating" ? "Selecting relevant knowledge and organizing learning activities. You can leave this page and return later." : `Your ${language} goals, preferences, starting point and assessment results are saved. Try again or change your starting point.`}</p>{preview === "generating" && <ol className={`mt-5 space-y-2 text-[12px] ${muted}`}><li>✓ Goals, preferences and starting point loaded</li><li>● Selecting knowledge items and dependencies</li><li>○ Arranging modules and review schedule</li></ol>}<div className="mt-5 flex flex-wrap gap-2">{preview === "generating" ? <button className={button} onClick={() => setPreview("current")}>Preview created plan</button> : <button className={button} onClick={() => setPreview("generating")}>{preview === "genfail" ? "Retry" : "Create learning path"}</button>}{preview !== "generating" && <button className={secondary} onClick={() => begin("quick")}>Change starting point</button>}{archived && <button className={secondary} onClick={() => setPanel("history")}>View plan history</button>}</div><p className={`mt-3 text-[11px] ${muted}`}>Design preview · no real program is created.</p></section><section className={`${card} p-5`}><h2 className="font-semibold">{language} profile</h2><p className={`mt-2 text-[12px] ${muted}`}>Goals, preferences and the starting point stay available during setup. No modules or progress are shown before a path exists.</p></section>{historyDrawer}</div>;

  if (noProgram) return <><div className="grid gap-6 lg:grid-cols-[minmax(0,1.4fr)_minmax(290px,.7fr)]"><section className={`${card} p-6`}><Badge tone={preview === "unfinished" ? "blue" : "neutral"}>{preview === "unfinished" ? "Assessment in progress" : "No current program"}</Badge><h2 className="mt-4 text-[22px] font-semibold">{preview === "unfinished" ? "Finish or resume your assessment" : `Choose where to start in ${language}`}</h2><p className={`mt-2 text-[13px] ${muted}`}>{preview === "unfinished" ? "Your sample assessment is unfinished. No learning path or completed result is shown yet." : "Choose a starting point. A claimed level is self-reported, and you can begin at A1 without a test."}</p>{archived && <p className="mt-4 rounded-md bg-[#f5f5f4] p-3 text-[12px]">Your previous plan, assessment results and learning history are saved in history.</p>}{preview === "unfinished" ? <button className={button + " mt-5"} onClick={() => go("Assessments")}>Resume assessment</button> : <div className="mt-5 grid gap-3 sm:grid-cols-3"><button className={secondary + " flex-col items-start text-left"} disabled={language !== "English" && language !== "Spanish"} onClick={() => begin("quick")}><b>Quick grammar screening</b><span className={muted}>{language === "English" || language === "Spanish" ? "Optional assessment" : "Unavailable for this language"}</span></button><button className={secondary + " flex-col items-start text-left"} onClick={() => begin("manual")}><b>Select a starting level</b><span className={muted}>Self-reported</span></button><button className={secondary + " flex-col items-start text-left"} onClick={() => begin("scratch")}><b>Start at A1</b><span className={muted}>Without an assessment</span></button></div>}<div className="mt-5 flex flex-wrap gap-2"><button className={secondary} onClick={() => go("Overview")}>Back to Overview</button>{archived && <button className={secondary} onClick={() => setPanel("history")}>View plan history</button>}</div></section><section className={`${card} p-5`}><h2 className="font-semibold">What appears here</h2><p className={`mt-2 text-[12px] ${muted}`}>Program modules and activities appear after a starting method is completed. Previous learning history remains separate.</p></section></div>{historyDrawer}</>;

  return <>
    <div className="grid items-start gap-6 min-[1200px]:grid-cols-[minmax(0,1.55fr)_minmax(290px,.75fr)]"><div className="space-y-6">
      <section className={`${card} p-6`}><div className="flex flex-wrap items-center justify-between gap-2"><Badge tone={completedPlan ? "green" : preview === "due" ? "amber" : "blue"}>{completedPlan ? "Path completed" : blocked ? "Last known plan" : preview === "due" ? "Review due" : "Active learning path"}</Badge><span className={`text-[11px] ${muted}`}>Sample plan · design preview</span></div><h2 className="mt-4 text-[22px] font-semibold">{plan.objective}</h2><p className={`mt-2 text-[13px] ${muted}`}>{completedPlan ? "All planned activities are complete. Completion does not certify a level; mastery is shown only where demonstrated." : `A sample ${language} learning path. Your next activity is highlighted below.`}</p>
        {preview === "insufficient" && <p className="mt-3 rounded-md bg-[#f5f5f4] p-3 text-[12px]">There is not enough evidence for skill estimates or trends yet.</p>}{preview === "missing-ai" && <p className="mt-3 rounded-md bg-[#fff8e9] p-3 text-[12px]">Existing content and history remain visible. AI-dependent actions need a configured provider.</p>}{blocked && <p className="mt-3 rounded-md bg-[#fff8e9] p-3 text-[12px]">This is a last-known sample. Live activity is unavailable until the service returns.</p>}
        <div className="mt-5 grid grid-cols-2 gap-3 sm:grid-cols-4">{[["Current focus", completedPlan ? "Review" : currentModule?.title || "—"], ["Weekly target", target || plan.target], ["Preferred session", duration || plan.session], ["Remaining", `${remaining} activities`]].map(([label, value]) => <div key={label} className="rounded-lg bg-[#f5f5f4] p-3"><p className={`text-[11px] ${muted}`}>{label}</p><b className="mt-1 block text-[12px]">{value}</b></div>)}</div>
        <div className="mt-4 flex flex-wrap gap-1.5">{plan.goals.map(goal => <span key={goal} className="rounded-full bg-[#f3f3f3] px-2 py-1 text-[11px]">{goal}</span>)}</div>
        <div className="mt-5 flex flex-wrap gap-2"><button className={button} disabled={blocked} onClick={() => completedPlan ? go("Practice") : next && currentModule && openDetails(currentModule, next)}>{completedPlan ? "Continue reviews" : next?.state === "In progress" ? "Resume activity" : "Start next activity"}</button><button className={secondary} disabled={blocked} onClick={() => setPanel("goals")}>Edit learning goals</button><button className={secondary} onClick={() => setPanel("history")}>View plan history</button><button className={secondary} disabled={blocked} onClick={startNewPlan}>Start a new plan</button></div><p className={`mt-3 text-[11px] ${muted}`}>Your previous plan, assessment results and learning history remain available. Deleting history is a separate action.</p>{savedPreferences && <p className="mt-2 text-[11px] text-[#24833b]" role="status">Preferences saved for this preview.</p>}</section>
      <section aria-label="Learning path summary" className="grid grid-cols-2 gap-3 md:grid-cols-4">{[["Completed activities", `${finishedCount} / ${all.length}`, "Activity completion"], ["Mastered knowledge items", `${masteredCount} items`, "Demonstrated mastery"], ["Review due", `${dueCount} activities`, "Spaced repetition"], ["Current module", currentModule ? `${completedPlan ? currentModule.activities.length : currentModule.activities.filter(activity => activity.state === "Completed").length} / ${currentModule.activities.length}` : "—", currentModule?.title || "All modules completed"]].map(([label, value, detail]) => <div key={label} className={`${card} p-4`}><p className={`text-[11px] ${muted}`}>{label}</p><p className="mt-2 text-[19px] font-semibold">{value}</p><p className={`mt-1 text-[11px] ${muted}`}>{detail} · sample</p></div>)}</section>
      {completedPlan && <section className={`${card} p-5`}><h2 className="font-semibold">Planned activities completed</h2><p className={`mt-2 text-[12px] ${muted}`}>Keep reviewing or choose a new starting point. Completing the plan does not certify a particular level.</p><button className={secondary + " mt-3"} onClick={startNewPlan}>Start a new plan</button></section>}
      <section className={card}><div className="flex flex-wrap items-start justify-between gap-3 border-b border-[#eee] px-5 py-4"><div><h2 className="text-[15px] font-semibold">Learning path</h2><p className={`mt-1 text-[11px] ${muted}`}>Ordered modules and activities · sample content</p></div><div className="flex flex-wrap items-center gap-2"><label className="text-[11px]">Skill <select aria-label="Filter path by skill" value={skill} onChange={event => setSkill(event.target.value)} className="ml-1 rounded-md border border-[#e6e6e3] bg-white px-2 py-1.5">{["All skills", ...skills].map(value => <option key={value}>{value}</option>)}</select></label><label className="text-[11px]">Type <select aria-label="Filter path by activity type" value={kind} onChange={event => setKind(event.target.value)} className="ml-1 rounded-md border border-[#e6e6e3] bg-white px-2 py-1.5">{["All types", "Learn", "Practice", "Review", "Apply", "Assess"].map(value => <option key={value}>{value}</option>)}</select></label>{(skill !== "All skills" || kind !== "All types") && <button className="text-[11px] underline" onClick={() => { setSkill("All skills"); setKind("All types"); }}>Clear</button>}</div></div>
        <div className="flex gap-1 border-b border-[#eee] px-5 pt-3" role="tablist" aria-label="Path views">{(["My path", "Review", "Completed"] as const).map(value => <button key={value} role="tab" aria-selected={tab === value} className={`rounded-t-md px-3 py-2 text-[12px] font-medium ${tab === value ? "bg-[#f3f3f1] text-[#292929]" : muted}`} onClick={() => setTab(value)}>{value}</button>)}</div>
        <div className="space-y-2 p-3">{!filtered.length && <p className={`rounded-lg border border-dashed p-6 text-center text-[12px] ${muted}`}>No activities match these filters.</p>}{filtered.map(({ module, index, activities }) => {
          const isOpen = !closed.includes(module.title) && (tab !== "My path" || opened.includes(module.title) || module.state === "In progress" || module === currentModule);
          const done = completedPlan ? module.activities.length : module.activities.filter(activity => activity.state === "Completed").length;
          const minutes = completedPlan ? 0 : module.activities.filter(activity => activity.state !== "Completed").reduce((sum, activity) => sum + activity.minutes, 0);
          const mastered = module.activities.filter(activity => activity.mastery === "Mastered").length;
          const hasDue = module.activities.some(activity => activity.review === "Due" || activity.review === "Overdue");
          const moduleState = completedPlan ? "Completed" : module.state === "Completed" && hasDue ? "Needs review" : module.state;
          return <article key={module.title} className="overflow-hidden rounded-xl border border-[#e8e8e8]"><button className="w-full p-4 text-left" aria-expanded={isOpen} onClick={() => { setOpened(current => isOpen ? current.filter(name => name !== module.title) : [...current, module.title]); setClosed(current => isOpen ? [...current, module.title] : current.filter(name => name !== module.title)); }}><div className="flex flex-wrap items-center gap-2"><span className={`text-[11px] ${muted}`}>Module {index + 1}</span><Badge tone={moduleState === "Needs review" ? "amber" : moduleState === "Completed" ? "green" : moduleState === "In progress" ? "blue" : "neutral"}>{moduleState}</Badge><span className={`ml-auto text-[11px] ${muted}`}>{isOpen ? "Collapse" : "Expand"}</span></div><h3 className="mt-2 text-[15px] font-semibold">{module.title}</h3><p className={`mt-1 text-[12px] ${muted}`}>{module.description}</p><p className={`mt-2 text-[11px] ${muted}`}><b className="font-medium text-[#282828]">{done} of {module.activities.length}</b> activities completed · {minutes ? `~${minutes} min remaining` : "No remaining workload"} · {mastered} mastered · {[...new Set(module.activities.map(activity => activity.skill))].join(", ")}</p><div className="mt-3 h-1 overflow-hidden rounded-full bg-[#ededeb]"><div className={`h-full ${moduleState === "In progress" ? "bg-[#5069ee]" : "bg-[#299141]"}`} style={{ width: `${Math.round(done / module.activities.length * 100)}%` }} /></div></button>
            {isOpen && <div className="border-t border-[#eee]"><p className={`px-4 pb-2 pt-3 text-[11px] ${muted}`}>Objective: {module.objective}</p><ol>{activities.map(activity => { const isNext = !completedPlan && activity === next && module === currentModule; const activityState = completedPlan ? "Completed" : activity.state; const knowledgeState = activity.review ? `Review ${activity.review.toLowerCase()}` : activity.mastery; return <li key={activity.title} className={`border-t border-[#eee] px-4 py-3 ${isNext ? "bg-[#f3f5ff]" : ""}`}><div className="flex flex-wrap items-center gap-3"><span aria-label={activityState} title={activityState} className={`flex size-6 items-center justify-center rounded-full ${activityState === "Completed" ? "bg-[#eaf7ec] text-[#24833b]" : "bg-[#f3f3f3] text-[#777]"}`}>{activityState === "Completed" ? <Check size={15} /> : <ChevronRight size={14} />}</span><div className="min-w-[180px] flex-1"><div className="flex flex-wrap items-center gap-2"><b className="text-[12px]">{activity.title}</b>{isNext && <Badge tone="blue">Next</Badge>}</div><p className={`mt-0.5 text-[11px] ${muted}`}>{activity.type} · {activity.skill} · {activity.minutes} min · {activity.knowledge}</p></div><div className="flex flex-wrap items-center gap-2"><Badge tone={activity.review ? "amber" : activity.mastery === "Mastered" ? "green" : activity.mastery === "Learning" ? "blue" : "neutral"}>{knowledgeState}</Badge>{isNext && <button className={button} disabled={blocked} onClick={() => openDetails(module, activity)}>{activity.state === "In progress" ? "Resume" : "Start"}</button>}<button className={secondary} aria-label={`Details: ${activity.title}`} onClick={() => openDetails(module, activity)}>Details</button></div></div>{activity.prerequisite && activityState !== "Completed" && <div className="ml-9 mt-2 rounded-md bg-[#fff8e9] px-3 py-2 text-[11px]"><b>Recommended first:</b> {activity.prerequisite[0]}. {activity.prerequisite[1]} <span className={muted}>You can still open this activity.</span></div>}</li>; })}</ol></div>}</article>;
        })}</div></section>
    </div><aside className="space-y-6">
      <section className={`${card} p-5`}><p className={`text-[11px] font-medium uppercase tracking-wide ${muted}`}>Up next</p>{next && currentModule ? <><h2 className="mt-2 font-semibold">{next.title}</h2><p className={`mt-1 text-[12px] ${muted}`}>{next.skill} · {next.minutes} minutes</p><p className="mt-3 text-[12px]">Recommended as the next step in “{currentModule.title}”.</p><div className="mt-4 flex flex-wrap gap-2"><button className={button} disabled={blocked} onClick={() => openDetails(currentModule, next)}>{next.state === "In progress" ? "Resume activity" : "Start activity"}</button><button className={secondary} onClick={() => go("Knowledge Base")}>View rule</button></div></> : <><h2 className="mt-2 font-semibold">Choose a next objective</h2><p className={`mt-2 text-[12px] ${muted}`}>All current modules are completed.</p><button className={secondary + " mt-3"} onClick={startNewPlan}>Start a new plan</button></>}</section>
      <section className={`${card} p-5`}><div className="flex items-center justify-between"><h2 className="font-semibold">Learning goals</h2><button className="text-[12px] underline" disabled={blocked} onClick={() => setPanel("goals")}>Edit</button></div><ul className={`mt-3 space-y-2 text-[12px] ${muted}`}>{plan.goals.map(goal => <li key={goal}>– {goal}</li>)}</ul></section>
      <section className={`${card} p-5`}><div className="flex flex-wrap items-center justify-between gap-2"><h2 className="font-semibold">Review queue</h2><button className={secondary} disabled={!reviewItems.length || blocked} onClick={() => { setTab("Review"); setClosed([]); }}>View path reviews</button></div>{reviewItems.length ? <ul className="mt-3 space-y-3 text-[12px]">{reviewItems.map((item, index) => <li key={index} className="border-t border-[#eee] pt-3"><div className="flex justify-between gap-2"><b>{item.title}</b><Badge tone={item.due === "Scheduled" ? "neutral" : "amber"}>{item.due}</Badge></div><p className={`mt-1 ${muted}`}>{item.reason} · ~{item.minutes} min · sample</p></li>)}</ul> : <p className={`mt-3 text-[12px] ${muted}`}>No reviews are due right now.</p>}{reviewItems.length > 0 && <button className="mt-3 text-[12px] underline" onClick={() => go("Practice")}>View all reviews in Practice</button>}</section>
      <section className={`${card} p-5`}><h2 className="font-semibold">Skill coverage</h2><p className={`mt-1 text-[11px] ${muted}`}>Activities in this path, not a measure of proficiency.</p><div className="mt-3 space-y-2">{skills.map(name => { const count = all.filter(({ activity }) => activity.skill === name).length; const max = Math.max(1, ...skills.map(s => all.filter(({ activity }) => activity.skill === s).length)); return <div key={name} className="flex items-center gap-2 text-[12px]"><span className="w-24">{name}</span><span className="h-1.5 flex-1 overflow-hidden rounded-full bg-[#eee]"><span className="block h-full bg-[#a1a1aa]" style={{ width: `${count / max * 100}%` }} /></span><span className={`w-5 text-right ${muted}`}>{count}</span></div>; })}</div></section>
      <section className={`${card} p-5`}><h2 className="font-semibold">Recent plan changes</h2>{plan.changes.length ? <ul className="mt-3 space-y-3">{plan.changes.map((change, index) => <li key={index} className="border-t border-[#eee] pt-3 text-[12px]"><b>{change.title}</b><p className={muted}>{change.reason}</p><p className={`mt-1 text-[11px] ${muted}`}>{change.when} · {change.item} · sample</p></li>)}</ul> : <p className={`mt-2 text-[12px] ${muted}`}>No sample changes yet.</p>}<button className="mt-3 text-[12px] underline" onClick={() => setPanel("history")}>View plan history</button></section>
    </aside></div>
    {historyDrawer}
    {panel === "activity" && selected && <Drawer title={selected.activity.title} kicker="Activity details · sample" close={closePanel} footer={<><button className={button} disabled={blocked} onClick={() => start(selected.activity)}>{selected.activity.state === "In progress" ? "Resume activity" : "Start activity"}</button><button className={secondary} onClick={() => { closePanel(); go("Knowledge Base"); }}>Open related rule</button><button className={secondary} disabled={aiBlocked} onClick={() => start(selected.activity)}>Practise this topic</button><button className={secondary} onClick={() => go("Assessments")}>Open assessments</button><button className={secondary} onClick={() => { setReviewAdded(current => current.includes(selected.activity.title) ? current : [...current, selected.activity.title]); closePanel(); setTab("Review"); setClosed([]); }}>Add to review</button>{aiBlocked && <p className={`w-full text-[11px] ${muted}`}>Actions requiring an AI provider are unavailable in this state.</p>}</>}><div className="flex flex-wrap gap-2">{[selected.activity.type, selected.activity.skill, `${selected.activity.minutes} min`, selected.activity.knowledge].map(value => <Badge key={value}>{value}</Badge>)}</div><div><p className={`text-[11px] ${muted}`}>Learning objective</p>Use “{selected.activity.knowledge}” accurately in {selected.activity.skill.toLowerCase()} tasks.</div><div><p className={`text-[11px] ${muted}`}>Why it is recommended</p>Part of “{selected.module.title}”, which supports the goal: {plan.goals[0]}.</div><div><p className={`text-[11px] ${muted}`}>Expected outcome</p>Apply {selected.activity.knowledge} in short {selected.activity.skill.toLowerCase()} tasks without prompts.</div><div><p className={`mb-1 text-[11px] ${muted}`}>Status</p><dl className="grid grid-cols-[auto_1fr] gap-x-4 gap-y-1"><dt className={muted}>Activity</dt><dd>{completedPlan ? "Completed" : selected.activity.state}</dd><dt className={muted}>Knowledge</dt><dd>{selected.activity.mastery}</dd><dt className={muted}>Review</dt><dd>{selected.activity.review || "None scheduled"}</dd></dl></div><div><p className={`text-[11px] ${muted}`}>Prerequisites</p>{selected.activity.prerequisite ? selected.activity.prerequisite.join(". ") : "No missing prerequisites."}</div><div><p className={`text-[11px] ${muted}`}>Previous attempts</p>{selected.activity.state === "Not started" ? "No attempts in this sample." : "Sample activity history is illustrative."}</div></Drawer>}
    {panel === "goals" && <Drawer title="Edit learning goals" kicker="Preferences · sample" close={closePanel} footer={<><button className={button} onClick={() => { setSavedPreferences(true); closePanel(); }}>Save changes</button><button className={secondary} onClick={closePanel}>Cancel</button></>}><label className="block">Learning purpose<select className="mt-1 block w-full rounded-md border border-[#ddd] p-2" value={purpose} onChange={event => setPurpose(event.target.value)}>{["Everyday and professional communication", "Travel", "Study", "Work"].map(value => <option key={value}>{value}</option>)}</select></label><fieldset><legend>Priority skills</legend><div className="mt-2 flex flex-wrap gap-2">{skills.map(name => <label key={name} className="flex items-center gap-2 rounded-full border border-[#ddd] px-3 py-1 text-[12px]"><input type="checkbox" checked={priorities.includes(name)} onChange={() => setPriorities(values => values.includes(name) ? values.filter(value => value !== name) : [...values, name])} />{name}</label>)}</div></fieldset><label className="block">Weekly available time<select className="mt-1 block w-full rounded-md border border-[#ddd] p-2" value={target || plan.target} onChange={event => setTarget(event.target.value)}>{["3 hours", "5 hours", "8 hours"].map(value => <option key={value}>{value}</option>)}</select></label><label className="block">Preferred session duration<select className="mt-1 block w-full rounded-md border border-[#ddd] p-2" value={duration || plan.session} onChange={event => setDuration(event.target.value)}>{["15 minutes", "20 minutes", "30 minutes", "45 minutes", "60 minutes"].map(value => <option key={value}>{value}</option>)}</select></label><label className="block">Explanation language<select className="mt-1 block w-full rounded-md border border-[#ddd] p-2" value={explanation} onChange={event => setExplanation(event.target.value)}>{["English", "Russian", "Spanish"].map(value => <option key={value}>{value}</option>)}</select></label><label className="block">Target proficiency (optional)<select className="mt-1 block w-full rounded-md border border-[#ddd] p-2" value={targetLevel} onChange={event => setTargetLevel(event.target.value)}>{["No target", "A2", "B1", "B2", "C1"].map(value => <option key={value}>{value}</option>)}</select></label><p className={`rounded-md bg-[#f5f5f4] p-3 text-[12px] ${muted}`}>Changes may update future activities. Your completed learning history is kept.</p></Drawer>}
  </>;
}
