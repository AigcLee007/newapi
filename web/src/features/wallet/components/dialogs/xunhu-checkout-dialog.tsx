/*
Copyright (C) 2023-2026 QuantumNous

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU Affero General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
GNU Affero General Public License for more details.

You should have received a copy of the GNU Affero General Public License
along with this program. If not, see <https://www.gnu.org/licenses/>.

For commercial licensing, please contact support@quantumnous.com
*/
import { ExternalLink } from 'lucide-react'
import { QRCodeSVG } from 'qrcode.react'
import { useTranslation } from 'react-i18next'

import { Dialog } from '@/components/dialog'
import { Button } from '@/components/ui/button'

import type { XunhuCheckout } from '../../types'

interface XunhuCheckoutDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  checkout: XunhuCheckout | null
}

export function XunhuCheckoutDialog({
  open,
  onOpenChange,
  checkout,
}: XunhuCheckoutDialogProps) {
  const { t } = useTranslation()
  const qrValue = checkout?.url_qrcode || checkout?.url

  if (!checkout || !qrValue) return null

  return (
    <Dialog
      open={open}
      onOpenChange={onOpenChange}
      title={t('Pay with Xunhu')}
      contentClassName='max-sm:w-[calc(100vw-1.5rem)] sm:max-w-sm'
      contentHeight='auto'
      bodyClassName='space-y-4'
      footer={
        checkout.url ? (
          <Button
            className='w-full'
            render={
              <a
                href={checkout.url}
                target='_blank'
                rel='noopener noreferrer'
              />
            }
          >
            <ExternalLink className='h-4 w-4' />
            {t('Open payment page')}
          </Button>
        ) : null
      }
    >
      <div className='flex flex-col items-center gap-3 py-2'>
        <div className='flex w-full justify-center rounded-md bg-white p-4'>
          <QRCodeSVG
            value={qrValue}
            size={220}
            className='h-auto w-full max-w-[220px]'
            role='img'
            aria-label={t('Xunhu payment QR code')}
          />
        </div>
      </div>
    </Dialog>
  )
}
