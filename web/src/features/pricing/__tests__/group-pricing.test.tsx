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
  createMemoryHistory,
  createRootRoute,
  createRouter,
  RouterProvider,
} from '@tanstack/react-router'
import { render, screen, within } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { describe, expect, it } from 'vitest'

import {
  DEFAULT_CURRENCY_CONFIG,
  useSystemConfigStore,
} from '@/stores/system-config-store'

import { GroupPricingLink } from '../components/group-pricing-link'
import { GroupPricingContent } from '../components/group-pricing-page'
import {
  buildGroupPricingSections,
  formatUSDReferencePrice,
} from '../lib/group-pricing'
import type { PricingModel } from '../types'

function pricingModel(
  modelName: string,
  enableGroups: string[],
  vendorName?: string,
  overrides: Partial<PricingModel> = {}
): PricingModel {
  return {
    id: modelName.length,
    model_name: modelName,
    vendor_name: vendorName,
    quota_type: 0,
    model_ratio: 1,
    completion_ratio: 2,
    enable_groups: enableGroups,
    ...overrides,
  }
}

const usableGroups = {
  basic: 'Basic access',
  premium: 'Premium access',
  empty: '',
  auto: 'Automatic',
}

async function renderWithRouter(component: React.ReactNode) {
  const router = createRouter({
    routeTree: createRootRoute({ component: () => component }),
    history: createMemoryHistory({ initialEntries: ['/'] }),
  })
  await router.load()
  return render(<RouterProvider router={router} />)
}

describe('group pricing sections', () => {
  it('lists each model under every enabled group and omits hidden or empty groups', () => {
    const models = [
      pricingModel('shared-model', ['basic', 'premium']),
      pricingModel('basic-model', ['basic']),
      pricingModel('unused-model', ['private']),
    ]

    const sections = buildGroupPricingSections(models, usableGroups, {
      basic: 0.8,
      premium: 1.5,
      empty: 3,
    })

    expect(sections).toEqual([
      {
        name: 'basic',
        description: 'Basic access',
        ratio: 0.8,
        models: [models[0], models[1]],
      },
      {
        name: 'premium',
        description: 'Premium access',
        ratio: 1.5,
        models: [models[0]],
      },
      {
        name: 'empty',
        description: '',
        ratio: 3,
        models: [],
      },
    ])
  })

  it('includes models enabled for all groups in every visible group', () => {
    const model = pricingModel('all-model', ['all'])
    const sections = buildGroupPricingSections(
      [model],
      { basic: usableGroups.basic, premium: usableGroups.premium },
      {}
    )

    expect(sections.map((section) => section.models)).toEqual([
      [model],
      [model],
    ])
  })

  it('filters model name and vendor text without leaving empty group sections', () => {
    const models = [
      pricingModel('claude-sonnet', ['basic'], 'Anthropic'),
      pricingModel('gpt-fast', ['premium'], 'OpenAI'),
    ]

    const sections = buildGroupPricingSections(
      models,
      {
        basic: usableGroups.basic,
        premium: usableGroups.premium,
      },
      { basic: 1, premium: 2 },
      'openai'
    )

    expect(sections).toEqual([
      {
        name: 'premium',
        description: 'Premium access',
        ratio: 2,
        models: [models[1]],
      },
    ])
  })

  it('keeps reference prices in USD when the site currency is CNY', () => {
    useSystemConfigStore.getState().setConfig({
      currency: {
        ...DEFAULT_CURRENCY_CONFIG,
        quotaDisplayType: 'CNY',
        usdExchangeRate: 7,
      },
    })

    try {
      expect(formatUSDReferencePrice(2)).toBe('$2')
    } finally {
      useSystemConfigStore.getState().setConfig({
        currency: { ...DEFAULT_CURRENCY_CONFIG },
      })
    }
  })
})

describe('group pricing page content', () => {
  it('exposes a visible link from the model marketplace to group pricing', async () => {
    await renderWithRouter(<GroupPricingLink />)

    const link = screen.getByRole('button', {
      name: 'Complete group pricing',
    })
    expect(link).toBeVisible()
    expect(link).toHaveAttribute('href', '/pricing/groups')
  })

  it('shows each group multiplier with customer and USD reference prices', async () => {
    const model = pricingModel('gpt-fast', ['default'], 'OpenAI', {
      model_ratio: 0.5,
      completion_ratio: 2,
      cache_ratio: 0.5,
      create_cache_ratio: 1.5,
    })

    await renderWithRouter(
      <GroupPricingContent
        models={[model]}
        usableGroup={{ default: 'Standard access' }}
        groupRatio={{ default: 0.12 }}
        priceRate={1}
        usdExchangeRate={1}
      />
    )

    const region = screen.getByRole('region', {
      name: 'Pricing for group default',
    })
    expect(region).toHaveAttribute('tabindex', '0')
    expect(region).toHaveClass('overflow-x-auto')
    const table = within(region).getByRole('table', {
      name: 'Pricing for group default',
    })
    expect(
      screen.getByRole('heading', { name: 'Complete group pricing' })
    ).toBeVisible()
    expect(within(table).getByText('gpt-fast')).toBeVisible()
    expect(within(table).getByText('0.12x')).toBeVisible()
    expect(within(table).getByText('$0.12')).toBeVisible()
    expect(within(table).getByText('$0.24')).toBeVisible()
    expect(within(table).getByText('$0.06')).toBeVisible()
    expect(within(table).getByText('$0.18')).toBeVisible()
    expect(within(table).getByText('$1')).toBeVisible()
    expect(within(table).getByText('$2')).toBeVisible()
    expect(within(table).getByText('$0.5')).toBeVisible()
    expect(within(table).getByText('$1.5')).toBeVisible()
  })

  it('shows expression pricing with each group multiplier and USD reference', async () => {
    const model = pricingModel('tiered-model', ['default'], undefined, {
      billing_mode: 'tiered_expr',
      billing_expr: 'tier("short", p * 0.5)',
      group_ratio: { default: 0.5 },
    })

    await renderWithRouter(
      <GroupPricingContent
        models={[model]}
        usableGroup={{ default: '' }}
        groupRatio={{ default: 0.5 }}
        priceRate={1}
        usdExchangeRate={1}
      />
    )

    const table = screen.getByRole('table')
    expect(screen.getByText('0.25')).toBeVisible()
    expect(screen.getByText('0.5')).toBeVisible()
    expect(
      within(table).getByRole('row', { name: /tiered-model/ })
    ).toHaveTextContent('0.5x')
  })

  it('shows configured groups that do not currently have enabled models', async () => {
    await renderWithRouter(
      <GroupPricingContent
        models={[]}
        usableGroup={{ unused: 'Available to users' }}
        groupRatio={{ unused: 1.5 }}
        priceRate={1}
        usdExchangeRate={1}
      />
    )

    expect(screen.getByRole('heading', { name: 'unused' })).toBeVisible()
    expect(screen.getByText('Available to users')).toBeVisible()
    expect(
      screen.getByText('No models are enabled for this group.')
    ).toBeVisible()
  })

  it('filters groups by model name and clears the search when there are no matches', async () => {
    const user = userEvent.setup()
    const models = [
      pricingModel('gpt-fast', ['default'], 'OpenAI'),
      pricingModel('claude-sonnet', ['premium'], 'Anthropic'),
    ]

    await renderWithRouter(
      <GroupPricingContent
        models={models}
        usableGroup={{
          default: '',
          premium: '',
        }}
        groupRatio={{ default: 1, premium: 2 }}
        priceRate={1}
        usdExchangeRate={1}
      />
    )

    const search = screen.getByRole('textbox', { name: 'Search models' })
    await user.type(search, 'gpt')
    expect(screen.getByText('gpt-fast')).toBeVisible()
    expect(screen.queryByText('claude-sonnet')).not.toBeInTheDocument()

    await user.clear(search)
    await user.type(search, 'missing')
    expect(
      screen.getByRole('heading', { name: 'No models found' })
    ).toBeVisible()
    await user.click(screen.getByRole('button', { name: 'Clear all filters' }))
    expect(screen.getByText('gpt-fast')).toBeVisible()
    expect(screen.getByText('claude-sonnet')).toBeVisible()
  })
})
