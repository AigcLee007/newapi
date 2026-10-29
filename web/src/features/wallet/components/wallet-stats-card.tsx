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
import { Activity, BarChart3, WalletCards } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { IconBadge, type IconBadgeTone } from '@/components/ui/icon-badge'
import { Skeleton } from '@/components/ui/skeleton'
import { formatNumber, formatQuota } from '@/lib/format'
import { cn } from '@/lib/utils'

import type { UserWalletData } from '../types'

interface WalletStatsCardProps {
  user: UserWalletData | null
  loading?: boolean
}

export function WalletStatsCard(props: WalletStatsCardProps) {
  const { t } = useTranslation()
  if (props.loading) {
    return (
      <div className='grid grid-cols-2 overflow-hidden rounded-lg border sm:grid-cols-3'>
        {['balance', 'usage', 'requests'].map((key, index) => (
          <div
            key={key}
            className={cn(
              'min-w-0 px-3 py-3 sm:px-5 sm:py-4',
              index === 0 &&
                'col-span-2 border-b bg-success/5 sm:col-span-1 sm:border-b-0',
              index > 0 && 'border-t sm:border-t-0',
              index === 1 && 'sm:border-l',
              index === 2 && 'border-l sm:border-l'
            )}
          >
            <Skeleton className='h-3.5 w-full' />
            <Skeleton className='mt-2 h-6 w-full sm:h-7' />
            <Skeleton className='mt-1.5 hidden h-3.5 w-24 md:block' />
          </div>
        ))}
      </div>
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

  return (
    <dl className='bg-card grid grid-cols-2 overflow-hidden rounded-lg border sm:grid-cols-3'>
      {stats.map((item, index) => (
        <div
          key={item.label}
          className={cn(
            'min-w-0 px-3 py-3 sm:px-5 sm:py-4',
            index === 0 &&
              'col-span-2 border-b bg-success/5 sm:col-span-1 sm:border-b-0',
            index > 0 && 'border-t sm:border-t-0',
            index === 1 && 'sm:border-l',
            index === 2 && 'border-l sm:border-l'
          )}
        >
          <div className='flex items-center gap-1.5 sm:gap-2.5'>
            <IconBadge tone={item.tone} size='stat'>
              <item.icon />
            </IconBadge>
            <dt className='text-muted-foreground truncate text-[11px] font-medium tracking-wider uppercase sm:text-xs'>
              {item.label}
            </dt>
          </div>

          <dd
            className={cn(
              'mt-1.5 font-mono font-bold tabular-nums sm:mt-2.5',
              index === 0
                ? 'text-foreground text-xl sm:text-2xl'
                : 'text-foreground text-base sm:text-xl'
            )}
          >
            {item.value}
          </dd>
          <div className='text-muted-foreground/60 mt-1 hidden text-xs md:block'>
            {item.description}
          </div>
        </div>
      ))}
    </dl>
  )
}
