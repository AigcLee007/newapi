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
  FileDown,
  Image as ImageIcon,
  KeyRound,
  PencilLine,
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

type EndpointGroup = {
  title: string
  description: string
  models: string[]
  endpoints: string[]
}

type DocsContent = {
  hero: {
    badge: string
    title: string
    description: string
    imageExamples: string
    exportMarkdown: string
  }
  base: {
    title: string
    description: string
    authorization: string
  }
  endpointGroups: EndpointGroup[]
  syncAsync: {
    title: string
    description: string
    syncTitle: string
    syncDescription: string
    syncPoints: string[]
    asyncTitle: string
    asyncDescription: string
    asyncPoints: string[]
  }
  parameters: {
    title: string
    description: string
    fieldHeader: string
    useHeader: string
    notesHeader: string
  }
  parameterRows: [string, string, string][]
  examplesTitle: string
  examplesDescription: string
  troubleshooting: {
    title: string
    description: string
    responseTitle: string
    responseDescription: string
    mistakesTitle: string
    mistakesDescription: string
    points: string[]
  }
  copyLabel: string
  copiedLabel: string
  badges: {
    sync: string
    async: string
    edit: string
    visionary: string
    gemini: string
  }
  examples: Example[]
}

function createDocsContent(props: {
  baseUrl: string
  isChinese: boolean
}): DocsContent {
  const { baseUrl, isChinese } = props

  const en: Omit<DocsContent, 'examples'> = {
    hero: {
      badge: 'Aittco API Documentation',
      title: 'Image APIs, clearly mapped.',
      description:
        'Use the right endpoint for each image workflow. Text-to-image, image editing, async tasks, Visionary image parameters, and Gemini native calls have different request shapes.',
      imageExamples: 'Image examples',
      exportMarkdown: 'Export Markdown',
    },
    base: {
      title: 'Base URL',
      description: 'Requests should use this site as the API host.',
      authorization: 'Authorization: Bearer sk-your-api-key',
    },
    endpointGroups: [
      {
        title: 'gpt-image-2',
        description:
          'Text-to-image must use /v1/images/generations. Image-to-image and image editing must use /v1/images/edits. Sync and async are both supported.',
        models: ['gpt-image-2'],
        endpoints: [
          'POST /v1/images/generations',
          'POST /v1/images/generations?async=true',
          'POST /v1/images/edits',
          'POST /v1/images/edits?async=true',
          'GET /v1/images/tasks/{task_id}',
        ],
      },
      {
        title: 'Visionary image models',
        description:
          'Nano_Banana_Pro uses provider-native image fields. imageSize supports only 2K and 4K; do not send 1K.',
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
          'Gemini supports native generateContent, OpenAI-compatible image generation, async generation, and async image editing.',
        models: [
          'gemini-3-pro-image-preview',
          'gemini-3.1-flash-image-preview',
        ],
        endpoints: [
          'POST /v1beta/models/{model}:generateContent',
          'POST /v1/images/generations',
          'POST /v1/images/generations?async=true',
          'POST /v1/images/edits?async=true',
          'GET /v1/images/tasks/{task_id}',
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
        'Text-to-image uses POST /v1/images/generations.',
        'Image-to-image or editing uses POST /v1/images/edits.',
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
      [
        'size',
        'gpt-image-2',
        'Example: 1024x1024. Use this for gpt-image-2 text-to-image and edits.',
      ],
      [
        'images',
        '/v1/images/edits',
        'Array of input image URLs for image-to-image or editing tasks. Do not send edit images to /v1/images/generations.',
      ],
      [
        'aspect_ratio',
        'Gemini / Visionary style',
        'Example: 1:1, 16:9, 9:16. Prefer this for Gemini and Visionary-style channels.',
      ],
      [
        'imageSize',
        'Gemini / Visionary style',
        'Gemini support depends on the upstream channel. Nano_Banana_Pro supports only 2K and 4K, not 1K.',
      ],
      [
        'Idempotency-Key',
        'Async header',
        'Recommended for async requests to avoid duplicate task submission.',
      ],
    ],
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
        'For gpt-image-2 text-to-image, use /v1/images/generations.',
        'For gpt-image-2 image-to-image or editing, use /v1/images/edits.',
        'Nano_Banana_Pro accepts imageSize 2K or 4K only. 1K will fail.',
        'If the model price is not configured, enable self-use mode or add model pricing before testing.',
      ],
    },
    copyLabel: 'Copy',
    copiedLabel: 'Copied',
    badges: {
      sync: 'Sync',
      async: 'Async',
      edit: 'Edit',
      visionary: 'Visionary',
      gemini: 'Gemini',
    },
  }

  const zh: Omit<DocsContent, 'examples'> = {
    hero: {
      badge: 'Aittco API 文档',
      title: '把生图接口讲清楚。',
      description:
        '不同生图流程必须使用不同接口。文生图、图生图/图片编辑、异步任务、Visionary 参数、Gemini 原生格式的请求结构都不一样。',
      imageExamples: '生图示例',
      exportMarkdown: '导出 Markdown',
    },
    base: {
      title: '基础地址',
      description: '所有请求都应该使用当前站点作为 API Host。',
      authorization: '鉴权方式：Authorization: Bearer sk-your-api-key',
    },
    endpointGroups: [
      {
        title: 'gpt-image-2',
        description:
          '文生图必须使用 /v1/images/generations。图生图/图片编辑必须使用 /v1/images/edits。同步和异步都支持。',
        models: ['gpt-image-2'],
        endpoints: en.endpointGroups[0].endpoints,
      },
      {
        title: 'Visionary 生图模型',
        description:
          'Nano_Banana_Pro 使用上游原生图片字段。imageSize 只支持 2K 和 4K，不支持 1K。',
        models: ['Nano_Banana_Pro'],
        endpoints: en.endpointGroups[1].endpoints,
      },
      {
        title: 'Gemini 生图模型',
        description:
          '支持 Gemini 原生 generateContent、OpenAI 兼容生图、异步生图，以及异步图片编辑。',
        models: en.endpointGroups[2].models,
        endpoints: en.endpointGroups[2].endpoints,
      },
    ],
    syncAsync: {
      title: '同步和异步生图有什么区别',
      description:
        '快速测试可以用同步接口。较慢的图片上游、生产流程、可能耗时几十秒的请求，建议使用异步接口。',
      syncTitle: '同步',
      syncDescription: 'HTTP 请求会一直等待，直到上游返回结果。',
      syncPoints: [
        '文生图使用 POST /v1/images/generations。',
        '图生图或图片编辑使用 POST /v1/images/edits。',
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
      ['prompt', '必填', '文生图或图片编辑的文本指令。'],
      ['n', '可选', '生成图片数量。多数上游建议保持为 1。'],
      [
        'size',
        'gpt-image-2',
        '例如 1024x1024。gpt-image-2 文生图和图片编辑都使用这个字段。',
      ],
      [
        'images',
        '/v1/images/edits',
        '输入图片 URL 数组，用于图生图或图片编辑。不要把编辑图片传到 /v1/images/generations。',
      ],
      [
        'aspect_ratio',
        'Gemini / Visionary 风格',
        '例如 1:1、16:9、9:16。Gemini 和 Visionary 风格渠道优先使用这个字段。',
      ],
      [
        'imageSize',
        'Gemini / Visionary 风格',
        'Gemini 是否支持取决于上游渠道。Nano_Banana_Pro 只支持 2K 和 4K，不支持 1K。',
      ],
      [
        'Idempotency-Key',
        '异步请求头',
        '异步请求推荐携带，避免网络重试造成重复任务。',
      ],
    ],
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
        '多数生图失败来自参数格式不匹配，或者轮询了错误的任务路径。',
      points: [
        '任务查询使用 /v1/images/tasks/{task_id}，不是 /v1/images/task/{task_id}。',
        'gpt-image-2 文生图使用 /v1/images/generations。',
        'gpt-image-2 图生图/图片编辑使用 /v1/images/edits。',
        'Nano_Banana_Pro 的 imageSize 只能传 2K 或 4K，传 1K 会失败。',
        '如果提示模型价格未配置，请先开启自用模式，或给该模型配置定价。',
      ],
    },
    copyLabel: '复制',
    copiedLabel: '已复制',
    badges: {
      sync: '同步',
      async: '异步',
      edit: '编辑',
      visionary: 'Visionary',
      gemini: 'Gemini',
    },
  }

  const content = isChinese ? zh : en
  return {
    ...content,
    examples: buildExamples(baseUrl, content, isChinese),
  }
}

function buildExamples(
  baseUrl: string,
  content: Pick<DocsContent, 'badges'>,
  isChinese: boolean
): Example[] {
  return [
    {
      id: 'gpt-image-sync',
      title: isChinese
        ? 'gpt-image-2 同步文生图'
        : 'gpt-image-2 synchronous text-to-image',
      badge: content.badges.sync,
      description: isChinese
        ? '文生图使用 OpenAI 兼容图片生成接口 /v1/images/generations。不要把图生图/编辑图片传到这个接口。'
        : 'Text-to-image uses the OpenAI-compatible /v1/images/generations endpoint. Do not send edit input images to this endpoint.',
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
        ? 'gpt-image-2 异步文生图'
        : 'gpt-image-2 asynchronous text-to-image',
      badge: content.badges.async,
      description: isChinese
        ? '文生图异步接口是在 /v1/images/generations 后添加 async=true。拿到 task_id 后轮询任务接口。'
        : 'Asynchronous text-to-image adds async=true to /v1/images/generations. Poll the returned task id afterwards.',
      code: `TASK_ID=$(curl -s "${baseUrl}/v1/images/generations?async=true" \\
  -H "Authorization: Bearer sk-your-api-key" \\
  -H "Content-Type: application/json" \\
  -H "Idempotency-Key: gpt-image-generation-$(date +%s)" \\
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
      id: 'gpt-image-edit-sync',
      title: isChinese
        ? 'gpt-image-2 同步图生图 / 图片编辑'
        : 'gpt-image-2 synchronous image editing',
      badge: content.badges.edit,
      description: isChinese
        ? '图生图或图片编辑必须使用 /v1/images/edits。接口用错会导致上游生图失败。'
        : 'Image-to-image or editing must use /v1/images/edits. Using the generation endpoint for edits can fail upstream.',
      code: `curl ${baseUrl}/v1/images/edits \\
  -H "Authorization: Bearer sk-your-api-key" \\
  -H "Content-Type: application/json" \\
  -d '{
    "model": "gpt-image-2",
    "prompt": "Keep the composition, make it watercolor style",
    "images": ["https://example.com/input.jpg"],
    "size": "1024x1024",
    "n": 1
  }'`,
    },
    {
      id: 'gpt-image-edit-async',
      title: isChinese
        ? 'gpt-image-2 异步图生图 / 图片编辑'
        : 'gpt-image-2 asynchronous image editing',
      badge: content.badges.async,
      description: isChinese
        ? '异步图片编辑是在 /v1/images/edits 后添加 async=true，然后轮询任务接口获取进度和图片 URL。'
        : 'Asynchronous image editing adds async=true to /v1/images/edits, then polls the task endpoint for progress and URLs.',
      code: `TASK_ID=$(curl -s "${baseUrl}/v1/images/edits?async=true" \\
  -H "Authorization: Bearer sk-your-api-key" \\
  -H "Content-Type: application/json" \\
  -H "Idempotency-Key: gpt-image-edit-$(date +%s)" \\
  -d '{
    "model": "gpt-image-2",
    "prompt": "Keep the product, replace the background with a clean studio wall",
    "images": ["https://example.com/input.jpg"],
    "size": "1024x1024",
    "n": 1
  }' | sed -n 's/.*"data":"\\([^"]*\\)".*/\\1/p')

curl -s "${baseUrl}/v1/images/tasks/$TASK_ID" \\
  -H "Authorization: Bearer sk-your-api-key"`,
    },
    {
      id: 'visionary-sync',
      title: isChinese
        ? 'Visionary Nano_Banana_Pro 同步生图'
        : 'Visionary Nano_Banana_Pro synchronous generation',
      badge: content.badges.visionary,
      description: isChinese
        ? 'Nano_Banana_Pro 使用原生比例字段。imageSize 只能传 2K 或 4K，不能传 1K。'
        : 'Nano_Banana_Pro uses native ratio fields. imageSize must be 2K or 4K; 1K is not supported.',
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
        ? '请求体和同步生图一致，只需要添加 async=true。这里示例使用 4K。'
        : 'Use the same body as synchronous generation and add async=true. This example uses 4K.',
      code: `TASK_ID=$(curl -s "${baseUrl}/v1/images/generations?async=true" \\
  -H "Authorization: Bearer sk-your-api-key" \\
  -H "Content-Type: application/json" \\
  -H "Idempotency-Key: visionary-$(date +%s)" \\
  -d '{
    "model": "Nano_Banana_Pro",
    "prompt": "A cinematic studio photo of a ceramic coffee cup",
    "aspect_ratio": "16:9",
    "imageSize": "4K",
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
      title: isChinese
        ? 'Gemini 异步生图'
        : 'Gemini asynchronous generation',
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

function createMarkdown(content: DocsContent, baseUrl: string) {
  const endpointSections = content.endpointGroups
    .map(
      (group) => `### ${group.title}

${group.description}

Models: ${group.models.map((model) => `\`${model}\``).join(', ')}

${group.endpoints.map((endpoint) => `- \`${endpoint}\``).join('\n')}`
    )
    .join('\n\n')

  const parameterTable = [
    `| ${content.parameters.fieldHeader} | ${content.parameters.useHeader} | ${content.parameters.notesHeader} |`,
    '|---|---|---|',
    ...content.parameterRows.map(
      ([field, use, notes]) => `| \`${field}\` | ${use} | ${notes} |`
    ),
  ].join('\n')

  const examples = content.examples
    .map(
      (example) => `### ${example.title}

${example.description}

\`\`\`bash
${example.code}
\`\`\``
    )
    .join('\n\n')

  return `# ${content.hero.badge}

${content.hero.description}

## ${content.base.title}

\`${baseUrl}\`

\`${content.base.authorization}\`

## Endpoints

${endpointSections}

## ${content.syncAsync.title}

${content.syncAsync.description}

### ${content.syncAsync.syncTitle}

${content.syncAsync.syncDescription}

${content.syncAsync.syncPoints.map((point) => `- ${point}`).join('\n')}

### ${content.syncAsync.asyncTitle}

${content.syncAsync.asyncDescription}

${content.syncAsync.asyncPoints.map((point) => `- ${point}`).join('\n')}

## ${content.parameters.title}

${content.parameters.description}

${parameterTable}

## ${content.examplesTitle}

${content.examplesDescription}

${examples}

## ${content.troubleshooting.title}

${content.troubleshooting.description}

### ${content.troubleshooting.responseTitle}

${content.troubleshooting.responseDescription}

\`\`\`json
{
  "status": "SUCCESS",
  "progress": "100%",
  "model": "gpt-image-2",
  "data": [
    {
      "url": "https://your-cos-domain/newapi/images/task-id/0.png"
    }
  ]
}
\`\`\`

### ${content.troubleshooting.mistakesTitle}

${content.troubleshooting.mistakesDescription}

${content.troubleshooting.points.map((point) => `- ${point}`).join('\n')}
`
}

function downloadMarkdown(content: DocsContent, baseUrl: string) {
  const markdown = createMarkdown(content, baseUrl)
  const blob = new Blob([markdown], { type: 'text/markdown;charset=utf-8' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = 'aittco-api-docs.md'
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)
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
              <h1 className='max-w-3xl text-4xl leading-[1.04] font-semibold tracking-normal text-balance md:text-6xl'>
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
              <Button
                type='button'
                variant='outline'
                onClick={() => downloadMarkdown(content, baseUrl)}
              >
                <FileDown className='size-4' />
                {content.hero.exportMarkdown}
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
  example: Example
  copyLabel: string
  copiedLabel: string
}) {
  return (
    <Card className='rounded-lg'>
      <CardHeader>
        <div className='flex flex-wrap items-start justify-between gap-3'>
          <div className='space-y-1'>
            <CardTitle className='flex items-center gap-2'>
              <PencilLine className='text-muted-foreground size-4' />
              {props.example.title}
            </CardTitle>
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
