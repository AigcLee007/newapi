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
import { AnimateInView } from '@/components/animate-in-view'

export function HowItWorks() {
  const { t } = useTranslation()

  const sections = [
    {
      title: t('Personal'),
      price: t('Free'),
      desc: t('A quiet starting point for chat, experiments, and image tasks.'),
      action: t('Try Aittco'),
      items: [
        t('Chat workspace'),
        t('Image task history'),
        t('Basic usage visibility'),
      ],
    },
    {
      title: t('Team'),
      price: t('Managed'),
      desc: t('Shared controls for people building on top of multiple models.'),
      action: t('Open console'),
      items: [
        t('Team access rules'),
        t('API keys and routing'),
        t('Budget and quota controls'),
      ],
    },
    {
      title: t('Platform'),
      price: t('Custom'),
      desc: t(
        'A private AI layer for products, workflows, and internal tools.'
      ),
      action: t('View settings'),
      items: [
        t('Provider abstraction'),
        t('Async result storage'),
        t('Detailed operational logs'),
      ],
    },
  ]

  return (
    <section className='bg-[#efe6d8] px-5 py-20 text-[#211f1b] md:px-8 md:py-28 dark:bg-[#1c1a16] dark:text-[#f4efe6]'>
      <div className='mx-auto max-w-6xl'>
        <AnimateInView className='mb-12 flex flex-col justify-between gap-6 md:flex-row md:items-end'>
          <div>
            <p className='mb-4 text-sm font-medium text-[#8a5a3b] dark:text-[#d6a47b]'>
              {t('Choose how to begin')}
            </p>
            <h2 className='max-w-2xl text-4xl leading-[1.05] font-medium md:text-6xl'>
              {t('Start small, then make it operational.')}
            </h2>
          </div>
          <p className='max-w-sm text-base leading-7 text-[#61594e] dark:text-[#cfc6b8]'>
            {t(
              'The public face stays simple. The controls are there when your team needs them.'
            )}
          </p>
        </AnimateInView>

        <div className='grid gap-px bg-[#2f2b24]/12 md:grid-cols-3 dark:bg-white/12'>
          {sections.map((section, index) => (
            <AnimateInView
              key={section.title}
              delay={index * 80}
              className='bg-[#efe6d8] p-7 md:p-8 dark:bg-[#1c1a16]'
            >
              <h3 className='text-xl font-medium'>{section.title}</h3>
              <p className='mt-4 text-3xl font-medium'>{section.price}</p>
              <p className='mt-4 min-h-20 text-sm leading-6 text-[#61594e] dark:text-[#cfc6b8]'>
                {section.desc}
              </p>
              <div className='my-7 h-px bg-[#2f2b24]/12 dark:bg-white/12' />
              <ul className='space-y-3 text-sm text-[#403b33] dark:text-[#e3d9ca]'>
                {section.items.map((item) => (
                  <li key={item} className='flex gap-3'>
                    <span className='mt-2 size-1.5 shrink-0 rounded-full bg-[#a0643f]' />
                    <span>{item}</span>
                  </li>
                ))}
              </ul>
              <p className='mt-8 text-sm font-medium text-[#8a5a3b] dark:text-[#d6a47b]'>
                {section.action}
              </p>
            </AnimateInView>
          ))}
        </div>
      </div>
    </section>
  )
}
