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
import { useTranslation } from 'react-i18next'
import { useSystemConfig } from '@/hooks/use-system-config'
import { Skeleton } from '@/components/ui/skeleton'

type AuthLayoutProps = {
  children: React.ReactNode
}

export function AuthLayout({ children }: AuthLayoutProps) {
  const { t } = useTranslation()
  const { systemName, logo, loading } = useSystemConfig()

  return (
    <div className='relative min-h-svh overflow-hidden bg-[#f3eee4] text-[#171511] dark:bg-[#151411] dark:text-[#f6f0e6]'>
      <Link
        to='/'
        className='absolute top-5 left-5 z-10 flex items-center gap-3 transition-opacity hover:opacity-80 sm:top-8 sm:left-8 lg:left-24'
      >
        <div className='relative h-8 w-8'>
          {loading ? (
            <Skeleton className='absolute inset-0 rounded-full' />
          ) : (
            <img
              src={logo || '/logo.png'}
              alt={t('Logo')}
              className='h-8 w-8 rounded-full object-cover'
            />
          )}
        </div>
        {loading ? (
          <Skeleton className='h-6 w-24' />
        ) : (
          <h1 className='text-lg font-extrabold tracking-normal'>
            {systemName}
          </h1>
        )}
      </Link>

      <div className='mx-auto grid min-h-svh w-full max-w-7xl items-center gap-12 px-5 pt-24 pb-12 md:px-8 lg:grid-cols-[minmax(0,1fr)_28rem] lg:gap-24 lg:pt-28'>
        <section className='hidden max-w-3xl lg:block'>
          <p className='mb-6 text-sm font-extrabold text-[#7f3a10] uppercase dark:text-[#d6a47b]'>
            {t('One place for every model')}
          </p>
          <h1 className='max-w-[46rem] text-[clamp(4.8rem,8vw,7rem)] leading-[0.92] font-extrabold tracking-normal'>
            {t('Welcome back to your workspace.')}
          </h1>
          <p className='mt-7 max-w-2xl text-xl leading-8 text-[#443e34] dark:text-[#cfc6b8]'>
            {t(
              'Sign in to route models, create images, and keep usage in order.'
            )}
          </p>

          <div className='mt-12 max-w-md divide-y divide-[#1d1b16]/12 border-y border-[#1d1b16]/12 dark:divide-white/10 dark:border-white/10'>
            <div className='flex items-center justify-between py-4 text-base font-extrabold'>
              <span>{t('Chat routing')}</span>
              <span className='text-sm text-[#62584b] dark:text-[#cfc6b8]'>
                {t('stable')}
              </span>
            </div>
            <div className='flex items-center justify-between py-4 text-base font-extrabold'>
              <span>{t('Image tasks')}</span>
              <span className='text-sm text-[#62584b] dark:text-[#cfc6b8]'>
                {t('COS ready')}
              </span>
            </div>
          </div>
        </section>

        <div className='mx-auto w-full max-w-[30rem] rounded-lg border border-[#1d1b16]/12 bg-[#fffdf8]/60 p-6 shadow-[0_22px_80px_rgba(52,43,28,0.08)] backdrop-blur-sm sm:p-8 dark:border-white/10 dark:bg-white/[0.04] dark:shadow-none'>
          <div className='mb-8 flex items-center gap-3 lg:hidden'>
            {loading ? (
              <Skeleton className='size-9 rounded-full' />
            ) : (
              <img
                src={logo || '/logo.png'}
                alt={t('Logo')}
                className='size-9 rounded-full object-cover'
              />
            )}
            <div>
              <div className='text-sm font-extrabold'>{systemName}</div>
              <div className='text-xs font-bold text-[#7f3a10] uppercase dark:text-[#d6a47b]'>
                {t('One place for every model')}
              </div>
            </div>
          </div>
          <div className='w-full'>{children}</div>
        </div>
      </div>
    </div>
  )
}
