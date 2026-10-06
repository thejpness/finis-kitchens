import { gsap } from "gsap";
import { ScrollTrigger } from "gsap/ScrollTrigger";
import { enhanceImages } from "./images";
import { motion } from "./settings";

/** Preserve the photographic chapters; only roomy desktops use the sticky canvas. */
export function enhanceProcessStory(section: HTMLElement, desktop: boolean, context: gsap.Context) {
  if (!desktop) return enhanceImages(section, "[data-story-photo]");
  const chapters = Array.from(section.querySelectorAll<HTMLElement>("[data-process-stage]"));
  const photos = chapters.map((chapter) => chapter.querySelector<HTMLElement>("[data-story-photo]")!);
  const layers = section.querySelector<HTMLElement>(".process-story__layers")!;
  let disposed = false;

  // Decode near the journey, not during the LCP request. Until every layer is
  // ready the static photographs stay beside their associated chapter.
  const observer = new IntersectionObserver(async ([entry]) => {
    if (!entry.isIntersecting) return;
    observer.disconnect();
    try {
      await Promise.all(photos.map(async (photo) => {
        const image = photo.querySelector("img")!;
        image.loading = "eager";
        await image.decode();
      }));
      if (disposed) return;
      // Async setup must join the owning media context so preference/resize
      // changes also revert these timelines, styles and ScrollTriggers.
      context.add(() => {
        photos.forEach((photo) => layers.append(photo));
        section.classList.add("is-enhanced");
        gsap.set(photos.slice(1), { autoAlpha: 0 });
        photos.slice(1).forEach((photo, i) => {
          gsap.timeline({
            defaults: { ease: "none" },
            scrollTrigger: {
              id: `motion-story-photo-${i + 1}`, trigger: chapters[i + 1],
              start: "top 82%", end: "top 38%", scrub: true,
            },
          })
            .fromTo(photo, { autoAlpha: 0 }, { autoAlpha: 1, duration: 1 }, 0)
            .fromTo(photo.querySelector("img"), { scale: motion.imageScale }, { scale: 1, duration: 1 }, 0)
            .fromTo(photo.querySelector(".process-story__veil"), { opacity: 0.5 }, { opacity: 0, duration: 1 }, 0);
        });
        ScrollTrigger.refresh();
      });
    } catch {
      // A failed decode keeps the complete editorial presentation and rail.
    }
  }, { rootMargin: "600px 0px" });
  observer.observe(section);

  return () => {
    disposed = true;
    observer.disconnect();
    section.classList.remove("is-enhanced");
    photos.forEach((photo, index) => chapters[index].prepend(photo));
  };
}
