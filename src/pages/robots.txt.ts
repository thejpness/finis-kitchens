import type { APIRoute } from "astro";

const fallbackSite = "https://finiskitchens.co.uk";

export const GET: APIRoute = ({ site }) => {
  if (import.meta.env.DEPLOYMENT_ENV === "staging") {
    return new Response("User-agent: *\nDisallow: /\n", {
      headers: { "Content-Type": "text/plain; charset=utf-8" },
    });
  }

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
