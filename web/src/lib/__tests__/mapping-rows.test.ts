/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as
published by the Free Software Foundation, either version 3 of the
License, or (at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { describe, expect, test } from 'vitest'

import { syncMappingRowsFromJson } from '../mapping-rows'

describe('syncMappingRowsFromJson', () => {
  test('preserves row ids when parsed entries have not changed', () => {
    const rows = syncMappingRowsFromJson('{"gpt-4":"gpt-4o"}', [], 'model-map')

    const nextRows = syncMappingRowsFromJson(
      '{\n  "gpt-4": "gpt-4o"\n}',
      rows,
      'model-map'
    )

    expect(nextRows).toHaveLength(1)
    expect(nextRows[0]?.id).toBe(rows[0]?.id)
  })

  test('keeps existing ids when one entry changes and creates ids for new entries', () => {
    const rows = syncMappingRowsFromJson('{"gpt-4":"gpt-4o"}', [], 'model-map')

    const nextRows = syncMappingRowsFromJson(
      '{"gpt-4":"gpt-4.1","claude":"claude-sonnet"}',
      rows,
      'model-map'
    )

    expect(nextRows).toHaveLength(2)
    expect(nextRows[0]?.id).toBe(rows[0]?.id)
    expect(nextRows[1]?.id).not.toBe(rows[0]?.id)
  })

  test('returns previous rows for invalid json', () => {
    const rows = syncMappingRowsFromJson('{"gpt-4":"gpt-4o"}', [], 'model-map')

    const nextRows = syncMappingRowsFromJson('{', rows, 'model-map')

    expect(nextRows).toEqual(rows)
  })
})
