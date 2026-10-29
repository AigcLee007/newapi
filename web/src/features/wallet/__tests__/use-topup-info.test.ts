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
import { renderHook, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, test, vi } from 'vitest'

import { getTopupInfo } from '../api'
import { useTopupInfo } from '../hooks/use-topup-info'
import type { TopupInfoResponse } from '../types'

vi.mock('../api', () => ({
  getTopupInfo: vi.fn(),
}))

describe('useTopupInfo', () => {
  beforeEach(() => vi.mocked(getTopupInfo).mockReset())

  test('preserves Xunhu provider fields from the top-up API', async () => {
    const response = {
      success: true,
      data: {
        enable_online_topup: false,
        enable_xunhu_topup: true,
        enable_stripe_topup: false,
        pay_methods: [
          {
            name: 'WeChat',
            type: 'wxpay',
            provider: 'xunhu',
            xunhu_type: 'WAP',
          },
        ],
        min_topup: 1,
        stripe_min_topup: 1,
        amount_options: [10],
        discount: {},
      },
    } satisfies TopupInfoResponse
    vi.mocked(getTopupInfo).mockResolvedValue(response)

    const { result } = renderHook(() => useTopupInfo())

    await waitFor(() => {
      expect(result.current.topupInfo?.pay_methods[0]).toMatchObject({
        provider: 'xunhu',
        xunhu_type: 'WAP',
      })
    })
  })
})
