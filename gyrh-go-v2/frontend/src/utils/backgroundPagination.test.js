import assert from 'node:assert/strict';
import { test } from 'node:test';
import {
  BACKGROUND_MANAGER_PAGE_SIZE,
  buildBackgroundPromptListUrl,
  getPageAfterRefresh,
  getTotalPages,
} from './backgroundPagination.js';

test('builds background prompt list URL with limit and offset', () => {
  assert.equal(BACKGROUND_MANAGER_PAGE_SIZE, 10);
  assert.equal(
    buildBackgroundPromptListUrl(1),
    '/api/v1/background-prompts?limit=10&offset=0',
  );
  assert.equal(
    buildBackgroundPromptListUrl(3),
    '/api/v1/background-prompts?limit=10&offset=20',
  );
});

test('adds category filter to background prompt list URL when category is positive', () => {
  assert.equal(
    buildBackgroundPromptListUrl(2, 6, { categoryId: 9 }),
    '/api/v1/background-prompts?limit=6&offset=6&category_id=9',
  );
  assert.equal(
    buildBackgroundPromptListUrl(1, 6, { categoryId: 0 }),
    '/api/v1/background-prompts?limit=6&offset=0',
  );
});

test('clamps invalid page values when building list URL', () => {
  assert.equal(
    buildBackgroundPromptListUrl(0),
    '/api/v1/background-prompts?limit=10&offset=0',
  );
  assert.equal(
    buildBackgroundPromptListUrl(-8),
    '/api/v1/background-prompts?limit=10&offset=0',
  );
});

test('computes total pages with a minimum of one page', () => {
  assert.equal(getTotalPages(0), 1);
  assert.equal(getTotalPages(1), 1);
  assert.equal(getTotalPages(10), 1);
  assert.equal(getTotalPages(11), 2);
});

test('sync refresh jumps back to the first page', () => {
  assert.equal(getPageAfterRefresh(4, { resetToFirstPage: true }), 1);
  assert.equal(getPageAfterRefresh(4, { resetToFirstPage: false }), 4);
});
