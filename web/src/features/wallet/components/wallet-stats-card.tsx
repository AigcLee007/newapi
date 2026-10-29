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
import { Activity, ArrowDownToLine, BarChart3, WalletCards } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { Button } from '@/components/ui/button'
import { IconBadge, type IconBadgeTone } from '@/components/ui/icon-badge'
import { Skeleton } from '@/components/ui/skeleton'
import { formatNumber, formatQuota } from '@/lib/format'

import type { UserWalletData } from '../types'

interface WalletStatsCardProps {
  user: UserWalletData | null
  loading?: boolean
}

export function WalletStatsCard(props: WalletStatsCardProps) {
  const { t } = useTranslation()
  if (props.loading) {
    return (
      <section
        aria-label={t('Wallet')}
        className='bg-card grid overflow-hidden rounded-lg border lg:grid-cols-[minmax(0,1.25fr)_minmax(0,1fr)]'
      >
        <div className='bg-foreground flex min-h-40 items-center justify-between gap-4 p-5 sm:p-7'>
          <div className='min-w-0 flex-1 space-y-3'>
            <Skeleton className='bg-background/20 h-4 w-32' />
            <Skeleton className='bg-background/20 h-10 w-48' />
            <Skeleton className='bg-background/20 h-8 w-32' />
          </div>
          <WalletCards className='text-background/30 hidden size-14 sm:block' />
        </div>
        <div className='grid grid-cols-2 divide-x'>
          {['usage', 'requests'].map((key) => (
            <div
              key={key}
              className='flex min-w-0 flex-col justify-center px-4 py-5 sm:px-6'
            >
              <Skeleton className='h-3.5 w-full' />
              <Skeleton className='mt-2 h-6 w-full sm:h-7' />
              <Skeleton className='mt-1.5 hidden h-3.5 w-24 md:block' />
            </div>
          ))}
        </div>
      </section>
    )
  }

  const stats: {
    label: string
    value: string
    description: string
    icon: typeof WalletCards
    tone: IconBadgeTone
  }[] = [
    {
      label: t('Current Balance'),
      value: formatQuota(props.user?.quota ?? 0),
      description: t('Remaining quota'),
      icon: WalletCards,
      tone: 'success',
    },
    {
      label: t('Total Usage'),
      value: formatQuota(props.user?.used_quota ?? 0),
      description: t('Total consumed quota'),
      icon: BarChart3,
      tone: 'info',
    },
    {
      label: t('API Requests'),
      value: formatNumber(props.user?.request_count ?? 0),
      description: t('Total requests made'),
      icon: Activity,
      tone: 'chart-4',
    },
  ]

  const balance = stats[0]
  const secondaryStats = stats.slice(1)

  return (
    <section
      aria-label={t('Wallet')}
      className='bg-card grid overflow-hidden rounded-lg border lg:grid-cols-[minmax(0,1.25fr)_minmax(0,1fr)]'
    >
      <div className='bg-foreground text-background flex min-h-40 items-center justify-between gap-4 p-5 sm:p-7'>
        <div className='min-w-0'>
          <dl className='min-w-0'>
            <dt className='text-background/70 flex items-center gap-2 text-xs font-semibold uppercase'>
              <WalletCards className='size-4' />
              {balance.label}
            </dt>
            <dd className='mt-4 font-mono text-3xl font-bold tabular-nums sm:text-4xl'>
              {balance.value}
            </dd>
          </dl>
          <Button
            variant='secondary'
            className='mt-5 gap-2'
            render={<a href='#wallet-add-funds' />}
          >
            <ArrowDownToLine />
            {t('Add Funds')}
          </Button>
        </div>
        <div className='text-background/15 hidden sm:block'>
          <WalletCards className='size-20 lg:size-24' />
        </div>
      </div>

      <dl className='grid grid-cols-2 divide-x'>
        {secondaryStats.map((item) => (
          <div
            key={item.label}
            className='flex min-w-0 flex-col justify-center px-4 py-5 sm:px-6'
          >
            <dt className='text-muted-foreground flex min-w-0 items-center gap-2 text-xs font-medium uppercase'>
              <IconBadge tone={item.tone} size='sm'>
                <item.icon />
              </IconBadge>
              <span className='truncate'>{item.label}</span>
            </dt>
            <dd className='text-foreground mt-3 truncate font-mono text-xl font-bold tabular-nums sm:text-2xl'>
              {item.value}
            </dd>
            <dd className='text-muted-foreground mt-1 text-xs'>
              {item.description}
            </dd>
          </div>
        ))}
      </dl>
    </section>
  )
}
