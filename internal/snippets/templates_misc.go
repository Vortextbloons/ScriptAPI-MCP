package snippets

func init() {
	AllDefinitions = append(AllDefinitions,
		SnippetDefinition{
			Type:        "equipment.equipment_scanner",
			Description: "Iterate all equipment slots and collect item data with slot-safe scanning",
			Category:    "equipment",
			Complexity:  "simple",
			Tags:        []string{"equipment", "inventory", "scanning", "player", "items"},
			Related:     []string{"runtime.profile_cache"},
			Notes:       []string{"Safely iterates all equipment slots with try/catch", "Use scanEquipmentWithData() to extract custom data per item", "getMainhandItem() returns item and equippable component"},
			JavaScript: `const ALL_EQUIPMENT_SLOTS = [
  EquipmentSlot.Head,
  EquipmentSlot.Chest,
  EquipmentSlot.Legs,
  EquipmentSlot.Feet,
  EquipmentSlot.Mainhand,
  EquipmentSlot.Offhand,
];

function scanEquipment(player) {
  const equippable = player.getComponent(EntityComponentTypes.Equippable);
  if (!equippable) return [];

  const results = [];
  for (const slot of ALL_EQUIPMENT_SLOTS) {
    try {
      const item = equippable.getEquipment(slot);
      if (item) results.push({ slot, item });
    } catch {
    }
  }
  return results;
}

function getMainhandItem(player) {
  const equippable = player.getComponent(EntityComponentTypes.Equippable);
  const item = equippable?.getEquipment(EquipmentSlot.Mainhand);
  return equippable && item ? { item, equippable } : undefined;
}

function scanEquipmentWithData(player, reader) {
  const results = [];
  for (const { slot, item } of scanEquipment(player)) {
    const data = reader(item);
    if (data !== undefined) results.push({ slot, data });
  }
  return results;
}`,
			TypeScript: `export const ALL_EQUIPMENT_SLOTS = [
  EquipmentSlot.Head,
  EquipmentSlot.Chest,
  EquipmentSlot.Legs,
  EquipmentSlot.Feet,
  EquipmentSlot.Mainhand,
  EquipmentSlot.Offhand,
];

export interface EquipmentEntry {
  slot: EquipmentSlot;
  item: ItemStack;
}

export function scanEquipment(player: Player): EquipmentEntry[] {
  const equippable = player.getComponent(EntityComponentTypes.Equippable) as any;
  if (!equippable) return [];

  const results: EquipmentEntry[] = [];
  for (const slot of ALL_EQUIPMENT_SLOTS) {
    try {
      const item = equippable.getEquipment(slot) as ItemStack | undefined;
      if (item) results.push({ slot, item });
    } catch {
    }
  }
  return results;
}

export function getMainhandItem(player: Player): { item: ItemStack; equippable: any } | undefined {
  const equippable = player.getComponent(EntityComponentTypes.Equippable) as any;
  const item = equippable?.getEquipment(EquipmentSlot.Mainhand) as ItemStack | undefined;
  return equippable && item ? { item, equippable } : undefined;
}

export function scanEquipmentWithData<T>(
  player: Player,
  reader: (item: ItemStack) => T | undefined,
): { slot: EquipmentSlot; data: T }[] {
  const results: { slot: EquipmentSlot; data: T }[] = [];
  for (const { slot, item } of scanEquipment(player)) {
    const data = reader(item);
    if (data !== undefined) results.push({ slot, data });
  }
  return results;
}`,
			JSImports:       []string{"EntityComponentTypes", "EquipmentSlot"},
			TSImports:       []string{},
			TSTypeImports:   []string{"ItemStack", "Player"},
			RequiredModules: []string{"@minecraft/server"},
		},
		SnippetDefinition{
			Type:        "item.lore_builder",
			Description: "Color-coded lore text builder with word wrap, block replacement, and pattern-based extraction",
			Category:    "item",
			Complexity:  "simple",
			Tags:        []string{"lore", "items", "text", "formatting", "color-codes"},
			Related:     []string{"equipment.equipment_scanner"},
			Notes:       []string{"Word-wrap with configurable width (default 34 chars)", "buildLoreBlock() creates structured lore sections", "mergeLore() replaces existing blocks by header pattern"},
			JavaScript: `const DEFAULT_WRAP_WIDTH = 34;

function wrapText(text, width = DEFAULT_WRAP_WIDTH) {
  const words = text.trim().split(/\s+/);
  const lines = [];
  let current = '';
  for (const word of words) {
    const next = current.length === 0 ? word : ` + "`" + `${current} ${word}` + "`" + `;
    if (next.length > width && current.length > 0) {
      lines.push(current);
      current = word;
    } else {
      current = next;
    }
  }
  if (current.length > 0) lines.push(current);
  return lines;
}

function buildLoreBlock(header, entries, options) {
  const wrap = options?.wrapWidth ?? DEFAULT_WRAP_WIDTH;
  const indent = options?.indent ?? '§7  ';
  const lines = [header];

  for (let i = 0; i < entries.length; i++) {
    const entry = entries[i];
    if (entry.label) {
      lines.push(` + "`" + `${entry.color ?? '§7'}${entry.label}` + "`" + `);
    }
    const descLines = wrapText(entry.description, wrap);
    for (const line of descLines) {
      lines.push(` + "`" + `${indent}${line}` + "`" + `);
    }
    if (options?.locked && i === entries.length - 1 && options.boundLabel) {
      lines.push(` + "`" + `${indent}§8${options.boundLabel}` + "`" + `);
    }
  }

  return lines;
}

function mergeLore(existing, newBlock) {
  const headerIndex = existing.findIndex(line => /^§[0-9a-f]/.test(line) && !line.startsWith('§r'));
  if (headerIndex === -1) {
    return [...existing, ...newBlock];
  }
  let endIndex = headerIndex + 1;
  while (endIndex < existing.length && existing[endIndex].startsWith('§7') || existing[endIndex].startsWith('§8')) {
    endIndex++;
  }
  return [...existing.slice(0, headerIndex), ...newBlock, ...existing.slice(endIndex)];
}

function stripLoreBlocks(lines, headerPattern) {
  const blocks = [];
  const clean = [];
  let i = 0;
  while (i < lines.length) {
    if (headerPattern.test(lines[i])) {
      const block = [lines[i]];
      i++;
      while (i < lines.length && !headerPattern.test(lines[i])) {
        block.push(lines[i]);
        i++;
      }
      blocks.push(block);
    } else {
      clean.push(lines[i]);
      i++;
    }
  }
  return { clean, blocks };
}`,
			TypeScript: `const DEFAULT_WRAP_WIDTH = 34;

export function wrapText(text: string, width = DEFAULT_WRAP_WIDTH): string[] {
  const words = text.trim().split(/\s+/);
  const lines: string[] = [];
  let current = '';
  for (const word of words) {
    const next = current.length === 0 ? word : ` + "`" + `${current} ${word}` + "`" + `;
    if (next.length > width && current.length > 0) {
      lines.push(current);
      current = word;
    } else {
      current = next;
    }
  }
  if (current.length > 0) lines.push(current);
  return lines;
}

export function buildLoreBlock(
  header: string,
  entries: { label?: string; description: string; color?: string }[],
  options?: { wrapWidth?: number; indent?: string; locked?: boolean; boundLabel?: string },
): string[] {
  const wrap = options?.wrapWidth ?? DEFAULT_WRAP_WIDTH;
  const indent = options?.indent ?? '§7  ';
  const lines: string[] = [header];

  for (let i = 0; i < entries.length; i++) {
    const entry = entries[i];
    if (entry.label) {
      lines.push(` + "`" + `${entry.color ?? '§7'}${entry.label}` + "`" + `);
    }
    const descLines = wrapText(entry.description, wrap);
    for (const line of descLines) {
      lines.push(` + "`" + `${indent}${line}` + "`" + `);
    }
    if (options?.locked && i === entries.length - 1 && options.boundLabel) {
      lines.push(` + "`" + `${indent}§8${options.boundLabel}` + "`" + `);
    }
  }

  return lines;
}

export function mergeLore(existing: string[], newBlock: string[]): string[] {
  const headerIndex = existing.findIndex(line => /^§[0-9a-f]/.test(line) && !line.startsWith('§r'));
  if (headerIndex === -1) {
    return [...existing, ...newBlock];
  }
  let endIndex = headerIndex + 1;
  while (endIndex < existing.length && existing[endIndex].startsWith('§7') || existing[endIndex].startsWith('§8')) {
    endIndex++;
  }
  return [...existing.slice(0, headerIndex), ...newBlock, ...existing.slice(endIndex)];
}

export function stripLoreBlocks(lines: string[], headerPattern: RegExp): { clean: string[]; blocks: string[][] } {
  const blocks: string[][] = [];
  const clean: string[] = [];
  let i = 0;
  while (i < lines.length) {
    if (headerPattern.test(lines[i])) {
      const block: string[] = [lines[i]];
      i++;
      while (i < lines.length && !headerPattern.test(lines[i])) {
        block.push(lines[i]);
        i++;
      }
      blocks.push(block);
    } else {
      clean.push(lines[i]);
      i++;
    }
  }
  return { clean, blocks };
}`,
			JSImports:       []string{},
			TSImports:       []string{},
			TSTypeImports:   []string{},
			RequiredModules: []string{"@minecraft/server"},
		},
		SnippetDefinition{
			Type:        "balance.scaled_value",
			Description: "Formula-based scaling with weighted random level rolling for game balance systems",
			Category:    "balance",
			Complexity:  "moderate",
			Tags:        []string{"scaling", "balance", "levels", "random", "weighted", "formula"},
			Related:     []string{},
			Notes:       []string{"Linear scaling with base + perLevel formula", "Weighted random rolling favors lower levels (squared distance)", "rollEffectLevel uses bonus rolls and keeps best result"},
			JavaScript: `function getScaledValue(axis, level) {
  const clamped = Math.max(1, Math.floor(level));
  return axis.base + (clamped - 1) * axis.perLevel;
}

function getScaledValueClamped(axis, level, min, max) {
  return Math.max(min, Math.min(max, getScaledValue(axis, level)));
}

function getScaledValues(axes, level) {
  const results = {};
  for (const [key, axis] of Object.entries(axes)) {
    results[key] = getScaledValue(axis, level);
  }
  return results;
}

function rollWeightedLevel(maxLevel) {
  const max = Math.max(1, Math.floor(maxLevel));
  let totalWeight = 0;
  const weights = [];
  for (let level = 1; level <= max; level++) {
    const weight = Math.max(1, (max - level + 1) ** 2);
    weights.push(weight);
    totalWeight += weight;
  }
  let roll = Math.random() * totalWeight;
  for (let i = 0; i < weights.length; i++) {
    roll -= weights[i];
    if (roll < 0) return i + 1;
  }
  return max;
}

function rollEffectLevel(maxLevel, tierBonusRolls = 1) {
  let best = 1;
  for (let i = 0; i < Math.max(1, tierBonusRolls); i++) {
    const rolled = rollWeightedLevel(maxLevel);
    if (rolled > best) best = rolled;
    if (best >= maxLevel) break;
  }
  return best;
}

function clamp(value, min, max) {
  return Math.max(min, Math.min(max, value));
}`,
			TypeScript: `export interface ScalingAxis {
  label: string;
  base: number;
  perLevel: number;
  unit: string;
}

export function getScaledValue(axis: ScalingAxis, level: number): number {
  const clamped = Math.max(1, Math.floor(level));
  return axis.base + (clamped - 1) * axis.perLevel;
}

export function getScaledValueClamped(axis: ScalingAxis, level: number, min: number, max: number): number {
  return Math.max(min, Math.min(max, getScaledValue(axis, level)));
}

export function getScaledValues(axes: Record<string, ScalingAxis>, level: number): Record<string, number> {
  const results: Record<string, number> = {};
  for (const [key, axis] of Object.entries(axes)) {
    results[key] = getScaledValue(axis, level);
  }
  return results;
}

export function rollWeightedLevel(maxLevel: number): number {
  const max = Math.max(1, Math.floor(maxLevel));
  let totalWeight = 0;
  const weights: number[] = [];
  for (let level = 1; level <= max; level++) {
    const weight = Math.max(1, (max - level + 1) ** 2);
    weights.push(weight);
    totalWeight += weight;
  }
  let roll = Math.random() * totalWeight;
  for (let i = 0; i < weights.length; i++) {
    roll -= weights[i];
    if (roll < 0) return i + 1;
  }
  return max;
}

export function rollEffectLevel(maxLevel: number, tierBonusRolls = 1): number {
  let best = 1;
  for (let i = 0; i < Math.max(1, tierBonusRolls); i++) {
    const rolled = rollWeightedLevel(maxLevel);
    if (rolled > best) best = rolled;
    if (best >= maxLevel) break;
  }
  return best;
}

export function clamp(value: number, min: number, max: number): number {
  return Math.max(min, Math.min(max, value));
}`,
			JSImports:       []string{},
			TSImports:       []string{},
			TSTypeImports:   []string{},
			RequiredModules: []string{"@minecraft/server"},
		},
		SnippetDefinition{
			Type:        "command.custom_slash_command",
			Description: "Custom command registration with permission levels using the unstable registry API",
			Category:    "command",
			Complexity:  "moderate",
			Tags:        []string{"commands", "slash", "permissions", "chat", "registry"},
			Related:     []string{},
			Notes:       []string{"Uses unstable custom command registry API from @minecraft/server", "Permission levels: Any, GameDirectors, Admin, Host, Owner", "Wrap form.show() calls in system.run() within handlers"},
			JavaScript: `const CommandPermissionLevel = { Any: 0, GameDirectors: 1, Admin: 2, Host: 3, Owner: 4 };
const CustomCommandStatus = { Success: 0, Failure: 1 };

const registeredCommands = new Map();

function isPlayer(entity) {
  return !!entity && typeof entity === 'object' && entity.typeId === 'minecraft:player';
}

function getPlayerFromOrigin(origin) {
  return isPlayer(origin.sourceEntity) ? origin.sourceEntity : undefined;
}

function safeRegisterCommand(registry, command, handler) {
  try {
    registry.registerCommand(command, (origin, args) => {
      const player = getPlayerFromOrigin(origin);
      if (!player) return { status: CustomCommandStatus.Failure, message: 'Command can only be used by a player.' };
      return handler(player, args);
    });
  } catch (error) {
    console.warn(` + "`" + `[Cmd] Skipping '${command.name}': ${error}` + "`" + `);
  }
}

function defineCommand(name, description, handler, options) {
  registeredCommands.set(name, handler);
  system.beforeEvents.startup.subscribe((startup) => {
    safeRegisterCommand(startup.customCommandRegistry, {
      name,
      description,
      permissionLevel: options?.permissionLevel ?? CommandPermissionLevel.Admin,
      cheatsRequired: options?.cheatsRequired ?? false,
    }, handler);
  });
}

function defineCommands(commands) {
  for (const cmd of commands) {
    defineCommand(cmd.name, cmd.description, cmd.handler, cmd.options);
  }
}

function success(message) {
  return { status: CustomCommandStatus.Success, message };
}

function failure(message) {
  return { status: CustomCommandStatus.Failure, message };
}`,
			TypeScript: `const CommandPermissionLevel = { Any: 0, GameDirectors: 1, Admin: 2, Host: 3, Owner: 4 } as const;
const CustomCommandStatus = { Success: 0, Failure: 1 } as const;

interface CommandDef {
  name: string;
  description: string;
  permissionLevel: number;
  cheatsRequired: boolean;
}

type CommandHandler = (player: Player, args: unknown[]) => { status: number; message?: string };

const registeredCommands = new Map<string, CommandHandler>();

function isPlayer(entity: unknown): entity is Player {
  return !!entity && typeof entity === 'object' && (entity as Player).typeId === 'minecraft:player';
}

function getPlayerFromOrigin(origin: { sourceEntity?: unknown }): Player | undefined {
  return isPlayer(origin.sourceEntity) ? origin.sourceEntity : undefined;
}

function safeRegisterCommand(
  registry: any,
  command: CommandDef,
  handler: CommandHandler,
): void {
  try {
    registry.registerCommand(command, (origin: any, args: unknown[]) => {
      const player = getPlayerFromOrigin(origin);
      if (!player) return { status: CustomCommandStatus.Failure, message: 'Command can only be used by a player.' };
      return handler(player, args);
    });
  } catch (error) {
    console.warn(` + "`" + `[Cmd] Skipping '${command.name}': ${error}` + "`" + `);
  }
}

export function defineCommand(name: string, description: string, handler: CommandHandler, options?: {
  permissionLevel?: number;
  cheatsRequired?: boolean;
}): void {
  registeredCommands.set(name, handler);
  (system as any).beforeEvents.startup.subscribe((startup: any) => {
    safeRegisterCommand(startup.customCommandRegistry, {
      name,
      description,
      permissionLevel: options?.permissionLevel ?? CommandPermissionLevel.Admin,
      cheatsRequired: options?.cheatsRequired ?? false,
    }, handler);
  });
}

export function defineCommands(commands: { name: string; description: string; handler: CommandHandler; options?: Parameters<typeof defineCommand>[3] }[]): void {
  for (const cmd of commands) {
    defineCommand(cmd.name, cmd.description, cmd.handler, cmd.options);
  }
}

export function success(message?: string) {
  return { status: CustomCommandStatus.Success, message };
}

export function failure(message: string) {
  return { status: CustomCommandStatus.Failure, message };
}`,
			JSImports:       []string{"system"},
			TSImports:       []string{"system"},
			TSTypeImports:   []string{"Player"},
			RequiredModules: []string{"@minecraft/server"},
		},
	)
}
