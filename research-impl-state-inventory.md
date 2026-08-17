# Implementation state inventory — what of the prior plan is actually in `main`

Research findings for [#506](https://github.com/wso2/labs-agentic-engineer/issues/506), parent map [#502](https://github.com/wso2/labs-agentic-engineer/issues/502).

Scope: the journey **projects list → create → project overview → spec writing → design derivation → design gate/approval → build kicked off**, across three layers (console / backend / agent), plus two cross-cutting sections (drift; specified-but-never-built) and a wait-state audit.

All paths are absolute-from-repo-root. Every claim carries a `file:line`. Where something does not exist, it says **not found** and names what was searched.

---

## Stage 0 — Projects list

### Console

- Route: `apps/console/src/routes/index.tsx` (24 lines) → `apps/console/src/features/projects/components/ProjectsList.tsx`.
- Header: `title="Projects"`, `subtitle="Everything AEP is building for you, one project per app."` — `ProjectsList.tsx:189-190`.
- **Empty state exists**: `ProjectsList.tsx:214-229` — `EmptyState` with `title="No projects yet"`, `description="Tell AEP what you want to build and it becomes your first project."`, action `Create project`. Gated on `isTrueEmpty` (`ProjectsList.tsx:182`), which distinguishes "no projects at all" from "search matched nothing"; the page-level `Create project` action is suppressed in the true-empty case so the centred button is the only one (`ProjectsList.tsx:191-200`).
- **Loading state**: bare `CircularProgress aria-label="Loading projects"` — `ProjectsList.tsx:202-205`. No skeleton cards.
- **Error state**: `Alert severity="error"` + `Retry` — `ProjectsList.tsx:206-214`.
- **Search-empty state**: `No projects match "…"` — `ProjectsList.tsx:241-243`.
- Per-card state chip comes from `projectChip(status)` — `apps/console/src/features/projects/lib/projectChip.ts:41-54`. Values: `No repository` / `Preparing repository` / `Repository error` (from `status.phase`), then `deliveryChip` (`Build failed`, `Building`, `Deploy failed`, `Deploying`, `Active`, `Built` — `projectChip.ts:59-74`), falling back to `specChip`.
- **The list has no notion of "an agent is working on this project's spec."** `projectChip.ts:34-40` explains why: `status.phase`'s terminal rung is `tasks` and the server emits nothing past it, so the chip deliberately reads build/deploy aggregates instead. There is no spec-in-progress rung anywhere in the chip.

### Backend

- `GET /projects` list + `GET /projects/{name}/status` (the polled aggregate). Status polling interval: `OVERVIEW_POLL_MS = 10_000` (`apps/console/src/features/projects/api/queries.ts:82`), and for the project status query `STATUS_ACTIVE_POLL_MS = 5_000` / `STATUS_IDLE_POLL_MS = 30_000` (`queries.ts:87-88`, applied at `queries.ts:139-140` and `queries.ts:184-185`).

### Agent

- No agent involvement at this stage.

---

## Stage 1 — Create project

### Console

- Route: `apps/console/src/routes/projects.new.tsx` → `apps/console/src/features/projects/components/ProjectCreate.tsx` (263 lines).
- **Two-step wizard, in-component state** (`step === "prompt"` → `"confirm"`):
  1. Prompt step: `h4` "What do you want to build?", body "Describe it in your own words — AEP turns your requirement into a project and starts deriving its design." (`ProjectCreate.tsx:151-157`); a multiline `TextField` with placeholder "e.g. A booking system for a small hair salon with staff calendars and SMS reminders" (`ProjectCreate.tsx:161`); a `Start` button; and three `EXAMPLE_PROMPTS` cards (`ProjectCreate.tsx:178-184`).
  2. Confirm step: "Name your project", the prompt echoed in quotes, project-name + repo-name fields with validation `"Lowercase letters, digits, and dashes; must start with a letter."` (`ProjectCreate.tsx:124`).
- On success it navigates to the **project overview**, not to the spec view — `ProjectCreate.tsx:132-140`.
- **The copy on the create screen promises something the platform does not then do.** "AEP turns your requirement into a project **and starts deriving its design**" (`ProjectCreate.tsx:155-156`) — nothing is derived on create. See Drift D1.
- **No client-side copy of the prompt is kept**; the comment at `ProjectCreate.tsx:130-134` records the decision: the BE persists it into `specs/.agentic-engineer.toml` and `/start` reads it back.

### Backend

- `CreateProject` writes the descriptor holding the idea — `services/aep-api/internal/projects/project_service.go:245-257`. The inline comment at `:245-253` states this is "the ONLY durable copy of the idea the user typed — it is what the /start flow reads back", "Written even with an empty prompt", and that on failure "/start degrades by asking for the idea instead".
- **`CreateProject` does NOT kick off any agent turn.** Searched the handler body for `genai`, `Turn`, `TurnSpec`, `start`: only the descriptor write appears. The only construction sites of `agentsvc.TurnSpec` in non-test code are `services/aep-api/internal/spec/start_command.go:72-87` (reached from `services/aep-api/internal/spec/genai_service.go:333`, i.e. an *inbound instruction*) and `services/aep-api/internal/delivery/task/plan.go:245` (task planning). **No server-initiated `/start` at create: not found.**

### Agent

- Not invoked. The idea sits in the descriptor until the console later sends `/start`.

---

## Stage 2 — Project overview

### Console

- Route: `apps/console/src/routes/projects.$projectName.index.tsx` (29 lines) → `apps/console/src/features/projects/components/ProjectOverview.tsx` (181 lines).
- Layout: `PageHeader` with avatar, display name, one `StatusChip` from `projectChip` and the GitHub repo link (`ProjectOverview.tsx:91-131`); then `OverviewPipeline`; then a two-column body — `RecentActivity` beside `Components` (`ProjectOverview.tsx:145-176`).
- **Loading state**: `Skeleton variant="rounded" height={96}` for the pipeline (`ProjectOverview.tsx:143`) and `height={120}` for components (`ProjectOverview.tsx:167`).
- **Error state**: `SectionError` with Retry, per section (`ProjectOverview.tsx:42-57`, used at `:139` and `:157`).
- **No empty state for a brand-new project**: there is no "you have no spec yet, here is what happens next" panel. What a fresh project shows is the pipeline's three cards and two empty columns.

#### The pipeline — `apps/console/src/features/projects/components/OverviewPipeline.tsx`

Three cards, Spec → Build → Deploy, joined by chevrons (`OverviewPipeline.tsx:220-259`).

- Spec stage view derives purely from `status.spec.{exists,version,dirty}` — `apps/console/src/features/projects/lib/pipeline.ts:52-58`. Its own comment at `pipeline.ts:23-25`: *"the spec stage in particular **has no stored status**: exists/version/dirty decide everything."*
  - `!exists` → `cta: true`, empty line → the CTA card.
  - `exists && !version` → `"draft · not published"`.
  - `dirty` → `"v1+" / "draft changes"`.
  - else → `"published"`.
- Build stage (`pipeline.ts:65-77`): `running`→`"building"`, `failed`→`"build failed"`, `succeeded`→`"built"`, default→ ghost `"waiting on spec"`.
- Deploy stage (`pipeline.ts:79-111`) with validation folded into the line.
- `SpecActionStage` (`OverviewPipeline.tsx:133-199`) renders **one button with two labels**: `"Continue spec"` when `engaged`, `"Generate spec"` otherwise (`OverviewPipeline.tsx:194`). Only the non-engaged path attaches `search: { generate: "requirements" }` (`OverviewPipeline.tsx:190`).
- `engaged` comes from `useAgentEngaged(orgHandle, projectName)` — `OverviewPipeline.tsx:218`.

**What state the overview can show:** spec exists / draft / dirty / published; build running/failed/succeeded; deploy + validation. **What it cannot show:** that an agent turn is currently running, that questions are waiting, how many, or how far along spec generation is. There is no "interviewing" state, no question count, and no progress signal of any kind on the overview. The nearest thing is the button label flipping to "Continue spec" — and that flip is computed from **browser-local chat state**, not from the server (see Drift D2).

### Backend

- `ProjectStatus` (generated: `apps/console/src/generated/aep-api.d.ts:1780` for `specStatus`; Go: `services/aep-api/internal/gen/models_gen.go:1235-1236`) exposes: `phase`, `repoStatus`, `repoErrorMessage`, `repoUrl`, `hasSpec`, `hasDesign`, `hasTasks`, `specStatus`, `designStatus`, and the three stage aggregates `spec` / `build` / `deploy`.
- `phase` enum, per the contract (`packages/contracts/api/v1/openapi.yaml:4330-4335`): `no-repo`, `repo-cloning`, `repo-error`, `prompt` (no spec), `spec` (spec, no design), `tasks` (both). The doc itself notes **`"tasks" is terminal`** and that delivery state lives in the build/deploy aggregates.
- `specStatus` enum, per the contract (`packages/contracts/api/v1/openapi.yaml:4346-4348`): **`"", draft, approved`** — three values, nothing else. Set in exactly one place: `services/aep-api/internal/projects/status_stages.go:365-371` — `approved` when a `v<N>` spec tag exists, `draft` when a spec exists unversioned, otherwise left `""`.
- `designStatus` is set to `approved` only when a legacy `v<N>-<M>` design tag exists — `status_stages.go:372-374`.
- **There is no "agent turn running" field on `ProjectStatus`. Not found** — searched `models_gen.go` / `openapi.yaml` for a turn/activity/interview field on the status aggregate.

### Agent

- Not invoked at this stage.

---

## Stage 3 — Spec writing (`/start` → PRD)

### Console

- Route: `apps/console/src/routes/projects.$projectName_.spec.tsx` (48 lines). Trailing-underscore route = full-screen workspace without the project header (`:22-24`). Search params validated at `:33-42`: `generate=requirements|design` and `connections=open`.
- Feature: `apps/console/src/features/spec/components/SpecView.tsx` (1108 lines) + `apps/console/src/features/agent-chat/*`.

#### How `/start` actually fires — still client-side

1. `SpecActionStage` navigates to the spec route with `?generate=requirements` (`OverviewPipeline.tsx:186-192`).
2. `AppLayout` sees the search param and opens the chat panel: `if (generate && projectName) setChatOpen(true)` — `apps/console/src/layouts/AppLayout.tsx:125`; it forwards the signal as `autoGenerate` (`AppLayout.tsx:363`).
3. `AgentChatPanel`'s one-shot effect sends the command: `sendRef.current(instructionFor(autoGenerate))` — `apps/console/src/features/agent-chat/components/AgentChatPanel.tsx:314`, where `instructionFor` returns the literal `START_COMMAND` / `DESIGN_COMMAND` (`AgentChatPanel.tsx:107-109`, importing from `@aep/contracts/commands` at `:65-68`).
4. It is guarded by `useAgentEngaged` held in a ref (`AgentChatPanel.tsx:288-315`) and held until `conversationReady` (`AgentChatPanel.tsx:302`).

So: **the `/start` injection is still client-side.** What moved server-side (relative to the state #485 described) is only the *idea*: the console no longer stores the prompt in `localStorage`; the server reads it from the descriptor and enriches `/start` (`AgentChatPanel.tsx:101-106`; generated client doc string at `apps/console/src/generated/aep-api.d.ts:2287`).

#### What the user sees during spec generation

- The `/start` string is added to the chat log as **the user's own message and rendered verbatim** — `apps/console/src/features/agent-chat/runTurn.ts:85`, rendered by `UserBlock` at `apps/console/src/features/agent-chat/components/MessageList.tsx:95-105` (`{message.content}` in a bubble). A newcomer who clicked a button labelled "Generate spec" sees a chat bubble from themselves reading `/start`.
- Because the feed is then non-empty, the canned empty state does **not** show; instead the tail placeholder shows: `showWorkingTail` (`AgentChatPanel.tsx:193-194`) → `WorkingIndicator label="Working…"` (`apps/console/src/features/agent-chat/components/WorkingIndicator.tsx:24-47`) — a pulsing dot and the word "Working…". That is the entire progress signal.
- Once the turn produces a block, `TurnFooter` renders `WorkingIndicator label="Working…"` while `status === "running"` (`apps/console/src/features/agent-chat/components/TurnBlock.tsx:50-52`), `"Turn failed"` on failure (`TurnBlock.tsx:53-66`), `"Turn committed" · "Open spec"` on success (`TurnBlock.tsx:68-90`).
- Tool activity renders on a vertical "activity rail" — `TurnBlock.tsx:35-41` + `apps/console/src/features/agent-chat/components/ActivityStep.tsx`. Same-file tool runs are collapsed by default (`AgentChatPanel.tsx:197-205`).
- Assistant prose renders as markdown (`TurnBlock.tsx:187-192`), skipping empty messages.
- The composer is disabled while the turn runs: `inputDisabled = isSending || Boolean(teammateRunning) || !conversationReady` (`AgentChatPanel.tsx:180`), with hint strings at `AgentChatPanel.tsx:184-190`.

#### The canned empty state — still present

`AgentChatPanel.tsx:477-507` renders, when `feed.length === 0`:
- a Sparkles avatar,
- `"Hi! I'm your Agent."` (`AgentChatPanel.tsx:485`),
- `"Ask me to edit this project's spec — I join the shared workspace and you can watch the files change live."` (`:486-489`),
- and three chips from `SUGGESTIONS` (`AgentChatPanel.tsx:80-84`): `"Draft the requirements for this project"`, `"Add acceptance criteria to the spec"`, `"Add a returns-policy section"`.

These are the off-domain canned chips #485 scoped for removal. They are still there, verbatim.

#### Questions

- Question tool calls are folded into `role: "question"` messages and rendered in chat only as a **pointer**, never as an answerable form: `QuestionsPointer` — `TurnBlock.tsx:94-130` — `"The agent has N questions"` / `"Answer them →"`. The comment at `TurnBlock.tsx:193-196` records the rule: every question is answered on the spec body's shared form.
- The panel **auto-navigates** to the spec view the moment a question arrives, once per `toolCallId` — `AgentChatPanel.tsx:225-238`.
- The form itself takes over the whole spec body: `SpecQuestionForm` — `apps/console/src/features/spec/components/SpecQuestionForm.tsx`, mounted at `SpecView.tsx:886-895` ahead of the file browser, gated on `roomQuestion && roomDoc`. Collaborative: every room participant sees and co-authors, only the asker can submit (`SpecView.tsx:882-885` comment).
- `awaiting` state and hint: `AgentChatPanel.tsx:166` and `:188-189` — `"Answer the agent's questions on the spec view, or type a reply…"`.

**The question form is by some distance the most complete surface in the journey** — `SpecQuestionForm.tsx`:
- Header: Sparkles icon, `"Quick questions"`, sub-line `"Everyone on this project can answer together — anyone can send the answers."` (`:280-291`).
- Per question (`QuestionBlock`, `:120-212`): question text, optional detail, `"Pick as many as apply"` caption for multi-select, `OptionCard`s carrying a `"Recommended"` chip and an always-visible description, plus a free-text field (`"Other — your own answer or extra context"`, or `"Your answer"` when the question is free-text-only) with helper `"This choice needs a typed answer — describe it here to continue."`.
- **A streaming state exists here and nowhere else in the journey**: `CircularProgress size={14}` + `"The agent is still writing questions — you can start answering."` (`:304-310`), with submit/skip disabled while streaming.
- Sticky footer (`:315-334`): `"Skip questions"` (`:330`) — which sends the literal `"Skip these questions — stop interviewing and proceed with your best assumptions, stating them."` (`:267`) — and `"Continue"` (`:333`), disabled until `allAnswered && !streaming`.

Note the asymmetry: the **skip valve has a button** (`SpecQuestionForm.tsx:330`, string at `:267`) and it is the single most consequential control in the flow — it is what makes the agent write the PRD from its own recommended answers tagged `*assumed*` (`skills/start/SKILL.md:63-67`). Nothing in the console explains that consequence at the point of the click, and nothing afterwards surfaces the resulting assumptions (see #487/#490 under section B).

#### Spec view file browser

- `SpecFileList` — `apps/console/src/features/spec/components/SpecFileList.tsx`. Empty groups render `emptyNote` (`SpecFileList.tsx:110-114`): `"Derivation failed"` when `failed`, `"Being derived…"` when `deriving`, else `"No files yet"`, rendered at `:178` and `:289`.
- Body fallback when nothing is selected — `SpecView.tsx:1069-1074`: `"The agents are shaping the spec — files appear here as they land."` when `deriving`, else `"Select a file to view its content."`.
- `deriving` / `failed` derive from `specStatus`: `SpecView.tsx:436-441`.
- **There are no ghost/placeholder nav entries for files not yet written.** Searched `SpecFileList.tsx` for `ghost`, `placeholder`, `Skeleton`: **not found**. The nav shows only files that exist; the journey ahead is invisible.

#### Loading / error / waiting states in the spec view

| State | Where | What |
|---|---|---|
| Spec listing loading | `SpecView.tsx:864-874` | bare `CircularProgress aria-label="Loading spec"` |
| Spec listing error | `SpecView.tsx:875-885` | `Alert` "Failed to load the spec" + Retry |
| File content loading | `SpecView.tsx:1014-1021`, `1070-1080` | bare `CircularProgress` |
| File content error | `SpecView.tsx:1001-1010`, `1041-1055` | `Alert` + Retry |
| Agent writing this file | `SpecView.tsx:985-1000` | `"Waiting for the agent to write {filename}…"` |
| Collab offline | `SpecView.tsx:668-676` | `StatusChip "solo session"` + tooltip "Collaboration server unreachable — editing solo; edits aren't shared or saved." |
| Turn failed banner | `SpecView.tsx:769-775` | `Alert` "The agent's last turn failed" — **dead code**, see Drift D3 |
| Flush error | `SpecView.tsx:853-861` | `Alert` from `collab.flushError` |

### Backend

- The console sends the raw line; the BFF classifies it. `services/aep-api/internal/spec/start_command.go:52` (`startCommand = "/start"`), `:65` (`slashCommandPattern`), `:72-88` (`turnSpecFor` → `TurnKindChat` / `TurnKindStart` / `TurnKindFlow`), reached from `services/aep-api/internal/spec/genai_service.go:333`.
- The stream is SSE, one turn per HTTP request (`services/agents/AGENTS.md`: "`POST /conversations/:id/turns` (SSE)"). The console attaches/re-attaches: mount-time re-attach and a `FOREIGN_TURN_POLL_MS` interval for a teammate's turn — `apps/console/src/features/agent-chat/useAgentChat.ts:183-209`, plus a `visibilitychange` catch-up at `:217-222`.
- Turn status is polled through `getActiveTurn(projectName)` (`useAgentChat.ts:191`, `:203`); a severed stream falls back to one authoritative status poll (`runTurn.ts:96-100`).

### Agent

- `/start` → `TurnSpec{Kind: start, Idea}` → `services/agents/src/prompts/turn.ts:41` — the whole instruction is `"Load the start skill and follow it."`, plus `IDEA_PREFIX` (`turn.ts:47`) and `SPEC_PATHS_RULE` (`turn.ts:57-58`).
- Eager skills for the `start` flow: `["grilling", "prd-contract"]` — `services/agents/src/prompts/turn.ts:104` (`FLOW_SUPPORTING_SKILLS`); the flow's own skill is always inlined.
- **System prompt narration rule** — `services/agents/src/agents/main/prompt.ts:72-74`:
  > `Narration: keep prose outside tool calls to a single short sentence by default. A LOADED skill may define the narration for its own flow (what to say as you work, and how to close) — when one does, follow the skill. When the instruction is fully applied, stop.`

  That is a **cap**, not a requirement. The prompt file's own header (`prompt.ts:26-33`) says it carries "ONLY the tool contract, the error-reaction table … and the narration meta-rule — stage behavior lives in skills".
- The prompt says nothing about flow tokens, labels, or headless posture. Headless is composed separately as `HEADLESS_NOTE` in `services/agents/src/prompts/turn.ts:72-73`, only when `TurnRequest.headless` is set.
- **`skills/start/SKILL.md`**:
  - one form only — `:16` (*"`/start` asks exactly once"*) and `:49-52` (*"There is no second form"*).
  - **the pre-question work is explicitly silent** — `:38-39`: *"The walk is **planning, not turns**: you take it silently, in full, before the user sees a single question."*
  - narration is **closing-only** — `:95-98`: *"Close with a one-paragraph summary of the decisions taken (calling out every `*assumed*` one), then point the user at the next step…"*
  - skip valve tags `*assumed*` — `:63-67`.
- **`skills/grilling/SKILL.md`**: defines `ask_question` / `ask_questions` (`:20-23`); ending states Converged / Skip valve / Headless (`:44-52`); and the rule that kills multi-round work — `:54-56`: *"How many forms an interview may spend is the calling skill's rule … converging early is always allowed, **asking again never is**."* No "session" concept: grep for `session` in that file returns **not found**.
- Question mechanism is a tool pair, not a bespoke channel: `services/agents/src/agents/main/tools/files.ts:87-181`, wire contract at `packages/agent-stream/src/contracts/sse-events.ts:116-190`. `ask_question` takes `{question, detail?, options (0–5, ≤1 recommended), multiSelect?}`; `ask_questions` takes 1–8 of them. `execute()` returns `{status: "awaiting_user_response", …}` and **the call ends the turn**; the answer arrives as the next turn's user message (`sse-events.ts:203-228`).
- **There is no narration tool, narration event, or progress-event type.** The SSE event set is `text-delta`, `tool-input-start/delta/end`, `tool-call`, `tool-result`, `tool-error`, `error`, `finish`, `manifest` — `packages/agent-stream/src/contracts/sse-events.ts:638-656`. Narration is whatever prose the model chooses to emit between tool calls. Grep for `narrat` in `services/agents/src` returns exactly two hits, both comments in `prompt.ts` (lines 30, 73).

---

## Stage 4 — Design derivation (`/design`)

### Console

- Same route and view. The CTA is in the spec header — `SpecView.tsx:743-765`: a contained button `"Generate design"` with a Sparkles icon.
- Its disabled predicate: `disabled={!hasRequirementsFiles || agentBusy || openQuestions > 0}` (`SpecView.tsx:759`). Tooltips explain each case (`SpecView.tsx:744-754`): "An agent is still working — Generate design is available once it finishes" / "`N` open questions block design — answer or defer them first" / "Derive the component design from your requirements" / "Generate requirements first".
- Open questions are counted **client-side by parsing the PRD markdown** — `countBlockingOpenQuestions` in `apps/console/src/features/spec/lib/openQuestions.ts`, called at `SpecView.tsx:451-454` over `prdContent.data`.
- A `Resolve open questions (N)` outlined warning button seeds `"/amend Resolve the open questions"` — `SpecView.tsx:763-772`. A `+ Feature` button seeds `"/amend Add a feature"` — `SpecView.tsx:726-730`.
- `generateDesign()` re-navigates to the same route with `search: { generate: "design" }` — `SpecView.tsx:768-774`; the chat panel then auto-sends `DESIGN_COMMAND` through the identical one-shot path as `/start`.
- `agentBusy` is derived from **collab room presence**, not from turn state: `collab.peers.some((p) => p.kind === "agent")` — `SpecView.tsx:477`.
- **What the user sees during derivation**: identical to spec generation — a `/design` bubble from themselves, the `WorkingIndicator` "Working…", and the activity rail as files land. Plus the file-list `"Being derived…"` note and the body line `"The agents are shaping the spec — files appear here as they land."` (`SpecView.tsx:1071-1072`) — but both are gated on `specStatus`, which the backend never sets to a deriving value (Drift D3).
- The `Actions ▾` menu in the chat composer (`AgentChatPanel.tsx:531-557`) is the other entry surface. Its items seed literal strings: `"/amend Add a feature"`, `"/amend Add an actor"`, prefill `"/amend Go deeper on "`, `"/amend Resolve the open questions"`, `"/design Start the next phase — delta pass, protect shipped components"`.

### Backend

- `/design` is classified by the same `turnSpecFor` path as any `/<skill>` token — `services/aep-api/internal/spec/start_command.go:72-88` → `TurnKindFlow{Skill: "design"}`. The BFF also gates MCP/web-search minting on recognising a design turn (documented in `start_command.go`'s header).
- `DESIGN_COMMAND = "/design"` is a named constant precisely because it is a console CTA, not merely typed text — `packages/contracts/commands/index.ts:63`.

### Agent

- `FLOW_SUPPORTING_SKILLS.design = ["cell-design", "architecture", "security-design", "openapi-conventions", "wireframes", "validation-criteria"]` — `services/agents/src/prompts/turn.ts:118`, all inlined eagerly with the rationale at `:108-121`.
- **`skills/design/SKILL.md:20-22`**: *"Design FROM `specs/requirements/prd.md`. **Do not interview the user again** and do not widen or narrow the scope: what the PRD says is what gets designed."*
- Open-questions gate — `skills/design/SKILL.md:24-26`: any PRD Open Question neither answered nor marked `deferred` blocks design; the skill stops and points at the amend flow. Deferred questions never block.
- Design close — `skills/design/SKILL.md:76-80`: one line per component; a **"Needs your input"** block listing only still-ambiguous dependencies; a one-line pointer to `specs/design/`.
- Per-dependency narration is the **only real narration in the whole journey**, and it belongs to the `architecture` skill — `skills/architecture/SKILL.md:322-332` ("Narrating the design turn": *"Narrate each dependency decision in one plain-prose line as the user watches live"*), with the close at `:336-340`.
- `skills/cell-design/SKILL.md:173`: *"Do not narrate this in chat — just make the writes."*
- **No fork scan. Not found** — searched `skills/` for `fork scan`, `fork-scan`, `forkScan`: zero matches.
- **`*assumed*` never reaches design.** `grep -rn assumed skills/` hits only `prd-contract/SKILL.md:40`, `grilling/SKILL.md:50,52`, `start/SKILL.md:15,66,96`. Not present in `design`, `amend`, `architecture`, `cell-design`, `task-planning`, `validation-criteria`, or `organization`.
- `skills/prd-contract/SKILL.md:35-40` scopes the token narrowly: `*assumed*` marks **skip-valve entries in the Product Decisions section only**; org-default decisions are "ordinary entries".

---

## Stage 5 — Design gate / approval

### Console

**There is no approval step.** The gate is: *do design files exist?*

- `const hasDesignFiles = files.some((f) => f.group === "designs")` — `SpecView.tsx:443`.
- The header CTA is a straight ternary on that one boolean — `SpecView.tsx:691-767`: design files present → `Build`; absent → `Generate design`. The comment at `SpecView.tsx:691-693` states the intent: *"the prominent action is always the next pipeline step — Generate design until a design exists, then Build. A dead disabled Build hid what to do next."*
- `Build` is disabled only while `agentBusy || buildPhase !== null` (`SpecView.tsx:711`), with the tooltip *"An agent is still working — Build is available once it finishes"* / *"Commit your latest changes and start building"* (`SpecView.tsx:696-701`).
- **Nothing asks the user to read, acknowledge, or approve the derived design.** There is no checklist, no diff, no "I've reviewed this" affordance, no summary-of-decisions surface. The derived files are browsable in the nav; whether the user opened any of them is not tracked and does not gate anything.
- The only ceremony is the **cut-version dialog** — `SpecView.tsx:812-838`: title `"Cut version {vN}"`, body *"Snapshots the PRD and design together as a git tag; the build runs against that snapshot, so you can keep editing afterwards."*, plus `Stories in scope:` and `Milestone:` lines, and the confirm `Cut {vN} & build`. The version shown is **predictive** — the comment at `SpecView.tsx:812-815` says the backend assigns the real tag at cut time.
- The **real** gate lives on the server and surfaces only as a **refusal**: a 422 from `POST /build` renders `gateRefusal` — `SpecView.tsx:777-810`: `Alert severity="warning"`, `AlertTitle "Build refused — the design isn't complete"`, one bullet per unmet condition, and a `Fix via chat` action that seeds `"/design Fix these build-gate refusals:\n…"` (`SpecView.tsx:786-798`). So a newcomer learns the gate's criteria only by tripping it.

### Backend

- The gate is enforced in the build service: a 422 with per-field details is what the console renders as the checklist (`SpecView.tsx:777-810` consumes `details: Array<{field?, message}>` at `SpecView.tsx:546-547`).
- `specStatus = "approved"` exists in the backend (`services/aep-api/internal/projects/status_stages.go:367-368`) but means only *"a `v<N>` spec tag exists"* — i.e. it is set **by cutting the tag during Build**, after the fact. It is not a user approval, and **the console never reads the value** (searched `apps/console/src` for `"approved"` against `specStatus`: the only consumers of `specStatus` are `SpecView.tsx:436-441`, which test for `pending`/`draft`/`in_progress`/`failed`).
- `designStatus = "approved"` is likewise a tag observation (`status_stages.go:372-374`) and is read nowhere in the console outside mocks.
- **There is no approve/gate endpoint. Not found** — the tags surface is read-only (`GET /projects/{projectName}/tags` → `services/aep-api/internal/spec/tags/handler.go:38`, returning `TagList{Tags, Latest, SpecDirty}` at `services/aep-api/internal/gen/models_gen.go:1602`). The tag is cut only as a **side effect of `POST /build`** (`services/aep-api/internal/delivery/build/build_service.go:263`, `s.tagger.TagSpec`). "Approval" in this system is not an act; it is an observation that a tag exists.
- The design domain declares an approval concept that **nothing enforces**: `ErrSpecNotApproved = errors.New("spec must be saved (tagged) before generating a design")` — `services/aep-api/internal/spec/design_service.go:30`, whose own doc comment at `:27-29` claims it is "surfaced (as 409 by the controller)" and that the design feature "is the only consumer". `grep -rn ErrSpecNotApproved services/aep-api --include=*.go` returns **only the declaration and its comment — no caller**. So the rule that a spec must be tagged before design derivation is documented, sentinel-ed, and never applied; the console correspondingly lets `Generate design` run against a wholly untagged spec (`SpecView.tsx:759` checks only `hasRequirementsFiles`, `agentBusy`, `openQuestions`).

### Agent

- The design skill's own "gate" is mechanical and lives at build time, not as a user-facing approval. `skills/design/SKILL.md:54` marks the validation-criteria step "never skip this" because it is the acceptance oracle.
- `skills/design/SKILL.md:70`: on a conflict with a shipped design, *"surface the conflict to the user; never silently redraw shipped"* components.

---

## Stage 6 — Build kicked off

### Console

The `Build` click runs a three-phase sequence — `onBuild` at `SpecView.tsx:484-522`:

1. `buildPhase = "committing"` → `collab.flush()` then `spec.refetch()` (`SpecView.tsx:486-493`). Button reads `"Committing…"` (`SpecView.tsx:716`).
2. `buildPhase = "checking"` → `preflight.refetch()` (`SpecView.tsx:494-495`). Button reads `"Checking…"`. On error, an explicit guard (`SpecView.tsx:496-511`) — the comment records that TanStack's `refetch()` resolves rather than throws, and falling through would "silently skip dependency provisioning (the exact #164 symptom)".
3. `data.needsInput` → open `BuildDependencyDrawer` (`SpecView.tsx:512-515`); otherwise open the cut dialog (`SpecView.tsx:516`).

Confirm → `runBuild` (`SpecView.tsx:527-...`): `buildPhase = "building"`, `build.mutateAsync({inputs: []})`, then **navigate to the project overview** (`SpecView.tsx:536-540`). Failure paths: 422 → `gateRefusal` checklist; anything else → `buildError` `Alert` (`SpecView.tsx:841-851`).

**What the user sees the instant build is kicked off:** they are dropped on the project overview. The overview's Build card renders `buildStageView(status)` — and until the polled status aggregate reports `build.status === "running"`, it renders the **default** branch: ghost tone, `version: ""`, line `"waiting on spec"` (`apps/console/src/features/projects/lib/pipeline.ts:74-75`). The status poll runs at `STATUS_ACTIVE_POLL_MS = 5_000` at best (`queries.ts:87`, `:139-140`). See Drift D6.

- The Builds page's own empty state reads `"No builds yet — publish your spec and click Build in the spec view to…"` — `apps/console/src/features/builds/components/BuildsPage.tsx:200-203`. **"Publish your spec" names an action that does not exist** in the console; the version is cut *by* Build. See Drift D7.

### Backend

- Build kickoff is `POST /projects/{projectName}/build` — `services/aep-api/internal/gen/server_gen.go:3205`, wired via `services/aep-api/internal/app/build_adapters.go`.
- It cuts the tag: `services/aep-api/internal/delivery/build/build_service.go:303` logs `"tag", res.Tag, "specStatus", res.Status`.
- **Build is not a `/<skill>` flow token.** `/build` appears nowhere in `packages/contracts/commands/index.ts` (which defines only `START_COMMAND` at `:60` and `DESIGN_COMMAND` at `:63`, plus `parseStartCommand` at `:73-80` and `parseFlowCommand` at `:100-104`).

### Agent

- Not involved in kickoff. Task planning is a separate turn kind: `TurnSpec{Kind: plan}` selects the `task-plan` toolset with **no file tools** (`services/agents/AGENTS.md`, "Tool sets"), instructions at `services/agents/src/agents/main/prompt.ts:231-257` and `PLAN_INSTRUCTION` at `services/agents/src/prompts/turn.ts:81`. Constructed at `services/aep-api/internal/delivery/task/plan.go:245`.

---

# A. DRIFT — where the console assumes what the backend/agent does not provide

## D1. The create screen promises derivation that never starts

`apps/console/src/features/projects/components/ProjectCreate.tsx:155-156` tells the user *"AEP turns your requirement into a project **and starts deriving its design**."* Then `accept()` navigates to the overview (`ProjectCreate.tsx:132-140`) and **nothing runs**: `services/aep-api/internal/projects/project_service.go:245-257` only writes the descriptor. The overview then shows a Spec card whose state is `cta: true` — `pipeline.ts:54` — i.e. a `Generate spec` button. The user was told derivation started; the platform is waiting for them to press a button. This is the first "clueless" moment and it is manufactured entirely by copy that outran the implementation.

## D2. "Is the agent working?" is answered from `localStorage`, not from the server

`OverviewPipeline.tsx:218` calls `useAgentEngaged(orgHandle, projectName)`. That hook reads the **browser-local** chat log: `useAgentEngaged.ts:69-76` → `getMessages(chatKey)` → `apps/console/src/features/agent-chat/chatStore.ts:121` (`localStorage.getItem(key)`).

Consequences, all real:
- A user on a **second device or browser** sees `Generate spec` for a project whose interview is already open — and the CTA attaches `?generate=requirements` (`OverviewPipeline.tsx:190`), injecting a second `/start`. `useAgentEngaged.ts:28-34` documents exactly what that costs: the start skill reads a fresh instruction as its skip valve and *"writes the PRD from its own recommended answers, tagged `*assumed*`, instead of the user's. Nothing errors; the interview is simply gone."*
- The doc comment at `useAgentEngaged.ts:47-50` claims project-scope accuracy because conversations are project-scoped since #430 — but that is accuracy *"up to the freshness triggers in `useAgentChat`"* (mount, foreign-turn poll, tab refocus, `useAgentChat.ts:183-222`). On the **overview**, `AgentChatPanel` is unmounted unless the user opens it, so none of those triggers run: the overview's `engaged` is whatever this browser last wrote.
- A turn that never settled (tab closed mid-flight) rehydrates as `in_flight` and reads engaged forever (`useAgentEngaged.ts:41-45`, accepted).

The backend has no field to ask instead — see the `ProjectStatus` enumeration above; **no turn/activity field exists**.

## D3. `specStatus`: the console tests for three values the backend never emits

Console (`SpecView.tsx:436-441`):
```
const specStatus = status.data?.specStatus;
const deriving = specStatus === "pending" || specStatus === "draft" || specStatus === "in_progress";
const failed = specStatus === "failed";
```
Backend (`services/aep-api/internal/projects/status_stages.go:365-371`) emits only `""`, `"draft"`, `"approved"`; the contract says so explicitly at `packages/contracts/api/v1/openapi.yaml:4346-4348` (`'"", draft, approved'`).

Therefore:
- **`failed` is never true.** The banner at `SpecView.tsx:769-775` — *"The agent's last turn failed / Your files are safe — everything already written remains browsable."* — is **dead code**. It has never rendered against a real backend. The `"Derivation failed"` empty-group note (`SpecFileList.tsx:110-111`) is likewise unreachable.
- **`deriving` collapses onto `specStatus === "draft"`**, which means only "a spec exists and no tag has been cut". So `"Being derived…"` (`SpecFileList.tsx:112-113`) and `"The agents are shaping the spec — files appear here as they land."` (`SpecView.tsx:1071-1072`) are shown for **every unpublished spec, permanently**, whether or not any agent is running. A user who came back a week later sees "Being derived…" beside an idle project.
- Neither `"approved"` (the only remaining real value) nor `designStatus` is read by the console at all.

**How this shipped:** the MSW fixtures emit the fictional values. `apps/console/src/mocks/fixtures/project.ts:76` (`specStatus: "pending"`), `:92` (`designStatus: "in_progress"`), `:104` (`specStatus: "failed"`), `:105` (`designStatus: "failed"`). The states were designed and demoed against mocks that do not match the contract the backend implements.

## D4. `agentBusy` is collab-room presence, not turn state

`SpecView.tsx:477`: `const agentBusy = collab.peers.some((p) => p.kind === "agent")`. This one boolean gates **Build**, **Generate design**, and **Regenerate design** (`SpecView.tsx:711`, `:759`, `:927`) and drives the `"Waiting for the agent to write …"` placeholder (`SpecView.tsx:985-1000`).

But an agent joins the Yjs room only while it is *writing files*. A `/start` turn spends its longest stretch — the coverage walk, explicitly silent per `skills/start/SKILL.md:38-39` — doing no writes, and a turn blocked on an unanswered `ask_questions` form has ended entirely (the tool call terminates the turn; `packages/agent-stream/src/contracts/sse-events.ts` semantics, confirmed in the agent inventory). In both windows `agentBusy` is **false**, so the console will happily offer `Generate design` while an interview is open. The only thing standing between that and a wrecked interview is `useAgentEngaged` — which, per D2, is per-browser.

Meanwhile the *chat panel's* notion of busy is a different variable entirely (`isSending` / `teammateRunning`, `AgentChatPanel.tsx:170-180`), computed from a third source (the chat feed). Three surfaces, three answers to "is the agent working".

## D5. The console renders a "design gate" the backend does not agree exists

The console's gate is `hasDesignFiles` — one file-group predicate (`SpecView.tsx:443`). The backend's gate is a set of manifest/story-coverage checks that only ever speak as a **422 refusal** (`SpecView.tsx:777-810`).

So the button arms on a criterion (files exist) that is strictly weaker than the criterion enforced (every story claimed by a component, every deployable component enriched, language not `TBD` — per `skills/design`'s and the build gate's rules). The predictable experience: `Build` looks ready, the cut dialog appears and is confirmed, and *then* the platform refuses. `Build refused — the design isn't complete` is the newcomer's first exposure to the gate's actual criteria. Nothing before the click discloses them.

## D6. Build kickoff has no hand-off confirmation — the user is dropped on the overview and left to infer

`runBuild` navigates to the **project overview** (`SpecView.tsx:536-540`) the moment `POST /build` resolves. Nothing marks the moment: no toast, no banner, no "build started" state, and no navigation to the Builds page where the run actually lives.

The backend does set the state synchronously — `Service.Run` (`services/aep-api/internal/delivery/build/build_service.go:208-305`) cuts the tag (`:263`), claims the version and admits the run row (`:271-300`) before returning `BuildResponse{Tag}`. And `useBuildProject`'s `onSuccess` invalidates `projectKeys.detail(projectName)` (`apps/console/src/features/projects/api/queries.ts:394-397`), which **is** a prefix of `projectKeys.status(name)` (`apps/console/src/features/projects/api/keys.ts:25-26`), so the status query does refetch immediately rather than waiting a poll interval.

So the gap is short — but it is filled by the wrong words. Until that refetch lands, `buildStageView`'s `default` branch renders the Build card as **ghost tone, no version, `"waiting on spec"`** (`apps/console/src/features/projects/lib/pipeline.ts:74-75`). The user who just confirmed *"Cut v1 & build"* watches a greyed card say *waiting on spec* and then silently flip to *building*. There is no optimistic "starting" state (`pipeline.ts:65-77` has no such branch) and no acknowledgement that their click was the cause. For the journey's terminal moment — the one #502 bounds the destination at — the console says nothing at all.

Worse, the rich narration that *does* exist for this phase is on a page the user was not sent to: `RunStory` / `RunHoldNotice` / `RunBusy` carry real explanatory copy — e.g. *"Planning &lt;milestone&gt; — Creating this version's issues in GitHub. Nothing is held and nothing is needed from you — the first build session dispatches as soon as the milestone is written."* (`apps/console/src/features/builds/lib/runView.ts:196-205`) and *"Waiting to dispatch the first build session…"* (`apps/console/src/features/builds/components/RunStory.tsx:386-389`). The best waiting-state copy in the product sits one navigation away from the user who most needs it.

## D6b. The gate refusal is mislabelled — a *spec* validation failure is reported as a *design* problem

The backend maps the tag-cut gate to **HTTP 400** with the message `"spec validation failed"` and one `ErrorDetail{Field: <file path>, Message: <code>: <message>}` per offending file — `services/aep-api/internal/delivery/build/build_service.go:326-334` (`mapTagError`).

The console renders those same details under `AlertTitle` **`"Build refused — the design isn't complete"`** (`SpecView.tsx:802`). It is a *spec* gate, over spec files, and the user is told their design is incomplete.

Two further mismatches in the same path:
- Both `queries.ts:379-381` and `SpecView.tsx:526` describe this as **"the build gate's 422"**. The backend returns **400**. It happens to work only because the console keys on the *presence* of `details` (`SpecView.tsx:540-541`, `queries.ts:382-385`), never on the status code — so the comments document a contract the server does not implement, and any future code that does branch on 422 will silently miss every refusal.
- `mapTagError` also emits 404 `"project repository not found"` and 409 `"project repository is not ready yet"` (`build_service.go:336-339`); neither carries `details`, so both fall through to the flat `buildError` alert (`SpecView.tsx:542-543`) with no guidance.

## D7. Vocabulary drift: "publish"

`BuildsPage.tsx:200-203` instructs the user to *"publish your spec and click Build"*. There is no publish action anywhere in the console — the spec header shows `"{vN} · published"` as a **derived chip** (`SpecView.tsx:639-647`, from `tags.data.latest`), and the version is cut **by** the Build ceremony (`SpecView.tsx:812-838`). The empty state names a prerequisite step that does not exist, pointing a newcomer at a control they will never find.

## D8. The user's own button-press renders as a slash command they never typed

`AgentChatPanel.tsx:314` sends the literal `START_COMMAND` / `DESIGN_COMMAND`; `runTurn.ts:85` writes it into the log as a `role: "user"` message; `MessageList.tsx:95-105` renders `{message.content}` verbatim in an own-message bubble. A newcomer who pressed *Generate spec* sees themselves apparently having typed `/start`. The `journal` mechanism that exists precisely to carry display text (`services/agents/AGENTS.md`, "Turn journal (#463)": *"the raw client-sent instruction + acting user, stored beside the transcript"*) stores the same raw string — it de-duplicates authorship, not vocabulary.

## D9. The console counts open questions by parsing markdown

`SpecView.tsx:451-454` calls `countBlockingOpenQuestions(prdContent.data.content)` (`apps/console/src/features/spec/lib/openQuestions.ts`) to decide whether `Generate design` is enabled (`SpecView.tsx:759`). The **agent** enforces the same gate independently from its own reading of the PRD (`skills/design/SKILL.md:24-26`). Two parsers, one markdown convention, no shared contract and no backend field. They can disagree — and when they do, the button is enabled and the skill stops with an explanation the user has to read out of the chat.

## D10. The panel auto-navigates on questions; the overview cannot say so

`AgentChatPanel.tsx:225-238` yanks the user to the spec view when a question arrives — but only if the panel is **mounted**. The panel is unmounted on the overview unless the user opened it (`AppLayout.tsx:354` `Collapse unmountOnExit`, per the comment at `AgentChatPanel.tsx:297-301`). A user sitting on the overview while their interview reaches its question form gets **no signal at all**: no badge, no card state, no toast. `pipeline.ts:52-58` has no branch for it, and `ProjectStatus` carries no field it could branch on.

---

# B. SPECIFIED-BUT-NEVER-BUILT

Each item below was searched for by name, by concept, and by likely symbol.

## From #485 — Spec view first run (BE-initiated `/start`, ghost nav, skeleton PRD)

| Specified | State |
|---|---|
| `POST /projects` starts the `/start` turn server-side at creation, idempotent, one per project | **No code found.** `services/aep-api/internal/projects/project_service.go:245-257` writes only the descriptor. The only non-test `agentsvc.TurnSpec` construction sites are `spec/start_command.go:72-87` (inbound instruction) and `delivery/task/plan.go:245` (planning). `/start` is still injected client-side at `AgentChatPanel.tsx:314`. |
| Overview Spec card shows live interview state ("interviewing — 4 questions waiting") | **No code found.** `pipeline.ts:52-58` has four branches, none of them an interview state; `pipeline.ts:23-25` states the spec stage has no stored status. The only live-ish signal is the two-label button at `OverviewPipeline.tsx:194`, computed per-browser (D2). |
| Chat rail shows the turn streaming from anywhere | **Partially built.** Streaming + re-attach exist (`useAgentChat.ts:183-222`), and the questions banner exists (`TurnBlock.tsx:94-130`, "The agent has N questions … Answer them →"). But the rail is unmounted unless opened (`AppLayout.tsx:354`), and nothing outside it signals the turn. |
| Live narration from second one ("Reading your idea… ✓ Consulted org defaults… Planning the interview") | **No code found, and the skill forbids it.** `skills/start/SKILL.md:38-39` makes the pre-question walk explicitly silent; `prompt.ts:72` caps prose at "a single short sentence by default". No narration tool or event exists (`sse-events.ts:638-656`). |
| File nav pre-renders the journey as ghost entries | **No code found.** Searched `SpecFileList.tsx` for `ghost`/`placeholder`/`Skeleton`: nothing. The nav renders only extant files. |
| Skeleton PRD (ghost section outline with shimmer) | **No code found.** The spec body's only loading affordance is `CircularProgress` (`SpecView.tsx:864-874`, `:1014-1021`, `:1070-1080`). |
| Ghosts graduate / writing pulse / relabel | **No code found** (there are no ghosts to graduate). |
| Canned empty-state chips removed on fresh projects | **Not done.** `"Hi! I'm your Agent."` and the three off-domain `SUGGESTIONS` are still present — `AgentChatPanel.tsx:80-84`, `:485-506`. |

## From #486 — Multi-round grilling sessions

| Specified | State |
|---|---|
| `grilling` skill session mode replacing "asking again never is allowed" | **Not built — the rule is still there, verbatim.** `skills/grilling/SKILL.md:56`: *"converging early is always allowed, asking again never is."* Grep for `session` in that file: **not found**. |
| Adaptive small batches / rounds | **No code found.** |
| Write-as-you-go between rounds | **No code found.** |
| "Finish — use recommendations" valve per round | **Not built** as a session control. The skip valve exists but is a one-shot user utterance — `skills/start/SKILL.md:63-71`, `skills/grilling/SKILL.md:46-50`. |
| Park/resume session chrome, area checklist header | **No code found.** Searched `apps/console/src/features/spec` and `features/agent-chat` for `session`: only `sessionStorage` (auth) and `EarlierSessions.tsx` (build runs). |
| `/start` end-of-turn offer ("grill me on: [areas]") | **No code found.** `skills/start/SKILL.md:95-98` closes with a summary + "run `/design`", no grilling offer. |
| Answer serialization gaining a decision-to-question link | **No code found.** `buildAnswerInstruction`/`buildAnswersInstruction` (`packages/agent-stream/src/contracts/sse-events.ts:203-228`) emit flat `Answer to "…"` / `Answers:` prose. |

## From #487 — Clickable assumptions (`*assumed*` chip)

| Specified | State |
|---|---|
| `*assumed*` rendered as an interactive chip in the spec editor | **No code found.** `grep -rn assumed apps/console/src` returns three hits, **all comments in tests and hook docs** (`useAgentEngaged.ts:33`, `useAgentEngaged.test.tsx:24`, `AgentChatPanel.test.tsx:169`). Nothing renders it; the editor treats it as ordinary markdown emphasis. |
| Chip click → grilling session scoped to that decision | **No code found** (no sessions exist; see #486). |
| prd-contract riders: end of line, one per line, valid in **any** PRD section | **Not built.** `skills/prd-contract/SKILL.md:35-40` still scopes the token to the **Product Decisions section only**, and only for skip-valve entries. |

## From #488 — PRD code lenses and the `refine` skill

| Specified | State |
|---|---|
| Code lenses (Add actor / Add feature / Deepen / Resolve / Grill) in the PRD editor | **No code found.** `grep -rni "codelens|code-lens|code lens" apps/console/src`: zero matches. |
| Structured intent envelope (`add-actor` / `add-feature` / `deepen` / `resolve`) | **No code found.** The entry surfaces still seed literal strings: `AgentChatPanel.tsx:547-552` and `SpecView.tsx:726-730`, `:763-772`. |
| `amend` → `refine` rename | **Not done.** `skills/refine/` does not exist (directory listing of `skills/`); the only `refine` token in the whole tree is Zod's `.refine(` at `services/agents/src/agents/main/tools/files.ts:127`. `skills/amend/` is still the skill. |
| Interview-before-artifact rule ("writes nothing until answers land") | **No code found** as an encoded rule. |
| Removal of the "what do you want to amend?" prologue | **Not applicable / not done** — the console still hands the agent an unstructured English string, which is the condition that produced the prologue. |

## From #489 — Design-time fork clarifications

| Specified | State |
|---|---|
| Silent fork scan of the PRD opening the design turn | **No code found.** Searched `skills/` for `fork scan` / `fork-scan` / `forkScan`: zero matches. |
| One grilling session over the forks found, auto-continuing into derivation | **Not built** (no session mode; and see next row). |
| Reversal of "Do not interview the user again" | **Not done — the rule is intact.** `skills/design/SKILL.md:20-22`. |
| Late forks written back to Product Decisions tagged `*assumed*` | **Not built.** `assumed` does not appear in `skills/design/SKILL.md` at all (grep over `skills/`). |
| Design summary must list every `*assumed*` decision or state "no forks" | **Not done.** `skills/design/SKILL.md:76-80` still closes with component lines + a **"Needs your input"** dependency block + a pointer — the exact informal line #489 set out to replace. |

## From #490 — Assumption count indicator

| Specified | State |
|---|---|
| Header badge "N assumptions" from parsing `*assumed*` | **No code found** (nothing parses the token; see #487). |
| Per-section counts in the file nav | **No code found.** |
| Scroll-to-first-chip | **No code found.** |

## From #377 / map #364 — the spec-agent redesign epic

Phases A, B, C, E are visible in `main` and consistent with the epic's checkboxes: canonical prompt strings in `packages/contracts/commands/index.ts` and server flow-token recognition (`services/aep-api/internal/spec/start_command.go`); the `start` / `grilling` / `amend` / `organization` skills plus `prd-contract`; the cell DSL and scaffold path (`skills/cell-design`, `skills/architecture`); milestone-per-version and `skills/task-planning` with `PLAN_INSTRUCTION` (`services/agents/src/prompts/turn.ts:81`) and the `task-plan` toolset (`prompt.ts:231-257`).

Two items marked complete on #377 are **not present**:

- **Phase F, "chat-top flow steppers"** — **no code found.** `grep -rni "flowstep|FlowStepper|flow-step" apps/console/src`: zero matches. `grep -rni "stepper" apps/console/src --include=*.tsx` finds exactly two: `apps/console/src/features/onboarding/components/OnboardingWizard.tsx:81-87` (the credentials wizard, out of scope per map #502) and `apps/console/src/features/alerts/components/AlertDetail.tsx:225-244` (an alert's derived stages). **The idea→build journey has no stepper anywhere.**
- **Phase F, "in-document pills"** — **no code found** in the spec editor (`apps/console/src/features/spec/collab/SpecMdEditor.tsx` and siblings; also the `*assumed*` search above).

The rest of Phase F **is** present: the enriched spec card (`OverviewPipeline.tsx:133-199`), the `Actions ▾` menu (`AgentChatPanel.tsx:531-557`), the header launchers (`SpecView.tsx:726-772`), the "Cut version" drawer (`SpecView.tsx:812-838`), the failure cards (`SpecView.tsx:841-861`) and the gate-refusal checklist (`SpecView.tsx:777-810`).

Phase D ("Design skills") is unchecked on #377 but the skills **do exist** in `skills/`: `design`, `cell-design`, `architecture`, `security-design`, `wireframes`, `openapi-conventions`, `validation-criteria` — all seven, all with `SKILL.md`.

---

# C. WHAT THE USER SEES DURING EVERY WAIT

| Wait | Duration character | What is on screen | Citation |
|---|---|---|---|
| **After Create → overview** | instant, but nothing starts | A `Generate spec` button. No indication the platform is idle-by-design after telling the user it "starts deriving". | `pipeline.ts:54`, `OverviewPipeline.tsx:194`, `ProjectCreate.tsx:155-156` |
| **Spec generation — the coverage walk (the long one, before questions)** | tens of seconds | A `/start` bubble attributed to the user, then a pulsing dot + `"Working…"`. Nothing else. The agent is **required to be silent** here. | `runTurn.ts:85`, `MessageList.tsx:95-105`, `AgentChatPanel.tsx:193-194`, `WorkingIndicator.tsx:24-47`, `skills/start/SKILL.md:38-39`, `prompt.ts:72` |
| **Spec generation — from the overview (panel closed)** | tens of seconds | **Nothing.** No card state, no badge, no count. | `pipeline.ts:52-58`, `AppLayout.tsx:354` |
| **Question form arrives** | user-blocking | Auto-navigation to the spec view (panel mounted only) + a full-body form headed `"Quick questions"`; in chat, a `"The agent has N questions / Answer them →"` pointer. Questions still streaming show `"The agent is still writing questions — you can start answering."` — **the only true streaming affordance in the journey**. This is the journey's best state by a wide margin. | `AgentChatPanel.tsx:225-238`, `TurnBlock.tsx:94-130`, `SpecView.tsx:886-895`, `SpecQuestionForm.tsx:280-291`, `:304-310` |
| **PRD writing (after answers land)** | tens of seconds | `"Working…"`, plus tool steps on the activity rail as files are written; the selected-file pane may show `"Waiting for the agent to write prd.md…"`. No outline, no section-by-section signal. | `TurnBlock.tsx:50-52`, `:35-41`, `SpecView.tsx:985-1000` |
| **Design derivation** | the longest wait in the journey | `"Working…"` + the activity rail, **plus** genuine per-dependency prose from the `architecture` skill (`✓ <capability>: using <choice>`) — the only real narration anywhere in the flow. `cell-design` is explicitly told not to narrate. Nothing names the seven-artifact lineup being walked, so the prose has no frame. | `TurnBlock.tsx:50-52`, `skills/architecture/SKILL.md:322-332`, `skills/cell-design/SKILL.md:173`, `turn.ts:118` |
| **Design derivation — from the overview** | minutes | **Nothing.** Same blind spot as spec generation. | `pipeline.ts:52-58` |
| **Build: committing / checking** | seconds | Button label changes: `"Committing…"` → `"Checking…"`, with a loading spinner on the button. This is the **best-instrumented wait in the journey**. | `SpecView.tsx:711-719`, `:486-495` |
| **Build: after confirm** | brief (invalidation-driven, not a full poll interval) | Navigate to overview. No toast, no banner, no confirmation. The Build card shows ghost `"waiting on spec"` and then flips to `"building"`. The genuinely good waiting copy (`"Planning <milestone>… the first build session dispatches as soon as the milestone is written"`) lives on the Builds page the user was **not** sent to. | `SpecView.tsx:536-540`, `pipeline.ts:74-75`, `queries.ts:394-397`, `keys.ts:25-26`, `builds/lib/runView.ts:196-205`, `builds/components/RunStory.tsx:386-389` |
| **Turn failure** | — | In chat: `"Turn failed"` footer + an error row. In the spec view: the `"The agent's last turn failed"` banner **never fires** (D3). | `TurnBlock.tsx:53-66`, `runTurn.ts:82-89`, `SpecView.tsx:769-775` |

**Summary of the waits:** of the journey's five real waits, **two are fully silent** (spec generation and design derivation, whenever the chat panel is not open), **two show a single undifferentiated `"Working…"` dot** regardless of what is happening or how far along it is, and **one — the build's commit/check sequence — is properly narrated**, by label, in the console, with no agent involved. The only agent-side narration that exists anywhere is the `architecture` skill's per-dependency lines during design.

---

## Appendix 1 — the backend surface the journey touches

Contract-first: `packages/contracts/api/v1/openapi.yaml` → `oapi-codegen` strict server at `services/aep-api/internal/gen/server_gen.go`, routed in `services/aep-api/internal/edge/server.go:92-124`. The console consumes `apps/console/src/generated/aep-api.d.ts`, which is **types only** (`openapi-typescript`: `paths`, `components`, `operations`) — there are no generated hooks; the runtime client is one `openapi-fetch` instance at `apps/console/src/api/client.ts:30` and hooks are hand-written per feature.

| Stage | Method + path | Handler |
|---|---|---|
| List | `GET /projects` | `services/aep-api/internal/projects/projectcrud/handler.go:37` |
| Create | `POST /projects` | `.../projectcrud/handler.go:53` → `.../projects/project_service.go:189-284` |
| Overview | `GET /projects/{p}` · `GET /projects/{p}/status` | `.../projectcrud/handler.go:73` · `:90` |
| Spec files | `GET /projects/{p}/files` · `/files/{path}` · `/files/bundle` · `POST /files/apply` | `services/aep-api/internal/spec/files/handler.go:48` · `:65` · `:89` · `:105` |
| Design deps | `GET /projects/{p}/design/dependencies` | `services/aep-api/internal/edge/handlers_design.go:47` |
| Collab | `GET /projects/{p}/spec/collab-session` · `GET /collab/validate` | `services/aep-api/internal/spec/collab/handler.go:42` · `:65` |
| Tags | `GET /projects/{p}/tags` | `services/aep-api/internal/spec/tags/handler.go:38` |
| Chat / turns | `POST .../agents/{convId}/messages` (202, detached) · `GET .../turns/active` · `GET .../turns/{id}` · `GET .../turns/{id}/stream` (SSE) | `services/aep-api/internal/spec/genaiturns/handler.go:66` · `:103` · `:94` · `:115` |
| Build | `POST /projects/{p}/build` · `GET /build/preflight` · `GET /builds` | `services/aep-api/internal/delivery/build/handlers.go:61` · `:92` · `:80` |
| Runs | `GET /builds/{tag}/runs` · `GET /runs/{id}/progress` (SSE) · `POST /runs/{id}/cancel` | `services/aep-api/internal/delivery/runread/handlers.go:48` · `:75` · `:87` |
| Activity | `GET /projects/{p}/activity` · `/activity/stream` (SSE) | `services/aep-api/internal/projects/activityfeed/handlers.go:46` · `:98` |

Two structural facts that shape the journey:

- **`/design` is not an endpoint.** Design derivation rides the same `create-turn` chat channel as spec writing, distinguished only by the flow token in `TurnInputBody.instruction` (`packages/contracts/api/v1/openapi.yaml:5465-5467`). So there is no server-side notion of "a design derivation is in progress" distinct from "some turn is running". **Not found:** any design-specific endpoint or status.
- **All aep-api real-time signalling is SSE, never websocket.** Searched the Go tree for `Upgrader` / `gorilla` / `websocket`: no matches. Turn streaming is `text/event-stream` with replay-from-index resumption (`genaiturns/handler.go:115`, `:264`, `:318`; 15s keep-alive at `:64`). The one true websocket in the journey — the Yjs spec room — belongs to the separate `services/collab`; aep-api only mints the session and authorizes access.
- **No validation endpoint exists.** The contract says so at `packages/contracts/api/v1/openapi.yaml:347` (*"The validation verdict is a RUN property and is read here; there is no separate validation endpoint"*); the console reads `specs/validation/validation-criteria.json` and `tests/validation/report.json` as ordinary spec files (`apps/console/src/features/validation/api/queries.ts:62`, `:78`).

## Appendix 2 — what a journey redesign can and cannot read today

**Available from the backend without new work:** `phase` (repo rungs only, terminal at `tasks`), `spec.{exists,version,dirty,design}`, `specStatus` (`""`/`draft`/`approved`), `designStatus` (`""`/`approved`), the build and deploy aggregates, project tags, spec file listings + contents, preflight, and the SSE turn stream + `getActiveTurn`.

**Not available and would have to be built:** any project-level "a turn is running" / "N questions waiting" / "which flow" state (today: inferred per-browser from `localStorage`, D2); any spec-lifecycle state between "draft" and "tag cut" (today: `specStatus` conflates every unpublished spec, D3); any user-approval fact distinct from "a tag exists" (today: none, D5/Stage 5); any pre-click view of the build gate's criteria (today: only the 422 refusal, D5); any structured progress or narration channel from the agent (today: none — `sse-events.ts:638-656`).
