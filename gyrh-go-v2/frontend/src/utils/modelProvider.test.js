import assert from 'node:assert/strict';
import { test } from 'node:test';
import { getModelLabel, getNextModel, getProviderForModel } from './modelProvider.js';

test('cycles Wan Gemini GPT', () => {
  assert.equal(getNextModel('W'), 'G');
  assert.equal(getNextModel('G'), 'GPT');
  assert.equal(getNextModel('GPT'), 'W');
});

test('maps models to backend providers', () => {
  assert.equal(getProviderForModel('W'), 'wan');
  assert.equal(getProviderForModel('G'), 'google');
  assert.equal(getProviderForModel('GPT'), '302-gpt-image');
});

test('returns display labels', () => {
  assert.equal(getModelLabel('W'), 'W');
  assert.equal(getModelLabel('G'), 'G');
  assert.equal(getModelLabel('GPT'), 'GPT');
});
