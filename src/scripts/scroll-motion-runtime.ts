import { gsap } from "gsap";
import { ScrollTrigger } from "gsap/ScrollTrigger";

export { gsap, ScrollTrigger };
type Owner = { active: boolean; triggers: Set<ScrollTrigger> };
const owners = new Set<Owner>();
let registered = false;
let awake = false;

// Both photographic sections share one deferred engine. A section may suspend
// its own work without stopping another section, and the engine sleeps when idle.
export function createScrollMotionOwner() {
  if (!registered) {
    gsap.registerPlugin(ScrollTrigger);
    registered = true;
    awake = true;
  }
  const owner: Owner = { active: false, triggers: new Set() };
  owners.add(owner);
  const sleepIfIdle = () => {
    if (awake && !Array.from(owners).some((item) => item.active)) {
      ScrollTrigger.disable(false);
      awake = false;
    }
  };
  const setActive = (active: boolean) => {
    if (owner.active === active) { sleepIfIdle(); return; }
    owner.active = active;
    if (active) {
      if (!awake) {
        ScrollTrigger.enable();
        awake = true;
        owners.forEach((item) => {
          if (!item.active) item.triggers.forEach((trigger) => trigger.disable(false));
        });
      }
      owner.triggers.forEach((trigger) => { trigger.enable(false, true); trigger.update(); });
    } else {
      owner.triggers.forEach((trigger) => trigger.disable(false));
      sleepIfIdle();
    }
  };
  return {
    setActive,
    add(trigger: ScrollTrigger) {
      owner.triggers.add(trigger);
      if (!owner.active) trigger.disable(false);
    },
    clear() { setActive(false); owner.triggers.clear(); },
    destroy() { setActive(false); owners.delete(owner); owner.triggers.clear(); sleepIfIdle(); },
  };
}
