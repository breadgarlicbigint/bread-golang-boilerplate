import * as React from "react";
import { Button, Heading, Text, Section, Hr, Row, Column } from "@react-email/components";
import { Layout, LayoutTokens } from "../components/layout";

const P = {
  name:         "__NAME__",
  dashboardUrl: "__DASHBOARD_URL__",
  docsUrl:      "__DOCS_URL__",
  preview:      "__PREVIEW__",
  heading:      "__HEADING__",
  greeting:     "__GREETING__",
  subheading:   "__SUBHEADING__",
  ctaPrimary:   "__CTA_PRIMARY__",
  ctaSecondary: "__CTA_SECONDARY__",
  helpNote:     "__HELP_NOTE__",
  // feature tokens
  feat1Icon:    "__FEAT1_ICON__",
  feat1Title:   "__FEAT1_TITLE__",
  feat1Desc:    "__FEAT1_DESC__",
  feat2Icon:    "__FEAT2_ICON__",
  feat2Title:   "__FEAT2_TITLE__",
  feat2Desc:    "__FEAT2_DESC__",
  feat3Icon:    "__FEAT3_ICON__",
  feat3Title:   "__FEAT3_TITLE__",
  feat3Desc:    "__FEAT3_DESC__",
};

export interface WelcomeProps {
  name?:          string;
  dashboardUrl?:  string;
  docsUrl?:       string;
  preview?:       string;
  heading?:       string;
  greeting?:      string;
  subheading?:    string;
  ctaPrimary?:    string;
  ctaSecondary?:  string;
  helpNote?:      string;
  feat1Icon?:     string;
  feat1Title?:    string;
  feat1Desc?:     string;
  feat2Icon?:     string;
  feat2Title?:    string;
  feat2Desc?:     string;
  feat3Icon?:     string;
  feat3Title?:    string;
  feat3Desc?:     string;
  brandName?:     string;
  footerCopyright?: string;
  footerIgnore?:  string;
  lang?:          string;
}

export default function Welcome({
  name          = P.name,
  dashboardUrl  = P.dashboardUrl,
  docsUrl       = P.docsUrl,
  preview       = P.preview,
  heading       = P.heading,
  greeting      = P.greeting,
  subheading    = P.subheading,
  ctaPrimary    = P.ctaPrimary,
  ctaSecondary  = P.ctaSecondary,
  helpNote      = P.helpNote,
  feat1Icon     = P.feat1Icon,
  feat1Title    = P.feat1Title,
  feat1Desc     = P.feat1Desc,
  feat2Icon     = P.feat2Icon,
  feat2Title    = P.feat2Title,
  feat2Desc     = P.feat2Desc,
  feat3Icon     = P.feat3Icon,
  feat3Title    = P.feat3Title,
  feat3Desc     = P.feat3Desc,
  brandName     = LayoutTokens.brandName,
  footerCopyright = LayoutTokens.footerCopyright,
  footerIgnore  = LayoutTokens.footerIgnore,
  lang          = "en",
}: WelcomeProps) {
  const features = [
    { icon: feat1Icon, title: feat1Title, desc: feat1Desc },
    { icon: feat2Icon, title: feat2Title, desc: feat2Desc },
    { icon: feat3Icon, title: feat3Title, desc: feat3Desc },
  ];

  return (
    <Layout preview={preview} brandName={brandName}
      footerCopyright={footerCopyright} footerIgnore={footerIgnore} lang={lang}>
      <Heading className="text-2xl font-bold text-gray-900 mt-0 mb-2">{heading}</Heading>
      <Text className="text-gray-600 text-base mt-0 mb-2">{greeting}</Text>
      <Text className="text-gray-600 text-base mt-0 mb-6">{subheading}</Text>

      {features.map((f, i) => (
        <Section key={i} className="mb-3">
          <Row>
            <Column className="w-8 text-xl">{f.icon}</Column>
            <Column>
              <Text className="text-gray-900 font-semibold text-sm m-0">{f.title}</Text>
              <Text className="text-gray-500 text-sm m-0">{f.desc}</Text>
            </Column>
          </Row>
        </Section>
      ))}

      <Hr className="border-gray-200 my-6" />

      <Section className="text-center">
        <Button href={dashboardUrl}
          className="bg-black text-white rounded-lg px-6 py-3 text-sm font-semibold no-underline mr-3">
          {ctaPrimary}
        </Button>
        <Button href={docsUrl}
          className="bg-white text-gray-900 border border-gray-300 rounded-lg px-6 py-3 text-sm font-semibold no-underline">
          {ctaSecondary}
        </Button>
      </Section>

      <Text className="text-gray-400 text-xs text-center mt-6 mb-0">{helpNote}</Text>
    </Layout>
  );
}

export function WelcomeText(p: WelcomeProps): string {
  return `${p.greeting ?? P.greeting}

${p.subheading ?? P.subheading}

• ${p.feat1Title ?? P.feat1Title}: ${p.feat1Desc ?? P.feat1Desc}
• ${p.feat2Title ?? P.feat2Title}: ${p.feat2Desc ?? P.feat2Desc}
• ${p.feat3Title ?? P.feat3Title}: ${p.feat3Desc ?? P.feat3Desc}

${p.ctaPrimary ?? P.ctaPrimary}: ${p.dashboardUrl ?? P.dashboardUrl}
${p.ctaSecondary ?? P.ctaSecondary}: ${p.docsUrl ?? P.docsUrl}

${p.helpNote ?? P.helpNote}

${p.brandName ?? LayoutTokens.brandName}`;
}
