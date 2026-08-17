import type { CorpusFixture } from "../models";

export const corpora: readonly CorpusFixture[] = [
  {
    id: "brazil-data-protection",
    label: "Brazilian Data Protection",
    eyebrowKey: "corpora.brazil.eyebrow",
    descriptionKey: "corpora.brazil.description",
    language: "pt",
    jurisdiction: "Brazil",
    sources: [
      {
        id: "lgpd-law",
        corpusId: "brazil-data-protection",
        kind: "pdf",
        title: "Lei Geral de Proteção de Dados Pessoais — Lei nº 13.709/2018",
        shortTitle: "LGPD — Lei nº 13.709/2018",
        publisher: "Presidência da República",
        publishedLabel: "Official legal text · 2018",
        status: "available",
        sections: [
          {
            id: "article-6",
            marker: "Art. 6",
            heading: "Princípios das atividades de tratamento",
            paragraphs: [
              "As atividades de tratamento de dados pessoais deverão observar a boa-fé e os princípios da finalidade, adequação, necessidade, livre acesso, qualidade dos dados, transparência, segurança, prevenção, não discriminação e responsabilização.",
              "O princípio da necessidade limita o tratamento ao mínimo necessário para a realização de suas finalidades, com dados pertinentes, proporcionais e não excessivos.",
            ],
          },
          {
            id: "article-18",
            marker: "Art. 18",
            heading: "Direitos do titular",
            paragraphs: [
              "O titular dos dados pessoais tem direito a obter do controlador, em relação aos dados por ele tratados, confirmação da existência de tratamento e acesso aos dados.",
              "Também pode solicitar correção, anonimização, bloqueio ou eliminação de dados desnecessários, excessivos ou tratados em desconformidade com a lei.",
            ],
          },
        ],
      },
      {
        id: "anpd-guide",
        corpusId: "brazil-data-protection",
        kind: "external",
        title: "Guia orientativo para definições dos agentes de tratamento",
        shortTitle: "ANPD guide to processing agents",
        publisher: "Autoridade Nacional de Proteção de Dados",
        publishedLabel: "Guidance · 2022",
        status: "available",
        externalUrl:
          "https://www.gov.br/anpd/pt-br/documentos-e-publicacoes/guia-agentes-de-tratamento.pdf",
        sections: [
          {
            id: "controller-definition",
            marker: "Section 2.1",
            heading: "Controlador",
            paragraphs: [
              "O controlador é o agente responsável por tomar as principais decisões referentes ao tratamento de dados pessoais e por definir a finalidade e os elementos essenciais desse tratamento.",
            ],
          },
        ],
      },
      {
        id: "anpd-broken-preview",
        corpusId: "brazil-data-protection",
        kind: "external",
        title: "ANPD regulatory agenda archive",
        shortTitle: "Regulatory agenda archive",
        publisher: "Autoridade Nacional de Proteção de Dados",
        publishedLabel: "Archive · preview unavailable",
        status: "unavailable",
        externalUrl:
          "https://www.gov.br/anpd/pt-br/assuntos/agenda-regulatoria",
        sections: [],
      },
    ],
    preparedAnswers: [
      {
        id: "lgpd-rights",
        prompts: [
          "What rights does a data subject have?",
          "Quais são os direitos do titular?",
        ],
        answer:
          "Under the prepared LGPD excerpt, a data subject can request confirmation that processing exists, access to personal data, correction, and measures concerning unnecessary, excessive, or unlawfully processed data.",
        citations: [
          {
            id: "citation-lgpd-18",
            sourceId: "lgpd-law",
            sectionId: "article-18",
            label: "LGPD, Art. 18",
          },
        ],
      },
      {
        id: "lgpd-controller",
        prompts: ["Who is the controller?", "Quem é o controlador?"],
        answer:
          "The controller is the processing agent that makes the principal decisions about personal-data processing, including its purpose and essential elements.",
        citations: [
          {
            id: "citation-anpd-controller",
            sourceId: "anpd-guide",
            sectionId: "controller-definition",
            label: "ANPD guide, Section 2.1",
          },
        ],
      },
    ],
    suggestedQuestions: [
      "What rights does a data subject have?",
      "Who is the controller?",
    ],
    failurePrompts: [
      "Simulate a failed response",
      "Simule uma resposta com falha",
    ],
  },
  {
    id: "eu-data-protection",
    label: "European Data Protection",
    eyebrowKey: "corpora.europe.eyebrow",
    descriptionKey: "corpora.europe.description",
    language: "en",
    jurisdiction: "European Union",
    sources: [
      {
        id: "gdpr-regulation",
        corpusId: "eu-data-protection",
        kind: "pdf",
        title: "General Data Protection Regulation — Regulation (EU) 2016/679",
        shortTitle: "GDPR — Regulation 2016/679",
        publisher: "European Parliament and Council",
        publishedLabel: "Official Journal · 2016",
        status: "available",
        sections: [
          {
            id: "article-5",
            marker: "Article 5",
            heading: "Principles relating to processing of personal data",
            paragraphs: [
              "Personal data shall be processed lawfully, fairly and in a transparent manner in relation to the data subject.",
              "Data shall be adequate, relevant and limited to what is necessary in relation to the purposes for which they are processed.",
            ],
          },
          {
            id: "article-15",
            marker: "Article 15",
            heading: "Right of access by the data subject",
            paragraphs: [
              "The data subject shall have the right to obtain from the controller confirmation as to whether or not personal data concerning them are being processed and, where that is the case, access to the personal data.",
            ],
          },
        ],
      },
      {
        id: "edpb-guidelines",
        corpusId: "eu-data-protection",
        kind: "external",
        title: "Guidelines 07/2020 on the concepts of controller and processor",
        shortTitle: "EDPB controller guidelines",
        publisher: "European Data Protection Board",
        publishedLabel: "Guidelines · 2021",
        status: "available",
        externalUrl:
          "https://www.edpb.europa.eu/our-work-tools/our-documents/guidelines/guidelines-072020-concepts-controller-and-processor-gdpr_en",
        sections: [
          {
            id: "controller-concept",
            marker: "Paragraph 20",
            heading: "Essential means of processing",
            paragraphs: [
              "A controller determines the purposes and essential means of processing. Essential means are closely linked to the purpose and scope of processing.",
            ],
          },
        ],
      },
    ],
    preparedAnswers: [
      {
        id: "gdpr-access",
        prompts: [
          "What does the right of access include?",
          "O que inclui o direito de acesso?",
        ],
        answer:
          "The prepared GDPR excerpt gives a data subject the right to obtain confirmation of processing and, where processing occurs, access to the personal data concerned.",
        citations: [
          {
            id: "citation-gdpr-15",
            sourceId: "gdpr-regulation",
            sectionId: "article-15",
            label: "GDPR, Article 15",
          },
        ],
      },
      {
        id: "gdpr-minimisation",
        prompts: [
          "What is data minimisation?",
          "O que é minimização de dados?",
        ],
        answer:
          "Data minimisation requires personal data to be adequate, relevant, and limited to what is necessary for the purposes of processing.",
        citations: [
          {
            id: "citation-gdpr-5",
            sourceId: "gdpr-regulation",
            sectionId: "article-5",
            label: "GDPR, Article 5(1)(c)",
          },
        ],
      },
    ],
    suggestedQuestions: [
      "What does the right of access include?",
      "What is data minimisation?",
    ],
    failurePrompts: [
      "Simulate a failed response",
      "Simule uma resposta com falha",
    ],
  },
] as const;

export function findCorpus(corpusId: string): CorpusFixture | undefined {
  return corpora.find((corpus) => corpus.id === corpusId);
}
