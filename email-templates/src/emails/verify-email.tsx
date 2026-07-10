import * as React from "react";
import { Button, Heading, Text, Section, Hr } from "@react-email/components";
import { Layout, LayoutTokens } from "../components/layout";

// ── Data tokens (dynamic values) ─────────────────────────────────────────────
// ── Text tokens  (i18n resolved by Go before sending) ────────────────────────
const P = {
  // data
  name:          "__NAME__",
  verifyLink:    "__VERIFY_LINK__",
  expireHrs:     "__EXPIRE_HRS__",
  // i18n text
  preview:       "__PREVIEW__",
  heading:       "__HEADING__",
  greeting:      "__GREETING__",
  body:          "__BODY__",
  ctaLabel:      "__CTA_LABEL__",
  copyLinkText:  "__COPY_LINK_TEXT__",
  expireNote:    "__EXPIRE_NOTE__",
  ignoreNote:    "__IGNORE_NOTE__",
};

export interface VerifyEmailProps {
  name?:         string;
  verifyLink?:   string;
  expireHrs?:    string;
  preview?:      string;
  heading?:      string;
  greeting?:     string;
  body?:         string;
  ctaLabel?:     string;
  copyLinkText?: string;
  expireNote?:   string;
  ignoreNote?:   string;
  // layout
  brandName?:      string;
  footerCopyright?:string;
  footerIgnore?:   string;
  lang?:           string;
}

export default function VerifyEmail({
  name          = P.name,
  verifyLink    = P.verifyLink,
  expireHrs     = P.expireHrs,
  preview       = P.preview,
  heading       = P.heading,
  greeting      = P.greeting,
  body          = P.body,
  ctaLabel      = P.ctaLabel,
  copyLinkText  = P.copyLinkText,
  expireNote    = P.expireNote,
  ignoreNote    = P.ignoreNote,
  brandName     = LayoutTokens.brandName,
  footerCopyright = LayoutTokens.footerCopyright,
  footerIgnore  = LayoutTokens.footerIgnore,
  lang          = "en",
}: VerifyEmailProps) {
  return (
    <Layout
      preview={preview}
      brandName={brandName}
      footerCopyright={footerCopyright}
      footerIgnore={footerIgnore}
      lang={lang}
    >
      <Heading className="text-2xl font-bold text-gray-900 mt-0 mb-2">
        {heading}
      </Heading>

      <Text className="text-gray-600 text-base mt-0 mb-2">{greeting}</Text>
      <Text className="text-gray-600 text-base mt-0 mb-6">{body}</Text>

      <Section className="text-center mb-6">
        <Button
          href={verifyLink}
          className="bg-black text-white rounded-lg px-6 py-3 text-sm font-semibold no-underline"
        >
          {ctaLabel}
        </Button>
      </Section>

      <Text className="text-gray-500 text-sm mb-0">{copyLinkText}</Text>
      <Text className="text-blue-600 text-sm break-all mt-1">{verifyLink}</Text>

      <Hr className="border-gray-200 my-6" />

      <Text className="text-gray-400 text-xs mb-1">{expireNote}</Text>
      <Text className="text-gray-400 text-xs mb-0">{ignoreNote}</Text>
    </Layout>
  );
}

export function VerifyEmailText(p: VerifyEmailProps): string {
  return `${p.greeting ?? P.greeting}

${p.body ?? P.body}

${p.ctaLabel ?? P.ctaLabel}: ${p.verifyLink ?? P.verifyLink}

${p.expireNote ?? P.expireNote}
${p.ignoreNote ?? P.ignoreNote}

${p.brandName ?? LayoutTokens.brandName}`;
}
