export const MODEL_SEQUENCE = ['W', 'G', 'GPT'];

export const MODEL_LABELS = {
  W: 'W',
  G: 'G',
  GPT: 'GPT',
};

export const MODEL_PROVIDERS = {
  W: 'wan',
  G: 'google',
  GPT: '302-gpt-image',
};

export function getNextModel(current) {
  const index = MODEL_SEQUENCE.indexOf(current);
  if (index === -1) return MODEL_SEQUENCE[0];
  return MODEL_SEQUENCE[(index + 1) % MODEL_SEQUENCE.length];
}

export function getModelLabel(model) {
  return MODEL_LABELS[model] || MODEL_LABELS.W;
}

export function getProviderForModel(model) {
  return MODEL_PROVIDERS[model] || MODEL_PROVIDERS.W;
}

export function isGPTModel(model) {
  return model === 'GPT';
}
