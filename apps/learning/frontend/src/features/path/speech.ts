/**
 * Speech recognition in the browser: free, no key, and good enough to hear
 * whether a phrase was said. It exists in Chrome and on Android, and only
 * partly on iPhone, so everything that uses it has a typed fallback.
 */

type RecognitionResult = { transcript: string }
type RecognitionEvent = { results: ArrayLike<ArrayLike<RecognitionResult>> }
type RecognitionError = { error: string }

interface Recognition {
  lang: string
  interimResults: boolean
  maxAlternatives: number
  onresult: ((event: RecognitionEvent) => void) | null
  onerror: ((event: RecognitionError) => void) | null
  onend: (() => void) | null
  start(): void
  stop(): void
}

type RecognitionConstructor = new () => Recognition

/** Tags the recognizer understands; a language not listed is tried as it is. */
const locales: Record<string, string> = {
  en: "en-US",
  es: "es-ES",
  fr: "fr-FR",
  de: "de-DE",
  it: "it-IT",
  pt: "pt-PT",
  pl: "pl-PL",
  ru: "ru-RU",
  uk: "uk-UA",
  be: "be-BY",
  zh: "zh-CN",
  ja: "ja-JP",
  ko: "ko-KR",
  tr: "tr-TR",
}

function constructorOf(): RecognitionConstructor | null {
  if (typeof window === "undefined") return null
  const scope = window as unknown as {
    SpeechRecognition?: RecognitionConstructor
    webkitSpeechRecognition?: RecognitionConstructor
  }

  return scope.SpeechRecognition ?? scope.webkitSpeechRecognition ?? null
}

/** Whether this browser can hear the learner at all. */
export function speechSupported(): boolean {
  return constructorOf() !== null
}

/**
 * Listens once and hands over what was heard. Returns a function that stops
 * listening. onError gets the browser's reason ("not-allowed" when the
 * microphone was refused, "no-speech" when nothing was said).
 */
export function listenOnce(
  language: string,
  handlers: {
    onText: (text: string) => void
    onError: (reason: string) => void
    onEnd: () => void
  },
): () => void {
  const Recognizer = constructorOf()
  if (!Recognizer) {
    handlers.onError("unsupported")
    handlers.onEnd()

    return () => undefined
  }

  const recognition = new Recognizer()
  recognition.lang = locales[language.split("-")[0].toLowerCase()] ?? language
  recognition.interimResults = false
  recognition.maxAlternatives = 1
  recognition.onresult = (event) => {
    const heard = event.results[0]?.[0]?.transcript
    if (heard) handlers.onText(heard)
  }
  recognition.onerror = (event) => handlers.onError(event.error)
  recognition.onend = handlers.onEnd

  try {
    recognition.start()
  } catch {
    handlers.onError("unavailable")
    handlers.onEnd()
  }

  return () => recognition.stop()
}
