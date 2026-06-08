import assert from 'node:assert';
import { validateImportData } from '../src/lib/util/importValidator.ts';

interface TestCase {
  name: string;
  fn: () => void;
}

const tests: TestCase[] = [];
function test(name: string, fn: () => void): void {
  tests.push({ name, fn });
}

// Happy paths
test('minimal valid data', () => {
  const result = validateImportData({ total: 45.50, participants: ['Alice', 'Bob'] });
  assert.strictEqual(result.total, 45.50);
  assert.deepStrictEqual(result.participants, ['Alice', 'Bob']);
  assert.strictEqual(result.subtotal, 45.50);
  assert.deepStrictEqual(result.items, []);
  assert.strictEqual(result.title, '');
});

test('full data with items and title', () => {
  const result = validateImportData({
    title: 'Dinner',
    total: 45.50,
    subtotal: 40.00,
    participants: ['Alice', 'Bob'],
    items: [
      { description: 'Pizza', amount: 20.00 },
      { description: 'Salad', amount: 15.00 },
    ],
  });
  assert.strictEqual(result.title, 'Dinner');
  assert.strictEqual(result.subtotal, 40.00);
  assert.strictEqual(result.items.length, 2);
  assert.strictEqual(result.items[0].description, 'Pizza');
  assert.strictEqual(result.items[0].amount, 20.00);
  assert.strictEqual(result.items[1].description, 'Salad');
});

test('subtotal defaults to total when omitted', () => {
  const result = validateImportData({ total: 50, participants: ['Alice'] });
  assert.strictEqual(result.subtotal, 50);
});

test('trims whitespace from participant names', () => {
  const result = validateImportData({ total: 10, participants: ['  Alice  ', ' Bob'] });
  assert.deepStrictEqual(result.participants, ['Alice', 'Bob']);
});

test('trims whitespace from item descriptions', () => {
  const result = validateImportData({
    total: 10,
    participants: ['Alice'],
    items: [{ description: '  Pizza  ', amount: 10 }],
  });
  assert.strictEqual(result.items[0].description, 'Pizza');
});

test('item with missing description defaults to empty string', () => {
  const result = validateImportData({
    total: 10,
    participants: ['Alice'],
    items: [{ amount: 10 }],
  });
  assert.strictEqual(result.items[0].description, '');
});

test('parses string numbers for total and amounts', () => {
  const result = validateImportData({
    total: '45.50',
    subtotal: '40.00',
    participants: ['Alice'],
    items: [{ description: 'Pizza', amount: '20.00' }],
  });
  assert.strictEqual(result.total, 45.50);
  assert.strictEqual(result.subtotal, 40.00);
  assert.strictEqual(result.items[0].amount, 20.00);
});

// Error cases
test('throws on null input', () => {
  assert.throws(() => validateImportData(null), /expected an object/);
});

test('throws on non-object input', () => {
  assert.throws(() => validateImportData('string'), /expected an object/);
  assert.throws(() => validateImportData(42), /expected an object/);
  assert.throws(() => validateImportData([]), /expected an object/);
});

test('throws when total is missing', () => {
  assert.throws(() => validateImportData({ participants: ['Alice'] }), /"total"/);
});

test('throws when total is NaN', () => {
  assert.throws(() => validateImportData({ total: 'abc', participants: ['Alice'] }), /"total"/);
});

test('throws when participants is missing', () => {
  assert.throws(() => validateImportData({ total: 10 }), /"participants"/);
});

test('throws when participants is empty array', () => {
  assert.throws(() => validateImportData({ total: 10, participants: [] }), /"participants"/);
});

test('throws when participants is not an array', () => {
  assert.throws(() => validateImportData({ total: 10, participants: 'Alice' }), /"participants"/);
});

test('throws when a participant is not a string', () => {
  assert.throws(() => validateImportData({ total: 10, participants: [123] }), /non-empty string/);
});

test('throws when a participant is an empty string', () => {
  assert.throws(() => validateImportData({ total: 10, participants: [''] }), /non-empty string/);
});

test('throws when item amount is missing', () => {
  assert.throws(
    () => validateImportData({ total: 10, participants: ['Alice'], items: [{ description: 'Pizza' }] }),
    /amount/,
  );
});

test('throws when item amount is NaN', () => {
  assert.throws(
    () => validateImportData({ total: 10, participants: ['Alice'], items: [{ description: 'Pizza', amount: 'free' }] }),
    /amount/,
  );
});

test('throws when an item is not an object', () => {
  assert.throws(
    () => validateImportData({ total: 10, participants: ['Alice'], items: ['Pizza'] }),
    /object/,
  );
});

// Run
let passed = 0;
let failed = 0;
for (const { name, fn } of tests) {
  try {
    fn();
    console.log(`✓ ${name}`);
    passed++;
  } catch (e) {
    const msg = e instanceof Error ? e.message : String(e);
    console.log(`✗ ${name}`);
    console.log(`  ${msg}`);
    failed++;
  }
}
console.log(`\n${passed} passed, ${failed} failed`);
if (failed > 0) process.exit(1);
