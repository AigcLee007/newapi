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

interface StatsProps {
  className?: string
}

export function Stats(_props: StatsProps) {
  const { t } = useTranslation()

  const lines = [
    t('Chat with private context'),
    t('Create images asynchronously'),
    t('Control spending and access'),
    t('Route requests without vendor lock-in'),
  ]

  return (
    <section className='bg-[#2f2a22] px-5 text-[#f6efe3] md:px-8 dark:bg-[#0f0e0c]'>
      <div className='mx-auto grid max-w-6xl gap-px bg-white/12 md:grid-cols-4'>
        {lines.map((line) => (
          <div
            key={line}
            className='bg-[#2f2a22] py-8 md:px-6 dark:bg-[#0f0e0c]'
          >
            <p className='max-w-[14rem] text-lg leading-7 md:text-xl'>{line}</p>
          </div>
        ))}
      </div>
    </section>
  )
}
