import { useEffect, useState } from "react";
import { BookOpen, Check, ClipboardCheck, Headphones, Mic, Pause, Play, RotateCcw } from "lucide-react";
import type { Language, Page } from "./WorkspacePages";
type Mode = "quick" | "full" | "skill" | "manual" | "scratch";
type Step = "choose" | "intro" | "question" | "paused" | "result" | "path";
type Skill = "Grammar" | "Vocabulary" | "Reading" | "Listening" | "Writing" | "Speaking" | "Pronunciation";
type Question = {
  kind: string;
  prompt: string;
  context?: string;
  options?: string[];
  expected?: string;
};
const card = "rounded-xl border border-[#e6e6e3] bg-white";
const muted = "text-[#777773]";
const button = "inline-flex min-h-9 items-center justify-center gap-2 rounded-md bg-[#292929] px-3.5 py-2 text-[12px] font-semibold text-white hover:bg-black focus-visible:outline-2 focus-visible:outline-offset-2 disabled:opacity-40";
const secondary = "inline-flex min-h-9 items-center justify-center gap-2 rounded-md border border-[#e6e6e3] bg-white px-3.5 py-2 text-[12px] font-medium hover:bg-[#f5f5f3] focus-visible:outline-2 focus-visible:outline-offset-2";
const skills: Skill[] = ["Grammar", "Vocabulary", "Reading", "Listening", "Writing", "Speaking", "Pronunciation"];
const available: Skill[] = ["Grammar"];
const bank: Record<Skill, Question[]> = {
  Grammar: [{
    kind: "Recognition",
    prompt: "She ___ a student.",
    options: ["am", "is", "are", "be"],
    expected: "is"
  }, {
    kind: "Independent recall",
    prompt: "Complete the sentence: Yesterday I ___ to school.",
    expected: "went"
  }],
  Vocabulary: [{
    kind: "Meaning in context",
    context: "We missed the train, so we took a bus.",
    prompt: "What does ‘missed’ mean here?",
    options: ["Could not catch", "Forgot", "Lost", "Avoided"],
    expected: "Could not catch"
  }, {
    kind: "Collocation",
    prompt: "Complete the phrase: ___ a decision.",
    expected: "make"
  }],
  Reading: [{
    kind: "Main idea",
    context: "Maya moved to a new city for work. At first she knew nobody, but joining a local book club helped her meet people. She now feels at home there.",
    prompt: "What is the main idea?",
    options: ["Moving is always difficult", "A group helped Maya settle in", "Maya changed jobs", "Book clubs are expensive"],
    expected: "A group helped Maya settle in"
  }, {
    kind: "Inference",
    context: "At first she knew nobody, but joining a local book club helped her meet people.",
    prompt: "How did Maya probably feel before joining the club?",
    options: ["Lonely", "Angry", "Bored", "Confident"],
    expected: "Lonely"
  }],
  Listening: [{
    kind: "Audio comprehension",
    prompt: "Listen and select what the speaker is asking for.",
    options: ["Directions", "A ticket", "A table", "The time"]
  }],
  Writing: [{
    kind: "Written production",
    prompt: "Write 3–4 sentences about a journey you took. Use at least one past tense form."
  }],
  Speaking: [{
    kind: "Communicative speaking",
    prompt: "Record a short answer about a recent trip."
  }],
  Pronunciation: [{
    kind: "Audio analysis",
    prompt: "Pronunciation needs validated acoustic evaluation; a transcript alone is insufficient."
  }]
};
export function AssessmentPrototype({
  language,
  initialMode,
  go,
  onPathCreated
}: {
  language: Language;
  initialMode: Mode;
  go: (page: Page) => void;
  onPathCreated: () => void;
}) {
  const storageKey = `magicpath.learning.assessment.${language}`;
  const [saved] = useState(() => {
    try {
      return JSON.parse(localStorage.getItem(storageKey) ?? "null") as {
        results?: Partial<Record<Skill, {
          answered: number;
          correct: number;
        }>>;
        history?: {
          skill: Skill;
          date: string;
          correct: number;
        }[];
      } | null;
    } catch {
      return null;
    }
  });
  const [step, setStep] = useState<Step>("choose");
  const [mode, setMode] = useState<Mode>(initialMode);
  const [skill, setSkill] = useState<Skill>("Grammar");
  const [question, setQuestion] = useState(0);
  const [answer, setAnswer] = useState("");
  const [results, setResults] = useState<Partial<Record<Skill, {
    answered: number;
    correct: number;
  }>>>(saved?.results ?? {});
  const [level, setLevel] = useState("A1");
  const [history, setHistory] = useState<{
    skill: Skill;
    date: string;
    correct: number;
  }[]>(saved?.history ?? []);
  const [showHistory, setShowHistory] = useState(false);
  useEffect(() => {
    localStorage.setItem(storageKey, JSON.stringify({
      results,
      history
    }));
  }, [storageKey, results, history]);
  const activeSkills: Skill[] = mode === "quick" ? ["Grammar"] : mode === "skill" ? [skill] : ["Grammar", "Vocabulary", "Reading", "Listening", "Writing", "Speaking", "Pronunciation"];
  const completed = activeSkills.filter(s => results[s]?.answered === bank[s].length);
  const current = bank[skill][question];
  const isChoice = !!current?.options;
  const start = () => {
    const previous = results[skill];
    if (previous?.answered === bank[skill].length) {
      const next = {
        ...results
      };
      delete next[skill];
      setResults(next);
      setQuestion(0);
    } else {
      setQuestion(previous?.answered ?? 0);
    }
    setAnswer("");
    setStep("question");
  };
  const submit = (skip = false) => {
    const q = bank[skill][question];
    const old = results[skill] ?? {
      answered: 0,
      correct: 0
    };
    const next = {
      answered: old.answered + 1,
      correct: old.correct + (!skip && q.expected && answer.trim().toLowerCase() === q.expected.toLowerCase() ? 1 : 0)
    };
    setResults({
      ...results,
      [skill]: next
    });
    setAnswer("");
    if (question + 1 < bank[skill].length) {
      setQuestion(question + 1);
      return;
    }
    setHistory(v => [{
      skill,
      date: "Today",
      correct: next.correct
    }, ...v]);
    setStep("result");
  };
  const chooseSkill = (next: Skill) => {
    setSkill(next);
    const previous = results[next];
    if (previous?.answered === bank[next].length) {
      const updated = {
        ...results
      };
      delete updated[next];
      setResults(updated);
      setQuestion(0);
    } else {
      setQuestion(previous?.answered ?? 0);
    }
    setAnswer("");
    setStep("question");
  };
  if (step === "choose") return <div className="max-w-[980px] space-y-5"><div><h2 className="text-[21px] font-semibold">Find your starting point</h2><p className={`mt-1 text-[13px] ${muted}`}>Assessment gives preliminary evidence and suspected gaps. It is not a certified exam or proof of mastery.</p></div><div className="grid gap-3 md:grid-cols-2">{([["quick", "Quick assessment", "Grammar screening · English and Spanish"], ["full", "Full assessment", "Planned · independent skill sections"], ["skill", "Skill assessment", "Planned · individual skill assessment"], ["manual", "Manual starting point", "Choose an approximate level yourself"], ["scratch", "Start from scratch", "Begin with the fundamentals, no assessment"]] as const).map(([id, title, desc]) => <button key={id} disabled={(id === "full" || id === "skill") || (id === "quick" && language !== "English" && language !== "Spanish")} className={`${card} min-h-[104px] p-4 text-left disabled:cursor-not-allowed disabled:opacity-50 ${mode === id ? "border-[#333] ring-1 ring-[#333]" : "hover:bg-[#fafaf9]"}`} onClick={() => setMode(id)}><span className="flex items-center justify-between gap-3"><b className="text-[14px]">{title}</b><span className={`text-[10px] ${id === "quick" ? "text-[#24833b]" : "text-[#777773]"}`}>{id === "quick" ? (language === "English" || language === "Spanish" ? "Grammar available" : "Unavailable") : id === "full" || id === "skill" ? "Unavailable" : "Available"}</span></span><span className={`mt-2 block text-[12px] ${muted}`}>{desc}</span></button>)}</div>{mode === "skill" && <label className="block text-[12px]">Choose a skill<select value={skill} onChange={e => setSkill(e.target.value as Skill)} className="ml-2 rounded-md border border-[#ddd] p-2">{skills.map(s => <option key={s}>{s}</option>)}</select></label>}{mode === "manual" && <label className="block text-[12px]">Approximate level<select value={level} onChange={e => setLevel(e.target.value)} className="ml-2 rounded-md border border-[#ddd] p-2">{["A1", "A2", "B1", "B2", "C1", "C2"].map(s => <option key={s}>{s}</option>)}</select></label>}<div className="flex flex-wrap items-center gap-3"><button className={button} onClick={() => {
        if (mode === "manual" || mode === "scratch") setStep("path");else setStep("intro");
      }}>Continue</button><span className={`text-[11px] ${muted}`}>You can start learning without an assessment.</span></div><div className="rounded-lg bg-[#f2f2f2] p-4 text-[12px]"><b>Product capability:</b> the current backend verifies Quick grammar for English and Spanish. Full, Skill and other skills remain unavailable in this prototype until verified end to end. The questions shown here are sample interactions, not backend results.</div>{history.length > 0 && <button className="text-[12px] underline" onClick={() => setShowHistory(!showHistory)}>{showHistory ? "Hide" : "View"} assessment history</button>}{showHistory && <div className={`${card} p-4 text-[12px]`}>{history.map((h, i) => <p key={i} className="border-b border-[#eee] py-2">{h.date} · {h.skill} · sample {h.correct}/{bank[h.skill].length} responses matched{history.slice(i + 1).find(previous => previous.skill === h.skill) && <span className="ml-2 font-semibold text-[#24833b]">({h.correct - history.slice(i + 1).find(previous => previous.skill === h.skill)!.correct >= 0 ? "+" : ""}{h.correct - history.slice(i + 1).find(previous => previous.skill === h.skill)!.correct} vs previous)</span>}</p>)}</div>}</div>;
  if (step === "path") return <div className={`${card} max-w-[720px] p-6`}><h2 className="text-[21px] font-semibold">Create your {language} learning path</h2><p className={`mt-2 text-[13px] ${muted}`}>{mode === "scratch" ? "Start with fundamentals." : mode === "manual" ? `Use your self-reported ${level} starting point.` : "Use the collected assessment evidence and keep unassessed skills unknown."}</p><div className="mt-5 rounded-lg bg-[#f4f4f3] p-4 text-[12px]">Goals: everyday communication · Routine: 3 hours per week · Explanations: English</div><div className="mt-5 flex gap-2"><button className={button} onClick={() => {
        onPathCreated();
        go("Learning Path");
      }}>Preview learning path</button><button className={secondary} onClick={() => setStep(mode === "manual" || mode === "scratch" ? "choose" : "result")}>Back</button></div><p className={`mt-4 text-[11px] ${muted}`}>Design preview: this action changes the prototype state only.</p></div>;
  if (step === "intro") return <div className="max-w-[760px] space-y-4"><button className={secondary} onClick={() => setStep("choose")}>Back to choices</button><section className={card}><div className="border-b border-[#eee] p-5"><p className={`text-[12px] ${muted}`}>{language} · AI Teacher</p><h2 className="mt-1 text-[21px] font-semibold">{mode === "quick" ? "Quick assessment" : mode === "full" ? "Full assessment" : `${skill} assessment`}</h2></div><div className="space-y-4 p-5"><p className={`text-[13px] ${muted}`}>{mode === "full" ? "Complete skill sections independently. Pause now and resume later; each completed section has separate evidence." : mode === "skill" ? "Repeat this skill assessment later to compare results. Earlier learning history stays intact." : "A short grammar screening. Questions and answers here are sample content; the application generates real questions through its assessment backend."}</p><div className="grid gap-3 sm:grid-cols-3">{[["Coverage", activeSkills.join(", ")], ["Progress", "Sample progress in this browser"], ["Result", "Preliminary evidence"]].map(([a, b]) => <div key={a} className="rounded-md bg-[#f3f3f1] p-3"><p className={`text-[11px] ${muted}`}>{a}</p><b className="text-[12px]">{b}</b></div>)}</div>{mode === "full" || mode === "quick" ? <div><h3 className="text-[13px] font-semibold">Skill sections</h3><div className="mt-2 grid gap-2 sm:grid-cols-2">{activeSkills.map(s => <button key={s} disabled={!available.includes(s)} className="flex items-center justify-between rounded-md border border-[#e6e6e3] p-3 text-left text-[12px] disabled:opacity-50" onClick={() => chooseSkill(s)}><span>{s}</span><span className={muted}>{results[s]?.answered === bank[s].length ? "Completed" : results[s]?.answered ? "Resume section" : available.includes(s) ? "Start section" : "Audio capability planned"}</span></button>)}</div></div> : available.includes(skill) ? <button className={button} onClick={start}>Start {skill} section</button> : <p className="rounded-md bg-[#fff8e9] p-3 text-[12px]">{skill} requires audio capture or audio evaluation. This section stays unassessed until those capabilities are supported.</p>}{completed.length > 0 && <button className={secondary} onClick={() => setStep("result")}>View partial result</button>}</div></section></div>;
  if (step === "paused") return <section className={`${card} max-w-[680px] space-y-4 p-6`}><Pause size={22} /><h2 className="text-[20px] font-semibold">Assessment paused</h2><p className={`text-[13px] ${muted}`}>You can resume the {skill.toLowerCase()} section from question {question + 1}. Completed sections remain visible in results.</p><div className="flex gap-2"><button className={button} onClick={() => setStep("question")}>Resume assessment</button><button className={secondary} onClick={() => setStep("intro")}>Back to sections</button></div></section>;
  if (step === "question") return <div className="mx-auto max-w-[720px] space-y-4"><div className="flex items-center justify-between"><p className={`text-[12px] ${muted}`}>{mode === "quick" ? "Quick" : mode === "full" ? "Full" : "Skill"} assessment · {skill}</p><button className={secondary} onClick={() => setStep("paused")}>Save and exit</button></div><section className={`${card} space-y-5 p-6`}><div className="flex justify-between"><p className={`text-[12px] ${muted}`}>Question {question + 1} of {bank[skill].length}</p><span className={`text-[11px] ${muted}`}>{current.kind}</span></div>{current.context && <p className="rounded-md bg-[#f5f5f4] p-4 text-[13px] leading-relaxed">{current.context}</p>}<h2 className="text-[20px] font-semibold">{current.prompt}</h2>{isChoice ? <fieldset className="space-y-2"><legend className="sr-only">Choose an answer</legend>{current.options!.map(o => <label key={o} className={`flex cursor-pointer items-center gap-3 rounded-lg border p-3.5 text-[13px] ${answer === o ? "border-[#333] bg-[#f4f4f2]" : "border-[#e7e7e5]"}`}><input type="radio" name="answer" checked={answer === o} onChange={() => setAnswer(o)} />{o}</label>)}</fieldset> : <label className="block text-[12px]">Your answer<textarea value={answer} onChange={e => setAnswer(e.target.value)} className="mt-1 min-h-20 w-full rounded-md border border-[#ddd] p-3 text-[13px]" placeholder="Write your answer" /></label>}<div className="flex justify-between"><button className={secondary} onClick={() => submit(true)}>I don't know</button><button className={button} disabled={!answer.trim()} onClick={() => submit()}>Continue</button></div><p className={`text-[11px] ${muted}`}>Correct answers stay hidden while an assessment is in progress. Sample responses are not sent to a backend.</p></section></div>;
  return <div className="grid items-start gap-6 lg:grid-cols-[minmax(0,1.3fr)_300px]"><section className={`${card} p-6`}><Check className="text-[#24833b]" size={23} /><h2 className="mt-3 text-[22px] font-semibold">Assessment results</h2><p className={`mt-1 text-[12px] ${muted}`}>{language} · {mode === "quick" ? "Quick" : mode === "full" ? "Full" : "Skill"} assessment · design simulation</p><div className="mt-5 rounded-lg bg-[#f3f3f1] p-4"><p className={`text-[11px] ${muted}`}>Recommended starting point</p><b className="text-[22px]">{mode === "skill" ? "Keep current path" : "A2 · preliminary"}</b><p className={`mt-1 text-[12px] ${muted}`}>This is illustrative, not a certified CEFR result. Unassessed skills have no estimate.</p></div><h3 className="mt-6 font-semibold">Skill sections</h3><div className="mt-2 divide-y divide-[#eee]">{activeSkills.map(s => <div key={s} className="flex items-center justify-between py-3 text-[12px]"><span>{s}</span><span className={muted}>{results[s]?.answered === bank[s].length ? `${results[s]!.answered} sample answers · section complete` : results[s]?.answered ? `${results[s]!.answered} sample answers · in progress` : "Not assessed"}</span></div>)}</div><h3 className="mt-5 font-semibold">Knowledge gaps</h3><p className={`mt-2 text-[12px] ${muted}`}>Past tense forms and contextual vocabulary are sample focus areas. An actual engine must derive gaps from validated evidence.</p><div className="mt-6 flex flex-wrap gap-2"><button className={button} onClick={() => setStep("path")}>Continue to learning path</button><button className={secondary} onClick={() => setStep("intro")}>Complete another section</button><button className={secondary} onClick={() => {
          setResults({});
          setQuestion(0);
          setStep("intro");
        }}><RotateCcw size={13} />Repeat assessment</button></div></section><aside className="space-y-5"><section className={`${card} p-5`}><h2 className="font-semibold">What this means</h2><p className={`mt-2 text-[12px] ${muted}`}>A short assessment creates provisional evidence. Mastery requires repeated performance over time.</p><button className="mt-3 text-[12px] underline" onClick={() => go("Knowledge Base")}>Explore Knowledge Base</button></section><section className={`${card} p-5`}><h2 className="font-semibold">Next assessments</h2><p className={`mt-2 text-[12px] ${muted}`}>Individual skill and full assessments are planned. Listening, speaking and pronunciation remain unassessed without audio support.</p><button className={`${secondary} mt-4`} disabled title="Skill assessment is not available yet">Skill assessment unavailable</button></section></aside></div>;
}