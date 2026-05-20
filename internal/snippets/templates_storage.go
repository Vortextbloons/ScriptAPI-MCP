package snippets

func init() {
	AllDefinitions = append(AllDefinitions,
		SnippetDefinition{
			Type:        "storage.dynamic_property_store",
			Description: "Schema-versioned JSON blob storage using world dynamic properties with data migration support",
			Category:    "storage",
			Complexity:  "moderate",
			Tags:        []string{"persistence", "dynamic-properties", "json", "schema", "migration"},
			Related:     []string{"storage.world_config"},
			Notes:       []string{"Uses world dynamic properties (limited storage capacity)", "Schema versioning enables data structure migration", "Data persists across world saves"},
			JavaScript: `function createStore(keyPrefix, options) {
  function worldKey(id) {
    return keyPrefix + id;
  }

  function isCurrentSchema(value) {
    if (!value || typeof value !== 'object') return false;
    const obj = value;
    if (obj.schema !== options.schema) {
      if (options.migration && typeof obj.schema === 'string') {
        const migrated = options.migration(obj);
        return options.validator(migrated);
      }
      return false;
    }
    return options.validator(value);
  }

  function get(id) {
    const raw = world.getDynamicProperty(worldKey(id));
    if (typeof raw !== 'string') return undefined;
    try {
      const parsed = JSON.parse(raw);
      return isCurrentSchema(parsed) ? parsed : undefined;
    } catch {
      return undefined;
    }
  }

  function set(id, data) {
    world.setDynamicProperty(worldKey(id), JSON.stringify({ ...data, schema: options.schema }));
  }

  function remove(id) {
    world.setDynamicProperty(worldKey(id), undefined);
  }

  function has(id) {
    return typeof world.getDynamicProperty(worldKey(id)) === 'string';
  }

  function getAllIds(searchPrefix) {
    const results = [];
    const prefix = searchPrefix ?? keyPrefix;
    for (const key of world.getDynamicPropertyIds()) {
      if (key.startsWith(prefix)) {
        results.push(key.slice(prefix.length));
      }
    }
    return results;
  }

  return { get, set, remove, has, getAllIds };
}`,
			TypeScript: `type SchemaValidator<T> = (value: unknown) => value is T;

interface StoreOptions<T> {
  schema: string;
  validator: SchemaValidator<T>;
  migration?: (raw: Record<string, unknown>) => T;
}

export function createStore<T extends Record<string, unknown>>(keyPrefix: string, options: StoreOptions<T>) {
  function worldKey(id: string): string {
    return keyPrefix + id;
  }

  function isCurrentSchema(value: unknown): value is T {
    if (!value || typeof value !== 'object') return false;
    const obj = value as Record<string, unknown>;
    if (obj.schema !== options.schema) {
      if (options.migration && typeof obj.schema === 'string') {
        const migrated = options.migration(obj);
        return options.validator(migrated);
      }
      return false;
    }
    return options.validator(value);
  }

  function get(id: string): T | undefined {
    const raw = world.getDynamicProperty(worldKey(id));
    if (typeof raw !== 'string') return undefined;
    try {
      const parsed = JSON.parse(raw) as unknown;
      return isCurrentSchema(parsed) ? parsed : undefined;
    } catch {
      return undefined;
    }
  }

  function set(id: string, data: T): void {
    world.setDynamicProperty(worldKey(id), JSON.stringify({ ...data, schema: options.schema }));
  }

  function remove(id: string): void {
    world.setDynamicProperty(worldKey(id), undefined);
  }

  function has(id: string): boolean {
    return typeof world.getDynamicProperty(worldKey(id)) === 'string';
  }

  function getAllIds(searchPrefix?: string): string[] {
    const results: string[] = [];
    const prefix = searchPrefix ?? keyPrefix;
    for (const key of world.getDynamicPropertyIds()) {
      if (key.startsWith(prefix)) {
        results.push(key.slice(prefix.length));
      }
    }
    return results;
  }

  return { get, set, remove, has, getAllIds };
}`,
			JSImports:       []string{"world"},
			TSImports:       []string{"world"},
			TSTypeImports:   []string{},
			RequiredModules: []string{"@minecraft/server"},
		},
		SnippetDefinition{
			Type:        "storage.world_config",
			Description: "World-level JSON config stored in dynamic properties with defaults and sanitization",
			Category:    "storage",
			Complexity:  "simple",
			Tags:        []string{"config", "persistence", "json", "dynamic-properties", "settings"},
			Related:     []string{"storage.dynamic_property_store"},
			Notes:       []string{"Minimal config pattern with sensible defaults", "Write merges with existing data (partial updates)", "Sanitize function optional but recommended for validation"},
			JavaScript: `function createWorldConfig(key, defaults, sanitize) {
  let cached;

  function read() {
    if (cached) return cached;
    const raw = world.getDynamicProperty(key);
    if (typeof raw !== 'string') {
      cached = { ...defaults };
      return cached;
    }
    try {
      const parsed = JSON.parse(raw);
      cached = sanitize ? sanitize(parsed) : { ...defaults, ...parsed };
      return cached;
    } catch {
      cached = { ...defaults };
      return cached;
    }
  }

  function write(patch) {
    const current = read();
    const merged = sanitize ? sanitize({ ...current, ...patch }) : { ...current, ...patch };
    world.setDynamicProperty(key, JSON.stringify(merged));
    cached = merged;
    return merged;
  }

  function reset() {
    world.setDynamicProperty(key, undefined);
    cached = undefined;
    return read();
  }

  return { read, write, reset };
}`,
			TypeScript: `export function createWorldConfig<T extends Record<string, unknown>>(
  key: string,
  defaults: T,
  sanitize?: (input: Partial<T>) => T,
) {
  let cached: T | undefined;

  function read(): T {
    if (cached) return cached;
    const raw = world.getDynamicProperty(key);
    if (typeof raw !== 'string') {
      cached = { ...defaults };
      return cached;
    }
    try {
      const parsed = JSON.parse(raw) as Partial<T>;
      cached = sanitize ? sanitize(parsed) : { ...defaults, ...parsed };
      return cached;
    } catch {
      cached = { ...defaults };
      return cached;
    }
  }

  function write(patch: Partial<T>): T {
    const current = read();
    const merged = sanitize ? sanitize({ ...current, ...patch }) : ({ ...current, ...patch } as T);
    world.setDynamicProperty(key, JSON.stringify(merged));
    cached = merged;
    return merged;
  }

  function reset(): T {
    world.setDynamicProperty(key, undefined);
    cached = undefined;
    return read();
  }

  return { read, write, reset };
}`,
			JSImports:       []string{"world"},
			TSImports:       []string{"world"},
			TSTypeImports:   []string{},
			RequiredModules: []string{"@minecraft/server"},
		},
	)
}
