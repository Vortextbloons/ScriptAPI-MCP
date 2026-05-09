package snippets

// SnippetDefinition holds the template data for one snippet type.
type SnippetDefinition struct {
	Type            string
	Description     string
	JavaScript      string
	TypeScript      string
	JSImports       []string
	TSImports       []string
	TSTypeImports   []string
	RequiredModules []string
}

var AllDefinitions = []SnippetDefinition{
	{
		Type:        "beforeEvents.playerBreakBlock",
		Description: "Subscribe to the beforeEvents.playerBreakBlock event",
		JavaScript: `world.beforeEvents.playerBreakBlock.subscribe((event) => {
  const { player, block, brokenBlockPermutation } = event;
  // your code here
});`,
		TypeScript: `world.beforeEvents.playerBreakBlock.subscribe((event: BlockBreakAfterEvent): void => {
  const { player, block, brokenBlockPermutation } = event;
  // your code here
});`,
		JSImports:       []string{"world"},
		TSImports:       []string{"world"},
		TSTypeImports:   []string{"BlockBreakAfterEvent"},
		RequiredModules: []string{"@minecraft/server"},
	},
	{
		Type:        "afterEvents.playerSpawn",
		Description: "Subscribe to the afterEvents.playerSpawn event",
		JavaScript: `world.afterEvents.playerSpawn.subscribe((event) => {
  const { player, initialSpawn } = event;
  // your code here
});`,
		TypeScript: `world.afterEvents.playerSpawn.subscribe((event: PlayerSpawnAfterEvent): void => {
  const { player, initialSpawn } = event;
  // your code here
});`,
		JSImports:       []string{"world"},
		TSImports:       []string{"world"},
		TSTypeImports:   []string{"PlayerSpawnAfterEvent"},
		RequiredModules: []string{"@minecraft/server"},
	},
	{
		Type:        "worldInitialize",
		Description: "Subscribe to the afterEvents.worldInitialize event",
		JavaScript: `world.afterEvents.worldInitialize.subscribe((event) => {
  const { propertyRegistry } = event;
  // your code here
});`,
		TypeScript: `world.afterEvents.worldInitialize.subscribe((event: WorldInitializeAfterEvent): void => {
  const { propertyRegistry } = event;
  // your code here
});`,
		JSImports:       []string{"world"},
		TSImports:       []string{"world"},
		TSTypeImports:   []string{"WorldInitializeAfterEvent"},
		RequiredModules: []string{"@minecraft/server"},
	},
	{
		Type:        "custom_item_template",
		Description: "Custom item component template with worldInitialize",
		JavaScript: `world.afterEvents.worldInitialize.subscribe((event) => {
  const { propertyRegistry } = event;

  propertyRegistry.registerCustomComponent("{{name}}", {
    onUse({ source }) {
      // your code here
    },
  });
});`,
		TypeScript: `world.afterEvents.worldInitialize.subscribe((event: WorldInitializeAfterEvent): void => {
  const { propertyRegistry } = event;

  propertyRegistry.registerCustomComponent("{{name}}", {
    onUse({ source }: ItemComponentUseEvent): void {
      // your code here
    },
  });
});`,
		JSImports:       []string{"world"},
		TSImports:       []string{"world"},
		TSTypeImports:   []string{"WorldInitializeAfterEvent", "ItemComponentUseEvent"},
		RequiredModules: []string{"@minecraft/server"},
	},
	{
		Type:        "custom_block_template",
		Description: "Custom block component template with worldInitialize",
		JavaScript: `world.afterEvents.worldInitialize.subscribe((event) => {
  const { propertyRegistry } = event;

  propertyRegistry.registerCustomComponent("{{name}}", {
    onPlayerDestroy({ player, block }) {
      // your code here
    },
  });
});`,
		TypeScript: `world.afterEvents.worldInitialize.subscribe((event: WorldInitializeAfterEvent): void => {
  const { propertyRegistry } = event;

  propertyRegistry.registerCustomComponent("{{name}}", {
    onPlayerDestroy({ player, block }: BlockComponentPlayerDestroyEvent): void {
      // your code here
    },
  });
});`,
		JSImports:       []string{"world"},
		TSImports:       []string{"world"},
		TSTypeImports:   []string{"WorldInitializeAfterEvent", "BlockComponentPlayerDestroyEvent"},
		RequiredModules: []string{"@minecraft/server"},
	},
	{
		Type:        "script_event_handler",
		Description: "Subscribe to system scriptEventReceive events",
		JavaScript: `system.afterEvents.scriptEventReceive.subscribe((event) => {
  const { id, message, sourceEntity, sourceBlock } = event;
  // your code here
});`,
		TypeScript: `system.afterEvents.scriptEventReceive.subscribe((event: ScriptEventReceiveEvent): void => {
  const { id, message, sourceEntity, sourceBlock } = event;
  // your code here
});`,
		JSImports:       []string{"system"},
		TSImports:       []string{"system"},
		TSTypeImports:   []string{"ScriptEventReceiveEvent"},
		RequiredModules: []string{"@minecraft/server"},
	},
}

func GetDefinition(snippetType string) (*SnippetDefinition, bool) {
	for _, d := range AllDefinitions {
		if d.Type == snippetType {
			return &d, true
		}
	}
	return nil, false
}
