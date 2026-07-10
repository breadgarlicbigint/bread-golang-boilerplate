/**
 * render.ts — Compiles React Email templates to HTML + plain-text files.
 * Output → pkg/email/dist/  (embedded in Go binary via //go:embed)
 *
 * Usage:
 *   npm run build
 *   EMAIL_DIST_DIR=/custom/path npm run build
 */
import React, { createElement } from "react";
import { render }               from "@react-email/render";
import { mkdirSync, writeFileSync } from "fs";
import { join }                  from "path";

// ── Template imports ──────────────────────────────────────────────────────────
import VerifyEmail,        { VerifyEmailText }       from "./emails/verify-email";
import ResetPassword,      { ResetPasswordText }     from "./emails/reset-password";
import Welcome,            { WelcomeText }           from "./emails/welcome";
import OtpCode,            { OtpCodeText }           from "./emails/otp-code";
import NotificationEmail,  { NotificationEmailText } from "./emails/notification";

// ── Template registry ─────────────────────────────────────────────────────────
interface TemplateEntry {
  component:    React.ComponentType<any>;
  textFn:       (props?: any) => string;
  defaultProps?: Record<string, string>;
}

const templates: Record<string, TemplateEntry> = {
  "verify-email":  { component: VerifyEmail,       textFn: VerifyEmailText       },
  "reset-password":{ component: ResetPassword,     textFn: ResetPasswordText     },
  "welcome":       { component: Welcome,           textFn: WelcomeText           },
  "otp-code":      { component: OtpCode,           textFn: OtpCodeText           },
  "notification":  { component: NotificationEmail, textFn: NotificationEmailText },
};

// ── Build ─────────────────────────────────────────────────────────────────────
// Wrapped in async IIFE to avoid top-level await (incompatible with CJS output).
(async () => {
  const distDir = process.env.EMAIL_DIST_DIR ?? join(__dirname, "../../pkg/email/dist");
  mkdirSync(distDir, { recursive: true });

  let ok = 0;
  let failed = 0;

  for (const [name, entry] of Object.entries(templates)) {
    try {
      // Render HTML via React Email
      const element = createElement(entry.component, entry.defaultProps ?? {});
      const html    = await render(element, { pretty: true });
      writeFileSync(join(distDir, `${name}.html`), html, "utf-8");

      // Render plain-text fallback
      const text = entry.textFn(entry.defaultProps ?? {});
      writeFileSync(join(distDir, `${name}.txt`), text, "utf-8");

      console.log(`  ✅  ${name}.html + ${name}.txt`);
      ok++;
    } catch (err) {
      console.error(`  ❌  ${name}: ${err}`);
      failed++;
    }
  }

  console.log(`\n▶ Email build complete — ${ok} rendered, ${failed} failed`);
  if (failed > 0) process.exit(1);
})();
