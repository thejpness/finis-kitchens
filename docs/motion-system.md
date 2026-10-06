# v1.3.0 — Motion & interaction system

## Audit and scope

The clean checkout was updated from `origin/main` at `43a4a21`. Main uses static Astro 7,
optimised local assets, npm and a committed lockfile. It has no client router, shared
scroll-animation runtime, release tags or changelog. `package.json` has retained its
starter version; the application release is recorded in the README.

The existing `feature/visual-updates-v1.2` branch contains a separate, unmerged visual
storytelling implementation. This release is based on main, preserving its photography,
approved copy, semantic headings, metadata, structured data and routes.

| Area | Main's behaviour | v1.3 treatment |
| --- | --- | --- |
| Homepage hero | Static priority image and copy; Explore arrow on hover | Visible staged arrival; glazing clears with native scroll on every screen size |
| Homepage process | Passive vertical rail below 72rem; horizontal rail above | Line draws toward the reading position; one current stage, completed nodes and small copy sequencing |
| Full process | Five explanatory rows | Row rule fills, number emphasis and a short heading/intro resolve; detailed lists stay still |
| Project card | Hover lift, zoom on both picture and image, moving CTA | Single image transform; fine-pointer hover, keyboard feedback, image reveal on touch |
| Featured/case-study imagery | Static imagery or hover zoom | Shared reveal on viewport entry, including touch; hover zoom resumes after reveal |
| Hero/process/budget/project arrows | Predominantly hover feedback | Keyboard focus parity; Explore affordance stays visible on touch |
| Navigation/buttons/form/footer links | Colour, underline, focus and menu feedback | Native touch/keyboard affordances; buttons also provide pressed feedback |
| Book thumbnails/project carousel | Explicit smooth scrolling | Normal interaction retained; instant scrolling for reduced motion |
| Header | Solid state after 8px; padding shrinks on scroll; nav hidden without JS | Contrast state retained; stable height; readable navigation fallback |

The implementation plan was to add one scoped GSAP layer, retain CSS for utility and
stage feedback, enhance the existing rails rather than pinning them, and apply photography
reveals selectively. Body sections, budget figures, detailed process lists, portraits,
the story book and decorative imagery remain still.

## Architecture and tuning

`BaseLayout.astro` loads `scripts/motion/index.ts` only on pages with motion hooks. GSAP
3.15.0 and ScrollTrigger are pinned npm dependencies bundled into local Astro assets.
The entry registers ScrollTrigger once and uses [GSAP matchMedia contexts](https://gsap.com/docs/v3/GSAP/gsap.matchMedia/)
for reduced motion and the existing 72rem horizontal-process breakpoint. Context reversion
restores GSAP styles and triggers; cleanup removes custom image/history listeners and
process classes. Resizing across the breakpoint does not replay the arrival.

Responsibilities are separated into `hero.ts`, `process.ts`, `images.ts` and `settings.ts`.
CSS owns layout, colour, utility transitions and the short stage resolve. The existing
120/180ms utility tokens remain; narrative activation adds a 550ms token. GSAP's central
settings contain equivalent timings, ease, scale and glazing strength.

### Signature

The existing priority hero image, responsive sources and crop remain. An additional
160px WebP (2,190 bytes) provides a soft-focus glass layer. Its blur is static; scrolling
changes the layer's opacity from 0.88 to 0 and settles the sharp photograph from 1.04 to 1.
No full-resolution filter, backdrop blur, pinning or parallax is animated. The existing
dark overlay stays in place for text/header contrast.

Eyebrow, headline, supporting text and Explore resolve in order over about 1.3 seconds.
Copy never drops below 72% opacity. Early scrolling completes the arrival immediately.
Explore keeps its native anchor, URL, focus and scrolling; it only reconciles the arrival.
The reveal reverses on upward scrolling. Hash navigation and restored history skip arrival.

Selected homepage/project, case-study lead/gallery and About origin photographs reveal
the final 8% of their image mask while settling from 1.03 to 1. This takes 900ms once,
without moving the layout container. A slow image waits until loaded to play its reveal.
CSS hover transitions are suspended during the reveal and restored afterwards.
Photographs already revealed stay settled across responsive/preference changes.

### Narrative

The homepage rail uses one scrubbed trigger. On mobile/tablet the reading position is
65% down the viewport; node distances determine activation, including unequal stage heights.
On desktop the horizontal line progresses as the rail moves from 85% to 35% of the viewport.
Track coordinates are read at setup/refresh only, not on every scroll frame. Number,
heading and copy resolve in place; previously visited stages retain a completed state.

The full process page has five row triggers. Each draws its lower rule as the row enters
and is read. Number, heading and introduction receive a short activation; explanatory
lists are always fully readable. No sections are pinned and no extra scroll distance is added.

### Accessibility and security

Reduced motion creates no motion triggers, displays the clear hero, removes masks/scale
and preserves static process rules and nodes. Live preference changes revert enhancement.
CSS also suppresses the glass layer and masking if the preference changes before JS reacts.

Without JavaScript all imagery, copy, routes and navigation remain visible. Header links
are the default; the mobile dialog is enhanced after its script has the required elements.
Focus indicators and the existing menu focus trap/Escape behaviour remain. Scroll-linked
visual state is decorative, with no live-region announcements or changed reading order.

The existing CSP permits local bundled scripts and style updates. No policy, security
header, environment, Docker or enquiry-service configuration was changed. There is no
CDN script, external animation request, eval requirement or tracking dependency.

## Validation

Production-build browser checks use the exact CSP from `docker-compose.yml` and local
loopback servers. Compression for the performance comparison matches Traefik's existing
compression middleware. Temporary test harnesses and screenshots are outside the repository.

- `npm run build`: passes; ten static pages generated with optimised images.
- `git diff --check`: passes. No separate frontend lint/typecheck/test scripts exist.
- Both existing Go service suites (`go test ./...`): pass; no enquiry changes made.
- Chromium: 320×568, 375×667, 390×844, 430×932, 768×1024, 1024×768,
  1366×768, 1440×900 and 1920×1080. No horizontal overflow, missing hero image,
  invisible copy, console warnings/errors or unexpected motion requests.
- Hero opening, partial/complete reveal, upward reversal and Explore; all five current/
  completed stages and exact line/node alignment; process and case-study navigation: pass.
- JavaScript disabled at 320, 390, 768 and 1440px across home, process, projects,
  case study, About, services and contact: headings/navigation visible; no overflow.
- Reduced motion and live preference changes; keyboard skip link, visible focus, mobile
  menu focus trap, Escape, touch activation, deep links and back/forward restoration: pass.
- Repeated portrait/landscape/desktop changes and rapid scroll to bottom/top: pass.
- Unavailable motion bundle leaves readable hero/process content; delayed photography
  still reveals and restores its styles; reduced-motion story-book controls: pass.
- WebKit automation could not complete with the available cached browser runtime;
  Safari/physical iOS validation remains a manual review item.

Changed files: README and this document; `package.json` and `package-lock.json`;
`scripts/motion/{index,settings,hero,process,images}.ts`; `styles/{motion,tokens,components}.css`;
`BaseLayout.astro`; `Header.astro`, `ProjectCard.astro`, `ProjectRail.astro`,
`sections/{HomeHero,ProcessSteps,ProjectPreviewGrid,BudgetGuide}.astro`,
`story/StoryBookViewer.astro`; and the `about`, `process`, `projects` and `projects/[slug]` pages.

The motion chunk is approximately 117KB minified / 45KB gzip and is shared only by pages
using it. Three cold Chromium samples at 390×844, 4× CPU throttling, 80ms latency and
1.6Mbps download gave a median LCP of 744ms on main and 728ms with motion. Both had CLS
0.00056. Both produced zero scroll frames above 34ms in these samples. These are local
lab measurements, not production Core Web Vitals or a substitute for physical-device review.

`npm audit` reports the same eight findings as main (one moderate, six high, one critical)
in existing dependencies. Adding GSAP introduces no additional findings. Dependency
remediation remains separate from this motion release.

## Review before merge

Review the opening glass strength and first scroll on a physical iPhone/Android phone,
particularly browser chrome, safe areas and GPU performance. Check the desktop arrival
at normal reading speed, process pacing, and the restrained image reveal on a real connection.
The final visual pass retained still body copy, quiet explanatory rows and short utility
feedback; no autoplay, looping decoration, custom scroll physics or loading gate was added.

The release is intended for feature-branch review only; do not deploy or merge as part
of this work.
