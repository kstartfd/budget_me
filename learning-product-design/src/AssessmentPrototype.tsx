import { useEffect, useRef, useState } from "react";
import { Check, Pause } from "lucide-react";
import type { Language, Page, Preview } from "./WorkspacePages";
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
type SkillResult = { answered: number; correct: number; skipped?: number };
type SavedAssessment = {
  results?: Partial<Record<Skill, SkillResult>>;
  history?: { skill: Skill; date: string; correct: number }[];
  paused?: boolean;
};
const card = "rounded-xl border border-[#e6e6e3] bg-white";
const muted = "text-[#646460]";
const button = "inline-flex min-h-11 items-center justify-center gap-2 rounded-md bg-[#292929] px-4 py-2 text-[13px] font-semibold text-white hover:bg-black focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[#292929] disabled:cursor-not-allowed disabled:opacity-40";
const secondary = "inline-flex min-h-11 items-center justify-center gap-2 rounded-md border border-[#d4d4d2] bg-white px-4 py-2 text-[13px] font-medium hover:bg-[#f5f5f3] focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[#292929] disabled:cursor-not-allowed disabled:opacity-40";
const headingFocus = "focus-visible:rounded-sm focus-visible:outline-2 focus-visible:outline-offset-4 focus-visible:outline-[#292929]";
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

// These Spanish sample questions already appear in the prototype's Practice view.
const spanishGrammar: Question[] = [{
  kind: "Choose an answer",
  prompt: "Choose the correct introduction.",
  options: ["Yo soy estudiante.", "Yo es estudiante.", "Yo ser estudiante."],
  expected: "Yo soy estudiante."
}, {
  kind: "Complete the sentence",
  prompt: "Ayer yo ___ al mercado.",
  expected: "fui"
}];
export function AssessmentPrototype({
  language,
  initialMode,
  preview,
  go,
  onPathCreated
}: {
  language: Language;
  initialMode: Mode;
  preview: Preview;
  go: (page: Page) => void;
  onPathCreated: () => void;
}) {
  const storageKey = `magicpath.learning.assessment.${language}`;
  const questions = language === "Spanish" ? spanishGrammar : bank.Grammar;
  const [initialStorage] = useState(() => {
    const read = (storage: Storage, key: string): SavedAssessment | null => {
      try {
        return JSON.parse(storage.getItem(key) ?? "null") as SavedAssessment | null;
      } catch {
        return null;
      }
    };
    const existing = read(localStorage, storageKey);
    const exampleKey = `${storageKey}.unfinished-preview`;
    const example = read(sessionStorage, exampleKey);
    const existingComplete = (existing?.results?.Grammar?.answered ?? 0) >= questions.length;
    // Keep the unfinished preview independent of any previously saved completed result.
    const useExample = existingComplete && (preview === "unfinished" || (preview === "generating" && example?.paused));
    return useExample
      ? { storage: sessionStorage, key: exampleKey, saved: example?.paused ? example : null }
      : { storage: localStorage, key: storageKey, saved: existing };
  });
  const saved = initialStorage.saved;
  const savedAnswers = saved?.results?.Grammar?.answered ?? 0;
  const initialResults = saved?.results ?? {};
  const initialPaused = preview === "unfinished" ||
    (initialMode === "quick" && (saved?.paused || savedAnswers > 0) && savedAnswers < questions.length);
  const [step, setStep] = useState<Step>(initialPaused ? "paused" : "choose");
  const [mode, setMode] = useState<Mode>(preview === "unfinished" ? "quick" : initialMode);
  const [question, setQuestion] = useState(initialPaused ? Math.min(savedAnswers, questions.length - 1) : 0);
  const [answer, setAnswer] = useState("");
  const [results, setResults] = useState<Partial<Record<Skill, SkillResult>>>(initialResults);
  const [isPaused, setIsPaused] = useState(Boolean(initialPaused || saved?.paused));
  const [level, setLevel] = useState("A1");
  const [history, setHistory] = useState<SavedAssessment["history"]>(saved?.history ?? []);
  const [showHistory, setShowHistory] = useState(false);
  const headingRef = useRef<HTMLHeadingElement>(null);

  useEffect(() => {
    initialStorage.storage.setItem(initialStorage.key, JSON.stringify({ results, history, paused: isPaused }));
  }, [initialStorage, results, history, isPaused]);

  useEffect(() => {
    headingRef.current?.focus();
  }, [step, question]);

  const result = results.Grammar;
  const answered = result?.answered ?? 0;
  const complete = answered >= questions.length;
  const insufficient = complete && (result?.skipped ?? 0) >= answered;
  const current = questions[question];
  const quickAvailable = language === "English" || language === "Spanish";

  const startOrResume = () => {
    if (complete) {
      setStep("result");
      return;
    }
    setQuestion(Math.min(answered, questions.length - 1));
    setAnswer("");
    setIsPaused(true);
    setStep("question");
  };

  const submit = (skip = false) => {
    const old = result ?? { answered: 0, correct: 0, skipped: 0 };
    const next: SkillResult = {
      answered: old.answered + 1,
      correct: old.correct + (!skip && current.expected && answer.trim().toLowerCase() === current.expected.toLowerCase() ? 1 : 0),
      skipped: (old.skipped ?? 0) + (skip ? 1 : 0)
    };
    setResults({ ...results, Grammar: next });
    setAnswer("");
    if (question + 1 < questions.length) {
      setQuestion(question + 1);
      return;
    }
    setIsPaused(false);
    setHistory(previous => [{ skill: "Grammar", date: "Today", correct: next.correct }, ...(previous ?? [])]);
    setStep("result");
  };

  const chooseStartingPoint = () => {
    setMode("manual");
    setStep("choose");
  };

  if (step === "choose") return (
    <div className="max-w-[980px] space-y-5">
      <div>
        <h2 ref={headingRef} tabIndex={-1} className={`text-[21px] font-semibold ${headingFocus}`}>Find your starting point</h2>
        <p className={`mt-1 text-[13px] leading-relaxed ${muted}`}>Choose a short grammar check or start without an assessment.</p>
      </div>
      <div className="grid gap-3 md:grid-cols-2">
        {([
          ["quick", "Quick assessment", "Grammar screening · English and Spanish"],
          ["full", "Full assessment", "Planned · independent skill sections"],
          ["skill", "Skill assessment", "Planned · individual skill assessment"],
          ["manual", "Manual starting point", "Choose an approximate level yourself"],
          ["scratch", "Start from scratch", "Begin at A1 without an assessment"]
        ] as const).map(([id, title, description]) => (
          <button
            key={id}
            type="button"
            disabled={id === "full" || id === "skill" || (id === "quick" && !quickAvailable)}
            aria-pressed={mode === id}
            className={`${card} min-h-[104px] p-4 text-left focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[#292929] disabled:cursor-not-allowed disabled:opacity-50 ${mode === id ? "border-[#333] ring-1 ring-[#333]" : "hover:bg-[#fafaf9]"}`}
            onClick={() => setMode(id)}
          >
            <span className="flex items-center justify-between gap-3">
              <b className="text-[14px]">{title}</b>
              <span className="text-right text-[12px] text-[#666661]">{id === "quick" ? (quickAvailable ? "Available" : "Unavailable") : id === "full" || id === "skill" ? "Unavailable" : "Available"}</span>
            </span>
            <span className={`mt-2 block text-[13px] ${muted}`}>{description}</span>
          </button>
        ))}
      </div>
      {mode === "manual" && <label className="block text-[13px]">Approximate level
        <select value={level} onChange={event => setLevel(event.target.value)} className="ml-2 min-h-10 rounded-md border border-[#aaa] bg-white p-2 focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[#292929]">
          {["A1", "A2", "B1", "B2", "C1", "C2"].map(option => <option key={option}>{option}</option>)}
        </select>
      </label>}
      <div className="flex flex-wrap items-center gap-3">
        <button className={button} disabled={(mode === "quick" && !quickAvailable) || mode === "full" || mode === "skill"} onClick={() => setStep(mode === "manual" || mode === "scratch" ? "path" : "intro")}>Continue</button>
        <span className={`text-[13px] ${muted}`}>You can start learning without an assessment.</span>
      </div>
      <div className="rounded-lg bg-[#f2f2f2] p-4 text-[13px] leading-relaxed">Quick assessment checks grammar for English and Spanish. Questions and results shown here are sample content.</div>
      {(history?.length ?? 0) > 0 && <button className={`${secondary}`} aria-expanded={showHistory} onClick={() => setShowHistory(!showHistory)}>{showHistory ? "Hide" : "View"} assessment history</button>}
      {showHistory && <div className={`${card} p-4 text-[13px]`}>{history?.map((item, index) => <p key={index} className="border-b border-[#eee] py-2">{item.date} · {item.skill} · sample result</p>)}</div>}
    </div>
  );

  if (step === "path") return (
    <section className={`${card} max-w-[720px] p-5 sm:p-6`}>
      <h2 ref={headingRef} tabIndex={-1} className={`text-[21px] font-semibold ${headingFocus}`}>Create your {language} learning path</h2>
      <p className={`mt-2 text-[13px] leading-relaxed ${muted}`}>
        {mode === "scratch" ? "Start at A1 without an assessment." : mode === "manual" ? `Use ${level} as your self-reported starting point.` : "Your grammar answers will help shape your learning path. Other skills have not been assessed."}
      </p>
      {isPaused && <p className="mt-4 rounded-lg bg-[#f4f4f3] p-4 text-[13px] leading-relaxed">Your assessment remains paused. You can resume it later from Assessments.</p>}
      <div className="mt-5 flex flex-wrap gap-2">
        <button className={button} onClick={() => { onPathCreated(); go("Learning Path"); }}>Create learning path</button>
        <button className={secondary} onClick={() => setStep(mode === "manual" || mode === "scratch" ? "choose" : "result")}>Back</button>
      </div>
    </section>
  );

  if (step === "intro") return (
    <div className="max-w-[760px] space-y-4">
      <button className={secondary} onClick={() => setStep("choose")}>Back to choices</button>
      <section className={card}>
        <div className="border-b border-[#eee] p-5">
          <p className={`text-[13px] ${muted}`}>{language} · Sample questions</p>
          <h2 ref={headingRef} tabIndex={-1} className={`mt-1 text-[21px] font-semibold ${headingFocus}`}>Quick grammar assessment</h2>
        </div>
        <div className="space-y-5 p-5">
          <p className={`text-[13px] leading-relaxed ${muted}`}>A short grammar check to help choose a starting point. You can pause and return later.</p>
          <div className="grid gap-3 sm:grid-cols-3">
            {[["Assessed", "Grammar"], ["Other skills", "Not assessed"], ["Result", "Preliminary"]].map(([label, value]) => <div key={label} className="rounded-md bg-[#f3f3f1] p-3"><p className={`text-[12px] ${muted}`}>{label}</p><b className="text-[13px]">{value}</b></div>)}
          </div>
          <div className="rounded-lg border border-[#e6e6e3] p-4">
            <div className="flex flex-wrap items-center justify-between gap-2"><h3 className="text-[14px] font-semibold">Grammar</h3><span className={`text-[13px] ${muted}`}>{complete ? (insufficient ? "Not enough information" : "Completed") : isPaused || answered ? "In progress" : "Not started"}</span></div>
            <button className={`${button} mt-4`} onClick={startOrResume}>{complete ? "View result" : isPaused || answered ? "Resume assessment" : "Start assessment"}</button>
          </div>
        </div>
      </section>
    </div>
  );

  if (step === "paused") return (
    <section className={`${card} max-w-[680px] space-y-4 p-5 sm:p-6`}>
      <Pause size={22} aria-hidden="true" />
      <div><p className={`text-[12px] ${muted}`}>{language} · Sample assessment</p><h2 ref={headingRef} tabIndex={-1} className={`mt-1 text-[20px] font-semibold ${headingFocus}`}>Assessment paused</h2></div>
      <p className={`text-[13px] leading-relaxed ${muted}`}>{answered ? "Your progress is saved. Resume when you’re ready." : "No answers submitted yet. You can resume when you’re ready."}</p>
      <div className="flex flex-wrap gap-2">
        <button className={button} onClick={startOrResume}>Resume assessment</button>
        {answered > 0 && <button className={secondary} onClick={() => setStep("result")}>View progress so far</button>}
        <button className={secondary} onClick={chooseStartingPoint}>Choose a starting point</button>
      </div>
    </section>
  );

  if (step === "question") return (
    <div className="mx-auto max-w-[720px] space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <p className={`text-[13px] ${muted}`}>Quick grammar assessment · {language}</p>
        <button className={secondary} onClick={() => { setIsPaused(true); setAnswer(""); setStep("paused"); }}>Save and exit</button>
      </div>
      <section className={`${card} space-y-5 p-5 sm:p-6`}>
        <div className="flex flex-wrap justify-between gap-2"><p className={`text-[13px] ${muted}`}>Question {question + 1} of {questions.length}</p><span className={`text-[12px] ${muted}`}>Sample question</span></div>
        {current.context && <p className="rounded-md bg-[#f5f5f4] p-4 text-[13px] leading-relaxed">{current.context}</p>}
        <h2 ref={headingRef} tabIndex={-1} className={`text-[20px] font-semibold ${headingFocus}`}>{current.prompt}</h2>
        {current.options ? <fieldset className="space-y-2"><legend className="mb-2 text-[13px] font-medium">Choose an answer</legend>{current.options.map(option => (
          <label key={option} className={`flex min-h-11 cursor-pointer items-center gap-3 rounded-lg border p-3.5 text-[13px] focus-within:outline-2 focus-within:outline-offset-2 focus-within:outline-[#292929] ${answer === option ? "border-[#333] bg-[#f4f4f2]" : "border-[#d4d4d2]"}`}>
            <input type="radio" name="answer" checked={answer === option} onChange={() => setAnswer(option)} />{option}
          </label>
        ))}</fieldset> : <label className="block text-[13px]">Your answer<textarea value={answer} onChange={event => setAnswer(event.target.value)} className="mt-1 min-h-20 w-full rounded-md border border-[#aaa] p-3 text-[13px] focus-visible:outline-2 focus-visible:outline-offset-2 focus-visible:outline-[#292929]" placeholder="Write your answer" /></label>}
        <div className="flex flex-wrap justify-between gap-2"><button className={secondary} onClick={() => submit(true)}>I don't know</button><button className={button} disabled={!answer.trim()} onClick={() => submit()}>Continue</button></div>
        <p className={`text-[12px] leading-relaxed ${muted}`}>Your submitted answers are saved. The questions in this preview are examples.</p>
      </section>
    </div>
  );

  const partial = !complete;
  return (
    <div className="grid max-w-[1050px] items-start gap-5 lg:grid-cols-[minmax(0,1fr)_290px]">
      <section className={`${card} p-5 sm:p-6`}>
        {partial || insufficient ? <Pause size={23} aria-hidden="true" /> : <Check className="text-[#24833b]" size={23} aria-hidden="true" />}
        <p className={`mt-3 text-[12px] ${muted}`}>{language} · Sample result</p>
        <h2 ref={headingRef} tabIndex={-1} className={`mt-1 text-[22px] font-semibold ${headingFocus}`}>{partial ? "Assessment in progress" : insufficient ? "Not enough information yet" : "Grammar screening complete"}</h2>
        <p className={`mt-2 text-[13px] leading-relaxed ${muted}`}>
          {partial ? "Your assessment is paused. There isn’t enough information for a starting point yet." : insufficient ? "There isn’t enough information to suggest a starting point. You can choose one yourself." : "This is a preliminary grammar check. It does not certify a level or demonstrate mastery."}
        </p>
        <div className="mt-5 rounded-lg bg-[#f3f3f1] p-4">
          <p className={`text-[12px] ${muted}`}>Starting point</p>
          <p className="mt-1 text-[15px] font-semibold">{partial || insufficient ? "Not enough information" : "Not shown in this sample"}</p>
          {!partial && !insufficient && <p className={`mt-1 text-[13px] leading-relaxed ${muted}`}>A real result may suggest a starting point when there is enough evidence.</p>}
        </div>
        <div className="mt-5 flex flex-wrap gap-2">
          {partial ? <><button className={button} onClick={startOrResume}>Resume assessment</button><button className={secondary} onClick={chooseStartingPoint}>Choose a starting point</button></> : insufficient ? <><button className={button} onClick={chooseStartingPoint}>Choose a starting point</button><button className={secondary} onClick={() => setStep("choose")}>Back to Assessments</button></> : <><button className={button} onClick={() => setStep("path")}>Continue to learning path</button><button className={secondary} onClick={() => setStep("choose")}>Back to Assessments</button></>}
        </div>
        <h3 className="mt-6 text-[14px] font-semibold">Assessment coverage</h3>
        <div className="mt-2 flex flex-wrap items-center justify-between gap-2 border-b border-[#e6e6e3] py-3 text-[13px]"><span>Grammar</span><span className="font-medium">{partial ? "In progress" : insufficient ? "Not enough information" : "Completed"}</span></div>
        <div className="py-3 text-[13px]"><p className="font-medium">Not assessed</p><p className={`mt-1 leading-relaxed ${muted}`}>Vocabulary, reading, listening, speaking, writing and pronunciation</p></div>
      </section>
      <aside className={`${card} p-5`}>
        <h3 className="text-[14px] font-semibold">What this means</h3>
        <p className={`mt-2 text-[13px] leading-relaxed ${muted}`}>This check covers grammar only. Your learning path can include skills that haven’t been assessed.</p>
      </aside>
    </div>
  );
}
