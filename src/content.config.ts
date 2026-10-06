import { defineCollection } from "astro:content";
import { z } from "astro/zod";
import { glob } from "astro/loaders";

const projects = defineCollection({
  loader: glob({
    pattern: "**/*.{md,mdx}",
    base: "./src/content/projects",
  }),
  schema: ({ image }) =>
    z.object({
      title: z.string(),
      description: z.string().optional(),
      location: z.string().optional(),
      style: z.string().optional(),
      projectType: z.string().optional(),
      completedAt: z.string().optional(),

      heroImage: image().optional(),
      heroAlt: z.string().trim().min(1).optional(),
      featureTeaser: z.string().trim().min(1).optional(),
      featureHeading: z.string().trim().min(1).optional(),
      featureParagraphs: z.array(z.string().trim().min(1)).optional(),
      featureLinkLabel: z.string().trim().min(1).optional(),
      introduction: z.object({
        title: z.string().trim().min(1),
        body: z.string().trim().min(1).optional(),
      }).optional(),
      galleryHeading: z.string().trim().min(1).optional(),
      gallery: z.array(z.object({
        image: image(),
        alt: z.string().trim().min(1),
        caption: z.string().trim().min(1).optional(),
      })).optional(),

      featured: z.boolean().default(false),
      order: z.number().optional(),
    }),
});

export const collections = { projects };
