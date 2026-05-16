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
import { useMemo, useState } from 'react'
import {
  BookOpen,
  CheckCircle2,
  Copy,
  ExternalLink,
  Image as ImageIcon,
  KeyRound,
  Route,
  Sparkles,
  Timer,
  Workflow,
} from 'lucide-react'
import { cn } from '@/lib/utils'
import { useStatus } from '@/hooks/use-status'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'
import { PublicLayout } from '@/components/layout'

const navLinks = [
  { title: 'Home', href: '/' },
  { title: 'Dashboard', href: '/dashboard' },
  { title: 'Models', href: '/pricing' },
  { title: 'Docs', href: '/docs' },
]

const endpointGroups = [
  {
    title: 'OpenAI image compatible',
    description: 'Best for gpt-image-2 and OpenAI-style image providers.',
    models: ['gpt-image-2'],
    endpoints: [
      'POST /v1/images/generations',
      'POST /v1/images/generations?async=true',
      'GET /v1/images/tasks/{task_id}',
    ],
  },
  {
    title: 'Visionary image models',
    description:
      'Use provider-native image fields. Nano_Banana_Pro expects ratio and imageSize style parameters.',
    models: ['Nano_Banana_Pro'],
    endpoints: [
      'POST /v1/images/generations',
      'POST /v1/images/generations?async=true',
      'GET /v1/images/tasks/{task_id}',
    ],
  },
  {
    title: 'Gemini image models',
    description:
      'Supports Gemini native generateContent plus OpenAI-compatible image generation and async editing.',
    models: ['gemini-3-pro-image-preview', 'gemini-3.1-flash-image-preview'],
    endpoints: [
      'POST /v1beta/models/{model}:generateContent',
      'POST /v1/images/generations',
      'POST /v1/images/generations?async=true',
      'POST /v1/images/edits?async=true',
    ],
  },
]

const parameterRows = [
  ['model', 'Required', 'The model name configured in the model square.'],
  ['prompt', 'Required', 'Text instruction for generation or editing.'],
  ['n', 'Optional', 'Number of images. Keep 1 for most upstream channels.'],
  ['size', 'OpenAI style', 'Example: 1024x1024. Use this for gpt-image-2.'],
  [
    'aspect_ratio',
    'Gemini / Visionary style',
    'Example: 1:1, 16:9, 9:16. Prefer this for Gemini and Visionary.',
  ],
  [
    'imageSize',
    'Gemini / Visionary style',
    'Example: 1K, 2K. Use this instead of size when the upstream expects native fields.',
  ],
  [
    'images',
    'Edit / reference image',
    'Array of input image URLs for image-to-image or editing tasks.',
  ],
  [
    'Idempotency-Key',
    'Async header',
    'Recommended for async requests to avoid duplicate task submission.',
  ],
]

function getOrigin() {
  if (typeof window === 'undefined') {
    return 'https://max.aittco.com'
  }
  return window.location.origin
}

export function Docs() {
  const { status } = useStatus()
  const baseUrl = useMemo(() => getOrigin(), [])
  const externalDocs = String(status?.docs_link || '').trim()

  const examples = useMemo(
    () => [
      {
        id: 'gpt-image-sync',
        title: 'gpt-image-2 synchronous generation',
        badge: 'Sync',
        description:
          'Use the OpenAI-compatible image endpoint. The response returns image data directly when the upstream finishes.',
        code: `curl ${baseUrl}/v1/images/generations \\
  -H "Authorization: Bearer sk-your-api-key" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "gpt-image-2",
    "prompt": "A tiny blue cube on a white table, simple product photo",
    "size": "1024x1024",
    "n": 1
  }'`,
      },
      {
        id: 'gpt-image-async',
        title: 'gpt-image-2 asynchronous generation',
        badge: 'Async',
        description:
          'Add async=true to create a background task. Poll the returned task id until status is SUCCESS or FAILED.',
        code: `TASK_ID=$(curl -s "${baseUrl}/v1/images/generations?async=true" \\
  -H "Authorization: Bearer sk-your-api-key" \\
  -H "Content-Type: application/json" \\
  -H "Idempotency-Key: image-async-$(date +%s)" \\
  -d '{
    "model": "gpt-image-2",
    "prompt": "A clean product photo of a glass perfume bottle",
    "size": "1024x1024",
    "n": 1
  }' | sed -n 's/.*"data":"\\([^"]*\\)".*/\\1/p')

curl -s "${baseUrl}/v1/images/tasks/$TASK_ID" \\
  -H "Authorization: Bearer sk-your-api-key"`,
      },
      {
        id: 'visionary-sync',
        title: 'Visionary Nano_Banana_Pro generation',
        badge: 'Visionary',
        description:
          'Nano_Banana_Pro uses native ratio fields. Prefer aspect_ratio and imageSize instead of size.',
        code: `curl ${baseUrl}/v1/images/generations \\
  -H "Authorization: Bearer sk-your-api-key" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "Nano_Banana_Pro",
    "prompt": "A premium product photo of a futuristic glass perfume bottle on a white studio background",
    "aspect_ratio": "1:1",
    "imageSize": "2K",
    "n": 1
  }'`,
      },
      {
        id: 'visionary-async',
        title: 'Visionary Nano_Banana_Pro asynchronous generation',
        badge: 'Async',
        description:
          'Use the same body as synchronous generation and add async=true. Query the task endpoint for progress and result URLs.',
        code: `TASK_ID=$(curl -s "${baseUrl}/v1/images/generations?async=true" \\
  -H "Authorization: Bearer sk-your-api-key" \\
  -H "Content-Type: application/json" \\
  -H "Idempotency-Key: visionary-$(date +%s)" \\
  -d '{
    "model": "Nano_Banana_Pro",
    "prompt": "A cinematic studio photo of a ceramic coffee cup",
    "aspect_ratio": "16:9",
    "imageSize": "2K",
    "n": 1
  }' | sed -n 's/.*"data":"\\([^"]*\\)".*/\\1/p')

curl -s "${baseUrl}/v1/images/tasks/$TASK_ID" \\
  -H "Authorization: Bearer sk-your-api-key"`,
      },
      {
        id: 'gemini-native',
        title: 'Gemini native generateContent',
        badge: 'Gemini',
        description:
          'Use this when the client already speaks Gemini format. The model name is part of the URL.',
        code: `curl ${baseUrl}/v1beta/models/gemini-3-pro-image-preview:generateContent \\
  -H "Authorization: Bearer sk-your-api-key" \\
  -H "Content-Type: application/json" \\
  -d '{
    "contents": [
      {
        "role": "user",
        "parts": [
          { "text": "Keep the composition, make it watercolor style" },
          {
            "fileData": {
              "mimeType": "image/jpeg",
              "fileUri": "https://example.com/input.jpg"
            }
          }
        ]
      }
    ],
    "generationConfig": {
      "responseModalities": ["IMAGE"],
      "imageConfig": {
        "aspectRatio": "16:9",
        "imageSize": "2K"
      }
    }
  }'`,
      },
      {
        id: 'gemini-image-sync',
        title: 'Gemini through /v1/images/generations',
        badge: 'Sync',
        description:
          'Use this when your client expects OpenAI-style image APIs but the selected upstream model is Gemini.',
        code: `curl ${baseUrl}/v1/images/generations \\
  -H "Authorization: Bearer sk-your-api-key" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "gemini-3.1-flash-image-preview",
    "prompt": "Create a clean 16:9 watercolor product background",
    "aspect_ratio": "16:9",
    "imageSize": "2K",
    "n": 1
  }'`,
      },
      {
        id: 'gemini-image-async',
        title: 'Gemini asynchronous generation',
        badge: 'Async',
        description:
          'Add async=true to run Gemini image generation as a background task and query progress later.',
        code: `TASK_ID=$(curl -s "${baseUrl}/v1/images/generations?async=true" \\
  -H "Authorization: Bearer sk-your-api-key" \\
  -H "Content-Type: application/json" \\
  -H "Idempotency-Key: gemini-image-$(date +%s)" \\
  -d '{
    "model": "gemini-3-pro-image-preview",
    "prompt": "A premium 2K product render on a warm neutral background",
    "aspect_ratio": "16:9",
    "imageSize": "2K",
    "n": 1
  }' | sed -n 's/.*"data":"\\([^"]*\\)".*/\\1/p')

curl -s "${baseUrl}/v1/images/tasks/$TASK_ID" \\
  -H "Authorization: Bearer sk-your-api-key"`,
      },
      {
        id: 'gemini-edit-async',
        title: 'Gemini asynchronous image editing',
        badge: 'Edit',
        description:
          'Use /v1/images/edits?async=true when you need image-to-image editing with input image URLs.',
        code: `TASK_ID=$(curl -s "${baseUrl}/v1/images/edits?async=true" \\
  -H "Authorization: Bearer sk-your-api-key" \\
  -H "Content-Type: application/json" \\
  -H "Idempotency-Key: gemini-edit-$(date +%s)" \\
  -d '{
    "model": "gemini-3-pro-image-preview",
    "prompt": "Keep the composition, make it watercolor style",
    "images": ["https://example.com/input.jpg"],
    "aspect_ratio": "16:9",
    "imageSize": "2K",
    "n": 1
  }' | sed -n 's/.*"data":"\\([^"]*\\)".*/\\1/p')

curl -s "${baseUrl}/v1/images/tasks/$TASK_ID" \\
  -H "Authorization: Bearer sk-your-api-key"`,
      },
    ],
    [baseUrl]
  )

  return (
    <PublicLayout navLinks={navLinks}>
      <div className='mx-auto w-full max-w-6xl space-y-8 pb-14'>
        <section className='grid gap-8 pt-8 lg:grid-cols-[1fr_20rem] lg:items-end'>
          <div className='space-y-5'>
            <Badge variant='outline' className='rounded-md'>
              Aittco API Documentation
            </Badge>
            <div className='space-y-4'>
              <h1 className='max-w-3xl text-4xl leading-[1.02] font-semibold tracking-normal text-balance md:text-6xl'>
                One accurate guide for your gateway.
              </h1>
              <p className='text-muted-foreground max-w-2xl text-base leading-7 md:text-lg'>
                Use these examples with your own API key and the models enabled
                in this gateway. Image generation supports synchronous requests,
                background tasks, task polling, Gemini native format, and
                provider-specific image parameters.
              </p>
            </div>
            <div className='flex flex-wrap gap-3'>
              <Button render={<a href='#image-examples' />}>
                <ImageIcon className='size-4' />
                Image examples
              </Button>
              {externalDocs ? (
                <Button
                  variant='outline'
                  render={
                    <a
                      href={externalDocs}
                      target='_blank'
                      rel='noopener noreferrer'
                    />
                  }
                >
                  <ExternalLink className='size-4' />
                  External reference
                </Button>
              ) : null}
            </div>
          </div>

          <Card className='rounded-lg'>
            <CardHeader>
              <CardTitle className='flex items-center gap-2'>
                <Route className='size-4' />
                Base URL
              </CardTitle>
              <CardDescription>
                Requests should use this site as the API host.
              </CardDescription>
            </CardHeader>
            <CardContent className='space-y-3'>
              <CodeInline value={baseUrl} />
              <div className='text-muted-foreground flex items-center gap-2 text-sm'>
                <KeyRound className='size-4' />
                Authorization: Bearer sk-your-api-key
              </div>
            </CardContent>
          </Card>
        </section>

        <section className='grid gap-4 md:grid-cols-3'>
          {endpointGroups.map((group) => (
            <Card key={group.title} className='rounded-lg'>
              <CardHeader>
                <CardTitle>{group.title}</CardTitle>
                <CardDescription>{group.description}</CardDescription>
              </CardHeader>
              <CardContent className='space-y-4'>
                <div className='flex flex-wrap gap-2'>
                  {group.models.map((model) => (
                    <Badge key={model} variant='secondary'>
                      {model}
                    </Badge>
                  ))}
                </div>
                <div className='space-y-2'>
                  {group.endpoints.map((endpoint) => (
                    <CodeInline key={endpoint} value={endpoint} />
                  ))}
                </div>
              </CardContent>
            </Card>
          ))}
        </section>

        <section className='space-y-4'>
          <SectionHeading
            icon={Workflow}
            title='How sync and async image calls differ'
            description='Use synchronous calls for quick tests. Use asynchronous calls for slower image providers, production workflows, and any request that may take tens of seconds.'
          />
          <div className='grid gap-4 md:grid-cols-2'>
            <Card className='rounded-lg'>
              <CardHeader>
                <CardTitle className='flex items-center gap-2'>
                  <Sparkles className='size-4' />
                  Synchronous
                </CardTitle>
                <CardDescription>
                  The HTTP request stays open until the upstream returns.
                </CardDescription>
              </CardHeader>
              <CardContent className='space-y-3 text-sm'>
                <Point>Use POST /v1/images/generations.</Point>
                <Point>Best for short tests or fast upstream channels.</Point>
                <Point>
                  The image URL appears in the response body directly.
                </Point>
              </CardContent>
            </Card>

            <Card className='rounded-lg'>
              <CardHeader>
                <CardTitle className='flex items-center gap-2'>
                  <Timer className='size-4' />
                  Asynchronous
                </CardTitle>
                <CardDescription>
                  The first response returns a task id. Results are retrieved
                  with the task endpoint.
                </CardDescription>
              </CardHeader>
              <CardContent className='space-y-3 text-sm'>
                <Point>Add async=true to generation or edit endpoints.</Point>
                <Point>Send Idempotency-Key to avoid duplicate tasks.</Point>
                <Point>
                  Poll GET /v1/images/tasks/{'{task_id}'} for progress and URLs.
                </Point>
              </CardContent>
            </Card>
          </div>
        </section>

        <section className='space-y-4'>
          <SectionHeading
            icon={BookOpen}
            title='Parameter format by model family'
            description='Different upstreams accept different image parameter names. Use the format below to avoid model errors.'
          />
          <Card className='rounded-lg'>
            <CardContent className='overflow-x-auto p-0'>
              <table className='w-full min-w-[48rem] text-sm'>
                <thead>
                  <tr className='border-b text-left'>
                    <th className='px-4 py-3 font-medium'>Field</th>
                    <th className='px-4 py-3 font-medium'>Use</th>
                    <th className='px-4 py-3 font-medium'>Notes</th>
                  </tr>
                </thead>
                <tbody>
                  {parameterRows.map(([field, use, notes]) => (
                    <tr key={field} className='border-b last:border-0'>
                      <td className='px-4 py-3 font-mono text-xs'>{field}</td>
                      <td className='px-4 py-3'>{use}</td>
                      <td className='text-muted-foreground px-4 py-3'>
                        {notes}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </CardContent>
          </Card>
        </section>

        <section id='image-examples' className='scroll-mt-24 space-y-4'>
          <SectionHeading
            icon={ImageIcon}
            title='Image API examples'
            description='Copy these examples, replace the API key, and keep the model name exactly as configured in the model square.'
          />
          <div className='grid gap-4'>
            {examples.map((example) => (
              <ExampleCard key={example.id} example={example} />
            ))}
          </div>
        </section>

        <section className='space-y-4'>
          <SectionHeading
            icon={CheckCircle2}
            title='Task response and troubleshooting'
            description='Async tasks are visible in the dashboard task logs. Result images are stored as URLs, and the dashboard shows addresses instead of loading images automatically.'
          />
          <div className='grid gap-4 md:grid-cols-2'>
            <Card className='rounded-lg'>
              <CardHeader>
                <CardTitle>Task query response</CardTitle>
                <CardDescription>
                  A finished task includes status, progress, model, and returned
                  image URLs.
                </CardDescription>
              </CardHeader>
              <CardContent>
                <CodeBlock
                  code={`{
  "status": "SUCCESS",
  "progress": "100%",
  "model": "gpt-image-2",
  "data": [
    {
      "url": "https://your-cos-domain/newapi/images/task-id/0.png"
    }
  ]
}`}
                />
              </CardContent>
            </Card>

            <Card className='rounded-lg'>
              <CardHeader>
                <CardTitle>Common mistakes</CardTitle>
                <CardDescription>
                  Most image failures come from mismatched parameter formats or
                  polling the wrong path.
                </CardDescription>
              </CardHeader>
              <CardContent className='space-y-3 text-sm'>
                <Point>
                  Use /v1/images/tasks/{'{task_id}'}, not /v1/images/task/
                  {'{task_id}'}.
                </Point>
                <Point>
                  Use size for gpt-image-2, but aspect_ratio and imageSize for
                  Gemini or Visionary-style image channels.
                </Point>
                <Point>
                  If the model price is not configured, enable self-use mode or
                  add model pricing before testing.
                </Point>
                <Point>
                  If a model has multiple channels, confirm each channel accepts
                  the same parameter format.
                </Point>
              </CardContent>
            </Card>
          </div>
        </section>
      </div>
    </PublicLayout>
  )
}

function SectionHeading(props: {
  icon: React.ElementType
  title: string
  description: string
}) {
  const Icon = props.icon
  return (
    <div className='space-y-2'>
      <div className='flex items-center gap-2'>
        <Icon className='text-muted-foreground size-5' />
        <h2 className='text-2xl font-semibold tracking-normal'>
          {props.title}
        </h2>
      </div>
      <p className='text-muted-foreground max-w-3xl text-sm leading-6'>
        {props.description}
      </p>
    </div>
  )
}

function Point(props: { children: React.ReactNode }) {
  return (
    <div className='flex gap-2'>
      <CheckCircle2 className='mt-0.5 size-4 shrink-0 text-emerald-600' />
      <span className='text-muted-foreground leading-6'>{props.children}</span>
    </div>
  )
}

function CodeInline(props: { value: string }) {
  return (
    <code className='bg-muted text-foreground block rounded-md px-3 py-2 font-mono text-xs break-all'>
      {props.value}
    </code>
  )
}

function ExampleCard(props: {
  example: {
    title: string
    badge: string
    description: string
    code: string
  }
}) {
  return (
    <Card className='rounded-lg'>
      <CardHeader>
        <div className='flex flex-wrap items-start justify-between gap-3'>
          <div className='space-y-1'>
            <CardTitle>{props.example.title}</CardTitle>
            <CardDescription>{props.example.description}</CardDescription>
          </div>
          <Badge variant='outline'>{props.example.badge}</Badge>
        </div>
      </CardHeader>
      <CardContent>
        <CodeBlock code={props.example.code} />
      </CardContent>
    </Card>
  )
}

function CodeBlock(props: { code: string; className?: string }) {
  const [copied, setCopied] = useState(false)

  async function copy() {
    await navigator.clipboard.writeText(props.code)
    setCopied(true)
    window.setTimeout(() => setCopied(false), 1200)
  }

  return (
    <div
      className={cn(
        'bg-foreground text-background overflow-hidden rounded-lg',
        props.className
      )}
    >
      <div className='flex items-center justify-between border-b border-white/10 px-4 py-2'>
        <span className='text-xs font-medium opacity-70'>Shell</span>
        <Button
          type='button'
          variant='ghost'
          size='sm'
          className='text-background hover:text-background hover:bg-white/10'
          onClick={() => void copy()}
        >
          <Copy className='size-3.5' />
          {copied ? 'Copied' : 'Copy'}
        </Button>
      </div>
      <pre className='overflow-x-auto p-4 text-xs leading-6'>
        <code>{props.code}</code>
      </pre>
    </div>
  )
}
