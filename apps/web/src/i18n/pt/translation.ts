import type { englishTranslation } from "../en/translation";

type TranslationShape = {
  [Section in keyof typeof englishTranslation]: {
    [Key in keyof (typeof englishTranslation)[Section]]: string;
  };
};

export const portugueseTranslation = {
  app: {
    brandTagline: "Pesquisa jurídica orientada por evidências",
    demonstration: "Demonstração técnica",
    notLegalAdvice: "Esta experiência não oferece aconselhamento jurídico.",
    skipToContent: "Ir para o conteúdo principal",
  },
  language: {
    label: "Idioma da interface",
    en: "Inglês",
    pt: "Português",
  },
  catalog: {
    kicker: "Coleções jurídicas selecionadas",
    title: "Escolha o limite da sua pesquisa.",
    introduction:
      "Cada corpus é um ambiente isolado de evidências. Selecione um antes de revisar fontes ou iniciar uma conversa.",
    collectionLabel: "Corpus jurídico",
    language: "Idioma",
    jurisdiction: "Jurisdição",
    sources: "Fontes",
    sourceCount: "{{count}} fonte",
    sourceCount_other: "{{count}} fontes",
    openCorpus: "Abrir corpus",
  },
  workspace: {
    backToCatalog: "Todos os corpus",
    activeCorpus: "Limite de evidências ativo",
    library: "Biblioteca de fontes",
    libraryDescription: "Documentos disponíveis neste corpus",
    chat: "Chat",
    source: "Fonte",
    modeLabel: "Modo de pesquisa",
  },
  tree: {
    label: "Fontes do corpus",
    pdfDocuments: "Documentos PDF",
    externalLinks: "Links externos",
    expand: "Expandir {{label}}",
    collapse: "Recolher {{label}}",
    pdfType: "PDF",
    externalType: "Link",
    available: "Disponível",
    unavailable: "Prévia indisponível",
    selected: "Selecionado",
  },
  viewer: {
    noSourceKicker: "Mesa de fontes",
    noSourceTitle: "Selecione uma fonte na biblioteca.",
    noSourceBody:
      "Documentos e links oficiais abrem aqui enquanto sua conversa permanece disponível.",
    metadata: "Detalhes da fonte",
    authority: "Autoridade",
    reference: "Referência oficial",
    currentLocation: "Localização atual",
    previous: "Localização anterior",
    next: "Próxima localização",
    page: "Página {{page}} de {{count}}",
    externalKicker: "Fonte externa oficial",
    preview: "Visão geral capturada",
    previewUnavailable: "Prévia indisponível",
    openOfficial: "Abrir fonte oficial",
    opensNewContext: "Abre a página HTTPS oficial em uma nova aba.",
  },
  chat: {
    regionLabel: "Conversa de pesquisa do corpus",
    kicker: "Assistente limitado ao corpus",
    title: "Pergunte com base nas evidências desta sala.",
    emptyBody:
      "Experimente uma pergunta preparada para inspecionar uma resposta estruturada e sua fonte.",
    userLabel: "Você",
    assistantLabel: "Norvii",
    simulated: "Resposta simulada",
    responding: "Revisando as evidências preparadas do corpus...",
    composerLabel: "Pergunta de pesquisa",
    composerPlaceholder: "Pergunte sobre este corpus...",
    send: "Enviar pergunta",
    citationLabel: "Abrir citação {{label}}",
    abstention:
      "As evidências preparadas nesta demonstração não sustentam uma resposta. O Norvii se abstém em vez de inferir uma conclusão jurídica.",
    failureTitle: "A resposta simulada não pôde ser concluída.",
    failureBody: "Sua pergunta foi preservada. Tente enviá-la novamente.",
    retry: "Tentar novamente",
    retryComplete:
      "A nova tentativa foi concluída. Nenhuma resposta foi adicionada porque este cenário preparado demonstra apenas a recuperação.",
    localData: "Dados locais de demonstração",
  },
  errors: {
    unknownCorpusKicker: "Sala de evidências desconhecida",
    unknownCorpusTitle: "Este corpus não foi encontrado.",
    unknownCorpusBody:
      "O endereço pode estar incompleto ou o corpus pode não estar mais disponível nesta demonstração.",
    returnToCatalog: "Voltar ao catálogo de corpus",
    unavailableCitation: "O local citado está indisponível.",
  },
} satisfies TranslationShape;
