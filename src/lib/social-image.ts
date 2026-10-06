import { getImage } from "astro:assets";
import type { ImageMetadata } from "astro";

/** Keep the complete photograph and its original ratio for existing share compositions. */
export async function socialImage(image?: ImageMetadata) {
  if (!image) return undefined;
  const result = await getImage({
    src: image,
    // Sharp's existing service prevents enlargement of smaller source images.
    width: 1200,
    format: "jpg",
    quality: 80,
  });
  return result.src;
}
