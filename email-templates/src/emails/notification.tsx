import * as React from "react";
import { Button, Heading, Text, Section } from "@react-email/components";
import { Layout, LayoutTokens } from "../components/layout";

const P = {
  name:     "__NAME__",
  title:    "__TITLE__",
  body:     "__BODY__",
  ctaLabel: "__CTA_LABEL__",
  ctaUrl:   "__CTA_URL__",
  preview:  "__PREVIEW__",
  greeting: "__GREETING__",
};

export interface NotificationProps {
  name?:     string;
  title?:    string;
  body?:     string;
  ctaLabel?: string;
  ctaUrl?:   string;
  preview?:  string;
  greeting?: string;
  brandName?: string;
  footerCopyright?: string;
  footerIgnore?: string;
  lang?:     string;
}

export default function NotificationEmail({
  name     = P.name,
  title    = P.title,
  body     = P.body,
  ctaLabel = P.ctaLabel,
  ctaUrl   = P.ctaUrl,
  preview  = P.preview,
  greeting = P.greeting,
  brandName = LayoutTokens.brandName,
  footerCopyright = LayoutTokens.footerCopyright,
  footerIgnore = LayoutTokens.footerIgnore,
  lang     = "en",
}: NotificationProps) {
  const showCTA = !!ctaUrl && ctaUrl !== P.ctaUrl;

  return (
    <Layout preview={preview} brandName={brandName}
      footerCopyright={footerCopyright} footerIgnore={footerIgnore} lang={lang}>
      <Heading className="text-2xl font-bold text-gray-900 mt-0 mb-2">{title}</Heading>
      <Text className="text-gray-600 text-base mt-0 mb-2">{greeting}</Text>
      <Text className="text-gray-600 text-base mt-0 mb-6">{body}</Text>

      {showCTA && (
        <Section className="text-center">
          <Button href={ctaUrl}
            className="bg-black text-white rounded-lg px-6 py-3 text-sm font-semibold no-underline">
            {ctaLabel}
          </Button>
        </Section>
      )}
    </Layout>
  );
}

export function NotificationEmailText(p: NotificationProps): string {
  return `${p.greeting ?? P.greeting}

${p.title ?? P.title}

${p.body ?? P.body}

${p.ctaUrl ? `${p.ctaLabel ?? P.ctaLabel}: ${p.ctaUrl}` : ""}

${p.brandName ?? LayoutTokens.brandName}`;
}
