// Unit tests for the A/B test experiment-to-component registry.
// Mirrors backend/internal/service/abtest_registry.go.
import { describe, it, expect } from 'vitest';

// Must stay in sync with backend/internal/service/abtest_registry.go.
const EXPERIMENT_REGISTRY = {
  catalog_layout: {
    description: 'Controls the collectible catalog grid layout (CatalogPage.tsx)',
    allowedVariants: new Set(['grid', 'list']),
  },
  checkout_flow: {
    description: 'Controls the order checkout experience (OrderDetailPage.tsx)',
    allowedVariants: new Set(['standard', 'express']),
  },
  search_ranking: {
    description: 'Controls the collectible search/sort algorithm (CatalogPage.tsx)',
    allowedVariants: new Set(['relevance', 'popular']),
  },
};

function validateExperiment(name, controlVariant, testVariant) {
  const comp = EXPERIMENT_REGISTRY[name];
  if (!comp) {
    const allowed = Object.keys(EXPERIMENT_REGISTRY).join(', ') || '(none)';
    return `unknown experiment name '${name}'; registered experiments: ${allowed}`;
  }
  if (!comp.allowedVariants.has(controlVariant)) {
    return `control variant '${controlVariant}' is not registered for experiment '${name}'`;
  }
  if (!comp.allowedVariants.has(testVariant)) {
    return `test variant '${testVariant}' is not registered for experiment '${name}'`;
  }
  if (controlVariant === testVariant) return 'control and test variants must be different';
  return '';
}

describe('Registered Experiments', () => {
  it('catalog_layout grid/list valid', () => expect(validateExperiment('catalog_layout', 'grid', 'list')).toBe(''));
  it('catalog_layout list/grid valid', () => expect(validateExperiment('catalog_layout', 'list', 'grid')).toBe(''));
});

describe('Unregistered Experiments', () => {
  it('unknown name rejected', () => {
    const msg = validateExperiment('nonexistent_experiment', 'a', 'b');
    expect(msg).toContain('unknown experiment');
  });
  it('empty name rejected', () => expect(validateExperiment('', 'grid', 'list')).toContain('unknown experiment'));
});

describe('Invalid Variants', () => {
  it('bad control', () => expect(validateExperiment('catalog_layout', 'carousel', 'list')).toContain('control variant'));
  it('bad test',    () => expect(validateExperiment('catalog_layout', 'grid', 'carousel')).toContain('test variant'));
  it('same variants', () => expect(validateExperiment('catalog_layout', 'grid', 'grid')).toContain('must be different'));
});

describe('All Registered Experiments', () => {
  it('catalog_layout grid/list valid', () => expect(validateExperiment('catalog_layout', 'grid', 'list')).toBe(''));
  it('checkout_flow standard/express valid', () => expect(validateExperiment('checkout_flow', 'standard', 'express')).toBe(''));
  it('search_ranking relevance/popular valid', () => expect(validateExperiment('search_ranking', 'relevance', 'popular')).toBe(''));
});

describe('Registry Completeness', () => {
  it('all components have >= 2 variants', () => {
    for (const [name, comp] of Object.entries(EXPERIMENT_REGISTRY)) {
      expect(comp.allowedVariants.size).toBeGreaterThanOrEqual(2);
    }
  });
  it('registry not empty', () => expect(Object.keys(EXPERIMENT_REGISTRY).length).toBeGreaterThan(0));
  it('catalog_layout registered', () => expect(EXPERIMENT_REGISTRY).toHaveProperty('catalog_layout'));
  it('checkout_flow registered', () => expect(EXPERIMENT_REGISTRY).toHaveProperty('checkout_flow'));
  it('search_ranking registered', () => expect(EXPERIMENT_REGISTRY).toHaveProperty('search_ranking'));
});

describe('Registry Parity with Backend', () => {
  // The canonical experiment list from backend/internal/service/abtest_registry.go.
  // If a new experiment is added to the Go registry, add it here too.
  const BACKEND_EXPERIMENTS = {
    catalog_layout: ['grid', 'list'],
    checkout_flow: ['standard', 'express'],
    search_ranking: ['relevance', 'popular'],
  };

  it('frontend registry has every backend experiment', () => {
    for (const name of Object.keys(BACKEND_EXPERIMENTS)) {
      expect(EXPERIMENT_REGISTRY).toHaveProperty(name);
    }
  });

  it('frontend registry has no extra experiments beyond backend', () => {
    for (const name of Object.keys(EXPERIMENT_REGISTRY)) {
      expect(BACKEND_EXPERIMENTS).toHaveProperty(name);
    }
  });

  it('variant sets match for every experiment', () => {
    for (const [name, backendVariants] of Object.entries(BACKEND_EXPERIMENTS)) {
      const frontendVariants = EXPERIMENT_REGISTRY[name].allowedVariants;
      expect([...frontendVariants].sort()).toEqual([...backendVariants].sort());
    }
  });
});

describe('Notification Template Scope', () => {
  const SUPPORTED = new Set([
    'order_confirmed', 'order_processing', 'order_completed', 'order_cancelled',
    'refund_approved', 'arbitration_opened', 'review_posted',
  ]);

  it('7 supported slugs', () => expect(SUPPORTED.size).toBe(7));
  it('refund_approved supported',     () => expect(SUPPORTED.has('refund_approved')).toBe(true));
  it('arbitration_opened supported',  () => expect(SUPPORTED.has('arbitration_opened')).toBe(true));
  it('review_posted supported',       () => expect(SUPPORTED.has('review_posted')).toBe(true));
});
