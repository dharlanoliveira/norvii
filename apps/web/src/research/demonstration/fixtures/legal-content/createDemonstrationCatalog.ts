import type { Corpus } from "../../../domain/models";
import {
  createResearchCatalog,
  type ResearchCatalog,
} from "../../../domain/researchCatalog";

const brazilCorpus: Corpus = {
  id: "brazil-data-protection",
  language: "pt",
  name: "Lei Geral de Proteção de Dados Pessoais",
  jurisdiction: "Brazil",
  summary:
    "A focused collection covering the Brazilian data protection framework.",
  suggestedQuestions: [
    "Quais são os princípios para o tratamento de dados pessoais?",
    "Quando um incidente deve ser comunicado?",
  ],
  sources: [
    {
      id: "lgpd",
      corpusId: "brazil-data-protection",
      kind: "pdf",
      title: "Lei 13.709/2018 - LGPD",
      authority: "Presidency of the Republic of Brazil",
      officialReference: "Law 13,709 of August 14, 2018",
      pageCount: 3,
      locations: [
        {
          id: "article-6",
          label: "Article 6",
          page: 1,
          content:
            "As atividades de tratamento de dados pessoais deverão observar a boa-fé e os princípios da finalidade, adequação, necessidade, livre acesso, qualidade dos dados, transparência, segurança, prevenção, não discriminação e responsabilização.",
        },
        {
          id: "article-18",
          label: "Article 18",
          page: 2,
          content:
            "O titular dos dados pessoais tem direito a obter do controlador informações sobre o tratamento e exercer os direitos previstos em lei.",
        },
        {
          id: "article-48",
          label: "Article 48",
          page: 3,
          content:
            "O controlador deverá comunicar à autoridade nacional e ao titular a ocorrência de incidente de segurança que possa acarretar risco ou dano relevante.",
        },
      ],
    },
    {
      id: "anpd-incident-regulation",
      corpusId: "brazil-data-protection",
      kind: "external",
      title: "ANPD security incident regulation",
      authority: "Brazilian Data Protection Authority",
      officialReference: "ANPD Resolution 15/2024",
      url: "https://www.gov.br/anpd/pt-br/assuntos/incidente-de-seguranca",
      preview: {
        status: "available",
        summary:
          "Official guidance on assessing and communicating security incidents.",
      },
      locations: [
        {
          id: "notification-overview",
          label: "Notification overview",
          content:
            "The authority describes the conditions and information expected for incident communication.",
        },
      ],
    },
  ],
  preparedResponses: [
    {
      id: "lgpd-principles",
      prompts: ["princípios", "tratamento de dados"],
      outcome: "answered",
      parts: [
        {
          type: "text",
          text: "Article 6 lists purpose, adequacy, necessity, free access, data quality, transparency, security, prevention, non-discrimination, and accountability among the principles that guide personal-data processing.",
        },
        {
          type: "citation",
          citation: {
            id: "lgpd-article-6",
            sourceId: "lgpd",
            locationId: "article-6",
            label: "LGPD, Article 6",
          },
        },
      ],
    },
    {
      id: "brazil-response-failure",
      prompts: ["simulate failure"],
      outcome: "failed",
      parts: [],
    },
  ],
};

const europeanCorpus: Corpus = {
  id: "eu-data-protection",
  language: "en",
  name: "European Data Protection Framework",
  jurisdiction: "European Union",
  summary:
    "A compact research collection centered on the GDPR and official EDPB guidance.",
  suggestedQuestions: [
    "What principles govern personal data processing?",
    "What is the controller responsible for?",
  ],
  sources: [
    {
      id: "gdpr",
      corpusId: "eu-data-protection",
      kind: "pdf",
      title: "General Data Protection Regulation",
      authority: "European Parliament and Council",
      officialReference: "Regulation (EU) 2016/679",
      pageCount: 3,
      locations: [
        {
          id: "article-5",
          label: "Article 5",
          page: 1,
          content:
            "Personal data shall be processed lawfully, fairly and transparently, collected for specified purposes, limited to what is necessary, accurate, retained only as needed, and secured appropriately.",
        },
        {
          id: "article-24",
          label: "Article 24",
          page: 2,
          content:
            "The controller shall implement appropriate technical and organisational measures and be able to demonstrate that processing is performed in accordance with the Regulation.",
        },
        {
          id: "article-33",
          label: "Article 33",
          page: 3,
          content:
            "A qualifying personal data breach shall be notified to the supervisory authority without undue delay and, where feasible, within 72 hours.",
        },
      ],
    },
    {
      id: "edpb-controller-guidelines",
      corpusId: "eu-data-protection",
      kind: "external",
      title: "Guidelines on controller and processor concepts",
      authority: "European Data Protection Board",
      officialReference: "EDPB Guidelines 07/2020",
      url: "https://www.edpb.europa.eu/our-work-tools/our-documents/guidelines/guidelines-072020-concepts-controller-and-processor-gdpr_en",
      preview: {
        status: "available",
        summary:
          "Official interpretative guidance on allocating controller and processor roles.",
      },
      locations: [
        {
          id: "role-overview",
          label: "Role overview",
          content:
            "The guidelines explain that functional analysis of actual influence determines controller and processor roles.",
        },
      ],
    },
    {
      id: "edpb-breach-guidelines",
      corpusId: "eu-data-protection",
      kind: "external",
      title: "Guidelines on personal data breach notification",
      authority: "European Data Protection Board",
      officialReference: "EDPB Guidelines 9/2022",
      url: "https://www.edpb.europa.eu/our-work-tools/our-documents/guidelines/guidelines-92022-personal-data-breach-notification_en",
      preview: {
        status: "unavailable",
        reason:
          "This demonstration does not embed the remote page. The official source remains available in a new browser context.",
      },
      locations: [],
    },
  ],
  preparedResponses: [
    {
      id: "gdpr-principles",
      prompts: ["principles", "personal data processing"],
      outcome: "answered",
      parts: [
        {
          type: "text",
          text: "Article 5 establishes lawfulness, fairness, transparency, purpose limitation, data minimisation, accuracy, storage limitation, integrity, confidentiality, and accountability as core processing principles.",
        },
        {
          type: "citation",
          citation: {
            id: "gdpr-article-5",
            sourceId: "gdpr",
            locationId: "article-5",
            label: "GDPR, Article 5",
          },
        },
      ],
    },
    {
      id: "controller-responsibility",
      prompts: ["controller responsible", "controller responsibility"],
      outcome: "answered",
      parts: [
        {
          type: "text",
          text: "The controller must implement appropriate measures and be able to demonstrate compliance with the Regulation.",
        },
        {
          type: "citation",
          citation: {
            id: "gdpr-article-24",
            sourceId: "gdpr",
            locationId: "article-24",
            label: "GDPR, Article 24",
          },
        },
      ],
    },
    {
      id: "eu-response-failure",
      prompts: ["simulate failure"],
      outcome: "failed",
      parts: [],
    },
  ],
};

export function createDemonstrationCatalog(): ResearchCatalog {
  return createResearchCatalog([brazilCorpus, europeanCorpus]);
}
