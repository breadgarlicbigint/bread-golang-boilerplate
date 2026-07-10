import * as React from "react";
import { Button, Heading, Text, Section, Hr } from "@react-email/components";
import { Layout, LayoutTokens } from "../components/layout";

const P = {
  name:           "__NAME__",
  resetLink:      "__RESET_LINK__",
  expireMins:     "__EXPIRE_MINS__",
  ipAddress:      "__IP_ADDRESS__",
  preview:        "__PREVIEW__",
  heading:        "__HEADING__",
  greeting:       "__GREETING__",
  body:           "__BODY__",
  ctaLabel:       "__CTA_LABEL__",
  copyLinkText:   "__COPY_LINK_TEXT__",
  securityTitle:  "__SECURITY_TITLE__",
  securityNote:   "__SECURITY_NOTE__",
  ignoreNote:     "__IGNORE_NOTE__",
};

export interface ResetPasswordProps {
  name?:          string;
  resetLink?:     string;
  expireMins?:    string;
  ipAddress?:     string;
  preview?:       string;
  heading?:       string;
  greeting?:      string;
  body?:          string;
  ctaLabel?:      string;
  copyLinkText?:  string;
  securityTitle?: string;
  securityNote?:  string;
  ignoreNote?:    string;
  brandName?:     string;
  footerCopyright?: string;
  footerIgnore?:  string;
  lang?:          string;
}

export default function ResetPassword({
  name          = P.name,
  resetLink     = P.resetLink,
  expireMins    = P.expireMins,
  ipAddress     = P.ipAddress,
  preview       = P.preview,
  heading       = P.heading,
  greeting      = P.greeting,
  body          = P.body,
  ctaLabel      = P.ctaLabel,
  copyLinkText  = P.copyLinkText,
  securityTitle = P.securityTitle,
  securityNote  = P.securityNote,
  ignoreNote    = P.ignoreNote,
  brandName     = LayoutTokens.brandName,
  footerCopyright = LayoutTokens.footerCopyright,
  footerIgnore  = LayoutTokens.footerIgnore,
  lang          = "en",
}: ResetPasswordProps) {
  return (
    <Layout preview={preview} brandName={brandName}
      footerCopyright={footerCopyright} footerIgnore={footerIgnore} lang={lang}>
      <Heading className="text-2xl font-bold text-gray-900 mt-0 mb-2">{heading}</Heading>
      <Text className="text-gray-600 text-base mt-0 mb-2">{greeting}</Text>
      <Text className="text-gray-600 text-base mt-0 mb-6">{body}</Text>

      <Section className="text-center mb-6">
        <Button href={resetLink}
          className="bg-black text-white rounded-lg px-6 py-3 text-sm font-semibold no-underline">
          {ctaLabel}
        </Button>
      </Section>

      <Text className="text-gray-500 text-sm mb-0">{copyLinkText}</Text>
      <Text className="text-blue-600 text-sm break-all mt-1 mb-6">{resetLink}</Text>

      <Hr className="border-gray-200 my-4" />

      <Section className="bg-amber-50 border border-amber-200 rounded-lg p-4 mb-4">
        <Text className="text-amber-800 text-sm font-semibold m-0">{securityTitle}</Text>
        <Text className="text-amber-700 text-sm mt-1 mb-0">{securityNote}</Text>
      </Section>

      <Text className="text-gray-400 text-xs mb-0">{ignoreNote}</Text>
    </Layout>
  );
}

export function ResetPasswordText(p: ResetPasswordProps): string {
  return `${p.greeting ?? P.greeting}

${p.body ?? P.body}

${p.ctaLabel ?? P.ctaLabel}: ${p.resetLink ?? P.resetLink}

${p.securityTitle ?? P.securityTitle}
${p.securityNote ?? P.securityNote}

${p.ignoreNote ?? P.ignoreNote}

${p.brandName ?? LayoutTokens.brandName}`;
}
