import { gsap } from "gsap";
import { ScrollTrigger } from "gsap/ScrollTrigger";

export function enhanceProcessRail(rail: HTMLElement) {
  const stages = Array.from(rail.querySelectorAll<HTMLElement>("[data-process-stage]"));
  const nodes = Array.from(rail.querySelectorAll<HTMLElement>("[data-process-node]"));
  const line = rail.querySelector<HTMLElement>("[data-process-line]");
  const track = line?.parentElement;
  if (!line || !track || stages.length < 2 || nodes.length !== stages.length) return () => {};
  let thresholds: number[] = [];
  let current = -2;

  const measure = () => {
    const railRect = rail.getBoundingClientRect();
    const centres = nodes.map((node) => {
      const rect = node.getBoundingClientRect();
      return { x: rect.left - railRect.left + rect.width / 2, y: rect.top - railRect.top + rect.height / 2 };
    });
    const first = centres[0];
    const last = centres[centres.length - 1];
    // Geometry is read only at setup/refresh, never on each scroll update.
    gsap.set(track, {
      left: first.x, top: first.y, bottom: "auto",
      width: 1, height: last.y - first.y,
    });
    thresholds = centres.map((centre) => (centre.y - first.y) / (last.y - first.y));
  };
  const activate = (progress: number, started: boolean) => {
    const next = started ? Math.max(0, thresholds.findLastIndex((value) => value <= progress + 0.001)) : -1;
    if (next === current) return;
    current = next;
    stages.forEach((stage, index) => {
      stage.classList.toggle("is-current", index === current);
      stage.classList.toggle("is-complete", index < current);
      if (index <= current) stage.classList.add("is-seen");
    });
  };

  measure();
  rail.classList.add("motion-ready");
  gsap.fromTo(line, { scaleY: 0 }, {
    scaleY: 1, ease: "none",
    scrollTrigger: {
      id: "motion-process-rail", trigger: nodes[0],
      endTrigger: nodes[nodes.length - 1],
      start: "center 65%", end: "center 65%",
      scrub: true, invalidateOnRefresh: true,
      onRefreshInit: measure,
      onRefresh: (self) => activate(self.progress, self.scroll() >= self.start),
      onUpdate: (self) => activate(self.progress, self.scroll() >= self.start),
    },
  });
  return () => {
    rail.classList.remove("motion-ready");
    stages.forEach((stage) => stage.classList.remove("is-current", "is-complete", "is-seen"));
  };
}

export function enhanceProcessPage(root: HTMLElement) {
  const stages = root.querySelectorAll<HTMLElement>("[data-process-row]");
  stages.forEach((stage, index) => {
    const rule = stage.querySelector("[data-process-rule]");
    if (!rule) return;
    gsap.fromTo(rule, { scaleX: 0 }, {
      scaleX: 1, ease: "none",
      scrollTrigger: {
        id: `motion-process-row-${index}`, trigger: stage,
        start: "top 75%", end: "bottom 55%", scrub: true,
        onUpdate: (self) => {
          stage.classList.toggle("is-current", self.progress > 0 && self.progress < 1);
          stage.classList.toggle("is-complete", self.progress === 1);
          if (self.progress > 0) stage.classList.add("is-seen");
        },
      },
    });
  });
  return () => stages.forEach((stage) => stage.classList.remove("is-current", "is-complete", "is-seen"));
}
