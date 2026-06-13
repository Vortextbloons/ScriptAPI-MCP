package tools

import (
	"encoding/json"
	"fmt"
	"strings"

	mcp "github.com/metoro-io/mcp-golang"
)

type GetBestPracticesInput struct {
	Category string `json:"category" mcp:"description='Filter: performance_principles, general, performance_optimization, patterns, or all (default)'"`
	Search   string `json:"search" mcp:"description='Keyword filter (e.g. watchdog, isValid, cache, tick, afterEvents)'"`
}

type BestPracticeEntry struct {
	Category string   `json:"category"`
	Title    string   `json:"title"`
	Content  string   `json:"content"`
	Tags     []string `json:"tags"`
}

type GetBestPracticesOutput struct {
	Results       []BestPracticeEntry `json:"results"`
	TotalCount    int                 `json:"total_count"`
	FilteredCount int                 `json:"filtered_count"`
}

var allBestPractices = []BestPracticeEntry{
	{
		Category: "performance_principles",
		Title:    "Gate expensive work behind feature flags or config checks",
		Content:  `Avoid running costly logic every tick. Wrap experimental or heavy features behind a config toggle, gamerule, or dynamic property check so they only execute when actually enabled.`,
		Tags:     []string{"performance", "feature-flags", "optimization"},
	},
	{
		Category: "performance_principles",
		Title:    "Return early when no work is needed",
		Content:  `Check preconditions at the top of your function and return immediately if nothing should happen. This avoids unnecessary computation and keeps your code readable.

Example:
if (someCondition) return;
// expensive work below`,
		Tags: []string{"performance", "early-return", "optimization"},
	},
	{
		Category: "performance_principles",
		Title:    "Cache derived data instead of rebuilding every cycle",
		Content:  `If you compute a value that doesn't change every tick (e.g. a filtered player list, a scoreboard lookup), cache it and reuse it. Rebuild only when the underlying data actually changes.

Example:
let cachedPlayers = [];
system.runInterval(() => {
    if (needsRebuild) {
        cachedPlayers = world.getAllPlayers().filter(p => p.hasTag("active"));
        needsRebuild = false;
    }
    // use cachedPlayers...
}, 1);`,
		Tags: []string{"performance", "caching", "optimization"},
	},
	{
		Category: "performance_principles",
		Title:    "Invalidate caches only when source data changes",
		Content:  `Use events to know when to refresh caches rather than clearing them on a timer. Subscribe to the relevant afterEvent (e.g. playerJoin, entityRemove) and set a dirty flag.

Example:
let needsRebuild = true;
world.afterEvents.playerJoin.subscribe(() => { needsRebuild = true; });`,
		Tags: []string{"performance", "caching", "events"},
	},
	{
		Category: "performance_principles",
		Title:    "Use Map, Set, or indexed records for repeated lookups",
		Content:  `Avoid scanning arrays with .find() or .filter() in hot paths. Convert lookup data into a Map (keyed by id) or a Set for O(1) access.

Example:
const playerMap = new Map();
for (const p of world.getAllPlayers()) {
    playerMap.set(p.id, p);
}
const target = playerMap.get(someId);`,
		Tags: []string{"performance", "data-structures", "optimization"},
	},
	{
		Category: "performance_principles",
		Title:    "Reuse per-cycle results like player lists",
		Content:  `If you call world.getAllPlayers() multiple times in the same tick, capture it once at the top and pass it around. Every call creates a new array.`,
		Tags:     []string{"performance", "reuse", "optimization"},
	},
	{
		Category: "performance_principles",
		Title:    "Move heavy or repeatable work into background jobs",
		Content:  `Offload complex calculations (pathfinding, large data processing) to system.run() spread across ticks so no single tick is blocked.`,
		Tags:     []string{"performance", "background", "time-slicing"},
	},
	{
		Category: "performance_principles",
		Title:    "Keep background jobs bounded so they yield regularly",
		Content:  `When splitting work across ticks, process a fixed chunk size per tick so the job eventually completes and doesn't starve other systems.

Example:
let index = 0;
const chunkSize = 50;
function processBatch() {
    const end = Math.min(index + chunkSize, items.length);
    for (; index < end; index++) { /* process item */ }
    if (index < items.length) system.run(processBatch);
}`,
		Tags: []string{"performance", "background", "time-slicing"},
	},
	{
		Category: "performance_principles",
		Title:    "Split large tasks into smaller chunks",
		Content:  `If a single operation takes more than a few milliseconds, break it into smaller pieces and schedule them across multiple ticks with system.run(). This prevents watchdog spikes.`,
		Tags:     []string{"performance", "time-slicing", "watchdog"},
	},
	{
		Category: "performance_principles",
		Title:    "Stagger periodic work so everything does not fire at once",
		Content:  `If you have multiple system.runInterval calls, offset their starting ticks so they don't all execute in the same tick, causing a CPU spike.`,
		Tags:     []string{"performance", "staggering", "optimization"},
	},
	{
		Category: "performance_principles",
		Title:    "Batch or debounce saves and writes",
		Content:  `Accumulate state changes and write them in batches rather than saving on every mutation. This reduces I/O overhead and improves throughput.`,
		Tags:     []string{"performance", "batching", "debounce"},
	},
	{
		Category: "performance_principles",
		Title:    "Track the earliest time a task can run again",
		Content:  `Instead of using a raw interval, store the next-allowed tick and skip execution until that tick arrives. This gives you precise control over cooldowns.`,
		Tags:     []string{"performance", "cooldowns", "scheduling"},
	},
	{
		Category: "performance_principles",
		Title:    "Avoid repeating expensive template, parsing, or formatting work",
		Content:  `Pre-build formatted strings, parse templates once, and cache compiled patterns. Don't re-parse or re-format the same thing every tick.`,
		Tags:     []string{"performance", "caching", "templates"},
	},
	{
		Category: "performance_principles",
		Title:    "Keep immediate permission checks and cancellation logic synchronous",
		Content:  `Use beforeEvents for cancellation scenarios. These run synchronously before the action occurs and are the only place you can cancel events like chat messages or block breaking.`,
		Tags:     []string{"performance", "events", "beforeEvents"},
	},
	{
		Category: "performance_principles",
		Title:    "Prefer one shared scheduler over many separate intervals",
		Content:  `Maintain a single tick loop that dispatches to registered tasks rather than creating dozens of system.runInterval calls. This centralizes timing and makes staggering easier.`,
		Tags:     []string{"performance", "scheduling", "architecture"},
	},
	{
		Category: "general",
		Title:    "Always assume a multiplayer environment",
		Content: `Even during solo local testing, scripts execute on the server-side architecture. Treating the game as single-player is a leading cause of bugs.

Bad:
const player = world.getAllPlayers()[0]; // crashes if no players or in multiplayer

Good:
for (const player of world.getAllPlayers()) {
    player.sendMessage("Server update completed.");
}`,
		Tags: []string{"general", "multiplayer", "safety"},
	},
	{
		Category: "general",
		Title:    "Enforce state cleanup on player disconnect",
		Content: `If you track per-player state (telemetry, scores, timers) in a Map or array, you must clean up when players leave. Failing to do so causes memory leaks.

Example:
world.afterEvents.playerLeave.subscribe((event) => {
    playerStateMap.delete(event.playerId);
    pendingTimers.delete(event.playerId);
});`,
		Tags: []string{"general", "memory", "cleanup"},
	},
	{
		Category: "general",
		Title:    "Implement isValid() guards on deferred callbacks",
		Content: `Asynchronous callbacks (system.runTimeout, promise resolutions) run out-of-sync with world state. An entity may have been destroyed by the time your callback fires.

Guard your deferred code:
system.runTimeout(() => {
    if (player.isValid()) {
        player.teleport({ x: 0, y: 100, z: 0 });
    }
}, 20);`,
		Tags: []string{"general", "safety", "async", "isValid"},
	},
	{
		Category: "general",
		Title:    "Favor afterEvents over beforeEvents",
		Content: `beforeEvents run synchronously before the action and are read-only (except for .cancel). afterEvents run in a deferred pipeline during a read-write state.

Rule of Thumb: Use afterEvents unless you specifically need to cancel an action via event.cancel.`,
		Tags: []string{"general", "events", "afterEvents", "beforeEvents"},
	},
	{
		Category: "performance_optimization",
		Title:    "Respect the Script Performance Watchdog",
		Content: `The Watchdog monitors script health:
- Spike Threshold (100ms default): Flags long-running single-tick execution.
- Hang Threshold (3000ms default): Force-kills execution on infinite loops.

Optimization: Avoid complex calculations in a single tick. Use system.run() to spread work across multiple ticks (time-slicing).`,
		Tags: []string{"optimization", "watchdog", "thresholds"},
	},
	{
		Category: "performance_optimization",
		Title:    "Use native API methods instead of runCommandAsync",
		Content: `Passing strings into the command parser (runCommandAsync / runCommand) introduces parsing overhead.

Bad:
player.runCommandAsync("say Hello!");

Good:
player.sendMessage("Hello!");

Use native methods (entity.addEffect, player.teleport, etc.) instead of command strings wherever possible.`,
		Tags: []string{"optimization", "native-api", "commands"},
	},
	{
		Category: "performance_optimization",
		Title:    "Time-slice intensive work across ticks",
		Content: `Spread heavy loops across multiple ticks to avoid watchdog spikes. Process a fixed chunk per tick until all work is done.

Example:
let i = 0;
function processChunk() {
    const end = Math.min(i + 10, allEntities.length);
    for (; i < end; i++) { /* process */ }
    if (i < allEntities.length) system.run(processChunk);
}
processChunk();`,
		Tags: []string{"optimization", "time-slicing", "watchdog"},
	},
	{
		Category: "patterns",
		Title:    "Single-tick dispatcher task scheduler",
		Content: `A lightweight single-tick-dispatcher task scheduler for Minecraft Bedrock Script API. No dependencies beyond @minecraft/server.

## Exports

| Export | Signature | Purpose |
|--------|-----------|---------|
| registerBackgroundTask | (id, intervalTicks, run, initialOffsetTicks?) => void | Register a recurring task that runs at most every N ticks. Supports dynamic intervals via a callback. |
| registerEveryTickTask | (id, run) => void | Register a task that runs every single tick. |

## Architecture

One global system.runInterval at 1 tick drives everything:
- EveryTickTask: runs unconditionally every tick
- BackgroundTask: runs only when currentTick >= nextDueTick; reschedules using currentTick + interval
- Only 2 background tasks fire per tick (configurable via MAX_BACKGROUND_TASKS_PER_TICK)
- Tasks are identified by string id — registering the same ID replaces the old task
- Each task is wrapped in try/catch — failures are logged but don't crash the loop
- Dispatcher lazily starts on the first registerBackgroundTask or registerEveryTickTask call

## Copyable implementation

import { system } from "@minecraft/server";

type TickInterval = number | (() => number);

type BackgroundTask = {
  id: string;
  run: () => void;
  intervalTicks: TickInterval;
  initialOffsetTicks?: number;
  nextDueTick: number;
};

type EveryTickTask = {
  id: string;
  run: () => void;
};

const MAX_BACKGROUND_TASKS_PER_TICK = 2;

const backgroundTasks = new Map<string, BackgroundTask>();
const everyTickTasks = new Map<string, EveryTickTask>();
let dispatcherStarted = false;

function normalizeTicks(value: number): number {
  return Math.max(1, Math.floor(value));
}

function resolveInterval(task: BackgroundTask): number {
  return normalizeTicks(typeof task.intervalTicks === "function" ? task.intervalTicks() : task.intervalTicks);
}

function safeRun(id: string, run: () => void): void {
  try {
    run();
  } catch (error) {
    console.warn("[Scheduler] Background task " + id + " failed: " + error);
  }
}

function runBackgroundSchedulerTick(): void {
  for (const task of everyTickTasks.values()) {
    safeRun(task.id, task.run);
  }

  let started = 0;
  for (const task of backgroundTasks.values()) {
    if (started >= MAX_BACKGROUND_TASKS_PER_TICK) break;
    if (system.currentTick < task.nextDueTick) continue;

    safeRun(task.id, task.run);
    task.nextDueTick = system.currentTick + resolveInterval(task);
    started++;
  }
}

function ensureBackgroundSchedulerStarted(): void {
  if (dispatcherStarted) return;
  dispatcherStarted = true;
  system.runInterval(runBackgroundSchedulerTick, 1);
}

export function registerBackgroundTask(id: string, intervalTicks: TickInterval, run: () => void, initialOffsetTicks = 0): void {
  const offset = Math.max(0, Math.floor(initialOffsetTicks));
  backgroundTasks.set(id, {
    id,
    run,
    intervalTicks,
    initialOffsetTicks: offset,
    nextDueTick: system.currentTick + offset,
  });
  ensureBackgroundSchedulerStarted();
}

export function registerEveryTickTask(id: string, run: () => void): void {
  everyTickTasks.set(id, { id, run });
  ensureBackgroundSchedulerStarted();
}

## Usage example

registerBackgroundTask("my-task", 20, () => {
  // runs every ~1 second
  doSomething();
});

registerBackgroundTask("dynamic-task", () => {
  // interval can vary each time
  return someConfig.intervalTicks;
}, () => {
  doDynamicThing();
}, 10); // initial 10-tick offset

registerEveryTickTask("urgent", () => {
  // runs every single tick
});

## Known limitations
- Synchronous only — tasks are plain () => void. Relies on tasks being very fast (<1ms each).
- No overlap protection — if the same task fires again before the previous invocation completes, they'll overlap. In practice, intervals >= 5 ticks and fast tasks avoid this.
- FIFO within budget — if multiple tasks are due and the budget is 2, they're processed first-in-first-out, not sorted by priority.
- No system.runJob support — heavier per-player/per-entry work needs that added as a separate pattern on top.`,
		Tags: []string{"patterns", "scheduler", "performance", "time-slicing", "system.run", "architecture"},
	},
}

func handleGetBestPractices(args GetBestPracticesInput) (*mcp.ToolResponse, error) {
	category := strings.ToLower(args.Category)
	search := strings.ToLower(strings.TrimSpace(args.Search))

	var results []BestPracticeEntry
	for _, entry := range allBestPractices {
		if category != "" && category != "all" && entry.Category != category {
			continue
		}
		if search != "" {
			titleLower := strings.ToLower(entry.Title)
			contentLower := strings.ToLower(entry.Content)
			tagMatch := false
			for _, tag := range entry.Tags {
				if strings.Contains(strings.ToLower(tag), search) {
					tagMatch = true
					break
				}
			}
			if !strings.Contains(titleLower, search) && !strings.Contains(contentLower, search) && !tagMatch {
				continue
			}
		}
		results = append(results, entry)
	}

	output := GetBestPracticesOutput{
		Results:       results,
		TotalCount:    len(allBestPractices),
		FilteredCount: len(results),
	}

	jsonOut, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return mcp.NewToolResponse(mcp.NewTextContent(fmt.Sprintf("Error serializing output: %v", err))), nil
	}
	return mcp.NewToolResponse(mcp.NewTextContent(string(jsonOut))), nil
}
