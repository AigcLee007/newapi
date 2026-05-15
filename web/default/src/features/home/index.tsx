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
import { useAuthStore } from '@/stores/auth-store'
import { Markdown } from '@/components/ui/markdown'
import { PublicLayout } from '@/components/layout'
import type { TopNavLink } from '@/components/layout/types'
import { Hero } from './components'
import { useHomePageContent } from './hooks'

const homeNavLinks: TopNavLink[] = [
  { title: 'Home', href: '/' },
  { title: 'Dashboard', href: '/dashboard' },
  { title: 'Models', href: '/pricing' },
  { title: 'Docs', href: 'https://docs.newapi.pro', external: true },
]

export function Home() {
  const { t } = useTranslation()
  const { auth } = useAuthStore()
  const isAuthenticated = !!auth.user
  const { content, isLoaded, isUrl } = useHomePageContent()

  if (!isLoaded) {
    return (
      <PublicLayout
        showMainContainer={false}
        navLinks={homeNavLinks}
        headerProps={{ showNavigation: false }}
        showThemeSwitch={false}
        showLanguageSwitcher={false}
        showNotifications={false}
      >
        <main className='flex min-h-screen items-center justify-center'>
          <div className='text-muted-foreground'>{t('Loading...')}</div>
        </main>
      </PublicLayout>
    )
  }

  if (content) {
    return (
      <PublicLayout
        showMainContainer={false}
        navLinks={homeNavLinks}
        headerProps={{ showNavigation: isAuthenticated }}
        showThemeSwitch={isAuthenticated}
        showLanguageSwitcher={isAuthenticated}
        showNotifications={isAuthenticated}
      >
        <main className='overflow-x-hidden'>
          {isUrl ? (
            <iframe
              src={content}
              className='h-screen w-full border-none'
              title={t('Custom Home Page')}
            />
          ) : (
            <div className='container mx-auto py-8'>
              <Markdown className='custom-home-content'>{content}</Markdown>
            </div>
          )}
        </main>
      </PublicLayout>
    )
  }

  if (!isAuthenticated) {
    return (
      <PublicLayout
        showMainContainer={false}
        navLinks={homeNavLinks}
        headerProps={{ showNavigation: false }}
        showThemeSwitch={false}
        showLanguageSwitcher={false}
        showNotifications={false}
      >
        <Hero isAuthenticated={false} />
      </PublicLayout>
    )
  }

  return (
    <PublicLayout showMainContainer={false} navLinks={homeNavLinks}>
      <Hero isAuthenticated />
    </PublicLayout>
  )
}
