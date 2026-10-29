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
import { useTranslation } from 'react-i18next'

import { GroupBadge } from '@/components/group-badge'
import { getCurrencyLabel } from '@/lib/currency'
import { cn } from '@/lib/utils'

import { DEFAULT_TOKEN_UNIT } from '../constants'
import { isDynamicPricingModel } from '../lib/dynamic-price'
import { formatUSDReferencePrice } from '../lib/group-pricing'
import {
  calculateTokenPrice,
  formatFixedPrice,
  formatGroupPrice,
} from '../lib/price'
import type { PricingModel, PriceType } from '../types'
import { ModelPriceCell } from './model-price-cell'

const BASE_PRICE_COLUMNS: Array<{ type: PriceType; label: string }> = [
  { type: 'input', label: 'Input' },
  { type: 'output', label: 'Output' },
]

export function GroupPricingTable(props: {
  group: string
  description: string
  ratio: number
  models: PricingModel[]
  priceRate: number
  usdExchangeRate: number
}) {
  const { t } = useTranslation()
  const priceColumns = [
    ...BASE_PRICE_COLUMNS,
    ...(props.models.some((model) => model.cache_ratio != null)
      ? [{ type: 'cache' as const, label: 'Cache' }]
      : []),
    ...(props.models.some((model) => model.create_cache_ratio != null)
      ? [{ type: 'create_cache' as const, label: 'Cache Write' }]
      : []),
    ...(props.models.some((model) => model.image_ratio != null)
      ? [{ type: 'image' as const, label: 'Image' }]
      : []),
    ...(props.models.some((model) => model.audio_ratio != null)
      ? [{ type: 'audio_input' as const, label: 'Audio In' }]
      : []),
    ...(props.models.some(
      (model) =>
        model.audio_ratio != null && model.audio_completion_ratio != null
    )
      ? [{ type: 'audio_output' as const, label: 'Audio Out' }]
      : []),
  ]
  const columnCount = priceColumns.length
  const currencyLabel = getCurrencyLabel()

  return (
    <section
      aria-labelledby={`group-${props.group}`}
      className='bg-card overflow-hidden rounded-md border border-emerald-700/35'
    >
      <header className='bg-muted/40 flex flex-wrap items-center gap-2 border-b px-4 py-2.5'>
        <div className='min-w-0'>
          <h2
            id={`group-${props.group}`}
            aria-label={props.group}
            className='flex min-w-0'
          >
            <GroupBadge
              group={props.group}
              ratio={props.ratio}
              showDot={false}
              copyable={false}
              size='sm'
            />
          </h2>
          {props.description && (
            <p className='text-muted-foreground mt-1 text-xs'>
              {props.description}
            </p>
          )}
        </div>
      </header>
      <div
        role='region'
        aria-label={t('Pricing for group {{group}}', { group: props.group })}
        tabIndex={0}
        className='focus-visible:outline-ring overflow-x-auto focus-visible:outline-2 focus-visible:outline-offset-[-2px]'
      >
        <table className='w-full min-w-[1024px] border-collapse text-sm'>
          <caption className='sr-only'>
            {t('Pricing for group {{group}}', { group: props.group })}
          </caption>
          <thead>
            <tr className='bg-muted/20 border-b'>
              <th
                rowSpan={2}
                scope='col'
                className='bg-card border-border/70 sticky left-0 z-20 min-w-[280px] border-r px-4 py-2.5 text-left font-medium md:static md:z-auto'
              >
                {t('Model')}
              </th>
              <th
                colSpan={columnCount}
                className='border-b border-emerald-700/20 bg-emerald-500/10 px-3 py-2 text-center font-semibold text-emerald-700 dark:text-emerald-300'
              >
                {t('Customer price')} ·{' '}
                {currencyLabel === 'Tokens' ? 'USD' : currencyLabel} /{' '}
                {t('1M token')}
              </th>
              <th
                colSpan={columnCount}
                className='border-border/70 border-b border-l px-3 py-2 text-center font-medium'
              >
                {t('USD reference')} · USD / {t('1M token')}
              </th>
              <th
                rowSpan={2}
                scope='col'
                className='border-border/70 min-w-28 border-l px-3 py-2 text-right font-medium'
              >
                {t('Discount ratio')}
              </th>
            </tr>
            <tr className='text-muted-foreground border-b'>
              {priceColumns.map((column) => (
                <th
                  key={`customer-${column.type}`}
                  scope='col'
                  className='min-w-28 bg-emerald-500/10 px-3 py-2 text-center text-xs font-normal'
                >
                  {t(column.label)}
                </th>
              ))}
              {priceColumns.map((column, index) => (
                <th
                  key={`usd-${column.type}`}
                  scope='col'
                  className={cn(
                    'min-w-28 px-3 py-2 text-center text-xs font-normal',
                    index === 0 && 'border-l border-border/70'
                  )}
                >
                  {t(column.label)}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {props.models.map((model) => (
              <tr
                key={`${props.group}-${model.id}-${model.model_name}`}
                className='border-border/70 border-b last:border-b-0'
              >
                <th
                  scope='row'
                  className='bg-card border-border/70 sticky left-0 z-10 border-r px-4 py-2.5 text-left align-middle font-medium md:static md:z-auto'
                >
                  <span className='block max-w-[360px] font-mono text-[13px] font-semibold break-words'>
                    {model.model_name}
                  </span>
                  {model.vendor_name && (
                    <span className='text-muted-foreground mt-1 block text-xs font-normal'>
                      {model.vendor_name}
                    </span>
                  )}
                  {model.description && (
                    <span className='text-muted-foreground mt-1 block max-w-[420px] text-xs leading-5 font-normal break-words'>
                      {model.description}
                    </span>
                  )}
                </th>
                {isDynamicPricingModel(model) ? (
                  <>
                    <td
                      colSpan={columnCount}
                      className='bg-emerald-500/10 px-3 py-2.5 text-center font-mono text-xs tabular-nums'
                    >
                      <ModelPriceCell
                        model={model}
                        options={{
                          tokenUnit: DEFAULT_TOKEN_UNIT,
                          priceRate: props.priceRate,
                          usdExchangeRate: props.usdExchangeRate,
                          selectedGroup: props.group,
                        }}
                        showExpression={false}
                      />
                    </td>
                    <td
                      colSpan={columnCount}
                      className='border-border/70 border-l px-3 py-2.5 text-center font-mono text-xs tabular-nums'
                    >
                      <ModelPriceCell
                        model={{
                          ...model,
                          group_ratio: {
                            ...model.group_ratio,
                            [props.group]: 1,
                          },
                        }}
                        options={{
                          tokenUnit: DEFAULT_TOKEN_UNIT,
                          priceRate: 1,
                          usdExchangeRate: 1,
                          selectedGroup: props.group,
                        }}
                        showExpression={false}
                      />
                    </td>
                  </>
                ) : (
                  <>
                    {priceColumns.map((column) => (
                      <td
                        key={`customer-${column.type}`}
                        className='bg-emerald-500/10 px-3 py-2.5 text-right font-mono text-xs font-semibold tabular-nums'
                      >
                        <GroupPriceValue
                          model={model}
                          group={props.group}
                          ratio={props.ratio}
                          type={column.type}
                          priceRate={props.priceRate}
                          usdExchangeRate={props.usdExchangeRate}
                          currency='customer'
                        />
                      </td>
                    ))}
                    {priceColumns.map((column, index) => (
                      <td
                        key={`usd-${column.type}`}
                        className={cn(
                          'px-3 py-2.5 text-right font-mono text-xs tabular-nums',
                          index === 0 && 'border-l border-border/70'
                        )}
                      >
                        <GroupPriceValue
                          model={model}
                          group={props.group}
                          ratio={props.ratio}
                          type={column.type}
                          priceRate={1}
                          usdExchangeRate={1}
                          currency='usd'
                        />
                      </td>
                    ))}
                  </>
                )}
                <td className='border-border/70 border-l px-3 py-2.5 text-right font-mono text-xs font-medium tabular-nums'>
                  {props.ratio}x
                </td>
              </tr>
            ))}
            {props.models.length === 0 && (
              <tr>
                <td
                  colSpan={columnCount * 2 + 2}
                  className='text-muted-foreground px-4 py-6 text-center'
                >
                  {t('No models are enabled for this group.')}
                </td>
              </tr>
            )}
          </tbody>
        </table>
      </div>
    </section>
  )
}

function GroupPriceValue(props: {
  model: PricingModel
  group: string
  ratio: number
  type: PriceType
  priceRate: number
  usdExchangeRate: number
  currency: 'customer' | 'usd'
}) {
  const { t } = useTranslation()

  if (props.model.quota_type === 1) {
    if (props.type !== 'input') return '—'

    return (
      <span>
        {props.currency === 'customer'
          ? formatFixedPrice(
              props.model,
              props.group,
              false,
              props.priceRate,
              props.usdExchangeRate,
              { [props.group]: props.ratio }
            )
          : formatUSDReferencePrice(props.model.model_price ?? 0)}
        <span className='text-muted-foreground ml-1 font-sans text-xs'>
          / {t('request')}
        </span>
      </span>
    )
  }

  if (props.type === 'cache' && props.model.cache_ratio == null) return '—'

  const groupRatio = props.currency === 'customer' ? props.ratio : 1
  const amount = calculateTokenPrice(props.model, props.type, groupRatio)
  if (!Number.isFinite(amount)) return '—'

  if (props.currency === 'usd') return formatUSDReferencePrice(amount)

  return formatGroupPrice(
    props.model,
    props.group,
    props.type,
    DEFAULT_TOKEN_UNIT,
    false,
    props.priceRate,
    props.usdExchangeRate,
    { [props.group]: props.ratio }
  )
}
