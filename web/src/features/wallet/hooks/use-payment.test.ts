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
import { act, renderHook } from '@testing-library/react'
import { beforeEach, describe, expect, test, vi } from 'vitest'

import { requestXunhuPayment } from '../api'
import { PAYMENT_TYPES } from '../constants'
import { requestPaymentAmount, usePayment } from './use-payment'

vi.mock('../api', async (importOriginal) => {
  const actual = await importOriginal<typeof import('../api')>()
  return {
    ...actual,
    requestXunhuPayment: vi.fn(),
  }
})

describe('payment amount routing', () => {
  beforeEach(() => vi.mocked(requestXunhuPayment).mockReset())

  test('requests the Xunhu surcharge in the payment amount preview', async () => {
    const requests: {
      amount: number
      provider?: string
      payment_method?: string
    }[] = []
    const amount = await requestPaymentAmount(
      10,
      'wxpay',
      {
        regular: async (request) => {
          requests.push(request)
          return {
            success: true,
            data: '10.20',
            base_amount: '10.00',
            fee_amount: '0.20',
            fee_percent: 2,
          }
        },
        stripe: async () => ({ success: true, data: '0' }),
        waffo: async () => ({ success: true, data: '0' }),
        waffoPancake: async () => ({ success: true, data: '0' }),
      },
      'xunhu'
    )

    expect(amount).toEqual({
      amount: 10.2,
      baseAmount: 10,
      feeAmount: 0.2,
      feePercent: 2,
    })
    expect(requests).toEqual([
      { amount: 10, provider: 'xunhu', payment_method: 'wxpay' },
    ])
  })

  test('uses the dedicated Waffo amount calculator', async () => {
    const calls: string[] = []
    const preview = await requestPaymentAmount(120, PAYMENT_TYPES.WAFFO, {
      regular: async () => {
        calls.push('regular')
        return { success: true, data: '1' }
      },
      stripe: async () => {
        calls.push('stripe')
        return { success: true, data: '2' }
      },
      waffo: async (request) => {
        calls.push(`waffo:${request.amount}`)
        return { success: true, data: '18.75' }
      },
      waffoPancake: async () => {
        calls.push('pancake')
        return { success: true, data: '4' }
      },
    })

    expect(preview).toEqual({ amount: 18.75 })
    expect(calls).toEqual(['waffo:120'])
  })

  test('redirects the reserved payment tab directly to Xunhu checkout', async () => {
    vi.mocked(requestXunhuPayment).mockResolvedValue({
      success: true,
      data: {
        url: 'https://pay.xunhupay.com/checkout',
        url_qrcode: 'weixin://pay/order',
      },
    })
    const paymentWindow = {
      closed: false,
      location: { href: 'about:blank' },
      close: vi.fn(),
    } as unknown as Window
    const { result } = renderHook(() => usePayment())
    let success = false

    await act(async () => {
      success = await result.current.processXunhuPayment(
        10,
        'wxpay',
        paymentWindow
      )
    })

    expect(success).toBe(true)
    expect(paymentWindow.location.href).toBe(
      'https://pay.xunhupay.com/checkout'
    )
    expect(paymentWindow.close).not.toHaveBeenCalled()
    expect(result.current.xunhuCheckout).toBeNull()
  })

  test('shows the QR checkout when the browser blocks the reserved payment tab', async () => {
    vi.mocked(requestXunhuPayment).mockResolvedValue({
      success: true,
      data: {
        url: 'https://pay.xunhupay.com/checkout',
        url_qrcode: 'weixin://pay/order',
      },
    })
    const { result } = renderHook(() => usePayment())

    await act(async () => {
      await result.current.processXunhuPayment(10, 'wxpay', null)
    })

    expect(result.current.xunhuCheckout).toEqual({
      url: 'https://pay.xunhupay.com/checkout',
      url_qrcode: 'weixin://pay/order',
    })
  })
})
