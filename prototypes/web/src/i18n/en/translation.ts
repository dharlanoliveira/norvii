export const englishTranslation = {
  brand: {
    descriptor: "Evidence-led legal research",
  },
  navigation: {
    corpora: "Corpora",
    backToCorpora: "All corpora",
  },
  language: {
    label: "Interface language",
    english: "English",
    portuguese: "Portuguese",
    contentLanguage: "Content language",
  },
  catalog: {
    kicker: "Research collections / prototype 01",
    title: "Choose the body of law before you ask the question.",
    introduction:
      "Each corpus is an isolated evidence boundary. Sources, citations, and conversations never cross between collections.",
    collectionLabel: "Legal corpus",
    sourceCount_one: "{{count}} source",
    sourceCount_other: "{{count}} sources",
    openCorpus: "Open corpus",
    principle: "One question. One corpus. Traceable evidence.",
  },
  corpora: {
    brazil: {
      eyebrow: "LGPD research collection",
      description:
        "A compact collection for examining lawful processing, data subject rights, and controller duties under Brazilian law.",
    },
    europe: {
      eyebrow: "GDPR research collection",
      description:
        "A focused collection for comparing processing principles, individual rights, and controller obligations in European Union law.",
    },
  },
  workspace: {
    corpusLabel: "Active corpus",
    sourcesLabel: "Source library",
    sourcesHint: "Select a source to read it beside your conversation.",
    pdfGroup: "PDF documents",
    externalGroup: "External links",
    pdfType: "PDF",
    externalType: "Link",
    available: "Available",
    unavailable: "Preview unavailable",
    treeLabel: "Sources in {{corpus}}",
    sourceSelected: "Selected source",
  },
  modes: {
    label: "Research mode",
    chat: "Chat",
    source: "Source",
  },
  viewer: {
    emptyKicker: "No source selected",
    emptyTitle: "Open a source from the library",
    emptyBody:
      "Documents and external references will appear here without interrupting your conversation.",
    documentLabel: "Document viewer",
    externalLabel: "External reference",
    publishedBy: "Published by",
    location: "Current location",
    previousSection: "Previous section",
    nextSection: "Next section",
    openOriginal: "Open original source",
    externalNotice: "The original source opens in a new browser tab.",
    previewHeading: "Saved research preview",
    unavailableTitle: "This preview is unavailable",
    unavailableBody:
      "The source remains in the corpus, but its saved preview cannot be displayed in this prototype.",
  },
  chat: {
    regionLabel: "Corpus research conversation",
    kicker: "Ask within this corpus",
    title: "Research with the sources in view.",
    emptyBody:
      "Ask a prepared question or use a suggestion. Norvii will cite the active corpus or decline to answer.",
    suggestionLabel: "Suggested question",
    composerLabel: "Question for this corpus",
    composerPlaceholder: "Ask about the active corpus...",
    send: "Send question",
    userLabel: "You",
    assistantLabel: "Norvii",
    simulated: "Deterministic prototype response",
    responding: "Reviewing the prepared evidence...",
    citationLabel: "Open citation {{label}}",
    failureTitle: "The prepared response could not be completed.",
    failureBody:
      "This deterministic failure leaves your question and corpus context intact. You can ask again or continue reviewing sources.",
    abstention:
      "I cannot support that answer with the prepared sources in this corpus. Try one of the suggested questions or review the source library.",
  },
  status: {
    prototype: "Interactive prototype",
    localData: "Local fixture data",
  },
  disclaimer: {
    title: "Prototype, not legal advice",
    body: "Answers and source excerpts are deterministic examples for product validation.",
  },
  errors: {
    unknownCorpusKicker: "Corpus not found",
    unknownCorpusTitle: "This research collection is not available.",
    unknownCorpusBody:
      "Return to the catalog and choose one of the two prepared legal corpora.",
    returnToCatalog: "Return to corpus catalog",
  },
} as const;

type DeepStringShape<Value> = {
  readonly [Key in keyof Value]: Value[Key] extends string
    ? string
    : DeepStringShape<Value[Key]>;
};

export type TranslationResource = DeepStringShape<typeof englishTranslation>;
