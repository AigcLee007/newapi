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
import { useMemo, useState } from 'react'
import { useTranslation } from 'react-i18next'

import { ErrorState } from '@/components/error-state'
import { PublicLayout } from '@/components/layout'
import { PageTransition } from '@/components/page-transition'

import { usePricingData } from '../hooks/use-pricing-data'
import { buildGroupPricingSections } from '../lib/group-pricing'
import type { PricingModel } from '../types'
import { EmptyState } from './empty-state'
import { GroupPricingTable } from './group-pricing-table'
import { LoadingSkeleton } from './loading-skeleton'
import { SearchBar } from './search-bar'

export function GroupPricingContent(props: {
  models: PricingModel[]
  usableGroup: Record<string, string>
  groupRatio: Record<string, number>
  priceRate: number
  usdExchangeRate: number
}) {
  const { t } = useTranslation()
  const [search, setSearch] = useState('')
  const sections = useMemo(
    () =>
      buildGroupPricingSections(
        props.models,
        props.usableGroup,
        props.groupRatio,
        search
      ),
    [props.groupRatio, props.models, props.usableGroup, search]
  )
  const visibleGroupCount = sections.length

  return (
    <PageTransition className='mx-auto w-full max-w-[1800px] px-4 pt-20 pb-8 sm:px-6 sm:pt-20 xl:px-8'>
      <header className='mb-4 border-b pb-3'>
        <div className='flex flex-wrap items-center justify-between gap-x-6 gap-y-3'>
          <div className='min-w-0'>
            <h1 className='text-xl font-semibold sm:text-2xl'>
              {t('Complete group pricing')}
            </h1>
            <p className='text-muted-foreground mt-0.5 text-sm'>
              {t('Browse every model price available in each group.')}
            </p>
          </div>
          <div className='flex w-full flex-wrap items-center justify-between gap-3 sm:w-auto sm:justify-end'>
            <SearchBar
              value={search}
              onChange={setSearch}
              onClear={() => setSearch('')}
              placeholder={t('Search models')}
              className='w-full max-w-[320px] sm:w-[280px]'
            />
            <p className='text-muted-foreground shrink-0 text-sm'>
              {t('{{count}} groups', { count: visibleGroupCount })}
            </p>
          </div>
        </div>
      </header>

      {sections.length === 0 ? (
        <EmptyState
          searchQuery={search}
          hasActiveFilters={Boolean(search.trim())}
          onClearFilters={() => setSearch('')}
        />
      ) : (
        <div className='space-y-4'>
          {sections.map((section) => (
            <GroupPricingTable
              key={section.name}
              group={section.name}
              description={section.description}
              ratio={section.ratio}
              models={section.models}
              priceRate={props.priceRate}
              usdExchangeRate={props.usdExchangeRate}
            />
          ))}
        </div>
      )}
    </PageTransition>
  )
}

export function GroupPricingPage() {
  const { t } = useTranslation()
  const pricing = usePricingData()

  if (pricing.isLoading) {
    return (
      <PublicLayout showMainContainer={false}>
        <div className='mx-auto w-full max-w-[1800px] px-4 pt-20 sm:px-6 sm:pt-20 xl:px-8'>
          <LoadingSkeleton viewMode='table' />
        </div>
      </PublicLayout>
    )
  }

  if (pricing.error) {
    return (
      <PublicLayout showMainContainer={false}>
        <ErrorState
          title={t('Failed to load model pricing')}
          description={String(pricing.error)}
          onRetry={() => void pricing.refetch()}
          className='mx-4 mt-16'
        />
      </PublicLayout>
    )
  }

  return (
    <PublicLayout showMainContainer={false}>
      <GroupPricingContent
        models={pricing.models}
        usableGroup={pricing.usableGroup}
        groupRatio={pricing.groupRatio}
        priceRate={pricing.priceRate}
        usdExchangeRate={pricing.usdExchangeRate}
      />
    </PublicLayout>
  )
}
