import type { APIRoute } from "astro";

const fallbackSite = "https://www.finiskitchens.co.uk";

export const GET: APIRoute = ({ site }) => {
  const siteUrl = site ?? new URL(fallbackSite);
  const sitemapUrl = new URL("/sitemap-index.xml", siteUrl);

  const body = `User-agent: *
Allow: /

Sitemap: ${sitemapUrl.href}
`;

  return new Response(body, {
    headers: {
      "Content-Type": "text/plain; charset=utf-8",
    },
  });
};
