/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { render, screen } from '@testing-library/react'
import { createInstance } from 'i18next'
import { I18nextProvider, initReactI18next } from 'react-i18next'
import { beforeEach, expect, test, vi } from 'vitest'

import en from '@/i18n/locales/en.json'

import { XunhuCheckoutDialog } from '../xunhu-checkout-dialog'

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

test('shows a scan-ready QR code and a direct payment link', () => {
  render(
    <I18nextProvider i18n={i18n}>
      <XunhuCheckoutDialog
        open
        onOpenChange={vi.fn()}
        checkout={{
          url: 'https://pay.xunhupay.com/checkout',
          url_qrcode: 'weixin://pay/order',
        }}
      />
    </I18nextProvider>
  )

  expect(
    screen.getByRole('img', { name: 'Xunhu payment QR code' })
  ).toBeVisible()
  const link = screen.getByRole('button', { name: 'Open payment page' })
  expect(link).toHaveAttribute('href', 'https://pay.xunhupay.com/checkout')
  expect(link).toHaveAttribute('target', '_blank')
})
