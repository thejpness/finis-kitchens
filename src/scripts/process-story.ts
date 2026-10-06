import { gsap, ScrollTrigger, createScrollMotionOwner } from "./scroll-motion-runtime";

const desktop = "(min-width: 68rem) and (min-height: 38rem) and (prefers-reduced-motion: no-preference)";
export async function initProcessStory(section: HTMLElement, signal: AbortSignal) {
  const chapters = Array.from(section.querySelectorAll<HTMLElement>(".process-story__chapter"));
  const photos = chapters.map((chapter) => chapter.querySelector<HTMLElement>(".process-story__photo")!);
  const layers = section.querySelector<HTMLElement>(".process-story__layers")!;
  const journey = section.querySelector<HTMLElement>(".process-story__journey")!;
  const thread = section.querySelector<HTMLElement>(".process-story__thread span")!;

  // Decode every layer before taking photographs out of the static reading order.
  // A failed image leaves the complete static presentation in place.
  await Promise.all(photos.map(async (photo) => {
    const image = photo.querySelector("img")!;
    image.loading = "eager";
    await image.decode();
  }));
  if (signal.aborted || !section.isConnected) return () => {};

  const owner = createScrollMotionOwner();
  const media = gsap.matchMedia();
  try {
    media.add(desktop, () => {
      owner.setActive(true);
      photos.forEach((photo) => layers.append(photo));
      section.classList.add("is-enhanced");
      gsap.set(photos.slice(1), { autoAlpha: 0 });
      const triggers: ScrollTrigger[] = [];

      // CSS provides all structural positioning. Each dissolve follows the actual
      // chapter position, so native scrolling and the ordered copy stay in sync.
      photos.forEach((photo, index) => {
        if (!index) return;
        const image = photo.querySelector("img")!;
        const veil = photo.querySelector(".process-story__veil")!;
        const transition = gsap.timeline({
          defaults: { ease: "none" },
          scrollTrigger: {
            trigger: chapters[index],
            start: "top 82%",
            end: "top 38%",
            scrub: 0.45,
          },
        });
        transition
          .fromTo(photo, { autoAlpha: 0 }, { autoAlpha: 1, duration: 1 }, 0)
          .fromTo(image, { scale: 1.035, xPercent: 0.6 }, { scale: 1, xPercent: 0, duration: 1.35 }, 0)
          .fromTo(veil, { opacity: 0.5 }, { opacity: 0.03, duration: 1 }, 0);
        // Keep the outgoing photograph underneath the dissolve. Each layer has
        // one opacity owner, including when the visitor reverses direction.
        triggers.push(transition.scrollTrigger!);
        owner.add(transition.scrollTrigger!);
      });

      chapters.forEach((chapter) => {
        const copy = chapter.querySelector(".process-story__copy")!;
        const body = chapter.querySelector(".process-story__body")!;
        const emphasis = gsap.timeline({
          defaults: { ease: "none" },
          scrollTrigger: { trigger: chapter, start: "top 85%", end: "bottom 15%", scrub: 0.35 },
        })
          .fromTo(copy, { opacity: 0.72 }, { opacity: 1, duration: 0.4 }, 0)
          .fromTo(body, { y: 8 }, { y: 0, duration: 0.4 }, 0)
          .to(copy, { opacity: 1, duration: 0.3 })
          .to(copy, { opacity: 0.72, duration: 0.3 });
        triggers.push(emphasis.scrollTrigger!);
        owner.add(emphasis.scrollTrigger!);
      });
      const progression = gsap.fromTo(thread, { scaleY: 0 }, {
        scaleY: 1, ease: "none",
        scrollTrigger: { trigger: journey, start: "top center", end: "bottom center", scrub: true },
      });
      triggers.push(progression.scrollTrigger!);
      owner.add(progression.scrollTrigger!);

      // Leave no process scroll handlers/scrubbing active away from the section.
      // Re-enabling the plugin remeasures once on re-entry, including any resize
      // that happened while its resize/scroll listeners were suspended.
      const relevance = new IntersectionObserver(([entry]) => {
        if (entry.isIntersecting) {
          owner.setActive(true);
          triggers.forEach((trigger) => trigger.update());
        } else owner.setActive(false);
      }, { rootMargin: "300px 0px" });
      relevance.observe(section);

      return () => {
        relevance.disconnect();
        owner.clear();
        section.classList.remove("is-enhanced");
        photos.forEach((photo, index) => chapters[index].prepend(photo));
      };
    });
    if (!window.matchMedia(desktop).matches) owner.setActive(false);
  } catch (error) {
    media.revert();
    owner.destroy();
    section.classList.remove("is-enhanced");
    photos.forEach((photo, index) => chapters[index].prepend(photo));
    throw error;
  }
  const dispose = () => { media.revert(); owner.destroy(); };
  signal.addEventListener("abort", dispose, { once: true });
  return () => {
    signal.removeEventListener("abort", dispose);
    dispose();
  };
}
