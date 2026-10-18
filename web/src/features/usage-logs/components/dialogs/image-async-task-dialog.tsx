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
import { useQuery } from '@tanstack/react-query'
import { ExternalLink, ImageIcon } from 'lucide-react'
import { useMemo } from 'react'
import { useTranslation } from 'react-i18next'

import { CopyButton } from '@/components/copy-button'
import { Dialog } from '@/components/dialog'
import { StatusBadge } from '@/components/status-badge'
import { Button } from '@/components/ui/button'
import { Label } from '@/components/ui/label'
import { Separator } from '@/components/ui/separator'
import { Skeleton } from '@/components/ui/skeleton'
import { formatLogQuota, formatTimestampToDate } from '@/lib/format'
import { cn } from '@/lib/utils'

import { getImageAsyncTaskDetail } from '../../api'
import { taskStatusMapper } from '../../lib/mappers'
import type { ImageAsyncTaskDetail, ImageAsyncResultObject } from '../../types'

interface ImageAsyncTaskDialogProps {
  taskId: string
  open: boolean
  onOpenChange: (open: boolean) => void
}

interface ImageResult {
  url: string
  source?: string
  contentType?: string
  size?: number
}

export function ImageAsyncTaskDialog(props: ImageAsyncTaskDialogProps) {
  const { t } = useTranslation()

  const query = useQuery({
    queryKey: ['image-async-task-detail', props.taskId],
    queryFn: async () => {
      const result = await getImageAsyncTaskDetail(props.taskId)
      if (!result.success || !result.data) {
        throw new Error(result.message || t('Failed to load task details'))
      }
      return result.data
    },
    enabled: props.open && !!props.taskId,
    refetchInterval: (query) => {
      const status = query.state.data?.status
      return status && status !== 'SUCCESS' && status !== 'FAILURE'
        ? 5000
        : false
    },
  })

  const detail = query.data
  const images = useMemo(() => collectImageResults(detail), [detail])
  const jsonText = useMemo(
    () => (detail ? JSON.stringify(detail.data ?? {}, null, 2) : ''),
    [detail]
  )

  return (
    <Dialog
      open={props.open}
      onOpenChange={props.onOpenChange}
      title={t('Image Async Task Details')}
      description={t(
        'View task progress, returned data, and stored image results.'
      )}
      contentClassName='sm:max-w-4xl'
    >
      {query.isLoading && (
        <div className='space-y-4 py-2'>
          <Skeleton className='h-24 w-full' />
          <Skeleton className='h-48 w-full' />
        </div>
      )}
      {query.isError && (
        <p className='text-destructive text-sm'>
          {query.error instanceof Error
            ? query.error.message
            : t('Failed to load task details')}
        </p>
      )}
      {detail && !query.isLoading && !query.isError && (
        <div className='space-y-5 py-2'>
          <SummarySection detail={detail} />

          <Separator />

          <section className='space-y-3'>
            <SectionTitle title={t('Returned Images')} />
            {images.length > 0 ? (
              <div className='space-y-3'>
                {images.map((image, index) => (
                  <div
                    key={image.url}
                    className='border-border/70 bg-muted/20 rounded-lg border p-3'
                  >
                    <div className='flex items-start justify-between gap-3'>
                      <div className='min-w-0 flex-1 space-y-2'>
                        <p className='text-muted-foreground text-xs font-medium'>
                          {t('Image')} #{index + 1}
                        </p>
                        <p className='bg-background text-foreground rounded-md border px-3 py-2 font-mono text-xs leading-relaxed break-all'>
                          {image.url}
                        </p>
                        <div className='text-muted-foreground flex flex-wrap gap-x-4 gap-y-1 text-xs'>
                          <span>{image.contentType || '-'}</span>
                          <span>
                            {image.size
                              ? `${(image.size / 1024).toFixed(1)} KB`
                              : image.source || '-'}
                          </span>
                        </div>
                      </div>
                      <div className='flex shrink-0 items-center gap-1'>
                        <CopyButton
                          value={image.url}
                          variant='ghost'
                          size='icon'
                          className='size-8'
                          iconClassName='size-4'
                          aria-label={t('Copy URL')}
                        />
                        <Button
                          nativeButton={false}
                          role='link'
                          variant='ghost'
                          size='icon'
                          className='size-8'
                          render={
                            <a
                              href={image.url}
                              target='_blank'
                              rel='noopener noreferrer'
                            />
                          }
                          title={t('Open image')}
                          aria-label={t('Open image')}
                        >
                          <ExternalLink className='size-4' />
                        </Button>
                      </div>
                    </div>
                  </div>
                ))}
              </div>
            ) : (
              <div className='text-muted-foreground border-border/70 bg-muted/20 flex items-center gap-2 rounded-lg border p-3 text-sm'>
                <ImageIcon className='size-4' />
                {t('No image URL is available yet.')}
              </div>
            )}
          </section>

          <Separator />

          <section className='space-y-3'>
            <div className='flex items-center justify-between gap-3'>
              <SectionTitle title={t('Returned JSON')} />
              <CopyButton value={jsonText} variant='outline' size='sm'>
                {t('Copy')}
              </CopyButton>
            </div>
            <pre className='bg-muted/40 border-border/70 max-h-80 overflow-auto rounded-lg border p-3 text-xs whitespace-pre-wrap'>
              {jsonText || '{}'}
            </pre>
          </section>

          <Separator />

          <DebugSection detail={detail} />
        </div>
      )}
    </Dialog>
  )
}

function SummarySection(props: { detail: ImageAsyncTaskDetail }) {
  const { t } = useTranslation()
  const detail = props.detail
  const rows = [
    [t('Task ID'), detail.task_id],
    [t('Model'), detail.model || '-'],
    [t('User'), detail.username || String(detail.user_id)],
    [t('Channel'), detail.channel_id ? `#${detail.channel_id}` : '-'],
    [t('Submit Time'), formatTaskTime(detail.submit_time)],
    [t('Start Time'), formatTaskTime(detail.start_time)],
    [t('Finish Time'), formatTaskTime(detail.finish_time)],
    [t('Cost'), formatLogQuota(detail.quota || 0)],
  ]

  return (
    <section className='space-y-3'>
      <div className='flex flex-wrap items-center justify-between gap-3'>
        <SectionTitle title={t('Task Summary')} />
        <StatusBadge
          label={t(taskStatusMapper.getLabel(detail.status, detail.status))}
          variant={taskStatusMapper.getVariant(detail.status)}
          size='sm'
          showDot
        />
      </div>
      <div className='grid gap-3 sm:grid-cols-2 lg:grid-cols-4'>
        {rows.map(([label, value]) => (
          <InfoItem key={label} label={label} value={value} />
        ))}
        <InfoItem label={t('Progress')} value={detail.progress || '-'} />
        <InfoItem label={t('Group')} value={detail.group || '-'} />
        <InfoItem
          label={t('Request Path')}
          value={detail.request?.path || '-'}
          className='sm:col-span-2'
        />
      </div>
      {detail.fail_reason && (
        <div className='border-destructive/30 bg-destructive/5 text-destructive rounded-lg border p-3 text-sm'>
          {detail.fail_reason}
        </div>
      )}
    </section>
  )
}

function DebugSection(props: { detail: ImageAsyncTaskDetail }) {
  const { t } = useTranslation()
  const debug = props.detail.debug
  const requestFields = props.detail.request?.fields
  let webhookStatus = t('Not configured')
  if (debug?.webhook_configured) {
    webhookStatus = debug.webhook_done ? t('Completed') : t('Pending')
  }
  let billingStatus = t('Pending')
  if (debug?.billing_finalized) {
    billingStatus = t('Finalized')
  } else if (debug?.billing_refunded) {
    billingStatus = t('Refunded')
  }
  return (
    <section className='space-y-3'>
      <SectionTitle title={t('Request and Worker')} />
      <div className='grid gap-3 sm:grid-cols-2 lg:grid-cols-4'>
        <InfoItem label={t('Worker Node')} value={debug?.worker_node || '-'} />
        <InfoItem label={t('Attempt')} value={String(debug?.attempt || 0)} />
        <InfoItem label={t('Webhook')} value={webhookStatus} />
        <InfoItem label={t('Billing')} value={billingStatus} />
      </div>
      {debug?.last_error && (
        <div className='border-destructive/30 bg-destructive/5 text-destructive rounded-lg border p-3 text-sm'>
          {debug.last_error}
        </div>
      )}
      {requestFields && (
        <div className='space-y-2'>
          <Label>{t('Request Fields')}</Label>
          <pre className='bg-muted/40 border-border/70 max-h-56 overflow-auto rounded-lg border p-3 text-xs whitespace-pre-wrap'>
            {JSON.stringify(requestFields, null, 2)}
          </pre>
        </div>
      )}
    </section>
  )
}

function SectionTitle(props: { title: string }) {
  return <h3 className='text-sm font-semibold'>{props.title}</h3>
}

function InfoItem(props: { label: string; value: string; className?: string }) {
  return (
    <div
      className={cn(
        'border-border/70 bg-muted/20 rounded-lg border p-3',
        props.className
      )}
    >
      <p className='text-muted-foreground text-xs'>{props.label}</p>
      <p className='overflow-wrap-anywhere mt-1 text-sm font-medium break-all'>
        {props.value}
      </p>
    </div>
  )
}

function formatTaskTime(timestamp?: number): string {
  return timestamp ? formatTimestampToDate(timestamp, 'seconds') : '-'
}

function collectImageResults(detail?: ImageAsyncTaskDetail): ImageResult[] {
  if (!detail) return []
  const results = new Map<string, ImageResult>()
  for (const object of detail.result_objects || []) {
    if (object.url) {
      results.set(object.url, fromResultObject(object))
    }
  }
  const data = detail.data
  if (data && typeof data === 'object' && 'data' in data) {
    const items = (data as { data?: unknown }).data
    if (Array.isArray(items)) {
      for (const item of items) {
        if (item && typeof item === 'object' && 'url' in item) {
          const url = (item as { url?: unknown }).url
          if (typeof url === 'string' && url && !results.has(url)) {
            results.set(url, { url })
          }
        }
      }
    }
  }
  return [...results.values()]
}

function fromResultObject(object: ImageAsyncResultObject): ImageResult {
  return {
    url: object.url || '',
    source: object.source,
    contentType: object.content_type,
    size: object.size,
  }
}
