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
import userEvent from '@testing-library/user-event'
import { beforeEach, describe, expect, test, vi } from 'vitest'

import { Docs } from '../index'

vi.mock('@/components/layout', () => ({
  PublicLayout: (props: { children: React.ReactNode }) => props.children,
}))

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string) => key,
    i18n: { resolvedLanguage: 'zhCN', language: 'zhCN' },
  }),
}))

describe('model API documentation', () => {
  beforeEach(() => {
    window.history.replaceState({}, '', '/docs')
  })

  test('lets users switch from image docs to model docs without losing either section', async () => {
    const user = userEvent.setup()
    render(<Docs />)

    expect(
      screen.getByRole('heading', { name: '把生图接口讲清楚。' })
    ).toBeVisible()
    const imageTab = screen.getByRole('tab', { name: 'Image API' })
    const modelTab = screen.getByRole('tab', { name: 'Language Model API' })

    await user.click(modelTab)
    const modelDocs = screen.getByRole('tabpanel')
    expect(
      within(modelDocs).getByRole('heading', {
        name: 'Connect to language models',
      })
    ).toBeVisible()
    for (const label of [
      'GPT / OpenAI-compatible',
      'Claude native Messages',
      'Gemini native API',
      'Grok / xAI',
    ]) {
      expect(within(modelDocs).getByText(label)).toBeVisible()
    }
    for (const endpoint of [
      '/v1/chat/completions',
      '/v1/messages',
      '/v1beta/models/{model}:generateContent',
    ]) {
      expect(
        within(modelDocs).getAllByText(
          (_, element) => element?.textContent?.includes(endpoint) ?? false
        ).length
      ).toBeGreaterThan(0)
    }
    expect(
      within(modelDocs).getAllByText(/x-api-key: \$API_KEY/).length
    ).toBeGreaterThan(0)
    expect(
      within(modelDocs).getAllByText(/x-goog-api-key: \$API_KEY/).length
    ).toBeGreaterThan(0)

    await user.click(imageTab)
    expect(
      screen.getByRole('heading', { name: '把生图接口讲清楚。' })
    ).toBeVisible()
    await user.click(modelTab)
    expect(
      within(screen.getByRole('tabpanel')).getByRole('heading', {
        name: 'Connect to language models',
      })
    ).toBeVisible()
  })
})
