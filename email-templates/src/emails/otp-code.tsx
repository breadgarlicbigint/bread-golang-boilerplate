import * as React from "react";
import { Heading, Text, Section, Hr } from "@react-email/components";
import { Layout, LayoutTokens } from "../components/layout";

const P = {
  name:          "__NAME__",
  code:          "__CODE__",
  expireMins:    "__EXPIRE_MINS__",
  purpose:       "__PURPOSE__",
  preview:       "__PREVIEW__",
  heading:       "__HEADING__",
  greeting:      "__GREETING__",
  expireNote:    "__EXPIRE_NOTE__",
  warningTitle:  "__WARNING_TITLE__",
  warningBody:   "__WARNING_BODY__",
};

export interface OtpCodeProps {
  name?:         string;
  code?:         string;
  expireMins?:   string;
  purpose?:      string;
  preview?:      string;
  heading?:      string;
  greeting?:     string;
  expireNote?:   string;
  warningTitle?: string;
  warningBody?:  string;
  brandName?:    string;
  footerCopyright?: string;
  footerIgnore?: string;
  lang?:         string;
}

export default function OtpCode({
  name         = P.name,
  code         = P.code,
  expireMins   = P.expireMins,
  purpose      = P.purpose,
  preview      = P.preview,
  heading      = P.heading,
  greeting     = P.greeting,
  expireNote   = P.expireNote,
  warningTitle = P.warningTitle,
  warningBody  = P.warningBody,
  brandName    = LayoutTokens.brandName,
  footerCopyright = LayoutTokens.footerCopyright,
  footerIgnore = LayoutTokens.footerIgnore,
  lang         = "en",
}: OtpCodeProps) {
  return (
    <Layout preview={preview} brandName={brandName}
      footerCopyright={footerCopyright} footerIgnore={footerIgnore} lang={lang}>
      <Heading className="text-2xl font-bold text-gray-900 mt-0 mb-2">{heading}</Heading>
      <Text className="text-gray-600 text-base mt-0 mb-6">{greeting}</Text>

      <Section className="bg-gray-50 border border-gray-200 rounded-xl py-6 text-center mb-6">
        <Text className="text-5xl font-bold tracking-[0.3em] text-gray-900 m-0 font-mono">
          {code}
        </Text>
      </Section>

      <Text className="text-gray-500 text-sm text-center mb-0">{expireNote}</Text>

      <Hr className="border-gray-200 my-6" />

      <Section className="bg-red-50 border border-red-100 rounded-lg p-4">
        <Text className="text-red-700 text-sm font-semibold m-0">{warningTitle}</Text>
        <Text className="text-red-600 text-sm mt-1 mb-0">{warningBody}</Text>
      </Section>
    </Layout>
  );
}

export function OtpCodeText(p: OtpCodeProps): string {
  return `${p.greeting ?? P.greeting}

${p.code ?? P.code}

${p.expireNote ?? P.expireNote}

${p.warningTitle ?? P.warningTitle}
${p.warningBody ?? P.warningBody}

${p.brandName ?? LayoutTokens.brandName}`;
}
