import * as React from "react";
import {
  Body, Container, Head, Html,
  Preview, Section, Text, Hr,
} from "@react-email/components";
import { Tailwind } from "@react-email/tailwind";

// Text tokens resolved by Go i18n before sending
export const LayoutTokens = {
  brandName:      "__BRAND_NAME__",
  footerCopyright:"__FOOTER_COPYRIGHT__",
  footerIgnore:   "__FOOTER_IGNORE__",
};

interface LayoutProps {
  preview:         string; // __PREVIEW__
  brandName?:      string;
  footerCopyright?:string;
  footerIgnore?:   string;
  lang?:           string; // "en" | "id" etc.
  children:        React.ReactNode;
}

export function Layout({
  preview,
  brandName       = LayoutTokens.brandName,
  footerCopyright = LayoutTokens.footerCopyright,
  footerIgnore    = LayoutTokens.footerIgnore,
  lang            = "en",
  children,
}: LayoutProps) {
  return (
    <Html lang={lang}>
      <Head />
      <Preview>{preview}</Preview>
      <Tailwind>
        <Body className="bg-gray-50 font-sans">
          <Container className="mx-auto py-10 px-4 max-w-[560px]">
            {/* Brand header */}
            <Section className="mb-8 text-center">
              <Text className="text-2xl font-bold text-gray-900 m-0">
                {brandName}
              </Text>
            </Section>

            {/* Card */}
            <Section className="bg-white rounded-xl shadow-sm border border-gray-100 px-10 py-8">
              {children}
            </Section>

            {/* Footer */}
            <Hr className="border-gray-200 my-6" />
            <Section className="text-center">
              <Text className="text-xs text-gray-400 m-0">{footerCopyright}</Text>
              <Text className="text-xs text-gray-400 mt-1 mb-0">{footerIgnore}</Text>
            </Section>
          </Container>
        </Body>
      </Tailwind>
    </Html>
  );
}
