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
import {
  formatBillingCurrencyFromUSD,
  getCurrencyDisplay,
} from '@/lib/currency'

import { EXCLUDED_GROUPS } from '../constants'
import type { PricingModel } from '../types'

export type GroupPricingSection = {
  name: string
  description: string
  ratio: number
  models: PricingModel[]
}

export function buildGroupPricingSections(
  models: PricingModel[],
  usableGroups: Record<string, string>,
  groupRatios: Record<string, number>,
  searchQuery = ''
): GroupPricingSection[] {
  const normalizedSearch = searchQuery.trim().toLowerCase()

  return Object.entries(usableGroups).flatMap(([name, group]) => {
    if (EXCLUDED_GROUPS.includes(name)) return []

    const groupModels = models.filter((model) => {
      const enabledGroups = model.enable_groups ?? []
      const enabledInGroup =
        enabledGroups.includes(name) || enabledGroups.includes('all')
      if (!enabledInGroup) return false
      if (!normalizedSearch) return true

      const searchableText = `${model.model_name} ${model.vendor_name ?? ''} ${model.vendor_description ?? ''}`
      return searchableText.toLowerCase().includes(normalizedSearch)
    })

    if (normalizedSearch && groupModels.length === 0) return []

    return [
      {
        name,
        description: group,
        ratio: groupRatios[name] ?? 1,
        models: groupModels,
      },
    ]
  })
}

export function formatUSDReferencePrice(amountUSD: number): string {
  const meta = getCurrencyDisplay().meta
  const exchangeRate =
    meta.kind === 'currency' || meta.kind === 'custom' ? meta.exchangeRate : 1
  const formattedAmount = formatBillingCurrencyFromUSD(
    amountUSD / exchangeRate,
    {
      showSymbol: false,
      digitsLarge: 4,
      digitsSmall: 6,
      abbreviate: false,
    }
  )

  return `$${formattedAmount}`
}
