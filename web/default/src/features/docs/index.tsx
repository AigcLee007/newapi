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
  Image as ImageIcon,
  KeyRound,
  Route,
  Sparkles,
  Timer,
  Workflow,
} from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { cn } from '@/lib/utils'
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

function getOrigin() {
  if (typeof window === 'undefined') {
    return 'https://max.aittco.com'
  }
  return window.location.origin
}

type Example = {
  id: string
  title: string
  badge: string
  description: string
  code: string
}

function createDocsContent(props: { baseUrl: string; isChinese: boolean }) {
  const { baseUrl, isChinese } = props

  const en = {
    hero: {
      badge: 'Aittco API Documentation',
      title: 'One accurate guide for your gateway.',
      description:
        'Use these examples with your own API key and the models enabled in this gateway. Image generation supports synchronous requests, background tasks, task polling, Gemini native format, and provider-specific image parameters.',
      imageExamples: 'Image examples',
    },
    base: {
      title: 'Base URL',
      description: 'Requests should use this site as the API host.',
      authorization: 'Authorization: Bearer sk-your-api-key',
    },
    endpointGroups: [
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
        models: [
          'gemini-3-pro-image-preview',
          'gemini-3.1-flash-image-preview',
        ],
        endpoints: [
          'POST /v1beta/models/{model}:generateContent',
          'POST /v1/images/generations',
          'POST /v1/images/generations?async=true',
          'POST /v1/images/edits?async=true',
        ],
      },
    ],
    syncAsync: {
      title: 'How sync and async image calls differ',
      description:
        'Use synchronous calls for quick tests. Use asynchronous calls for slower image providers, production workflows, and any request that may take tens of seconds.',
      syncTitle: 'Synchronous',
      syncDescription:
        'The HTTP request stays open until the upstream returns.',
      syncPoints: [
        'Use POST /v1/images/generations.',
        'Best for short tests or fast upstream channels.',
        'The image URL appears in the response body directly.',
      ],
      asyncTitle: 'Asynchronous',
      asyncDescription:
        'The first response returns a task id. Results are retrieved with the task endpoint.',
      asyncPoints: [
        'Add async=true to generation or edit endpoints.',
        'Send Idempotency-Key to avoid duplicate tasks.',
        'Poll GET /v1/images/tasks/{task_id} for progress and URLs.',
      ],
    },
    parameters: {
      title: 'Parameter format by model family',
      description:
        'Different upstreams accept different image parameter names. Use the format below to avoid model errors.',
      fieldHeader: 'Field',
      useHeader: 'Use',
      notesHeader: 'Notes',
    },
    parameterRows: [
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
    ] as [string, string, string][],
    examplesTitle: 'Image API examples',
    examplesDescription:
      'Copy these examples, replace the API key, and keep the model name exactly as configured in the model square.',
    troubleshooting: {
      title: 'Task response and troubleshooting',
      description:
        'Async tasks are visible in the dashboard task logs. Result images are stored as URLs, and the dashboard shows addresses instead of loading images automatically.',
      responseTitle: 'Task query response',
      responseDescription:
        'A finished task includes status, progress, model, and returned image URLs.',
      mistakesTitle: 'Common mistakes',
      mistakesDescription:
        'Most image failures come from mismatched parameter formats or polling the wrong path.',
      points: [
        'Use /v1/images/tasks/{task_id}, not /v1/images/task/{task_id}.',
        'Use size for gpt-image-2, but aspect_ratio and imageSize for Gemini or Visionary-style image channels.',
        'If the model price is not configured, enable self-use mode or add model pricing before testing.',
        'If a model has multiple channels, confirm each channel accepts the same parameter format.',
      ],
    },
    copyLabel: 'Copy',
    copiedLabel: 'Copied',
    badges: {
      sync: 'Sync',
      async: 'Async',
      visionary: 'Visionary',
      gemini: 'Gemini',
      edit: 'Edit',
    },
    examples: [] as Example[],
  }

  const zh = {
    ...en,
    hero: {
      badge: 'Aittco API 文档',
      title: '一份真正适配本站的接口指南。',
      description:
        '使用你自己的 API 密钥和本站已启用的模型即可调用。当前生图能力支持同步请求、异步任务、任务轮询、Gemini 原生格式，以及不同上游模型自己的图片参数格式。',
      imageExamples: '生图示例',
    },
    base: {
      title: '基础地址',
      description: '所有请求都应该使用当前站点作为 API Host。',
      authorization: '鉴权方式：Authorization: Bearer sk-your-api-key',
    },
    endpointGroups: [
      {
        title: 'OpenAI 兼容生图',
        description: '适合 gpt-image-2 以及 OpenAI 风格的图片上游。',
        models: ['gpt-image-2'],
        endpoints: en.endpointGroups[0].endpoints,
      },
      {
        title: 'Visionary 生图模型',
        description:
          '使用上游原生图片字段。Nano_Banana_Pro 推荐使用 ratio / imageSize 这类参数。',
        models: ['Nano_Banana_Pro'],
        endpoints: en.endpointGroups[1].endpoints,
      },
      {
        title: 'Gemini 生图模型',
        description:
          '同时支持 Gemini 原生 generateContent、OpenAI 兼容生图接口，以及异步图片编辑。',
        models: en.endpointGroups[2].models,
        endpoints: en.endpointGroups[2].endpoints,
      },
    ],
    syncAsync: {
      title: '同步和异步生图有什么区别',
      description:
        '快速测试可以用同步接口；更慢的图片上游、生产流程、可能耗时几十秒的请求，建议使用异步接口。',
      syncTitle: '同步',
      syncDescription: 'HTTP 请求会一直等待，直到上游返回结果。',
      syncPoints: [
        '使用 POST /v1/images/generations。',
        '适合短时间测试或响应较快的上游渠道。',
        '图片 URL 会直接出现在本次响应体里。',
      ],
      asyncTitle: '异步',
      asyncDescription:
        '第一次响应只返回任务 ID，后续通过任务查询接口获取进度和结果。',
      asyncPoints: [
        '在生图或图片编辑接口后添加 async=true。',
        '建议发送 Idempotency-Key，避免重复提交同一个任务。',
        '轮询 GET /v1/images/tasks/{task_id} 查询进度和图片 URL。',
      ],
    },
    parameters: {
      title: '不同模型家族的参数格式',
      description:
        '不同上游支持的图片参数名不完全一样。按下面格式填写，可以减少模型参数错误。',
      fieldHeader: '字段',
      useHeader: '用途',
      notesHeader: '说明',
    },
    parameterRows: [
      ['model', '必填', '模型广场里配置的模型名称。'],
      ['prompt', '必填', '生图或图片编辑的文本指令。'],
      ['n', '可选', '生成图片数量。多数上游建议保持为 1。'],
      ['size', 'OpenAI 风格', '例如 1024x1024。gpt-image-2 推荐使用这个字段。'],
      [
        'aspect_ratio',
        'Gemini / Visionary 风格',
        '例如 1:1、16:9、9:16。Gemini 和 Visionary 优先使用这个字段。',
      ],
      [
        'imageSize',
        'Gemini / Visionary 风格',
        '例如 1K、2K。当上游要求原生字段时，用它替代 size。',
      ],
      [
        'images',
        '编辑 / 参考图',
        '输入图片 URL 数组，用于图生图或图片编辑任务。',
      ],
      [
        'Idempotency-Key',
        '异步请求头',
        '异步请求推荐携带，避免网络重试造成重复任务。',
      ],
    ] as [string, string, string][],
    examplesTitle: '生图 API 示例',
    examplesDescription:
      '复制下面示例，替换 API 密钥，并确保模型名和模型广场中的配置完全一致。',
    troubleshooting: {
      title: '任务返回和常见问题',
      description:
        '异步任务可以在后台任务日志中查看。返回图片以 URL 保存，后台只显示地址，不会自动加载图片。',
      responseTitle: '任务查询返回',
      responseDescription:
        '完成后的任务会包含状态、进度、模型，以及返回图片 URL。',
      mistakesTitle: '常见错误',
      mistakesDescription:
        '多数生图失败来自参数格式不匹配，或轮询了错误的任务路径。',
      points: [
        '任务查询使用 /v1/images/tasks/{task_id}，不是 /v1/images/task/{task_id}。',
        'gpt-image-2 使用 size；Gemini 或 Visionary 风格渠道使用 aspect_ratio 和 imageSize。',
        '如果提示模型价格未配置，请先开启自用模式，或给该模型配置定价。',
        '如果同一个模型配置了多个渠道，请确认每个渠道都接受相同参数格式。',
      ],
    },
    copyLabel: '复制',
    copiedLabel: '已复制',
    badges: {
      sync: '同步',
      async: '异步',
      visionary: 'Visionary',
      gemini: 'Gemini',
      edit: '编辑',
    },
    examples: [] as Example[],
  }

  const content = isChinese ? zh : en
  content.examples = buildExamples(baseUrl, content, isChinese)
  return content
}

function buildExamples(
  baseUrl: string,
  content: {
    badges: {
      sync: string
      async: string
      visionary: string
      gemini: string
      edit: string
    }
  },
  isChinese: boolean
): Example[] {
  return [
    {
      id: 'gpt-image-sync',
      title: isChinese
        ? 'gpt-image-2 同步生图'
        : 'gpt-image-2 synchronous generation',
      badge: content.badges.sync,
      description: isChinese
        ? '使用 OpenAI 兼容图片接口。上游完成后，本次请求直接返回图片数据。'
        : 'Use the OpenAI-compatible image endpoint. The response returns image data directly when the upstream finishes.',
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
      title: isChinese
        ? 'gpt-image-2 异步生图'
        : 'gpt-image-2 asynchronous generation',
      badge: content.badges.async,
      description: isChinese
        ? '添加 async=true 创建后台任务。拿到 task_id 后轮询任务接口，直到状态为 SUCCESS 或 FAILED。'
        : 'Add async=true to create a background task. Poll the returned task id until status is SUCCESS or FAILED.',
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
      title: isChinese
        ? 'Visionary Nano_Banana_Pro 生图'
        : 'Visionary Nano_Banana_Pro generation',
      badge: content.badges.visionary,
      description: isChinese
        ? 'Nano_Banana_Pro 使用原生比例字段，建议使用 aspect_ratio 和 imageSize，不要只传 size。'
        : 'Nano_Banana_Pro uses native ratio fields. Prefer aspect_ratio and imageSize instead of size.',
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
      title: isChinese
        ? 'Visionary Nano_Banana_Pro 异步生图'
        : 'Visionary Nano_Banana_Pro asynchronous generation',
      badge: content.badges.async,
      description: isChinese
        ? '请求体和同步生图一致，只需要添加 async=true。后续通过任务接口查询进度和结果 URL。'
        : 'Use the same body as synchronous generation and add async=true. Query the task endpoint for progress and result URLs.',
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
      title: isChinese
        ? 'Gemini 原生 generateContent'
        : 'Gemini native generateContent',
      badge: content.badges.gemini,
      description: isChinese
        ? '当客户端本身使用 Gemini 格式时，用这个接口。模型名写在 URL 里。'
        : 'Use this when the client already speaks Gemini format. The model name is part of the URL.',
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
      title: isChinese
        ? 'Gemini 通过 /v1/images/generations 调用'
        : 'Gemini through /v1/images/generations',
      badge: content.badges.sync,
      description: isChinese
        ? '当客户端希望使用 OpenAI 风格图片接口，但上游模型是 Gemini 时，用这个方式。'
        : 'Use this when your client expects OpenAI-style image APIs but the selected upstream model is Gemini.',
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
      title: isChinese ? 'Gemini 异步生图' : 'Gemini asynchronous generation',
      badge: content.badges.async,
      description: isChinese
        ? '添加 async=true 后，Gemini 生图会作为后台任务运行，稍后通过任务接口查询。'
        : 'Add async=true to run Gemini image generation as a background task and query progress later.',
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
      title: isChinese
        ? 'Gemini 异步图片编辑'
        : 'Gemini asynchronous image editing',
      badge: content.badges.edit,
      description: isChinese
        ? '需要图生图或图片编辑时，使用 /v1/images/edits?async=true，并传入输入图片 URL。'
        : 'Use /v1/images/edits?async=true when you need image-to-image editing with input image URLs.',
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
  ]
}

export function Docs() {
  const { i18n } = useTranslation()
  const baseUrl = useMemo(() => getOrigin(), [])
  const isChinese =
    i18n.resolvedLanguage?.toLowerCase().startsWith('zh') ||
    i18n.language?.toLowerCase().startsWith('zh')

  const content = useMemo(
    () => createDocsContent({ baseUrl, isChinese }),
    [baseUrl, isChinese]
  )

  return (
    <PublicLayout navLinks={navLinks}>
      <div className='mx-auto w-full max-w-6xl space-y-8 pb-14'>
        <section className='grid gap-8 pt-8 lg:grid-cols-[1fr_20rem] lg:items-end'>
          <div className='space-y-5'>
            <Badge variant='outline' className='rounded-md'>
              {content.hero.badge}
            </Badge>
            <div className='space-y-4'>
              <h1 className='max-w-3xl text-4xl leading-[1.02] font-semibold tracking-normal text-balance md:text-6xl'>
                {content.hero.title}
              </h1>
              <p className='text-muted-foreground max-w-2xl text-base leading-7 md:text-lg'>
                {content.hero.description}
              </p>
            </div>
            <div className='flex flex-wrap gap-3'>
              <Button render={<a href='#image-examples' />}>
                <ImageIcon className='size-4' />
                {content.hero.imageExamples}
              </Button>
            </div>
          </div>

          <Card className='rounded-lg'>
            <CardHeader>
              <CardTitle className='flex items-center gap-2'>
                <Route className='size-4' />
                {content.base.title}
              </CardTitle>
              <CardDescription>{content.base.description}</CardDescription>
            </CardHeader>
            <CardContent className='space-y-3'>
              <CodeInline value={baseUrl} />
              <div className='text-muted-foreground flex items-center gap-2 text-sm'>
                <KeyRound className='size-4' />
                {content.base.authorization}
              </div>
            </CardContent>
          </Card>
        </section>

        <section className='grid gap-4 md:grid-cols-3'>
          {content.endpointGroups.map((group) => (
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
            title={content.syncAsync.title}
            description={content.syncAsync.description}
          />
          <div className='grid gap-4 md:grid-cols-2'>
            <Card className='rounded-lg'>
              <CardHeader>
                <CardTitle className='flex items-center gap-2'>
                  <Sparkles className='size-4' />
                  {content.syncAsync.syncTitle}
                </CardTitle>
                <CardDescription>
                  {content.syncAsync.syncDescription}
                </CardDescription>
              </CardHeader>
              <CardContent className='space-y-3 text-sm'>
                {content.syncAsync.syncPoints.map((point) => (
                  <Point key={point}>{point}</Point>
                ))}
              </CardContent>
            </Card>

            <Card className='rounded-lg'>
              <CardHeader>
                <CardTitle className='flex items-center gap-2'>
                  <Timer className='size-4' />
                  {content.syncAsync.asyncTitle}
                </CardTitle>
                <CardDescription>
                  {content.syncAsync.asyncDescription}
                </CardDescription>
              </CardHeader>
              <CardContent className='space-y-3 text-sm'>
                {content.syncAsync.asyncPoints.map((point) => (
                  <Point key={point}>{point}</Point>
                ))}
              </CardContent>
            </Card>
          </div>
        </section>

        <section className='space-y-4'>
          <SectionHeading
            icon={BookOpen}
            title={content.parameters.title}
            description={content.parameters.description}
          />
          <Card className='rounded-lg'>
            <CardContent className='overflow-x-auto p-0'>
              <table className='w-full min-w-[48rem] text-sm'>
                <thead>
                  <tr className='border-b text-left'>
                    <th className='px-4 py-3 font-medium'>
                      {content.parameters.fieldHeader}
                    </th>
                    <th className='px-4 py-3 font-medium'>
                      {content.parameters.useHeader}
                    </th>
                    <th className='px-4 py-3 font-medium'>
                      {content.parameters.notesHeader}
                    </th>
                  </tr>
                </thead>
                <tbody>
                  {content.parameterRows.map(([field, use, notes]) => (
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
            title={content.examplesTitle}
            description={content.examplesDescription}
          />
          <div className='grid gap-4'>
            {content.examples.map((example) => (
              <ExampleCard
                key={example.id}
                example={example}
                copyLabel={content.copyLabel}
                copiedLabel={content.copiedLabel}
              />
            ))}
          </div>
        </section>

        <section className='space-y-4'>
          <SectionHeading
            icon={CheckCircle2}
            title={content.troubleshooting.title}
            description={content.troubleshooting.description}
          />
          <div className='grid gap-4 md:grid-cols-2'>
            <Card className='rounded-lg'>
              <CardHeader>
                <CardTitle>{content.troubleshooting.responseTitle}</CardTitle>
                <CardDescription>
                  {content.troubleshooting.responseDescription}
                </CardDescription>
              </CardHeader>
              <CardContent>
                <CodeBlock
                  copyLabel={content.copyLabel}
                  copiedLabel={content.copiedLabel}
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
                <CardTitle>{content.troubleshooting.mistakesTitle}</CardTitle>
                <CardDescription>
                  {content.troubleshooting.mistakesDescription}
                </CardDescription>
              </CardHeader>
              <CardContent className='space-y-3 text-sm'>
                {content.troubleshooting.points.map((point) => (
                  <Point key={point}>{point}</Point>
                ))}
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
  copyLabel: string
  copiedLabel: string
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
        <CodeBlock
          code={props.example.code}
          copyLabel={props.copyLabel}
          copiedLabel={props.copiedLabel}
        />
      </CardContent>
    </Card>
  )
}

function CodeBlock(props: {
  code: string
  className?: string
  copyLabel: string
  copiedLabel: string
}) {
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
          {copied ? props.copiedLabel : props.copyLabel}
        </Button>
      </div>
      <pre className='overflow-x-auto p-4 text-xs leading-6'>
        <code>{props.code}</code>
      </pre>
    </div>
  )
}
