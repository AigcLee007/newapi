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
import type { TFunction } from 'i18next'
import { describe, expect, test } from 'vitest'

import {
  getApiKeyFormDefaultValues,
  getApiKeyFormSchema,
} from '../api-key-form'

const t = ((key: string) => key) as TFunction
const schema = getApiKeyFormSchema(t)
const defaults = { ...getApiKeyFormDefaultValues(false), name: 'Custom key' }

describe('custom API key validation', () => {
  test('accepts an empty key for automatic generation and valid key characters', () => {
    expect(schema.safeParse({ ...defaults, key: '' }).success).toBe(true)
    expect(
      schema.safeParse({ ...defaults, key: 'Abcdef0123456789._-' }).success
    ).toBe(true)
  })

  test('accepts keys at the 16 and 128 character boundaries', () => {
    expect(schema.safeParse({ ...defaults, key: 'a'.repeat(16) }).success).toBe(
      true
    )
    expect(
      schema.safeParse({ ...defaults, key: 'a'.repeat(128) }).success
    ).toBe(true)
  })

  test.each([
    ['keys shorter than 16 characters', 'a'.repeat(15)],
    ['keys longer than 128 characters', 'a'.repeat(129)],
    ['characters outside the allowed set', 'abcdef012345678+'],
  ])('rejects %s', (_caseName, key) => {
    expect(schema.safeParse({ ...defaults, key }).success).toBe(false)
  })
})
