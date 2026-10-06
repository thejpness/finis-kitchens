import { gsap } from "gsap";
import { ScrollTrigger } from "gsap/ScrollTrigger";
import { enhanceHero } from "./hero";
import { enhanceProcessRail, enhanceProcessPage } from "./process";
import { enhanceImages } from "./images";
import { motion } from "./settings";
import { enhanceProcessStory } from "./story";
import { enhanceMontage } from "./montage";

gsap.registerPlugin(ScrollTrigger);

export function startMotion(root: HTMLElement) {
  const media = gsap.matchMedia();
  media.add({
    reduce: "(prefers-reduced-motion: reduce)",
    normal: "(prefers-reduced-motion: no-preference)",
    desktop: motion.desktopHero,
    story: motion.desktopStory,
    montage: motion.desktopMontage,
  }, (context) => {
    if (context.conditions?.reduce) return;
    const desktop = Boolean(context.conditions?.desktop);
    const cleanups: Array<() => void> = [];
    const hero = root.querySelector<HTMLElement>("[data-motion-hero]");
    if (hero) cleanups.push(enhanceHero(hero, desktop));
    const story = root.querySelector<HTMLElement>("[data-motion-story]");
    if (story) {
      cleanups.push(enhanceProcessRail(story.querySelector<HTMLElement>(".process-story__journey")!));
      cleanups.push(enhanceProcessStory(story, Boolean(context.conditions?.story), context));
    }
    const montage = root.querySelector<HTMLElement>("[data-motion-montage]");
    if (montage) cleanups.push(enhanceMontage(montage, Boolean(context.conditions?.montage)));
    const journey = root.querySelector<HTMLElement>("[data-motion-journey]");
    if (journey) cleanups.push(enhanceProcessPage(journey));
    cleanups.push(enhanceImages(root));
    return () => cleanups.forEach((cleanup) => cleanup());
  }, root);

  return () => media.revert();
}
