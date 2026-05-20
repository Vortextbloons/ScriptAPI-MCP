package snippets

func init() {
	AllDefinitions = append(AllDefinitions,
		SnippetDefinition{
			Type:        "ui.action_form_wizard",
			Description: "Multi-step modal state machine for action forms with push-based dynamic step injection",
			Category:    "ui",
			Complexity:  "complex",
			Tags:        []string{"forms", "ui", "multi-step", "wizard", "state-machine"},
			Related:     []string{"interaction.item_interaction_handler"},
			Notes:       []string{"Uses @minecraft/server-ui module", "Supports push-based dynamic step injection via push action kind", "Async form.show() requires system.run() wrapping"},
			JavaScript: `import { ActionFormData, ActionFormResponse } from "@minecraft/server-ui";

function runWizard(
  player,
  initialState,
  steps,
  onComplete,
  onCancel,
) {
  runStep(0, player, initialState, steps, onComplete, onCancel);
}

function runStep(
  index,
  player,
  state,
  steps,
  onComplete,
  onCancel,
) {
  if (index >= steps.length) {
    onComplete(state, player);
    return;
  }

  const step = steps[index];
  const form = new ActionFormData().title(step.title);
  if (step.body) form.body(step.body);
  for (const btn of step.buttons) form.button(btn);

  system.run(() => {
    form.show(player).then((response) => {
      if (response.canceled || response.selection === undefined) {
        onCancel?.(state, player);
        return;
      }

      const action = step.onSelect(state, response.selection, player);

      switch (action.kind) {
        case 'next':
          runStep(index + 1, player, state, steps, onComplete, onCancel);
          break;
        case 'complete':
          onComplete(action.result, player);
          break;
        case 'cancel':
          onCancel?.(state, player);
          break;
        case 'push':
          const allSteps = [...action.steps, ...steps.slice(index + 1)];
          runStep(0, player, action.state, allSteps, onComplete, onCancel);
          break;
      }
    });
  });
}`,
			TypeScript: `import { ActionFormData, ActionFormResponse } from "@minecraft/server-ui";

export interface WizardStep<TState> {
  title: string;
  body?: string;
  buttons: string[];
  onSelect(state: TState, selection: number, player: Player): WizardAction<TState>;
}

export type WizardAction<TState> =
  | { kind: 'next'; step: WizardStep<TState> }
  | { kind: 'complete'; result: TState }
  | { kind: 'cancel' }
  | { kind: 'push'; steps: WizardStep<TState>[]; state: TState };

export function runWizard<TState>(
  player: Player,
  initialState: TState,
  steps: WizardStep<TState>[],
  onComplete: (state: TState, player: Player) => void,
  onCancel?: (state: TState, player: Player) => void,
): void {
  runStep(0, player, initialState, steps, onComplete, onCancel);
}

function runStep<TState>(
  index: number,
  player: Player,
  state: TState,
  steps: WizardStep<TState>[],
  onComplete: (state: TState, player: Player) => void,
  onCancel?: (state: TState, player: Player) => void,
): void {
  if (index >= steps.length) {
    onComplete(state, player);
    return;
  }

  const step = steps[index];
  const form = new ActionFormData().title(step.title);
  if (step.body) form.body(step.body);
  for (const btn of step.buttons) form.button(btn);

  system.run(() => {
    form.show(player).then((response: ActionFormResponse) => {
      if (response.canceled || response.selection === undefined) {
        onCancel?.(state, player);
        return;
      }

      const action = step.onSelect(state, response.selection, player);

      switch (action.kind) {
        case 'next':
          runStep(index + 1, player, state, steps, onComplete, onCancel);
          break;
        case 'complete':
          onComplete(action.result, player);
          break;
        case 'cancel':
          onCancel?.(state, player);
          break;
        case 'push':
          const allSteps = [...action.steps, ...steps.slice(index + 1)];
          runStep(0, player, action.state, allSteps, onComplete, onCancel);
          break;
      }
    });
  });
}`,
			JSImports:       []string{"system"},
			TSImports:       []string{"system"},
			TSTypeImports:   []string{"Player"},
			RequiredModules: []string{"@minecraft/server", "@minecraft/server-ui"},
		},
		SnippetDefinition{
			Type:        "interaction.item_interaction_handler",
			Description: "Block-click-to-UI pipeline for station-style item modification with stale-reference safety",
			Category:    "ui",
			Complexity:  "complex",
			Tags:        []string{"interaction", "blocks", "ui", "items", "station", "crafting"},
			Related:     []string{"ui.action_form_wizard"},
			Notes:       []string{"Combines block interaction with UI forms", "Handles stale container references by re-reading after async boundary", "Auto-cleanup cooldowns every 100 ticks"},
			JavaScript: `import { ActionFormData, ActionFormResponse } from "@minecraft/server-ui";

const interactionCooldowns = new Map();
const STALE_TICK_THRESHOLD = 5;

function getInventoryContainer(player) {
  const inv = player.getComponent('minecraft:inventory');
  return inv?.container;
}

function getHeldItem(player) {
  const container = getInventoryContainer(player);
  if (!container) return undefined;
  const slot = player.selectedSlotIndex;
  const item = container.getItem(slot);
  return item ? { slot, item } : undefined;
}

function registerStationHandler(handler) {
  const cooldownTicks = handler.cooldownTicks ?? 20;
  const blockSet = new Set(handler.blockIds);

  world.beforeEvents.playerInteractWithBlock.subscribe((event) => {
    if (!blockSet.has(event.block.typeId)) return;

    const player = event.player;
    const cdKey = 'station|' + player.id + '|' + handler.blockIds[0];
    if ((interactionCooldowns.get(cdKey) ?? 0) > system.currentTick) {
      event.cancel = true;
      return;
    }
    interactionCooldowns.set(cdKey, system.currentTick + cooldownTicks);
    event.cancel = true;

    system.run(async () => {
      const held = getHeldItem(player);
      const ctx = {
        player,
        blockId: event.block.typeId,
        heldItem: held?.item,
        isStale: false,
      };

      const form = new ActionFormData()
        .title(handler.title)
        .body(handler.body ?? '');
      for (const btn of handler.getButtons(ctx)) form.button(btn);

      const response = await form.show(player);
      if (response.canceled || response.selection === undefined) return;

      const refreshed = getHeldItem(player);
      if (!refreshed) {
        player.sendMessage('§cHeld item changed or vanished.');
        return;
      }
      ctx.heldItem = refreshed.item;
      ctx.isStale = false;

      const result = await handler.onSelect(ctx, response.selection);
      if (result.message) player.sendMessage(result.message);
    });
  });

  system.runInterval(() => {
    const now = system.currentTick;
    for (const [key, expires] of interactionCooldowns) {
      if (expires <= now) interactionCooldowns.delete(key);
    }
  }, 100);
}`,
			TypeScript: `import { ActionFormData, ActionFormResponse } from "@minecraft/server-ui";

export interface InteractionContext {
  player: Player;
  blockId: string;
  heldItem?: ItemStack;
  isStale: boolean;
}

export interface StationHandler {
  blockIds: string[];
  cooldownTicks?: number;
  title: string;
  body?: string;
  getButtons(ctx: InteractionContext): string[];
  onSelect(ctx: InteractionContext, selection: number): Promise<{ success: boolean; message: string }>;
}

const interactionCooldowns = new Map<string, number>();
const STALE_TICK_THRESHOLD = 5;

function getInventoryContainer(player: Player) {
  const inv = player.getComponent('minecraft:inventory') as any;
  return inv?.container;
}

function getHeldItem(player: Player): { slot: number; item: ItemStack } | undefined {
  const container = getInventoryContainer(player);
  if (!container) return undefined;
  const slot = player.selectedSlotIndex;
  const item = container.getItem(slot);
  return item ? { slot, item } : undefined;
}

export function registerStationHandler(handler: StationHandler): void {
  const cooldownTicks = handler.cooldownTicks ?? 20;
  const blockSet = new Set(handler.blockIds);

  world.beforeEvents.playerInteractWithBlock.subscribe((event) => {
    if (!blockSet.has(event.block.typeId)) return;

    const player = event.player;
    const cdKey = 'station|' + player.id + '|' + handler.blockIds[0];
    if ((interactionCooldowns.get(cdKey) ?? 0) > system.currentTick) {
      event.cancel = true;
      return;
    }
    interactionCooldowns.set(cdKey, system.currentTick + cooldownTicks);
    event.cancel = true;

    system.run(async () => {
      const held = getHeldItem(player);
      const ctx: InteractionContext = {
        player,
        blockId: event.block.typeId,
        heldItem: held?.item,
        isStale: false,
      };

      const form = new ActionFormData()
        .title(handler.title)
        .body(handler.body ?? '');
      for (const btn of handler.getButtons(ctx)) form.button(btn);

      const response: ActionFormResponse = await form.show(player);
      if (response.canceled || response.selection === undefined) return;

      const refreshed = getHeldItem(player);
      if (!refreshed) {
        player.sendMessage('§cHeld item changed or vanished.');
        return;
      }
      ctx.heldItem = refreshed.item;
      ctx.isStale = false;

      const result = await handler.onSelect(ctx, response.selection);
      if (result.message) player.sendMessage(result.message);
    });
  });

  system.runInterval(() => {
    const now = system.currentTick;
    for (const [key, expires] of interactionCooldowns) {
      if (expires <= now) interactionCooldowns.delete(key);
    }
  }, 100);
}`,
			JSImports:       []string{"system", "world"},
			TSImports:       []string{"system", "world"},
			TSTypeImports:   []string{"Player", "ItemStack"},
			RequiredModules: []string{"@minecraft/server", "@minecraft/server-ui"},
		},
	)
}
