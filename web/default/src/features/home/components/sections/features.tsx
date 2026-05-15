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
import { FileText, Image, MessageSquare, Users } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { AnimateInView } from '@/components/animate-in-view'

interface FeaturesProps {
  className?: string
}

export function Features(_props: FeaturesProps) {
  const { t } = useTranslation()

  const features = [
    {
      icon: <MessageSquare className='size-5' />,
      title: t('Think with a workspace'),
      desc: t(
        'Start from conversation, then move naturally into keys, logs, and operational context when you need it.'
      ),
    },
    {
      icon: <Image className='size-5' />,
      title: t('Create without waiting around'),
      desc: t(
        'Long-running image work can finish in the background and return durable links for later review.'
      ),
    },
    {
      icon: <FileText className='size-5' />,
      title: t('Bring your own model stack'),
      desc: t(
        'Keep one clean interface while your private provider choices stay behind the scenes.'
      ),
    },
    {
      icon: <Users className='size-5' />,
      title: t('Share the same controls'),
      desc: t(
        'Give teammates a calm place to work while budgets, permissions, and usage remain governed.'
      ),
    },
  ]

  return (
    <section className='bg-[#f7f3ea] px-5 py-20 text-[#211f1b] md:px-8 md:py-28 dark:bg-[#161512] dark:text-[#f4efe6]'>
      <div className='mx-auto max-w-6xl'>
        <AnimateInView className='max-w-2xl'>
          <p className='mb-4 text-sm font-medium text-[#8a5a3b] dark:text-[#d6a47b]'>
            {t('What Aittco helps with')}
          </p>
          <h2 className='text-4xl leading-[1.05] font-medium md:text-6xl'>
            {t('One place for the work before, during, and after AI responds.')}
          </h2>
        </AnimateInView>

        <div className='mt-16 grid gap-px bg-[#2f2b24]/12 md:grid-cols-2 dark:bg-white/12'>
          {features.map((feature, index) => (
            <AnimateInView
              key={feature.title}
              delay={index * 80}
              className='bg-[#f7f3ea] p-8 md:p-10 dark:bg-[#161512]'
            >
              <div className='mb-8 flex size-11 items-center justify-center rounded-full bg-[#dfd3c2] text-[#5f422f] dark:bg-white/10 dark:text-[#e8d6c1]'>
                {feature.icon}
              </div>
              <h3 className='text-2xl font-medium'>{feature.title}</h3>
              <p className='mt-4 max-w-md text-base leading-7 text-[#61594e] dark:text-[#cfc6b8]'>
                {feature.desc}
              </p>
            </AnimateInView>
          ))}
        </div>
      </div>
    </section>
  )
}
