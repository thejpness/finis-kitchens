import { gsap } from "gsap";
import { ScrollTrigger } from "gsap/ScrollTrigger";
import { motion } from "./settings";

export function enhanceHero(hero: HTMLElement, desktop: boolean) {
  const glazing = hero.querySelector<HTMLElement>("[data-hero-glazing]");
  const image = hero.querySelector(".home-hero__image");
  const copy = hero.querySelectorAll(".home-hero__eyebrow, .home-hero__title, .home-hero__supporting, .home-hero__scroll");
  const explore = hero.querySelector<HTMLAnchorElement>(".home-hero__scroll");
  if (!glazing || !image) return () => {};

  // Copy is always perceptible, including during arrival. A deep link or
  // restored history entry starts with finished copy rather than replaying it.
  const navigation = performance.getEntriesByType("navigation")[0] as PerformanceNavigationTiming | undefined;
  const arrival = gsap.timeline({ paused: true });
  if (!hero.dataset.motionArrived && window.scrollY < 40 && !location.hash && navigation?.type !== "back_forward") {
    arrival.fromTo(copy, { opacity: 0.72, y: 8 }, {
      opacity: 1, y: 0, duration: motion.signature, stagger: motion.stagger,
      ease: motion.ease, clearProps: "opacity,transform",
    }).play();
  }
  hero.dataset.motionArrived = "true";

  let arrivalFinished = false;
  const settleArrival = () => {
    if (arrivalFinished) return;
    arrival.progress(1).pause();
    arrivalFinished = true;
  };
  gsap.timeline({
    defaults: { ease: "none" },
    scrollTrigger: {
      id: "motion-hero", trigger: hero, start: 0,
      end: () => Math.min(hero.offsetHeight, window.innerHeight) * (desktop ? motion.heroReveal.desktop : motion.heroReveal.mobile),
      scrub: true, invalidateOnRefresh: true,
      onUpdate: (self) => { if (self.progress > 0.025) settleArrival(); },
    },
  })
    .fromTo(glazing, { opacity: motion.glazingOpacity }, { opacity: 0 }, 0)
    .fromTo(image, { scale: motion.heroScale }, { scale: 1 }, 0);

  // Let the browser own the anchor, focus, URL and scroll. The same scrubbed
  // reveal follows Explore's native movement, including an interrupted scroll.
  explore?.addEventListener("click", settleArrival);
  const reconcile = (event: PageTransitionEvent) => {
    if (event.persisted) { settleArrival(); ScrollTrigger.refresh(); }
  };
  window.addEventListener("pageshow", reconcile);
  return () => {
    explore?.removeEventListener("click", settleArrival);
    window.removeEventListener("pageshow", reconcile);
  };
}
