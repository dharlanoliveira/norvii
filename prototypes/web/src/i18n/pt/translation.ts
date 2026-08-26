import type { TranslationResource } from "../en/translation";

export const portugueseTranslation: TranslationResource = {
  brand: {
    descriptor: "Pesquisa jurídica orientada por evidências",
  },
  navigation: {
    corpora: "Corpora",
    backToCorpora: "Todos os corpora",
  },
  language: {
    label: "Idioma da interface",
    english: "Inglês",
    portuguese: "Português",
    contentLanguage: "Idioma do conteúdo",
  },
  catalog: {
    kicker: "Coleções de pesquisa · protótipo 01",
    title: "Escolha o conjunto jurídico antes de fazer a pergunta.",
    introduction:
      "Cada corpus é uma fronteira isolada de evidências. Fontes, citações e conversas nunca se misturam entre as coleções.",
    collectionLabel: "Corpus jurídico",
    sourceCount_one: "{{count}} fonte",
    sourceCount_other: "{{count}} fontes",
    openCorpus: "Abrir corpus",
    principle: "Uma pergunta. Um corpus. Evidências rastreáveis.",
  },
  corpora: {
    brazil: {
      eyebrow: "Coleção de pesquisa da LGPD",
      description:
        "Uma coleção compacta para examinar tratamento lícito, direitos dos titulares e deveres do controlador na legislação brasileira.",
    },
    europe: {
      eyebrow: "Coleção de pesquisa do GDPR",
      description:
        "Uma coleção focada na comparação de princípios de tratamento, direitos individuais e obrigações do controlador na União Europeia.",
    },
    antiCorruption: {
      eyebrow: "Coleção de pesquisa anticorrupção",
      description:
        "Uma coleção sintética segura para explorar cenários de responsabilidade empresarial e contratação pública no protótipo.",
    },
    fairHousing: {
      eyebrow: "Coleção de pesquisa de moradia justa",
      description:
        "Uma coleção sintética segura para explorar cenários de adaptações por deficiência no protótipo.",
    },
  },
  workspace: {
    corpusLabel: "Corpus ativo",
    sourcesLabel: "Biblioteca de fontes",
    sourcesHint: "Selecione uma fonte para lê-la ao lado da conversa.",
    pdfGroup: "Documentos PDF",
    externalGroup: "Links externos",
    pdfType: "PDF",
    externalType: "Link",
    available: "Disponível",
    unavailable: "Prévia indisponível",
    treeLabel: "Fontes em {{corpus}}",
    sourceSelected: "Fonte selecionada",
  },
  modes: {
    label: "Modo de pesquisa",
    chat: "Chat",
    source: "Fonte",
  },
  viewer: {
    emptyKicker: "Nenhuma fonte selecionada",
    emptyTitle: "Abra uma fonte da biblioteca",
    emptyBody:
      "Documentos e referências externas aparecerão aqui sem interromper sua conversa.",
    documentLabel: "Visualizador de documento",
    externalLabel: "Referência externa",
    publishedBy: "Publicado por",
    location: "Localização atual",
    previousSection: "Seção anterior",
    nextSection: "Próxima seção",
    openOriginal: "Abrir fonte original",
    externalNotice: "A fonte original abre em uma nova aba do navegador.",
    previewHeading: "Prévia de pesquisa salva",
    unavailableTitle: "Esta prévia está indisponível",
    unavailableBody:
      "A fonte continua no corpus, mas sua prévia salva não pode ser exibida neste protótipo.",
  },
  chat: {
    regionLabel: "Conversa de pesquisa no corpus",
    kicker: "Pergunte dentro deste corpus",
    title: "Pesquise com as fontes à vista.",
    emptyBody:
      "Faça uma pergunta preparada ou use uma sugestão. O Norvii citará o corpus ativo ou recusará a resposta.",
    suggestionLabel: "Pergunta sugerida",
    composerLabel: "Pergunta para este corpus",
    composerPlaceholder: "Pergunte sobre o corpus ativo…",
    send: "Enviar pergunta",
    userLabel: "Você",
    assistantLabel: "Norvii",
    simulated: "Resposta determinística do protótipo",
    responding: "Revisando as evidências preparadas…",
    citationLabel: "Abrir citação {{label}}",
    failureTitle: "A resposta preparada não pôde ser concluída.",
    failureBody:
      "Esta falha determinística mantém sua pergunta e o contexto do corpus. Você pode perguntar novamente ou continuar consultando as fontes.",
    abstention:
      "Não posso sustentar essa resposta com as fontes preparadas neste corpus. Tente uma das perguntas sugeridas ou consulte a biblioteca de fontes.",
  },
  status: {
    prototype: "Protótipo interativo",
    localData: "Dados locais de demonstração",
  },
  disclaimer: {
    title: "Protótipo, não aconselhamento jurídico",
    body: "Respostas e trechos de fontes são exemplos determinísticos para validação do produto.",
  },
  errors: {
    unknownCorpusKicker: "Corpus não encontrado",
    unknownCorpusTitle: "Esta coleção de pesquisa não está disponível.",
    unknownCorpusBody:
      "Volte ao catálogo e escolha um dos corpora jurídicos preparados.",
    returnToCatalog: "Voltar ao catálogo de corpora",
  },
};
