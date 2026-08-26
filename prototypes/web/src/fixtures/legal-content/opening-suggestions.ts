import type {
  CorpusFixture,
  CorpusReleaseFixture,
  LanguageCode,
  OpeningSuggestionFixture,
  OpeningSuggestionSetFixture,
  PreparedAnswer,
} from "../models";

interface CorpusSuggestionFixture {
  readonly corpusId: string;
  readonly activeRelease: CorpusReleaseFixture;
  readonly openingSuggestionSet: OpeningSuggestionSetFixture;
}

function createSuggestionSet(
  corpusId: string,
  snapshotId: string,
  snapshotManifestSha256: string,
  questions: readonly (readonly [string, string, string])[],
): OpeningSuggestionSetFixture {
  const suggestions: OpeningSuggestionFixture[] = questions.flatMap(
    ([caseId, englishQuestion, portugueseQuestion], index) => [
      {
        caseId,
        rank: index + 1,
        language: "en",
        question: englishQuestion,
      },
      {
        caseId,
        rank: index + 1,
        language: "pt",
        question: portugueseQuestion,
      },
    ],
  );

  return {
    corpusId,
    snapshotId,
    snapshotManifestSha256,
    suggestions,
  };
}

function createCorpusSuggestionFixture(
  corpusId: string,
  snapshotId: string,
  snapshotManifestSha256: string,
  questions: Parameters<typeof createSuggestionSet>[3],
): CorpusSuggestionFixture {
  return {
    corpusId,
    activeRelease: { snapshotId, snapshotManifestSha256 },
    openingSuggestionSet: createSuggestionSet(
      corpusId,
      snapshotId,
      snapshotManifestSha256,
      questions,
    ),
  };
}

const lgpdSuggestionFixture = createCorpusSuggestionFixture(
  "brazil-data-protection",
  "lgpd-snapshot-v1",
  "prototype-lgpd-manifest-v1",
  [
    [
      "lgpd-controller",
      "Which role decides the purposes of personal-data processing?",
      "Qual papel decide as finalidades do tratamento de dados pessoais?",
    ],
    [
      "lgpd-access",
      "What can a data subject request from the controller?",
      "O que o titular pode solicitar ao controlador?",
    ],
    [
      "lgpd-necessity",
      "How does this prototype describe the necessity principle?",
      "Como este protótipo descreve o princípio da necessidade?",
    ],
    [
      "lgpd-correction",
      "Which correction request does the synthetic LGPD fixture model?",
      "Qual pedido de correção a fixture sintética da LGPD modela?",
    ],
    [
      "lgpd-transparency",
      "What transparency topic appears in the synthetic LGPD fixture?",
      "Qual tema de transparência aparece na fixture sintética da LGPD?",
    ],
  ],
);

const europeanDataProtectionSuggestionFixture = createCorpusSuggestionFixture(
  "eu-data-protection",
  "gdpr-snapshot-v1",
  "prototype-gdpr-manifest-v1",
  [
    [
      "gdpr-access",
      "What does the GDPR right of access allow a person to obtain?",
      "O que o direito de acesso do GDPR permite que uma pessoa obtenha?",
    ],
    [
      "gdpr-minimisation",
      "How does the GDPR describe data minimisation?",
      "Como o GDPR descreve a minimização de dados?",
    ],
    [
      "gdpr-controller",
      "Which party determines the purposes and essential means in this GDPR fixture?",
      "Qual parte determina as finalidades e os meios essenciais nesta fixture do GDPR?",
    ],
    [
      "gdpr-lawfulness",
      "What lawful-processing theme does the synthetic GDPR fixture present?",
      "Qual tema de tratamento lícito a fixture sintética do GDPR apresenta?",
    ],
    [
      "gdpr-transparency",
      "How is transparent processing represented in this GDPR fixture?",
      "Como o tratamento transparente é representado nesta fixture do GDPR?",
    ],
  ],
);

const antiCorruptionSuggestionFixture = createCorpusSuggestionFixture(
  "brazil-anti-corruption-white-collar-crime",
  "anti-corruption-snapshot-v1",
  "prototype-anti-corruption-manifest-v1",
  [
    [
      "brac-liability",
      "When can a legal entity be held liable for a corrupt act?",
      "Quando uma pessoa jurídica pode ser responsabilizada por um ato corrupto?",
    ],
    [
      "brac-undue-advantage",
      "What is an undue advantage in a public-procurement context?",
      "O que é uma vantagem indevida em um contexto de contratação pública?",
    ],
    [
      "brac-public-official",
      "Which public-official scenario does the synthetic anti-corruption fixture model?",
      "Qual cenário de agente público a fixture sintética anticorrupção modela?",
    ],
    [
      "brac-controls",
      "What internal-control topic appears in the synthetic anti-corruption fixture?",
      "Qual tema de controle interno aparece na fixture sintética anticorrupção?",
    ],
    [
      "brac-reporting",
      "How does the synthetic anti-corruption fixture frame reporting concerns?",
      "Como a fixture sintética anticorrupção enquadra preocupações de reporte?",
    ],
  ],
);

const fairHousingSuggestionFixture = createCorpusSuggestionFixture(
  "us-fair-housing-disability-accommodations",
  "fair-housing-snapshot-v1",
  "prototype-fair-housing-manifest-v1",
  [
    [
      "fair-housing-accommodation",
      "What is a reasonable accommodation in housing?",
      "O que é uma adaptação razoável em moradia?",
    ],
    [
      "fair-housing-assistance-animal",
      "How can an assistance animal relate to a housing accommodation?",
      "Como um animal de assistência pode se relacionar a uma adaptação em moradia?",
    ],
    [
      "fair-housing-discrimination",
      "Which disability-discrimination scenario does the synthetic housing fixture model?",
      "Qual cenário de discriminação por deficiência a fixture sintética de moradia modela?",
    ],
    [
      "fair-housing-modification",
      "How does the synthetic fixture distinguish a housing modification?",
      "Como a fixture sintética distingue uma modificação em moradia?",
    ],
    [
      "fair-housing-request",
      "What accommodation-request topic appears in the synthetic housing fixture?",
      "Qual tema de pedido de adaptação aparece na fixture sintética de moradia?",
    ],
  ],
);

export const corpusSuggestionFixtures = [
  lgpdSuggestionFixture,
  europeanDataProtectionSuggestionFixture,
  antiCorruptionSuggestionFixture,
  fairHousingSuggestionFixture,
] as const;

export function findCorpusSuggestionFixture(
  corpusId: string,
): CorpusSuggestionFixture | undefined {
  return corpusSuggestionFixtures.find(
    (fixture) => fixture.corpusId === corpusId,
  );
}

export function createSuggestionPreparedAnswers(
  suggestionSet: OpeningSuggestionSetFixture,
): readonly PreparedAnswer[] {
  return suggestionSet.suggestions.map((suggestion) => ({
    id: `opening-suggestion-${suggestion.caseId}-${suggestion.language}`,
    prompts: [suggestion.question],
    answer: `This is the prepared synthetic prototype answer for ${suggestion.caseId} in the selected corpus.`,
    citations: [],
  }));
}

export function resolveOpeningSuggestions(
  corpus: Pick<CorpusFixture, "id" | "activeRelease" | "openingSuggestionSet">,
  language: LanguageCode,
): readonly OpeningSuggestionFixture[] {
  const release = corpus.activeRelease;
  const suggestionSet = corpus.openingSuggestionSet;

  if (
    !release ||
    !suggestionSet ||
    suggestionSet.corpusId !== corpus.id ||
    suggestionSet.snapshotId !== release.snapshotId ||
    suggestionSet.snapshotManifestSha256 !== release.snapshotManifestSha256
  ) {
    return [];
  }

  return suggestionSet.suggestions
    .filter((suggestion) => suggestion.language === language)
    .toSorted((left, right) => left.rank - right.rank);
}
