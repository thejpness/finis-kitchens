import { gsap, createScrollMotionOwner } from "./scroll-motion-runtime";

export async function initKitchenMontage(section: HTMLElement, signal: AbortSignal) {
  const images = Array.from(section.querySelectorAll<HTMLImageElement>("img"));
  await Promise.all(images.map(async (image) => { image.loading = "eager"; await image.decode(); }));
  if (signal.aborted || !section.isConnected) return () => {};
  const viewport = section.querySelector<HTMLElement>(".kitchen-montage__viewport")!;
  const track = section.querySelector<HTMLElement>(".kitchen-montage__track")!;
  const bands = Array.from(section.querySelectorAll<HTMLElement>(".kitchen-montage__band"));
  const restoreFocus = () => {
    if (window.matchMedia("(prefers-reduced-motion: no-preference)").matches) viewport.tabIndex = 0;
    else viewport.removeAttribute("tabindex");
  };
  const owner = createScrollMotionOwner();
  const media = gsap.matchMedia();
  try {
    media.add({ desktop: "(min-width: 48rem)", mobile: "(max-width: 47.99rem)", motion: "(prefers-reduced-motion: no-preference)" }, (context) => {
      if (!context.conditions?.motion) return;
      owner.setActive(true);
      section.classList.add("is-enhanced");
      viewport.removeAttribute("tabindex");
      const targets = context.conditions.desktop ? [track] : bands;
      targets.forEach((target, index) => {
        const travel = () => context.conditions?.desktop
          ? Math.min(target.scrollWidth * 0.34, Math.max(0, target.scrollWidth - innerWidth + innerWidth * 0.04))
          : Math.max(0, target.scrollWidth - innerWidth + 24);
        const drift = gsap.fromTo(target, { x: () => -innerWidth * (index ? 0.12 : 0.04) }, {
          x: () => -travel(), ease: "none",
          scrollTrigger: {
            trigger: context.conditions.desktop ? section : target,
            start: context.conditions.desktop ? "top bottom" : "top 70%",
            end: context.conditions.desktop ? "bottom top" : "top 12%",
            scrub: 0.55, invalidateOnRefresh: true,
          },
        });
        owner.add(drift.scrollTrigger!);
      });
      const relevance = new IntersectionObserver(([entry]) => owner.setActive(entry.isIntersecting), { rootMargin: "300px 0px" });
      relevance.observe(section);
      return () => { relevance.disconnect(); owner.clear(); section.classList.remove("is-enhanced"); restoreFocus(); };
    });
    if (window.matchMedia("(prefers-reduced-motion: reduce)").matches) owner.setActive(false);
  } catch (error) { media.revert(); owner.destroy(); section.classList.remove("is-enhanced"); restoreFocus(); throw error; }
  const dispose = () => { media.revert(); owner.destroy(); };
  signal.addEventListener("abort", dispose, { once: true });
  return () => { signal.removeEventListener("abort", dispose); dispose(); };
}
