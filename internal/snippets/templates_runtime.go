package snippets

func init() {
	AllDefinitions = append(AllDefinitions,
		SnippetDefinition{
			Type:        "runtime.plugin_registry",
			Description: "Event-dispatching plugin system with typed runtime hooks for combat, interaction, and tick-based effects",
			Category:    "runtime",
			Complexity:  "complex",
			Tags:        []string{"events", "combat", "player", "hooks", "dispatch"},
			Related:     []string{"runtime.background_scheduler", "runtime.cooldown_manager"},
			Notes:       []string{"Requires early startup initialization", "Subscribe all events at server startup using initializePluginEngine()"},
			JavaScript: `const runtimeRegistry = new Map();
const inputRuntimes = [];

function registerRuntime(key, rt) {
  runtimeRegistry.set(key, rt);
  if (rt.onInput) inputRuntimes.push(rt);
}

function dispatchMeleeHit(attacker, victim, effects, event) {
  for (const effect of effects) {
    runtimeRegistry.get(effect.key)?.onMeleeHit?.(attacker, victim, effect, event);
  }
}

function dispatchHurt(defender, effects, event) {
  for (const effect of effects) {
    runtimeRegistry.get(effect.key)?.onHurt?.(defender, effect, event);
  }
}

function dispatchProjectileHitEntity(attacker, effects, event) {
  for (const effect of effects) {
    runtimeRegistry.get(effect.key)?.onProjectileHitEntity?.(attacker, effect, event);
  }
}

function dispatchProjectileHitBlock(attacker, effects, event) {
  for (const effect of effects) {
    runtimeRegistry.get(effect.key)?.onProjectileHitBlock?.(attacker, effect, event);
  }
}

function dispatchBlockBreak(player, effects, event) {
  for (const effect of effects) {
    runtimeRegistry.get(effect.key)?.onBlockBreak?.(player, effect, event);
  }
}

function dispatchBeforeBlockBreak(player, effects, event) {
  for (const effect of effects) {
    runtimeRegistry.get(effect.key)?.onBeforeBlockBreak?.(player, effect, event);
  }
}

function dispatchBlockUse(player, effects, event) {
  for (const effect of effects) {
    runtimeRegistry.get(effect.key)?.onBlockUse?.(player, effect, event);
  }
}

function dispatchEntityKill(player, effects, event) {
  for (const effect of effects) {
    runtimeRegistry.get(effect.key)?.onEntityKill?.(player, effect, event);
  }
}

function processTickEffects() {
  for (const rt of runtimeRegistry.values()) {
    if (!rt.onTick) continue;
    if (rt.hasTickWork?.() || !rt.tickEffectKeys) {
      rt.onTick();
    }
  }
}

function processMoveEffects(player, activeKeys) {
  for (const rt of runtimeRegistry.values()) {
    if (!rt.onMove) continue;
    if (rt.hasMoveWork?.(player) || !rt.moveEffectKeys || rt.moveEffectKeys.some(k => activeKeys.has(k))) {
      rt.onMove(player);
    }
  }
}

function dispatchInput(event) {
  for (const rt of inputRuntimes) {
    rt.onInput?.(event);
  }
}

function initializePluginEngine(collectEffects) {
  world.afterEvents.entityHurt.subscribe(event => system.run(() => {
    const { damagingEntity, cause } = event.damageSource;
    if (cause === EntityDamageCause.entityAttack && damagingEntity?.typeId === 'minecraft:player') {
      const attacker = damagingEntity;
      dispatchMeleeHit(attacker, event.hurtEntity, collectEffects(attacker), event);
    }
    if (event.hurtEntity.typeId === 'minecraft:player') {
      const defender = event.hurtEntity;
      dispatchHurt(defender, collectEffects(defender), event);
    }
  }));

  world.afterEvents.projectileHitEntity.subscribe(event => system.run(() => {
    const source = event.source;
    if (source?.typeId !== 'minecraft:player') return;
    dispatchProjectileHitEntity(source, collectEffects(source), event);
  }));

  world.afterEvents.projectileHitBlock.subscribe(event => system.run(() => {
    const source = event.source;
    if (source?.typeId !== 'minecraft:player') return;
    dispatchProjectileHitBlock(source, collectEffects(source), event);
  }));

  world.afterEvents.playerButtonInput.subscribe(event => system.run(() => dispatchInput(event)));

  world.beforeEvents.playerInteractWithBlock.subscribe(event => system.run(() => {
    dispatchBlockUse(event.player, collectEffects(event.player), event);
  }));

  world.afterEvents.playerBreakBlock.subscribe(event => system.run(() => {
    dispatchBlockBreak(event.player, collectEffects(event.player), event);
  }));

  world.beforeEvents.playerBreakBlock.subscribe(event => {
    dispatchBeforeBlockBreak(event.player, collectEffects(event.player), event);
  });

  world.afterEvents.entityDie.subscribe(event => system.run(() => {
    const killer = event.damageSource.damagingEntity;
    if (killer?.typeId === 'minecraft:player') {
      dispatchEntityKill(killer, collectEffects(killer), event);
    }
  }));
}`,
			TypeScript: `export interface PluginEntry {
  key: string;
  level: number;
}

export interface PluginRuntime {
  tickEffectKeys?: readonly string[];
  moveEffectKeys?: readonly string[];
  hasTickWork?(): boolean;
  hasMoveWork?(player: Player): boolean;
  onMeleeHit?(player: Player, victim: Entity, effect: PluginEntry, event: EntityHurtAfterEvent): void;
  onProjectileHitEntity?(attacker: Player, effect: PluginEntry, event: ProjectileHitEntityAfterEvent): void;
  onProjectileHitBlock?(attacker: Player, effect: PluginEntry, event: ProjectileHitBlockAfterEvent): void;
  onBlockBreak?(player: Player, effect: PluginEntry, event: PlayerBreakBlockAfterEvent): void;
  onBeforeBlockBreak?(player: Player, effect: PluginEntry, event: PlayerBreakBlockBeforeEvent): void;
  onBlockUse?(player: Player, effect: PluginEntry, event: PlayerInteractWithBlockBeforeEvent): void;
  onTick?(): void;
  onMove?(player: Player): void;
  onHurt?(defender: Player, effect: PluginEntry, event: EntityHurtAfterEvent): void;
  onEntityKill?(player: Player, effect: PluginEntry, event: EntityDieAfterEvent): void;
  onInput?(event: PlayerButtonInputAfterEvent): void;
}

export const runtimeRegistry = new Map<string, PluginRuntime>();
const inputRuntimes: PluginRuntime[] = [];

export function registerRuntime(key: string, rt: PluginRuntime): void {
  runtimeRegistry.set(key, rt);
  if (rt.onInput) inputRuntimes.push(rt);
}

export function dispatchMeleeHit(attacker: Player, victim: Entity, effects: PluginEntry[], event: EntityHurtAfterEvent): void {
  for (const effect of effects) {
    runtimeRegistry.get(effect.key)?.onMeleeHit?.(attacker, victim, effect, event);
  }
}

export function dispatchHurt(defender: Player, effects: PluginEntry[], event: EntityHurtAfterEvent): void {
  for (const effect of effects) {
    runtimeRegistry.get(effect.key)?.onHurt?.(defender, effect, event);
  }
}

export function dispatchProjectileHitEntity(attacker: Player, effects: PluginEntry[], event: ProjectileHitEntityAfterEvent): void {
  for (const effect of effects) {
    runtimeRegistry.get(effect.key)?.onProjectileHitEntity?.(attacker, effect, event);
  }
}

export function dispatchProjectileHitBlock(attacker: Player, effects: PluginEntry[], event: ProjectileHitBlockAfterEvent): void {
  for (const effect of effects) {
    runtimeRegistry.get(effect.key)?.onProjectileHitBlock?.(attacker, effect, event);
  }
}

export function dispatchBlockBreak(player: Player, effects: PluginEntry[], event: PlayerBreakBlockAfterEvent): void {
  for (const effect of effects) {
    runtimeRegistry.get(effect.key)?.onBlockBreak?.(player, effect, event);
  }
}

export function dispatchBeforeBlockBreak(player: Player, effects: PluginEntry[], event: PlayerBreakBlockBeforeEvent): void {
  for (const effect of effects) {
    runtimeRegistry.get(effect.key)?.onBeforeBlockBreak?.(player, effect, event);
  }
}

export function dispatchBlockUse(player: Player, effects: PluginEntry[], event: PlayerInteractWithBlockBeforeEvent): void {
  for (const effect of effects) {
    runtimeRegistry.get(effect.key)?.onBlockUse?.(player, effect, event);
  }
}

export function dispatchEntityKill(player: Player, effects: PluginEntry[], event: EntityDieAfterEvent): void {
  for (const effect of effects) {
    runtimeRegistry.get(effect.key)?.onEntityKill?.(player, effect, event);
  }
}

export function processTickEffects(): void {
  for (const rt of runtimeRegistry.values()) {
    if (!rt.onTick) continue;
    if (rt.hasTickWork?.() || !rt.tickEffectKeys) {
      rt.onTick();
    }
  }
}

export function processMoveEffects(player: Player, activeKeys: Set<string>): void {
  for (const rt of runtimeRegistry.values()) {
    if (!rt.onMove) continue;
    if (rt.hasMoveWork?.(player) || !rt.moveEffectKeys || rt.moveEffectKeys.some(k => activeKeys.has(k))) {
      rt.onMove(player);
    }
  }
}

export function dispatchInput(event: PlayerButtonInputAfterEvent): void {
  for (const rt of inputRuntimes) {
    rt.onInput?.(event);
  }
}

export function initializePluginEngine(collectEffects: (player: Player) => PluginEntry[]): void {
  world.afterEvents.entityHurt.subscribe(event => system.run(() => {
    const { damagingEntity, cause } = event.damageSource;
    if (cause === EntityDamageCause.entityAttack && damagingEntity?.typeId === 'minecraft:player') {
      const attacker = damagingEntity as Player;
      dispatchMeleeHit(attacker, event.hurtEntity, collectEffects(attacker), event);
    }
    if (event.hurtEntity.typeId === 'minecraft:player') {
      const defender = event.hurtEntity as Player;
      dispatchHurt(defender, collectEffects(defender), event);
    }
  }));

  world.afterEvents.projectileHitEntity.subscribe(event => system.run(() => {
    const source = event.source;
    if (source?.typeId !== 'minecraft:player') return;
    dispatchProjectileHitEntity(source as Player, collectEffects(source as Player), event);
  }));

  world.afterEvents.projectileHitBlock.subscribe(event => system.run(() => {
    const source = event.source;
    if (source?.typeId !== 'minecraft:player') return;
    dispatchProjectileHitBlock(source as Player, collectEffects(source as Player), event);
  }));

  world.afterEvents.playerButtonInput.subscribe(event => system.run(() => dispatchInput(event)));

  world.beforeEvents.playerInteractWithBlock.subscribe(event => system.run(() => {
    dispatchBlockUse(event.player, collectEffects(event.player), event);
  }));

  world.afterEvents.playerBreakBlock.subscribe(event => system.run(() => {
    dispatchBlockBreak(event.player, collectEffects(event.player), event);
  }));

  world.beforeEvents.playerBreakBlock.subscribe(event => {
    dispatchBeforeBlockBreak(event.player, collectEffects(event.player), event);
  });

  world.afterEvents.entityDie.subscribe(event => system.run(() => {
    const killer = event.damageSource.damagingEntity;
    if (killer?.typeId === 'minecraft:player') {
      dispatchEntityKill(killer as Player, collectEffects(killer as Player), event);
    }
  }));
}`,
			JSImports:       []string{"system", "world"},
			TSImports:       []string{"system", "world"},
			TSTypeImports:   []string{"Player", "Entity", "EntityHurtAfterEvent", "EntityDieAfterEvent", "PlayerBreakBlockBeforeEvent", "PlayerBreakBlockAfterEvent", "PlayerInteractWithBlockBeforeEvent", "ProjectileHitEntityAfterEvent", "ProjectileHitBlockAfterEvent", "PlayerButtonInputAfterEvent"},
			RequiredModules: []string{"@minecraft/server"},
		},
		SnippetDefinition{
			Type:        "runtime.background_scheduler",
			Description: "Tick-budgeted task scheduler that throttles background work and supports every-tick and interval-based tasks",
			Category:    "runtime",
			Complexity:  "moderate",
			Tags:        []string{"scheduling", "ticks", "timers", "background", "throttling"},
			Related:     []string{"runtime.plugin_registry", "runtime.profile_cache", "runtime.cooldown_manager"},
			Notes:       []string{"Throttles background tasks to maxTasksPerTick (default 3)", "Every-tick tasks run unconditionally each tick", "Safe error isolation per task using try/catch"},
			JavaScript: `let maxTasksPerTick = 3;
const backgroundTasks = new Map();
const everyTickTasks = new Map();
let dispatcherStarted = false;

function normalizeTicks(value) {
  return Math.max(1, Math.floor(value));
}

function resolveInterval(task) {
  return normalizeTicks(typeof task.intervalTicks === 'function' ? task.intervalTicks() : task.intervalTicks);
}

function safeRun(id, run) {
  try {
    run();
  } catch (error) {
    console.warn('[Scheduler] Task ' + id + ' failed: ' + error);
  }
}

function runSchedulerTick() {
  for (const task of everyTickTasks.values()) {
    safeRun(task.id, task.run);
  }

  let started = 0;
  for (const task of backgroundTasks.values()) {
    if (started >= maxTasksPerTick) break;
    if (system.currentTick < task.nextDueTick) continue;
    safeRun(task.id, task.run);
    task.nextDueTick = system.currentTick + resolveInterval(task);
    started += 1;
  }
}

function ensureDispatcherStarted() {
  if (dispatcherStarted) return;
  dispatcherStarted = true;
  system.runInterval(runSchedulerTick, 1);
}

function setMaxTasksPerTick(count) {
  maxTasksPerTick = Math.max(1, Math.floor(count));
}

function registerBackgroundTask(id, intervalTicks, run, initialOffsetTicks = 0) {
  backgroundTasks.set(id, {
    id,
    run,
    intervalTicks,
    nextDueTick: system.currentTick + Math.max(0, Math.floor(initialOffsetTicks)),
  });
  ensureDispatcherStarted();
}

function registerEveryTickTask(id, run) {
  everyTickTasks.set(id, { id, run });
  ensureDispatcherStarted();
}

function unregisterTask(id) {
  backgroundTasks.delete(id);
  everyTickTasks.delete(id);
}`,
			TypeScript: `type TickInterval = number | (() => number);

interface BackgroundTask {
  id: string;
  run: () => void;
  intervalTicks: TickInterval;
  nextDueTick: number;
}

interface EveryTickTask {
  id: string;
  run: () => void;
}

let maxTasksPerTick = 3;
const backgroundTasks = new Map<string, BackgroundTask>();
const everyTickTasks = new Map<string, EveryTickTask>();
let dispatcherStarted = false;

function normalizeTicks(value: number): number {
  return Math.max(1, Math.floor(value));
}

function resolveInterval(task: BackgroundTask): number {
  return normalizeTicks(typeof task.intervalTicks === 'function' ? task.intervalTicks() : task.intervalTicks);
}

function safeRun(id: string, run: () => void): void {
  try {
    run();
  } catch (error) {
    console.warn('[Scheduler] Task ' + id + ' failed: ' + error);
  }
}

function runSchedulerTick(): void {
  for (const task of everyTickTasks.values()) {
    safeRun(task.id, task.run);
  }

  let started = 0;
  for (const task of backgroundTasks.values()) {
    if (started >= maxTasksPerTick) break;
    if (system.currentTick < task.nextDueTick) continue;
    safeRun(task.id, task.run);
    task.nextDueTick = system.currentTick + resolveInterval(task);
    started += 1;
  }
}

function ensureDispatcherStarted(): void {
  if (dispatcherStarted) return;
  dispatcherStarted = true;
  system.runInterval(runSchedulerTick, 1);
}

export function setMaxTasksPerTick(count: number): void {
  maxTasksPerTick = Math.max(1, Math.floor(count));
}

export function registerBackgroundTask(
  id: string,
  intervalTicks: TickInterval,
  run: () => void,
  initialOffsetTicks = 0,
): void {
  backgroundTasks.set(id, {
    id,
    run,
    intervalTicks,
    nextDueTick: system.currentTick + Math.max(0, Math.floor(initialOffsetTicks)),
  });
  ensureDispatcherStarted();
}

export function registerEveryTickTask(id: string, run: () => void): void {
  everyTickTasks.set(id, { id, run });
  ensureDispatcherStarted();
}

export function unregisterTask(id: string): void {
  backgroundTasks.delete(id);
  everyTickTasks.delete(id);
}`,
			JSImports:       []string{"system"},
			TSImports:       []string{"system"},
			TSTypeImports:   []string{},
			RequiredModules: []string{"@minecraft/server"},
		},
		SnippetDefinition{
			Type:        "runtime.profile_cache",
			Description: "Player-scoped data cache with TTL invalidation based on game ticks",
			Category:    "runtime",
			Complexity:  "simple",
			Tags:        []string{"cache", "performance", "player", "ttl", "optimization"},
			Related:     []string{"equipment.equipment_scanner"},
			Notes:       []string{"TTL-based invalidation using system.currentTick", "Use for frequently read computed player data", "Call invalidate() on relevant events to force refresh"},
			JavaScript: `class ProfileCache {
  cache = new Map();
  ttl;

  constructor(ttlTicks = 10) {
    this.ttl = Math.max(1, ttlTicks);
  }

  get(player, builder) {
    const tick = system.currentTick;
    const cached = this.cache.get(player.id);
    if (cached && tick - cached.tick <= this.ttl) {
      return cached.data;
    }
    const data = builder(player);
    this.cache.set(player.id, { tick, data });
    return data;
  }

  invalidate(playerId) {
    this.cache.delete(playerId);
  }

  invalidateAll() {
    this.cache.clear();
  }

  isStale(playerId) {
    const cached = this.cache.get(playerId);
    if (!cached) return true;
    return system.currentTick - cached.tick > this.ttl;
  }
}`,
			TypeScript: `interface CachedEntry<T> {
  tick: number;
  data: T;
}

export class ProfileCache<T> {
  private cache = new Map<string, CachedEntry<T>>();
  private ttl: number;

  constructor(ttlTicks = 10) {
    this.ttl = Math.max(1, ttlTicks);
  }

  get(player: Player, builder: (player: Player) => T): T {
    const tick = system.currentTick;
    const cached = this.cache.get(player.id);
    if (cached && tick - cached.tick <= this.ttl) {
      return cached.data;
    }
    const data = builder(player);
    this.cache.set(player.id, { tick, data });
    return data;
  }

  invalidate(playerId: string): void {
    this.cache.delete(playerId);
  }

  invalidateAll(): void {
    this.cache.clear();
  }

  isStale(playerId: string): boolean {
    const cached = this.cache.get(playerId);
    if (!cached) return true;
    return system.currentTick - cached.tick > this.ttl;
  }
}`,
			JSImports:       []string{"Player", "system"},
			TSImports:       []string{"system"},
			TSTypeImports:   []string{"Player"},
			RequiredModules: []string{"@minecraft/server"},
		},
		SnippetDefinition{
			Type:        "runtime.cooldown_manager",
			Description: "Tick-based cooldown tracking with prefix-based clearing and automatic cleanup",
			Category:    "runtime",
			Complexity:  "simple",
			Tags:        []string{"cooldown", "ticks", "timing", "throttle"},
			Related:     []string{"runtime.background_scheduler"},
			Notes:       []string{"Tick-based timing (not real-time seconds)", "Cleanup runs automatically via startCooldownCleanup()", "Use prefix-based clearing for grouped cooldowns"},
			JavaScript: `const cooldowns = new Map();
let lastCleanupTick = 0;

function isCooldownActive(key) {
  return (cooldowns.get(key) ?? 0) > system.currentTick;
}

function setCooldown(key, ticks) {
  cooldowns.set(key, system.currentTick + ticks);
}

function getRemainingTicks(key) {
  return Math.max(0, (cooldowns.get(key) ?? 0) - system.currentTick);
}

function clearCooldown(key) {
  cooldowns.delete(key);
}

function clearCooldownsByPrefix(prefix) {
  for (const key of cooldowns.keys()) {
    if (key.startsWith(prefix)) {
      cooldowns.delete(key);
    }
  }
}

function clearExpiredCooldowns(now = system.currentTick) {
  for (const [key, readyAt] of cooldowns.entries()) {
    if (readyAt <= now) cooldowns.delete(key);
  }
}

function startCooldownCleanup(interval = 100) {
  system.runInterval(() => {
    clearExpiredCooldowns();
  }, interval);
}`,
			TypeScript: `const cooldowns = new Map<string, number>();
let lastCleanupTick = 0;

export function isCooldownActive(key: string): boolean {
  return (cooldowns.get(key) ?? 0) > system.currentTick;
}

export function setCooldown(key: string, ticks: number): void {
  cooldowns.set(key, system.currentTick + ticks);
}

export function getRemainingTicks(key: string): number {
  return Math.max(0, (cooldowns.get(key) ?? 0) - system.currentTick);
}

export function clearCooldown(key: string): void {
  cooldowns.delete(key);
}

export function clearCooldownsByPrefix(prefix: string): void {
  for (const key of cooldowns.keys()) {
    if (key.startsWith(prefix)) {
      cooldowns.delete(key);
    }
  }
}

export function clearExpiredCooldowns(now: number = system.currentTick): void {
  for (const [key, readyAt] of cooldowns.entries()) {
    if (readyAt <= now) cooldowns.delete(key);
  }
}

export function startCooldownCleanup(interval = 100): void {
  system.runInterval(() => {
    clearExpiredCooldowns();
  }, interval);
}`,
			JSImports:       []string{"system"},
			TSImports:       []string{"system"},
			TSTypeImports:   []string{},
			RequiredModules: []string{"@minecraft/server"},
		},
	)
}
