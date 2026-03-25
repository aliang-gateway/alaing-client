import assert from 'node:assert/strict';
import test from 'node:test';

import { extractUsagePagination, extractUsageRecords } from './dashboardData.js';

test('extractUsageRecords accepts direct array payloads', () => {
  const records = [{ id: 1 }, { id: 2 }];
  assert.deepEqual(extractUsageRecords(records), records);
});

test('extractUsageRecords accepts paginated payloads', () => {
  const records = [{ id: 1001, model: 'claude-sonnet-4-20250514' }];

  assert.deepEqual(
    extractUsageRecords({
      items: records,
      total: 1,
      page: 1
    }),
    records
  );

  assert.deepEqual(
    extractUsageRecords({
      data: {
        items: records,
        total: 1
      }
    }),
    records
  );
});

test('extractUsagePagination reads pagination metadata from payloads', () => {
  assert.deepEqual(
    extractUsagePagination({
      items: [{ id: 1 }],
      total: 42,
      page: 2,
      page_size: 20,
      total_pages: 3
    }),
    {
      page: 2,
      pageSize: 20,
      total: 42,
      totalPages: 3
    }
  );

  assert.deepEqual(
    extractUsagePagination({
      data: {
        items: [{ id: 1 }],
        pagination: {
          total: 8,
          page: 1,
          page_size: 5,
          pages: 2
        }
      }
    }, 1, 5),
    {
      page: 1,
      pageSize: 5,
      total: 8,
      totalPages: 2
    }
  );
});
