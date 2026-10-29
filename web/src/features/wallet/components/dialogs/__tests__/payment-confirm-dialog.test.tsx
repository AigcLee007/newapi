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
import { render, screen } from '@testing-library/react'
import { createInstance } from 'i18next'
import { I18nextProvider, initReactI18next } from 'react-i18next'
import { afterEach, beforeEach, expect, test, vi } from 'vitest'

import en from '@/i18n/locales/en.json'

import { PaymentConfirmDialog } from '../payment-confirm-dialog'

const i18n = createInstance()
const initialInnerWidth = window.innerWidth

beforeEach(async () => {
  Object.defineProperty(window, 'innerWidth', {
    configurable: true,
    value: 320,
  })
  await i18n.use(initReactI18next).init({
    lng: 'en',
    fallbackLng: false,
    nsSeparator: false,
    resources: { en },
    interpolation: { escapeValue: false },
  })
})

afterEach(() => {
  Object.defineProperty(window, 'innerWidth', {
    configurable: true,
    value: initialInnerWidth,
  })
})

test('shows the Xunhu fee breakdown and a full-width confirm button on mobile', () => {
  render(
    <I18nextProvider i18n={i18n}>
      <PaymentConfirmDialog
        open
        onOpenChange={vi.fn()}
        onConfirm={vi.fn()}
        topupAmount={10}
        paymentAmount={10.2}
        paymentBaseAmount={10}
        paymentFeeAmount={0.2}
        paymentFeePercent={2}
        paymentMethod={{
          name: 'WeChat',
          type: 'wxpay',
          provider: 'xunhu',
        }}
        calculating={false}
        processing={false}
      />
    </I18nextProvider>
  )

  expect(screen.getByText('Payment amount')).toBeVisible()
  expect(screen.getByText('Processing fee (2%)')).toBeVisible()
  expect(screen.getByText('Total payment')).toBeVisible()
  expect(screen.getByText('$10.00')).toBeVisible()
  expect(screen.getByText('$0.20')).toBeVisible()
  expect(screen.getByText('$10.20')).toBeVisible()
  const confirmButton = screen.getByRole('button', {
    name: /Confirm Payment.*10\.20/,
  })
  expect(confirmButton).toBeVisible()
  expect(confirmButton).toHaveClass('w-full', 'max-w-full', 'whitespace-normal')
  expect(
    confirmButton.closest('[data-slot="alert-dialog-footer"]')
  ).toHaveClass('flex-col-reverse', 'sm:flex-col')
})
