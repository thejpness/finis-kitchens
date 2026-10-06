import { gsap } from "gsap";

/** One desktop reel or two touch-sized bands, following ordinary vertical scroll. */
export function enhanceMontage(section: HTMLElement, desktop: boolean) {
  const viewport = section.querySelector<HTMLElement>(".kitchen-montage__viewport")!;
  const track = section.querySelector<HTMLElement>(".kitchen-montage__track")!;
  const bands = Array.from(section.querySelectorAll<HTMLElement>(".kitchen-montage__band"));
  section.classList.add("is-enhanced");
  const targets = desktop ? [track] : bands;
  targets.forEach((target, index) => {
    gsap.fromTo(target, { x: () => -viewport.clientWidth * (index ? 0.12 : 0.04) }, {
      x: () => -Math.max(0, target.scrollWidth - viewport.clientWidth + (desktop ? viewport.clientWidth * 0.04 : 24)),
      ease: "none",
      scrollTrigger: {
        id: `motion-montage-${index}`, trigger: target,
        endTrigger: desktop ? section : target,
        start: desktop ? "top 85%" : "top 70%",
        end: desktop ? "bottom top" : "top 12%", scrub: true, invalidateOnRefresh: true,
      },
    });
  });
  // Offscreen lateral photographs need to be ready before drifting into view.
  // Loading remains lazy until this section approaches the viewport.
  const observer = new IntersectionObserver(([entry]) => {
    if (!entry.isIntersecting) return;
    section.querySelectorAll<HTMLImageElement>("img").forEach((image) => { image.loading = "eager"; });
    observer.disconnect();
  }, { rootMargin: "600px 0px" });
  observer.observe(section);
  return () => {
    observer.disconnect();
    section.classList.remove("is-enhanced");
  };
}
