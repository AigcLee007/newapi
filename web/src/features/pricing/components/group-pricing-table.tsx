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
      className='overflow-hidden rounded-lg border'
    >
      <header className='bg-muted/40 flex flex-wrap items-center justify-between gap-3 border-b px-4 py-3'>
        <div className='min-w-0'>
          <h2 id={`group-${props.group}`} className='font-medium'>
            {props.group}
          </h2>
          {props.description && (
            <p className='text-muted-foreground mt-0.5 text-sm'>
              {props.description}
            </p>
          )}
        </div>
        <GroupBadge group={props.group} ratio={props.ratio} />
      </header>
      <div
        role='region'
        aria-label={t('Pricing for group {{group}}', { group: props.group })}
        tabIndex={0}
        className='focus-visible:outline-ring overflow-x-auto focus-visible:outline-2 focus-visible:outline-offset-[-2px]'
      >
        <table className='w-full min-w-[880px] border-collapse text-sm'>
          <thead>
            <tr className='border-b'>
              <th
                rowSpan={2}
                scope='col'
                className='min-w-[220px] px-4 py-2 text-left font-medium'
              >
                {t('Model')}
              </th>
              <th
                colSpan={columnCount}
                className='px-3 py-2 text-center font-medium'
              >
                {t('Customer price')} ·{' '}
                {currencyLabel === 'Tokens' ? 'USD' : currencyLabel} /{' '}
                {t('1M token')}
              </th>
              <th
                colSpan={columnCount}
                className='px-3 py-2 text-center font-medium'
              >
                {t('USD reference')} · USD / {t('1M token')}
              </th>
              <th
                rowSpan={2}
                scope='col'
                className='min-w-24 px-3 py-2 text-right font-medium'
              >
                {t('Discount ratio')}
              </th>
            </tr>
            <tr className='text-muted-foreground border-b'>
              {priceColumns.map((column) => (
                <th
                  key={`customer-${column.type}`}
                  scope='col'
                  className='min-w-24 px-3 py-2 text-right text-xs font-normal'
                >
                  {t(column.label)}
                </th>
              ))}
              {priceColumns.map((column) => (
                <th
                  key={`usd-${column.type}`}
                  scope='col'
                  className='min-w-24 px-3 py-2 text-right text-xs font-normal'
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
                className='border-b last:border-b-0'
              >
                <th scope='row' className='px-4 py-3 text-left font-medium'>
                  <span className='block max-w-[320px] break-words'>
                    {model.model_name}
                  </span>
                  {model.vendor_name && (
                    <span className='text-muted-foreground mt-1 block text-xs font-normal'>
                      {model.vendor_name}
                    </span>
                  )}
                </th>
                {isDynamicPricingModel(model) ? (
                  <>
                    <td
                      colSpan={columnCount}
                      className='bg-emerald-500/5 px-3 py-3 text-center'
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
                    <td colSpan={columnCount} className='px-3 py-3 text-center'>
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
                        className='bg-emerald-500/5 px-3 py-3 text-right font-mono tabular-nums'
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
                    {priceColumns.map((column) => (
                      <td
                        key={`usd-${column.type}`}
                        className='px-3 py-3 text-right font-mono tabular-nums'
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
                <td className='px-3 py-3 text-right font-mono tabular-nums'>
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
