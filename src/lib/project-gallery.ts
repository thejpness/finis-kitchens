import type { ImageMetadata } from "astro";
import { getImage } from "astro:assets";

interface GalleryImage {
  image: ImageMetadata;
  alt: string;
  caption?: string;
}

interface ResolvedGalleryImage extends GalleryImage {
  aspectRatio: number;
}

// The container is 76rem wide, with the existing fluid 1–1.75rem side padding.
export const fullImageSizes = "(min-width: 79.5rem) 76rem, (min-width: 50rem) calc(100vw - 3.5rem), (min-width: 28.5715rem) 93vw, calc(100vw - 2rem)";

const ratio = (item: ResolvedGalleryImage) => item.aspectRatio;

/** Adjacent photographs share a row in their natural proportions. Panoramas stand alone. */
export async function galleryRows(images: GalleryImage[]) {
  // Read original dimensions from the public image API's attributes. Accessing
  // dimensions on the import itself also retains that source photograph in dist.
  const resolved = await Promise.all(images.map(async (item) => {
    const metadata = await getImage({ src: item.image, format: "jpg" });
    return {
      ...item,
      aspectRatio: Number(metadata.attributes.width) / Number(metadata.attributes.height),
    };
  }));
  const rows: ResolvedGalleryImage[][] = [];
  for (let index = 0; index < resolved.length;) {
    const first = resolved[index++];
    const next = resolved[index];
    if (ratio(first) < 1.8 && next && ratio(next) < 1.8) {
      rows.push([first, next]);
      index++;
    } else {
      rows.push([first]);
    }
  }
  return rows.map((items) => {
    const totalRatio = items.reduce((sum, item) => sum + ratio(item), 0);
    const portrait = items.length === 1 && ratio(items[0]) < 1;
    return {
      portrait,
      // Fractions below 1fr can leave part of a CSS-grid row unoccupied.
      columns: items.map((item) => `minmax(0, ${(100 * ratio(item) / totalRatio).toFixed(6)}fr)`).join(" "),
      items: items.map((item) => {
        const share = ratio(item) / totalRatio;
        // Row gap is clamp(1rem, 2.5vw, 2rem); pair widths follow their aspect ratios.
        const sizes = items.length === 2
          ? `(min-width: 79.5rem) ${(74 * share).toFixed(4)}rem, (min-width: 50rem) calc((97.5vw - 3.5rem) * ${share}), (min-width: 48rem) ${(90.5 * share).toFixed(4)}vw, ${fullImageSizes}`
          : portrait ? `(min-width: 48rem) 38rem, ${fullImageSizes}` : fullImageSizes;
        return { ...item, sizes };
      }),
    };
  });
}
