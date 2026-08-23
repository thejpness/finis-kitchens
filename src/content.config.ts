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
      gallery: z.array(image()).optional(),

      featured: z.boolean().default(false),
      order: z.number().optional(),
    }),
});

export const collections = { projects };