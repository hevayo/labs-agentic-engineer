# Ground-truth walkthrough — idea → build kicked off

Ticket [#504](https://github.com/wso2/labs-agentic-engineer/issues/504) · map
[#502](https://github.com/wso2/labs-agentic-engineer/issues/502)

**Walked** 2026-08-17, 11:30–11:41 UTC, on the instance rebuilt by
[#503](https://github.com/wso2/labs-agentic-engineer/issues/503) — commit `078cd1ca`,
console `http://localhost:8090`, org `default`, **0 projects** at the start.

**Idea used:** *"A lending library for a neighbourhood tool shed where members list
tools, borrow them, and get reminded when they are due back"* — deliberately not one of
the three example prompts, so the flow had to handle an unseen domain.

**Outcome: the journey completes.** Spec v1 published, build v1 started, tag `v1` cut on
`HevayoFactory/lending-library-neighbourhood`. Eleven minutes wall-clock, **5 agent
turns, 217s of agent time**, 12 user interactions.

> **Provenance caveat — read before using the "confusion" observations.** This
> walkthrough was driven by the agent, not by a human. Screens, states, waits, timings,
> dead ends and vocabulary are directly observed and are reliable. Anything below marked
> **[agent-read]** is the agent's inference about where a newcomer would lose the
> thread, and is *weaker evidence* than a human's own reaction. The ticket asks for the
> walker's words at the moment of confusion; those are missing and should be added by a
> human before #507 leans on them.

---

## Timeline

| Time (UTC) | Event | Duration |
|---|---|---|
| 11:30:30 | Project created, GitHub repo created | ~5s |
| 11:31:26 | Turn 1 — `/start` → 5 interview questions | **21.8s** |
| 11:34:31 | Turn 2 — answers → `prd.md` written | **30.3s** |
| 11:35:41 | Turn 3 — `/design` → full design + validation | **144.8s** |
| 11:39:55 | Turn 4 — resolve `transactional-email` dependency | 5.9s |
| 11:40:28 | Turn 5 — answer SendGrid → design.json repinned | 14.2s |
| ~11:41 | Build kicked off — spec v1 published, build v1 building | — |

---

## Step by step

### 0. Login — Thunder gate `01-landing.png`

Out of scope per #502, recorded only as the boundary. `admin`/`admin` on a
**Thunder-branded** page ("Flexible Identity Platform", "Centralizes identity
management for both on-prem and cloud environments"), URL carrying
`?showInsecureWarning=true`. Nothing on it says AEP.

### 1. Projects list — the journey's start `02-projects-list.png`

Empty state is **good**: "Projects / Everything AEP is building for you, one project per
app", then "No projects yet / Tell AEP what you want to build and it becomes your first
project" and a single primary **Create project**.

Sidebar at this level: Projects, Alerts, Settings. One obvious next action, no ambiguity.

### 2. Create — the idea `03-create-prompt.png`

"What do you want to build?" with a free-text box, placeholder *"e.g. A booking system
for a small hair salon…"*, **Start** disabled until text is entered, and three example
cards (Online store / Workout tracker / Invoicing tool).

Decided *for* the user: nothing. Clean.

### 3. Create — naming `04-create-confirm.png`

"Name your project", the typed idea quoted back verbatim, and two prefilled fields:

- **Project name** `lending-library-neighbourhood` — "Suggested from your prompt — change
  it if you like."
- **Repository name**, prefixed `github.com/HevayoFactory/` — "Holds the project's specs
  and source; the organization is fixed."

Decided *for* the user: the slug, the repo name, the GitHub org, and repo visibility
(private, from `.env` — never surfaced). Both name fields are editable and explain
themselves. This screen is the clearest in the whole flow.

### 4. Create → overview — first wait `05-creating-inflight.png`, `06-after-create.png`

~5s. **In-flight state shows the unchanged form** — no spinner captured on the create
button; the page simply sits, then navigates. On landing, the pipeline area renders as a
**blank grey slab** for several seconds before resolving (`06`), with "No activity yet"
and "No components yet" beneath it.

### 5. Project overview `07-overview-settled.png`

Resolves to: status chip **"Starting"**, repo link, and a three-card pipeline —
**Spec** (with a highlighted **Generate spec** CTA) → **Build** ("waiting on spec") →
**Deploy** ("nothing deployed"). Below: Recent activity, Components.

The pipeline is the best orientation device in the product: three stages, current one
outlined, each unreached stage saying what it waits for. This is ADR-0014's pattern.

Two observations:

- **The idea is nowhere on this page.** The user typed a sentence describing a tool
  library; the overview shows a slug and a Generate-spec button. The prompt is persisted
  to `specs/.agentic-engineer.toml` but never displayed. **[agent-read]** the one thing
  the user contributed is invisible at the moment they're asked to proceed.
- **The sidebar jumps from 3 items to 7** — Overview, Spec, Builds, Deployments,
  Validation, Issues, Settings — the instant a project exists. Five of the seven are
  empty and will stay empty for the next ten minutes. Nothing indicates which one is
  live now. **[agent-read]**
- Status chip says **"Starting"** with no explanation of what is starting or when it
  stops. It is still "Starting" while the user is expected to click Generate spec.

### 6. Generate spec — **the silent wait** `08-genspec-t2s.png`

Clicking **Generate spec** navigates to `/spec` and shows, for **~22 seconds**:

- REQUIREMENTS / DESIGNS / VALIDATION — all three "**No files yet**"
- centre panel — "**Select a file to view its content.**"
- agent rail — "**Hi! I'm your Agent.**" and three chips: *Draft the requirements for
  this project*, *Add acceptance criteria to the spec*, and **"Add a returns-policy
  section"**
- composer **disabled**, placeholder unchanged ("Ask the agent to edit the spec…"),
  Send disabled, Actions disabled, Generate design disabled

**There is no positive indication that anything is running.** The only signal is that
controls are greyed out. Three panels actively assert emptiness while a turn is
executing. This is #506's "fully silent wait", reproduced.

**The returns-policy chip is off-domain** — this is a tool-lending library. #483 decided
canned chips die on fresh projects; #506 found it unbuilt; here it is, offering an
e-commerce action on a neighbourhood tool shed. **[agent-read]** a newcomer reading
those three chips would reasonably conclude the platform has not understood the idea.

### 7. The interview `09-genspec-t2m.png`, `10-questions-full.png`

At ~22s the centre panel becomes **"Quick questions"** — *"Everyone on this project can
answer together — anyone can send the answers."* Five questions, each with 2–3 options,
one tagged **Recommended**, each option carrying a one-line consequence. Example:

> *Does the tool shed need a dedicated coordinator/admin role, or is lending fully
> peer-to-peer between members?*
> — **Peer-to-peer only** *(Recommended)*: Any member can list tools and any member can
> borrow directly from the owner; no oversight role.
> — **Coordinator role**: A neighbourhood coordinator/admin approves new tool listings,
> resolves disputes, and can suspend members who don't return items.

All five were domain-specific and genuinely load-bearing (admin role, owner approval,
reminder channel, overdue policy, single vs multi-shed). **This is the strongest part of
the product.** `Skip questions` sits beside a `Continue` that stays disabled until every
question is answered.

**The chat rail shows the user's message as the literal string `/start`.** The user never
typed it. **[agent-read]** a raw slash command attributed to "You" invites the reading
that there is a command language you are expected to know.

### 8. Answers → PRD `11-after-answers-t3s.png`, `12-prd-written.png`

On Continue the answers are echoed into the rail as a Q:A list and a "**Working…**" dot
appears — the undifferentiated pulsing dot. Left panel still "No files yet" throughout;
**no skeleton, no ghost entries**. 30.3s.

Result: `prd.md` under REQUIREMENTS with Problem Statement / Solution / Actors / User
Stories / Product Decisions / Out of Scope / Open Questions / Further Notes. Quality is
high and the non-recommended choices were honoured (email *and* in-app; restrict
borrowing). One deferred Open Question, correctly marked as not blocking design.

**Two defects at this moment:**

1. **DESIGNS and VALIDATION now read "*Being derived…*" while nothing is running.**
   Verified live: `specStatus: "draft"`, `select count(*) from agent_turns where
   status='running'` = **0**. The console's `deriving` branch matches `draft`
   (`SpecView.tsx:435-441`), so this text is permanent for every unpublished spec. The
   user is told two artifact groups are in progress, indefinitely, when nothing is.
   This is #506's dead-state finding reproduced end-to-end.
2. **The agent tells the user to run a slash command.** Its closing line: *"Next step:
   review `specs/requirements/prd.md`, then run `/design` when ready."* There is no CLI.
   The actual affordance is a **Generate design** button that appeared in the header.
   The agent speaks CLI, the console speaks buttons, and they never reconcile.

### 9. Generate design — the long wait `13-design-t4s.png`, `14-design-done.png`

**144.8s** — by far the longest wait. Here the product does better:

- centre panel shows "**Waiting for the agent to lay out the architecture…**"
- a "**Designing…**" chip appears
- the agent **narrates**: *"Now security.md, openapi.yaml, wireframes.dsl, and
  validation-criteria.json — all independent, in parallel."*

This is the `skills/architecture` narration — the only genuine narration in the journey,
exactly as #506 predicted from the code. The chat again shows `/design` as a message
from "You".

Result is substantial: Architecture (rendered cell diagram), `design.md`, `security.md`,
per-component `Design Overview` / `API Spec` / `Wireframe` for **tool-shed-api** and
**tool-shed-webapp**, plus `Validation Criteria`. The Generate-design button becomes
**Build** ("Commit your latest changes and start building").

### 10. The design gate — **there isn't one** `14-design-done.png`

Nothing asks the user to read, review, or approve the design. The button simply flips to
**Build**. `apps/console/PRD.md` documents step 3 of the loop as *"Design gate
(blocking, in the Console): a developer reviews and approves the derived design before
any coding agent starts."* **That gate does not exist**, matching #506's reading of
`SpecView.tsx:443`.

Worse, at this exact moment the agent has written in the rail:

> **Needs your input:** `transactional-email` dependency on `tool-shed-api` — no vendor
> is registered or implied by the requirements; candidates are SendGrid, Postmark, or
> Resend — tell me which to pin.

The agent is explicitly blocked. The **Build** button is enabled and is the brightest
element on screen. The blocking question is visible only as prose in a side rail that
can be collapsed. **[agent-read]** the obvious action and the correct action point in
opposite directions.

### 11. Build → dependency drawer `16`, `17`, `18`

Pressing **Build** shows "Checking…", then opens a right-hand drawer, **"Dependencies to
resolve"**:

- **transactional-email** — *"More than one candidate fits — resolve which one to use."*
  → **Resolve via chat**
- **tool-shed-db** — `postgres-cnpg`, resolved, config shown
- **tool-shed-auth** — `thunder-app`, resolved, used by both components
- footer: **Cancel** and **Continue** (disabled)

So a gate *does* exist — but it is a **dependency** gate, not a design review. It is
also the first time the blocking issue is presented as blocking, one click *after* the
user committed to building.

### 12. Resolve via chat — and the loop back `19`, `20`, `21`

"Resolve via chat" auto-sends *Let's resolve the "transactional-email" dependency on
"tool-shed-api".* and the agent returns one clean question with three options (SendGrid
*Recommended* / Postmark / Resend), each with a real trade-off. Answered SendGrid; 5.9s
+ 14.2s; `design.json` repinned, `candidates` removed.

**The drawer then closes and drops the user back on the spec view.** The build is not
resumed. The user must find and press **Build** again. Nothing says so.

### 13. Build, second attempt — the hard stop `22`, `23`, `24`

The drawer reopens with **sendgrid** now listed and an **empty `SENDGRID_API_KEY`
field**. Continue is disabled until it is filled.

The only per-dependency action (⋮) is **"Discuss in chat & modify"**. There is no skip,
no defer, no stub, no "fill this in later".

**To kick off a build you must produce a real third-party credential.** A developer
evaluating AEP for the first time is stopped here unless they own a SendGrid account —
caused by nothing they asked for: the reminder feature came from *their* idea, but the
vendor was introduced by the design step.

Entering the obviously fake `SG.placeholder-not-a-real-key` **enables Continue
immediately** — the value is not validated at all.

### 14. Build kicked off — destination `25`, `26`

Continue → ~18s → lands on the **project overview**:

- chip **"Building"**
- **Spec v1 — published** · **Build v1 — building** · **Deploy — nothing deployed**
- Recent activity: *"Admin published spec v1 and started build — just now"*
- rail confirms the SendGrid pin and "Turn committed"

Version numbers appear for the first time here, unexplained. The landing is otherwise
the clearest state transition in the flow: three stages, two with versions, one waiting.

---

## Cross-cutting findings

### The waits

| Wait | Duration | What the user sees |
|---|---|---|
| Project create | ~5s | Unchanged form, then a blank slab on the overview |
| **`/start` interview** | **21.8s** | **Nothing. Three "No files yet" panels + an off-domain greeting** |
| Answers → PRD | 30.3s | One "Working…" dot; file lists still say "No files yet" |
| **`/design`** | **144.8s** | "Waiting for the agent to lay out the architecture…", "Designing…" chip, **real narration** |
| Dependency turns | 6s / 14s | "Working…" dot |
| Build kickoff | ~18s | Drawer with both buttons disabled |

Two of six are effectively silent; the longest one is the best narrated. The narration
that exists comes from one skill (`architecture`), not from the platform.

### Vocabulary a newcomer meets, and where it is explained

| Term | First seen | Explained? |
|---|---|---|
| Project | Projects list | Yes — "one project per app" |
| Spec | Overview pipeline | No |
| Requirements / Designs / Validation | Spec file tree | No |
| PRD | `prd.md` heading | No — the acronym is never expanded |
| Design / Architecture / cell diagram | Design output | No |
| Component (`tool-shed-api`) | Design output | No |
| Dependency, "resolve", "pin", candidates | Build drawer | Partially |
| Version / v1 / "published" | After build starts | No |
| Build vs Deploy | Pipeline | Only as "waiting on…" / "nothing deployed" |
| `/start`, `/design` | Chat rail, agent prose | No — and no CLI exists |
| "Starting" (project status) | Overview | No |
| Milestone, task, run cycle | *never surfaced* | n/a |

### Defects reproduced live

1. **"Being derived…" is permanent** for DESIGNS and VALIDATION on any unpublished spec,
   including when no turn is running (`specStatus: draft`, 0 running turns).
2. **Off-domain canned chips** on a fresh project ("Add a returns-policy section").
3. **Raw slash commands (`/start`, `/design`) shown as the user's own messages**, and the
   agent instructing the user to "run `/design`" when no CLI exists.
4. **No design review or approval anywhere**, contradicting `apps/console/PRD.md`.
5. **Dependency values are not validated** — a placeholder string unblocks the build.
6. **Resolve-via-chat does not return you to the build** you were in the middle of.

### What is genuinely good, and should survive #507

- The **overview pipeline** (Spec → Build → Deploy, current stage outlined, unreached
  stages state what they wait for) — the ADR-0014 pattern the ledger flagged as the
  answer already in the repo.
- The **interview**: adaptive, domain-specific, Recommended defaults, per-option
  consequences, `Continue` gated on completeness.
- The **naming screen**: everything prefilled, everything editable, each field explains
  what it controls.
- The **projects-list empty state**: one action, plainly worded.
- The **design output**: architecture diagram, per-component API spec and wireframes,
  security doc, validation criteria — 145 seconds for a lot of substance.
- The **dependency drawer** as a concept: it shows what was auto-resolved, not only what
  is blocked.
