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
import { AnimateInView } from '@/components/animate-in-view'

interface CTAProps {
  className?: string
  isAuthenticated?: boolean
}

export function CTA(props: CTAProps) {
  const { t } = useTranslation()

  return (
    <section className='bg-[#f7f3ea] px-5 py-20 text-[#211f1b] md:px-8 md:py-28 dark:bg-[#161512] dark:text-[#f4efe6]'>
      <AnimateInView className='mx-auto max-w-6xl'>
        <div className='border-y border-[#2f2b24]/12 py-12 md:flex md:items-center md:justify-between md:gap-10 dark:border-white/12'>
          <div>
            <p className='mb-4 text-sm font-medium text-[#8a5a3b] dark:text-[#d6a47b]'>
              {t('Ready when you are')}
            </p>
            <h2 className='max-w-2xl text-4xl leading-[1.05] font-medium md:text-6xl'>
              {t('Open a calmer way to work with AI.')}
            </h2>
          </div>
          <Button
            className='mt-8 h-11 rounded-full bg-[#211f1b] px-5 text-[#fbf7ef] hover:bg-[#343027] md:mt-0 dark:bg-[#f4efe6] dark:text-[#161512] dark:hover:bg-white'
            render={
              <Link to={props.isAuthenticated ? '/dashboard' : '/sign-up'} />
            }
          >
            {props.isAuthenticated ? t('Go to Dashboard') : t('Try Aittco')}
            <ArrowRight className='ml-2 size-4' />
          </Button>
        </div>
      </AnimateInView>
    </section>
  )
}
