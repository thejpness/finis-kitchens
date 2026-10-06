import { gsap } from "gsap";
import { ScrollTrigger } from "gsap/ScrollTrigger";
import { enhanceHero } from "./hero";
import { enhanceProcessRail, enhanceProcessPage } from "./process";
import { enhanceImages } from "./images";
import { motion } from "./settings";

gsap.registerPlugin(ScrollTrigger);

export function startMotion(root: HTMLElement) {
  const media = gsap.matchMedia();
  media.add({
    reduce: "(prefers-reduced-motion: reduce)",
    normal: "(prefers-reduced-motion: no-preference)",
    desktop: motion.desktopRail,
  }, (context) => {
    if (context.conditions?.reduce) return;
    const desktop = Boolean(context.conditions?.desktop);
    const cleanups: Array<() => void> = [];
    const hero = root.querySelector<HTMLElement>("[data-motion-hero]");
    if (hero) cleanups.push(enhanceHero(hero, desktop));
    root.querySelectorAll<HTMLElement>("[data-motion-process]").forEach((rail) => {
      cleanups.push(enhanceProcessRail(rail, desktop));
    });
    const journey = root.querySelector<HTMLElement>("[data-motion-journey]");
    if (journey) cleanups.push(enhanceProcessPage(journey));
    cleanups.push(enhanceImages(root));
    return () => cleanups.forEach((cleanup) => cleanup());
  }, root);

  return () => media.revert();
}
