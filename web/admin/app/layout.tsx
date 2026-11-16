import { AppToasts } from "@/components/providers/toast-provider";
import { StoreProvider } from "@/components/providers/store-provider";
import { ThemeSync } from "@/components/theme/theme-sync";
import { themeInitScript } from "@/lib/theme";
import type { Metadata } from "next";
import { Outfit, Roboto_Mono } from "next/font/google";
import Script from "next/script";
import "./globals.css";

const outfit = Outfit({
  variable: "--font-outfit",
  subsets: ["latin"],
  weight: ["300", "400", "500", "600"],
});

const robotoMono = Roboto_Mono({
  variable: "--font-roboto-mono",
  subsets: ["latin"],
  weight: ["400"],
});

export const metadata: Metadata = {
  title: "Obelisk Admin",
  description: "Obelisk middleware appliance control plane",
};

export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html lang="en" suppressHydrationWarning>
      <body
        className={`${outfit.variable} ${robotoMono.variable} font-sans antialiased`}
      >
        <Script
          id="obelisk-theme-init"
          strategy="beforeInteractive"
          dangerouslySetInnerHTML={{ __html: themeInitScript }}
        />
        <ThemeSync />
        <StoreProvider>
          {children}
          <AppToasts />
        </StoreProvider>
      </body>
    </html>
  );
}