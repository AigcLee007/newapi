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
import { render, screen, within } from '@testing-library/react'
import { createInstance } from 'i18next'
import { I18nextProvider, initReactI18next } from 'react-i18next'
import { beforeEach, expect, test } from 'vitest'

import en from '@/i18n/locales/en.json'

import { WalletStatsCard } from '../wallet-stats-card'

const i18n = createInstance()

beforeEach(async () => {
  await i18n.use(initReactI18next).init({
    lng: 'en',
    fallbackLng: false,
    nsSeparator: false,
    resources: { en },
    interpolation: { escapeValue: false },
  })
})

test('presents current balance before usage and request metrics as definitions', () => {
  render(
    <I18nextProvider i18n={i18n}>
      <WalletStatsCard
        user={{
          id: 1,
          username: 'lee',
          quota: 50_000_000,
          used_quota: 8_000_000,
          request_count: 2481,
          aff_quota: 0,
          aff_history_quota: 0,
          aff_count: 0,
          group: 'default',
        }}
      />
    </I18nextProvider>
  )

  expect(screen.getAllByRole('term').map((term) => term.textContent)).toEqual([
    'Current Balance',
    'Total Usage',
    'API Requests',
  ])
  const terms = screen.getAllByRole('term')
  expect(screen.getAllByRole('definition')).toHaveLength(terms.length * 2)
  for (const term of terms) {
    expect(
      within(term.parentElement as HTMLElement).getAllByRole('definition')
    ).toHaveLength(2)
  }
})
