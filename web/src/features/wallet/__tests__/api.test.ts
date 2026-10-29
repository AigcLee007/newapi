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
import { beforeEach, describe, expect, test, vi } from 'vitest'

import { api } from '@/lib/api'

import { requestPayment, requestXunhuPayment } from '../api'

vi.mock('@/lib/api', () => ({
  api: { post: vi.fn() },
}))

describe('wallet payment endpoints', () => {
  beforeEach(() => vi.mocked(api.post).mockReset())

  test('keeps Epay on its existing payment endpoint', async () => {
    vi.mocked(api.post).mockResolvedValueOnce({
      data: {
        message: 'success',
        data: { pid: 'merchant' },
        url: 'https://epay.example/pay',
      },
    } as never)
    const request = { amount: 50, payment_method: 'alipay' }

    const response = await requestPayment(request)

    expect(api.post).toHaveBeenCalledWith(
      '/api/user/pay',
      request,
      expect.objectContaining({ skipBusinessError: true })
    )
    expect(response.url).toBe('https://epay.example/pay')
  })

  test('sends Xunhu payment methods to the Xunhu endpoint', async () => {
    vi.mocked(api.post).mockResolvedValueOnce({
      data: {
        message: 'success',
        data: {
          url: 'https://pay.xunhupay.com/checkout',
          url_qrcode: 'weixin://pay/order',
        },
      },
    } as never)
    const request = { amount: 50, payment_method: 'alipay' }

    const response = await requestXunhuPayment(request)

    expect(api.post).toHaveBeenCalledWith(
      '/api/user/xunhu/pay',
      request,
      expect.objectContaining({ skipBusinessError: true })
    )
    expect(response.data?.url_qrcode).toBe('weixin://pay/order')
  })
})
