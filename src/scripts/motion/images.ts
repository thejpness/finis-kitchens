import { gsap } from "gsap";
import { ScrollTrigger } from "gsap/ScrollTrigger";
import { motion } from "./settings";

export function enhanceImages(root: HTMLElement) {
  const cleanups: Array<() => void> = [];
  root.querySelectorAll<HTMLElement>("[data-motion-image]").forEach((frame) => {
    // A new responsive context should keep photography that has already
    // entered settled, including after a reduced-motion preference change.
    if (frame.dataset.motionRevealed) return;
    const image = frame.querySelector<HTMLImageElement>("img");
    if (!image) return;
    // A partial inset keeps the photograph perceptible. No layout changes and
    // no hidden content if enhancement fails or an image has yet to decode.
    let entered = false;
    // A slow image should get its reveal when it arrives, rather than spending
    // the animation on an empty container. CSS hover zoom resumes afterwards.
    gsap.set(image, { transition: "none" });
    const reveal = gsap.timeline({
      paused: true,
      onStart: () => { frame.dataset.motionRevealed = "true"; },
      defaults: { duration: motion.signature, ease: motion.ease },
    })
      .fromTo(frame, { clipPath: "inset(0% 0% 8% 0%)" }, { clipPath: "inset(0% 0% 0% 0%)", clearProps: "clipPath" }, 0)
      .fromTo(image, { scale: motion.imageScale }, { scale: 1, clearProps: "transform,transition" }, 0);
    const playWhenReady = () => { if (entered && image.complete && image.naturalWidth > 0) reveal.play(); };
    image.addEventListener("load", playWhenReady, { once: true });
    ScrollTrigger.create({ trigger: frame, start: "top 90%", once: true, onEnter: () => {
      entered = true;
      playWhenReady();
    } });
    cleanups.push(() => image.removeEventListener("load", playWhenReady));
  });
  return () => cleanups.forEach((cleanup) => cleanup());
}
