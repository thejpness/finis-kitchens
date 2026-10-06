/** Seconds and GSAP eases. CSS utility transitions remain in tokens.css. */
export const motion = {
  signature: 0.9,
  stagger: 0.12,
  ease: "power2.out",
  heroScale: 1.04,
  glazingOpacity: 0.88,
  imageScale: 1.03,
  heroReveal: { desktop: 0.22, mobile: 0.2 },
  desktopHero: "(min-width: 54rem)",
  desktopStory: "(min-width: 68rem) and (min-height: 38rem)",
  desktopMontage: "(min-width: 48rem)",
} as const;
