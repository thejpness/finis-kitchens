// astro.config.mjs
import { defineConfig } from "astro/config";
import sitemap from "@astrojs/sitemap";

const site = process.env.PUBLIC_SITE_ORIGIN ?? "https://finiskitchens.co.uk";

export default defineConfig({
  site,
  integrations: [
    sitemap({
      filter: (page) => {
        return !page.includes("/admin") && !page.includes("/api");
      },
    }),
  ],
});
