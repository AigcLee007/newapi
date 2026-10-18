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

export type MappingEditorRow<TValueField extends string> = {
  id: string
  key: string
} & Record<TValueField, string>

type ParsedMappingEntry = {
  key: string
  value: string
}

function createRowId(
  scope: string,
  index: number,
  usedIds: Set<string>
): string {
  let candidate = `${scope}-${index}`
  let suffix = 1

  while (usedIds.has(candidate)) {
    candidate = `${scope}-${index}-${suffix}`
    suffix += 1
  }

  usedIds.add(candidate)
  return candidate
}

function parseMappingJson(json: string): ParsedMappingEntry[] | undefined {
  if (!json.trim()) return []

  try {
    const parsed = JSON.parse(json)
    if (!parsed || typeof parsed !== 'object' || Array.isArray(parsed)) {
      return undefined
    }

    return Object.entries(parsed).map(([key, value]) => ({
      key,
      value:
        typeof value === 'object' && value !== null
          ? JSON.stringify(value)
          : String(value),
    }))
  } catch {
    return undefined
  }
}

export function syncMappingRowsFromJson<TValueField extends string>(
  json: string,
  previousRows: MappingEditorRow<TValueField>[],
  idScope: string,
  valueField: TValueField = 'value' as TValueField
): MappingEditorRow<TValueField>[] {
  const entries = parseMappingJson(json)
  if (!entries) return previousRows

  const usedIds = new Set<string>()
  const unusedPreviousRows = [...previousRows]

  return entries.map((entry, index) => {
    let previousIndex = unusedPreviousRows.findIndex(
      (row) => row.key === entry.key && row[valueField] === entry.value
    )

    if (previousIndex < 0) {
      previousIndex = unusedPreviousRows.findIndex(
        (row) => row.key === entry.key
      )
    }

    const previousRow =
      previousIndex >= 0 ? unusedPreviousRows.splice(previousIndex, 1)[0] : null
    const id =
      previousRow?.id && !usedIds.has(previousRow.id)
        ? previousRow.id
        : createRowId(idScope, index, usedIds)

    usedIds.add(id)

    return {
      id,
      key: entry.key,
      [valueField]: entry.value,
    } as MappingEditorRow<TValueField>
  })
}
