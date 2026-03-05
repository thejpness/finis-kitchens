// src/content/config.ts
import { defineCollection, z } from "astro:content";

const projects = defineCollection({
  type: "content",
  schema: ({ image }) =>
    z.object({
      title: z.string(),
      location: z.string().optional(),
      style: z.string().optional(),
      completedAt: z.string().optional(),

      // ✅ ImageMetadata, usable directly in <Picture src={...} />
      heroImage: image(),

      // ✅ future-proof: gallery can be real images too
      gallery: z.array(image()).optional(),

      featured: z.boolean().default(false),
      order: z.number().optional(),
    }),
});

export const collections = { projects };