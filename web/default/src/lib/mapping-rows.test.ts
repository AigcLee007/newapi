import assert from 'node:assert/strict'
import { describe, test } from 'node:test'
import { syncMappingRowsFromJson } from './mapping-rows.ts'

describe('syncMappingRowsFromJson', () => {
  test('preserves row ids when parsed entries have not changed', () => {
    const rows = syncMappingRowsFromJson(
      '{"gpt-4":"gpt-4o"}',
      [],
      'model-map'
    )

    const nextRows = syncMappingRowsFromJson(
      '{\n  "gpt-4": "gpt-4o"\n}',
      rows,
      'model-map'
    )

    assert.equal(nextRows.length, 1)
    assert.equal(nextRows[0]?.id, rows[0]?.id)
  })

  test('keeps existing ids when one entry changes and creates ids for new entries', () => {
    const rows = syncMappingRowsFromJson(
      '{"gpt-4":"gpt-4o"}',
      [],
      'model-map'
    )

    const nextRows = syncMappingRowsFromJson(
      '{"gpt-4":"gpt-4.1","claude":"claude-sonnet"}',
      rows,
      'model-map'
    )

    assert.equal(nextRows.length, 2)
    assert.equal(nextRows[0]?.id, rows[0]?.id)
    assert.notEqual(nextRows[1]?.id, rows[0]?.id)
  })

  test('returns previous rows for invalid json', () => {
    const rows = syncMappingRowsFromJson(
      '{"gpt-4":"gpt-4o"}',
      [],
      'model-map'
    )

    const nextRows = syncMappingRowsFromJson('{', rows, 'model-map')

    assert.deepEqual(nextRows, rows)
  })
})
