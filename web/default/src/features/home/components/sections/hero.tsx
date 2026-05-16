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
import { Link } from '@tanstack/react-router'
import { ArrowRight } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { Button } from '@/components/ui/button'

interface HeroProps {
  className?: string
  isAuthenticated?: boolean
}

export function Hero(props: HeroProps) {
  const { t } = useTranslation()
  const primaryTarget = props.isAuthenticated ? '/dashboard' : '/sign-in'
  const primaryLabel = props.isAuthenticated
    ? t('Open workspace')
    : t('Sign in')

  return (
    <section
      className={[
        'relative z-10 flex min-h-screen overflow-hidden bg-[#f3eee4] px-5 pt-24 text-[#171511]',
        'md:px-8 md:pt-28 dark:bg-[#151411] dark:text-[#f6f0e6]',
        props.className,
      ]
        .filter(Boolean)
        .join(' ')}
    >
      <div
        aria-hidden
        className='absolute inset-x-0 top-0 -z-10 h-px bg-[#1d1b16]/10 dark:bg-white/10'
      />
      <div className='mx-auto grid w-full max-w-7xl items-center gap-14 pb-14 md:grid-cols-[minmax(0,1fr)_27rem] md:gap-20 md:pb-18 lg:gap-24'>
        <div className='max-w-[52rem]'>
          <p
            className='landing-animate-fade-up mb-6 text-sm font-extrabold text-[#7f3a10] uppercase dark:text-[#d6a47b]'
            style={{ animationDelay: '0ms' }}
          >
            {t('One place for every model')}
          </p>
          <h1
            className='landing-animate-fade-up max-w-[820px] text-[clamp(3.45rem,6.4vw,5.75rem)] leading-[1.03] font-extrabold tracking-normal'
            style={{ animationDelay: '70ms' }}
          >
            {t('Route every request with intent.')}
          </h1>
          <p
            className='landing-animate-fade-up mt-8 max-w-[46rem] text-xl leading-8 text-[#443e34] md:text-[1.3rem] md:leading-9 dark:text-[#cfc6b8]'
            style={{ animationDelay: '140ms' }}
          >
            {t(
              'Give your team one clear entrance to choose models, control costs, create images, and track usage without exposing complex configuration.'
            )}
          </p>
          <div
            className='landing-animate-fade-up mt-10 flex flex-wrap items-center gap-6'
            style={{ animationDelay: '210ms' }}
          >
            <Button
              className='h-12 rounded-full bg-[#1d1b16] px-6 text-base font-extrabold text-[#fffaf0] hover:bg-[#343027] dark:bg-[#f4efe6] dark:text-[#161512] dark:hover:bg-white'
              render={<Link to={primaryTarget} />}
            >
              {primaryLabel}
              <ArrowRight className='ml-2 size-4' />
            </Button>
            {props.isAuthenticated && (
              <Link
                to='/keys'
                className='text-base font-bold text-[#242018] transition-colors hover:text-[#7f3a10] dark:text-[#ded5c8] dark:hover:text-[#f0c19d]'
              >
                {t('Manage tokens')}
              </Link>
            )}
          </div>
        </div>

        <div
          className='landing-animate-fade-up rounded-lg border border-[#1d1b16]/12 bg-[#fffdf8]/55 p-6 shadow-[0_22px_80px_rgba(52,43,28,0.08)] backdrop-blur-sm dark:border-white/10 dark:bg-white/[0.04] dark:shadow-none'
          style={{ animationDelay: '260ms' }}
        >
          <div className='mb-5 text-sm font-extrabold text-[#7f3a10] uppercase dark:text-[#d6a47b]'>
            {t('Today in Aittco')}
          </div>
          <div className='divide-y divide-[#1d1b16]/12 dark:divide-white/10'>
            <StatusRow label={t('Chat routing')} value={t('stable')} />
            <StatusRow label={t('Image tasks')} value={t('COS ready')} />
            <StatusRow label={t('Usage guardrails')} value={t('active')} />
          </div>
          <div className='mt-6 grid grid-cols-3 gap-3'>
            <Metric value='42' label={t('models')} />
            <Metric value='3' label={t('routes')} />
            <Metric value='1' label={t('bill')} />
          </div>
        </div>
      </div>
    </section>
  )
}

function StatusRow(props: { label: string; value: string }) {
  return (
    <div className='flex items-center justify-between gap-4 py-4 text-base font-extrabold'>
      <span>{props.label}</span>
      <span className='text-sm font-bold text-[#62584b] dark:text-[#cfc6b8]'>
        {props.value}
      </span>
    </div>
  )
}

function Metric(props: { value: string; label: string }) {
  return (
    <div className='rounded-lg bg-[#ebe3d5] p-4 dark:bg-white/10'>
      <strong className='block text-2xl leading-none'>{props.value}</strong>
      <span className='mt-2 block text-xs text-[#6d6457] dark:text-[#cfc6b8]'>
        {props.label}
      </span>
    </div>
  )
}
