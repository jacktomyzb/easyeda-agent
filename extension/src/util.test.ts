/**
 * Unit tests for pure helpers in `util.ts`.
 *
 * Run with: `npm test` (node:test via ts-node, no EasyEDA runtime needed).
 */

import assert from 'node:assert/strict';
import { test } from 'node:test';

import { ActionError } from './protocol';
import { normalizeWirePoints } from './util';

test('normalizeWirePoints: flat input is returned unchanged', () => {
	assert.deepEqual(normalizeWirePoints([195, 350, 215, 350]), [195, 350, 215, 350]);
});

test('normalizeWirePoints: nested pairs are flattened', () => {
	assert.deepEqual(normalizeWirePoints([[195, 350], [215, 350]]), [195, 350, 215, 350]);
});

test('normalizeWirePoints: nested and flat yield identical create args', () => {
	const nested = normalizeWirePoints([[100, 200], [100, 300], [150, 300]]);
	const flat = normalizeWirePoints([100, 200, 100, 300, 150, 300]);
	assert.deepEqual(nested, flat);
	assert.deepEqual(flat, [100, 200, 100, 300, 150, 300]);
});

test('normalizeWirePoints: missing / empty points throws', () => {
	assert.throws(() => normalizeWirePoints(undefined), ActionError);
	assert.throws(() => normalizeWirePoints([]), ActionError);
	assert.throws(() => normalizeWirePoints('nope'), ActionError);
});

test('normalizeWirePoints: odd-length or too-short flat input throws', () => {
	assert.throws(() => normalizeWirePoints([1, 2]), ActionError); // only one point
	assert.throws(() => normalizeWirePoints([1, 2, 3]), ActionError); // odd length
});

test('normalizeWirePoints: malformed nested entry throws', () => {
	assert.throws(() => normalizeWirePoints([[1, 2], [3]]), ActionError); // not a pair
	assert.throws(() => normalizeWirePoints([[1, 2, 3], [4, 5, 6]]), ActionError); // triple
});

test('normalizeWirePoints: non-finite coordinates throw', () => {
	assert.throws(() => normalizeWirePoints([1, 2, NaN, 4]), ActionError);
	assert.throws(() => normalizeWirePoints([[1, 2], [Infinity, 4]]), ActionError);
});
