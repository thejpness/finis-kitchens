# v1.3.0 — Motion & interaction system

## Release scope and integration

This is one release combining the earlier visual-storytelling work with the responsive
motion system. The photographic process chapters, Kitchen Montage, associated assets,
approved copy and Sussex SEO refinements are part of v1.3.0.

Starting revisions (all share main as their common base):

| Branch | Revision |
| --- | --- |
| `main` | `43a4a2104c0f1846fa761f2b72c81659c6e6a71d` |
| `feature/visual-updates-v1.2` | `1b4c6839f649bfda23d7a076d57e420e6580254c` |
| `feat/v1.3-motion-system` | `091836f867a6363bd505ad5a820c4736482d8176` |

The clean release branch integrates the visual branch through a non-squashed merge.
The visual branch supplies the composition/content baseline; v1.3 supplies the motion
architecture. Main had no intervening remote changes at the initial fetch. Application
release naming follows the README; the npm starter version stays `0.0.1`.

Three files conflicted:

- `HomeHero.astro`: retain the blue kitchen, crop and typography; use the approved
  v1.3 glazing and staged arrival, preserve the earlier approved supporting copy.
  Remove the competing timed CSS atmosphere/image/copy animations.
- `ProcessSteps.astro`: preserve the photographic five-chapter journey and roomy
  desktop canvas. Add the v1.3 measured progress rail, current/completed nodes, stage
  numbers, short copy activation and touch photography reveals. The richer chapter
  composition supersedes the simpler horizontal five-card rail.
- `process.astro`: preserve the earlier labels, editorial typography, vertical spine,
  explanatory copy and dedicated OG image. Add v1.3 row rules/activation, restrained
  numbers and optimised photographs associated with the five stages.

Other overlaps were reviewed: homepage section order is hero, daily-life introduction,
budgets, process journey, project, montage, closing and footer. No sections are duplicated.
BaseLayout keeps the v1.3 motion loader and the visual branch's SEO changes. Featured/
case-study/About images keep the selected reveals alongside approved copy. Both branches
use the same GSAP version; the final lockfile retains the established dependency versions.

## Architecture

Static Astro 7 renders the complete semantic site. `BaseLayout.astro` dynamically loads
`scripts/motion/index.ts` only on pages with motion hooks. GSAP 3.15.0 and ScrollTrigger
are pinned npm dependencies bundled into local Astro assets. ScrollTrigger is registered
once; one scoped matchMedia context owns reduced motion and the responsive choreography.
No global enable/disable calls, custom scroll physics, pinning or client router are added.

| Module | Responsibility |
| --- | --- |
| `hero.ts` | Visible arrival sequence and reversible native-scroll glass clearance |
| `process.ts` | Measured chapter rail and calmer full-process row progression |
| `story.ts` | Deferred desktop photo dissolves, mobile/tablet photo reveals |
| `montage.ts` | Desktop photographic reel or two touch-sized drifting bands |
| `images.ts` | Selected, once-only image mask/scale reveal |
| `settings.ts` | Shared timing, easing, scales and responsive/reveal thresholds |

The earlier `scroll-motion-runtime.ts`, `process-story.ts` and `kitchen-montage.ts`
are removed. Their useful canvas/dissolve and reel behaviour is migrated to `story.ts`
and `montage.ts`. The previous owner registry, global ScrollTrigger suspension, per-section
media contexts, copy-opacity timelines and component bootstrap/history handlers are gone.
The montage's executable inline bootstrap is removed; CSP does not permit it.

CSS owns layout, colour, utility transitions and the short narrative copy resolve.
Existing 120/180ms utility tokens remain; narrative activation uses the 550ms token.
GSAP owns scroll-linked transforms/opacity and selected masks. Reversion restores styles,
triggers, photograph placement and process classes. Async photo setup explicitly joins
its owning media context, and disposed contexts ignore late image decoding.

## Visual behaviour

### Hero

The initial glazing strength remains 0.88. A static 160px WebP (2,190 bytes) provides
soft-focus diffusion above the sharp priority photograph. Only its opacity animates;
the full-resolution LCP photograph never receives a blur filter. Dark overlays preserve
copy/header contrast. The image settles from 1.04 to 1.

Clearance starts immediately with physical scroll progress and completes at 22% of the
smaller hero/viewport height on desktop (54rem and above), 20% below that breakpoint.
At 1440×900 this is 198px; at 390×844 it is about 169px. Half the glass has cleared by
10–11% travel, leaving most of the hero available for the clear blue-kitchen payoff.
No wheel counting or timed defrost is used.

Eyebrow, headline, supporting text and Explore resolve in order over about 1.3 seconds.
Copy remains at least 72% opaque. Early scrolling finishes arrival immediately. Explore
keeps the native anchor, URL, focus and scroll behaviour; upward scrolling reverses
clearance. Hash navigation and restored history skip the copy entrance.

### Process journeys

The homepage retains all five photographic chapters. A single vertical rail follows
node centres, with the reading position 65% down the viewport. Geometry is read at setup/
refresh only. The current stage gains emphasis; visited stages retain completion and
number/heading/copy receive a small stagger. Nodes align beside the copy on every layout.

At 68rem and a minimum 38rem height, the earlier sticky photographic canvas remains.
Its four reversible dissolves use opacity, a 3% scale settle and a soft veil. Photographs
move into the decorative canvas only after decoding near the section. The ordered text
stays in its semantic order. Narrower/shorter screens retain each photo beside its own
chapter, with the shared once-only reveal. Tablet uses a consistent photo/copy column so
the progression spine stays continuous. No chapter is pinned or given artificial scroll.

The full `/process` page retains the five detailed stages, labels and vertical spine.
Each row draws its lower rule and activates its number, heading and introduction.
Associated photography uses the selected reveal; explanatory lists stay still.

### Photography and montage

Selected project/case-study/About and narrow-screen process photographs reveal the final
8% of their mask while settling from 1.03 to 1 over 900ms. Containers never move. Slow
images wait until loaded; completed images stay settled after resizing/preference changes.
Hover transitions are suspended during the reveal and restored afterwards.

Kitchen Montage preserves all six photographs, varied sizes, staggered offsets and crops.
Desktop drifts one reel as its photographs enter; touch screens drift two bands through
ordinary vertical scrolling. The reel travels far enough to show its closing photograph.
Montage photographs do not also receive individual masks or zooms. Reduced motion and
JavaScript-disabled visitors receive the complete, intentionally composed static gallery.

Body sections, budget figures, detailed process lists, portraits and decorative imagery
remain still. Header contrast state is retained with stable height, visible focus/pressed
feedback and the improved mobile-menu/JavaScript-disabled navigation behaviour.

## Accessibility and security

Reduced motion creates no motion triggers, displays the clear hero, removes masks/scale
and restores static process photographs and montage. Live preference changes revert the
entire enhancement. Without JavaScript all imagery, headings, copy, links and navigation
remain visible; no animation class or hidden content is required by server-rendered HTML.
The mobile menu retains its keyboard focus trap, Escape and focus return. Decorative
progress state has no live-region announcements and does not change text reading order.

The production CSP, security headers, Docker bases, environments and enquiry architecture
are unchanged. Motion is locally bundled; there is no CDN animation script, eval,
tracking, new external resource or arbitrary inline script requirement. Existing approved
copy changes on Contact do not alter the browser payload or endpoint. Canonicals and
sitemap behaviour are unchanged. Each page has one LocalBusiness schema, with East and
West Sussex coverage, and its intended OG image exists.

## Validation

Production browser checks use local loopback servers with the exact CSP from
`docker-compose.yml` and compression matching Traefik. Diagnostic code, screenshots,
performance results and temporary baseline builds live outside the repository.

- `npm install`: succeeds; established dependency versions and GSAP pin retained.
- `npm run build`: passes; ten static pages and optimised local images generated.
- `git diff --check`: passes. No frontend lint/typecheck/test scripts are configured.
- Both existing Go service suites (`go test ./...`): pass; no service changes made.
- Chromium: 320×568, 375×667, 390×844, 430×932, 768×1024, 1024×768,
  1366×768, 1440×900 and 1920×1080. Rendered hero, chapters, montage, projects,
  full process, spacing/crops and closing composition reviewed. No horizontal overflow,
  missing imagery, invisible copy, console/CSP errors or unexpected motion requests.
- Opening glass, first-scroll clarity, complete reveal by 25%, reversal, Explore,
  stable header height, all five current/completed states and rail alignment: pass.
- Full-process/project/case-study links, imagery loading and once-only masks: pass.
- JavaScript disabled at 320, 390, 768 and 1440px across home, process, projects,
  case study, About, services and contact: navigation, copy and major photos visible.
- Reduced motion/live changes, keyboard skip link and visible focus, mobile focus trap,
  Escape/focus return, touch, deep links and back/forward restoration: pass.
- Repeated portrait/landscape/desktop changes, rapid bottom/top scrolling, unavailable
  motion bundle, delayed photography and reduced-motion story-book controls: pass.
- Runtime inspection: eight homepage desktop triggers after photo setup, up to ten on
  mobile, no duplicates or listener growth through repeated responsive changes. Live
  reduced motion removes all triggers and restores photographs to their chapters.
- Cached WebKit could not create a page: its protocol rejects `PushAPIEnabled` from
  the installed Playwright client. Safari/iOS is not claimed as validated.

The shared motion chunk is 119,294 bytes minified / 45,772 bytes gzip, versus
117,183 / 45,186 on the previous v1.3 branch: about 2.1KB raw / 0.6KB gzip growth.
New photography has responsive AVIF/WebP sources, explicit dimensions and lazy loading.
Desktop process layers decode together only near the journey; lateral montage images
are promoted near their section. The hero keeps its original priority sources.

Three cold Chromium pairs at 390×844, 4× CPU throttling, 80ms latency and 1.6Mbps download
measured median LCP of 612ms for the previous v1.3 build and 636ms for the integration.
Both measured CLS 0.00056 and zero initial-scroll frames above 34ms. A separate main
comparison measured 572ms / 624ms with the same CLS. These small local samples vary with
host load; they detect obvious regressions and do not represent production Core Web Vitals.
Three-second process and montage scroll samples at 390px and 1440px with 4× CPU
throttling recorded no long tasks, layout shifts or frames above 34ms (about 180
frames per section). Physical GPU/device performance still needs human review.

`npm audit` reports the same eight existing findings as the starting branches:
one moderate, six high and one critical, in Astro, devalue, http-cache-semantics,
js-yaml, sharp, smol-toml, source-map-js and undici. None are introduced by GSAP or this
integration. The existing Astro advisory and other dependency remediation require
separate follow-up; no automatic broad dependency upgrade was performed.

## Physical-device review

Review first-scroll glass clearance on iPhone/Safari and Android, including browser
chrome, safe areas and real touch/GPU performance. Confirm desktop wheel/trackpad pacing,
photographic chapter dissolves and the montage's crop/drift at normal reading speed.
Inspect selected image reveals over a real mobile connection. The release ends at a
validated Git main merge; staging and production deployment are separate operations.
