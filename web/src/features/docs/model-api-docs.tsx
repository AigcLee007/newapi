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
import { BookOpen, FileDown, KeyRound, Route } from 'lucide-react'
import { useTranslation } from 'react-i18next'

import { CopyButton } from '@/components/copy-button'
import { Badge } from '@/components/ui/badge'
import { Button } from '@/components/ui/button'
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from '@/components/ui/card'

type ModelExample = {
  title: string
  description: string
  code: string
  language: 'bash' | 'python'
}

type ModelFamily = {
  title: string
  protocol: string
  description: string
  endpoints: string[]
  notes: string[]
}

type Translator = (key: string) => string

function createModelExamples(baseUrl: string, t: Translator): ModelExample[] {
  const prefix = `export API_BASE="${baseUrl}"
export API_KEY="YOUR_AITTCO_USER_API_KEY"
export MODEL="MODEL_NAME_FROM_DASHBOARD"`

  return [
    {
      title: t('GPT: OpenAI SDK'),
      description: t(
        'Point the OpenAI SDK to this gateway with base_url and use an Aittco user key. Set the AITTCO_API_KEY environment variable before running the example.'
      ),
      language: 'python',
      code: `import os
from openai import OpenAI

client = OpenAI(
    base_url="${baseUrl}/v1",
    api_key=os.environ["AITTCO_API_KEY"],
)

response = client.chat.completions.create(
    model="MODEL_NAME_FROM_DASHBOARD",
    messages=[{"role": "user", "content": "Hello!"}],
)
print(response.choices[0].message.content)`,
    },
    {
      title: t('GPT / Grok: OpenAI-compatible request'),
      description: t(
        'Use Chat Completions for GPT or Grok models enabled for your account. Set MODEL to the exact dashboard model name.'
      ),
      language: 'bash',
      code: `${prefix}

curl --fail-with-body "$API_BASE/v1/chat/completions" \\
  -H "Authorization: Bearer $API_KEY" \\
  -H "Content-Type: application/json" \\
  -d @- <<JSON
{"model":"$MODEL","messages":[{"role":"user","content":"Hello!"}]}
JSON`,
    },
    {
      title: t('OpenAI-compatible: streaming output'),
      description: t(
        'For models that support streaming, set stream to true and read the response as server-sent events.'
      ),
      language: 'bash',
      code: `${prefix}

curl -N --fail-with-body "$API_BASE/v1/chat/completions" \\
  -H "Authorization: Bearer $API_KEY" \\
  -H "Content-Type: application/json" \\
  -d @- <<JSON
{"model":"$MODEL","stream":true,"messages":[{"role":"user","content":"Write one short greeting."}]}
JSON`,
    },
    {
      title: t('Claude: native Messages API'),
      description: t(
        'Use the Anthropic Messages request format. Send your Aittco key in x-api-key with the required anthropic-version header.'
      ),
      language: 'bash',
      code: `${prefix}

curl --fail-with-body "$API_BASE/v1/messages" \\
  -H "x-api-key: $API_KEY" \\
  -H "anthropic-version: 2023-06-01" \\
  -H "content-type: application/json" \\
  -d @- <<JSON
{"model":"$MODEL","max_tokens":1024,"messages":[{"role":"user","content":"Hello!"}]}
JSON`,
    },
    {
      title: t('Claude: streaming Messages API'),
      description: t(
        'For Claude models that support streaming, set stream to true and consume the Messages event stream.'
      ),
      language: 'bash',
      code: `${prefix}

curl -N --fail-with-body "$API_BASE/v1/messages" \\
  -H "x-api-key: $API_KEY" \\
  -H "anthropic-version: 2023-06-01" \\
  -H "content-type: application/json" \\
  -d @- <<JSON
{"model":"$MODEL","max_tokens":1024,"stream":true,"messages":[{"role":"user","content":"Hello!"}]}
JSON`,
    },
    {
      title: t('Gemini: native generateContent'),
      description: t(
        'Gemini native requests put the model name in the URL and use the contents and parts request format.'
      ),
      language: 'bash',
      code: `${prefix}

curl --fail-with-body "$API_BASE/v1beta/models/$MODEL:generateContent" \\
  -H "x-goog-api-key: $API_KEY" \\
  -H "Content-Type: application/json" \\
  -d @- <<JSON
{"contents":[{"role":"user","parts":[{"text":"Hello!"}]}]}
JSON`,
    },
    {
      title: t('Gemini: native streaming output'),
      description: t(
        'Use streamGenerateContent with alt=sse for Gemini models that support streaming.'
      ),
      language: 'bash',
      code: `${prefix}

curl -N --fail-with-body "$API_BASE/v1beta/models/$MODEL:streamGenerateContent?alt=sse" \\
  -H "x-goog-api-key: $API_KEY" \\
  -H "Content-Type: application/json" \\
  -d @- <<JSON
{"contents":[{"role":"user","parts":[{"text":"Hello!"}]}]}
JSON`,
    },
  ]
}

function createFamilies(t: Translator): ModelFamily[] {
  return [
    {
      title: t('GPT / OpenAI-compatible'),
      protocol: 'OpenAI-compatible',
      description: t(
        'Use the OpenAI-compatible Chat Completions format for GPT models enabled on the platform. Other models may use this protocol when their channel supports it.'
      ),
      endpoints: ['GET /v1/models', 'POST /v1/chat/completions'],
      notes: [
        t('Set the OpenAI SDK base_url to https://max.aittco.com/v1.'),
        t(
          'Model features and supported fields depend on the model and channel configuration.'
        ),
      ],
    },
    {
      title: t('Claude native Messages'),
      protocol: 'Anthropic Messages',
      description: t(
        'Use the Messages request format for native Claude clients. Chat Completions may also work when the selected channel supports protocol conversion.'
      ),
      endpoints: ['GET /v1/models', 'POST /v1/messages'],
      notes: [
        t('Send x-api-key and anthropic-version: 2023-06-01.'),
        t(
          'The x-api-key value must be an Aittco user key, not an Anthropic provider key.'
        ),
      ],
    },
    {
      title: t('Gemini native API'),
      protocol: 'Gemini generateContent',
      description: t(
        'Use Gemini native clients and the contents/parts request format. The model name is part of the URL.'
      ),
      endpoints: [
        'GET /v1beta/models',
        'POST /v1beta/models/{model}:generateContent',
        'POST /v1beta/models/{model}:streamGenerateContent?alt=sse',
      ],
      notes: [
        t(
          'Authenticate with x-goog-api-key or Authorization: Bearer using your Aittco user key.'
        ),
        t('The native request body differs from OpenAI messages.'),
      ],
    },
    {
      title: t('Grok / xAI'),
      protocol: 'OpenAI-compatible',
      description: t(
        'Call Grok models enabled on the platform through the OpenAI-compatible Chat Completions endpoint.'
      ),
      endpoints: ['GET /v1/models', 'POST /v1/chat/completions'],
      notes: [
        t(
          'Use the same base_url and authentication as the GPT OpenAI-compatible example.'
        ),
        t('Set model to the exact Grok model name enabled for your account.'),
      ],
    },
  ]
}

function createMarkdown(
  baseUrl: string,
  t: Translator,
  families: ModelFamily[],
  examples: ModelExample[]
) {
  const endpointSections = families
    .map((family) => {
      const endpoints = family.endpoints
        .map((endpoint) => `- \`${endpoint}\``)
        .join('\n')
      const notes = family.notes.map((note) => `- ${note}`).join('\n')
      return `### ${family.title}\n\n${family.description}\n\n${endpoints}\n\n${notes}`
    })
    .join('\n\n')
  const exampleSections = examples
    .map(
      (example) =>
        `### ${example.title}\n\n${example.description}\n\n\`\`\`${example.language}\n${example.code}\n\`\`\``
    )
    .join('\n\n')

  return [
    `# ${t('Aittco Model API Integration Guide')}`,
    t(
      'Gateway: https://max.aittco.com. Use an Aittco user API key and a model name enabled for your account.'
    ),
    `## ${t('Base URL and authentication')}`,
    `- Gateway: \`${baseUrl}\``,
    `- OpenAI-compatible base_url: \`${baseUrl}/v1\``,
    `- ${t('Create a user API key in the Aittco dashboard. Do not use an upstream provider key or publish your key in source code, browser code, repositories, or logs.')}`,
    `- ${t('Replace MODEL with the exact model name shown in your dashboard.')}`,
    `## ${t('Protocols and endpoints')}`,
    endpointSections,
    `## ${t('Request examples')}`,
    exampleSections,
    `## ${t('Compatibility and troubleshooting')}`,
    `- ${t('Streaming, tools, image input, JSON mode, and context length depend on the model, upstream channel, and protocol conversion path.')}`,
    `- ${t('For 401, check the API key and authentication header. For 403 or an unavailable model, verify the model name and account access. For 400, check the request format and supported fields. For 429, check quota and rate limits.')}`,
    `- ${t('Claude Messages and Gemini native request bodies are not interchangeable with OpenAI Chat Completions. Use the matching example for native clients.')}`,
  ].join('\n\n')
}

function downloadMarkdown(
  baseUrl: string,
  t: Translator,
  families: ModelFamily[],
  examples: ModelExample[]
) {
  const markdown = createMarkdown(baseUrl, t, families, examples)
  const url = URL.createObjectURL(
    new Blob([markdown], { type: 'text/markdown;charset=utf-8' })
  )
  const anchor = document.createElement('a')
  anchor.href = url
  anchor.download = 'aittco-model-api.md'
  document.body.appendChild(anchor)
  anchor.click()
  document.body.removeChild(anchor)
  URL.revokeObjectURL(url)
}

export function ModelApiDocs(props: { baseUrl: string }) {
  const { t } = useTranslation()
  const families = createFamilies(t)
  const examples = createModelExamples(props.baseUrl, t)
  const copyLabel = t('Copy')

  return (
    <div className='space-y-8'>
      <section className='grid gap-8 pt-8 lg:grid-cols-[1fr_20rem] lg:items-end'>
        <div className='space-y-5'>
          <Badge variant='outline' className='rounded-md'>
            Aittco API
          </Badge>
          <div className='space-y-4'>
            <h1 className='max-w-3xl text-4xl leading-[1.04] font-semibold text-balance md:text-5xl'>
              {t('Connect to language models')}
            </h1>
            <p className='text-muted-foreground max-w-3xl text-base leading-7'>
              {t(
                'Use one gateway for GPT, Claude, Gemini, and Grok. Choose a protocol supported by the model and authenticate with your Aittco user key.'
              )}
            </p>
          </div>
          <Button
            type='button'
            variant='outline'
            onClick={() =>
              downloadMarkdown(props.baseUrl, t, families, examples)
            }
          >
            <FileDown className='size-4' />
            {t('Export model guide')}
          </Button>
        </div>

        <Card className='rounded-lg'>
          <CardHeader>
            <CardTitle className='flex items-center gap-2'>
              <Route className='size-4' />
              {t('Gateway')}
            </CardTitle>
            <CardDescription>
              {t('Use /v1 as the OpenAI-compatible SDK base_url.')}
            </CardDescription>
          </CardHeader>
          <CardContent className='space-y-3'>
            <code className='bg-muted block rounded-md px-3 py-2 font-mono text-xs break-all'>
              {props.baseUrl}/v1
            </code>
            <div className='text-muted-foreground flex items-center gap-2 text-sm'>
              <KeyRound className='size-4' />
              {t('Use a user API key created in the Aittco dashboard.')}
            </div>
          </CardContent>
        </Card>
      </section>

      <section className='space-y-4'>
        <div className='flex items-center gap-2'>
          <BookOpen className='text-muted-foreground size-5' />
          <h2 className='text-2xl font-semibold'>{t('Protocol overview')}</h2>
        </div>
        <div className='grid gap-4 md:grid-cols-2'>
          {families.map((family) => (
            <Card key={family.title} className='rounded-lg'>
              <CardHeader>
                <div className='flex flex-wrap items-center justify-between gap-2'>
                  <CardTitle>{family.title}</CardTitle>
                  <Badge variant='secondary'>{family.protocol}</Badge>
                </div>
                <CardDescription>{family.description}</CardDescription>
              </CardHeader>
              <CardContent className='space-y-3 text-sm'>
                {family.endpoints.map((endpoint) => (
                  <code
                    key={endpoint}
                    className='bg-muted block rounded-md px-3 py-2 font-mono text-xs break-all'
                  >
                    {endpoint}
                  </code>
                ))}
                {family.notes.map((note) => (
                  <p key={note} className='text-muted-foreground leading-6'>
                    {note}
                  </p>
                ))}
              </CardContent>
            </Card>
          ))}
        </div>
      </section>

      <section className='space-y-4'>
        <div className='flex items-center gap-2'>
          <KeyRound className='text-muted-foreground size-5' />
          <h2 className='text-2xl font-semibold'>
            {t('Model names and API keys')}
          </h2>
        </div>
        <Card className='rounded-lg'>
          <CardContent className='space-y-3 pt-6 text-sm leading-6'>
            <p>
              {t(
                'Check the Aittco dashboard for the exact model name, group, permissions, and availability. The model field must match a name enabled for your account; model visibility can differ by account.'
              )}
            </p>
            <p>
              {t(
                'OpenAI-compatible clients can call GET /v1/models. Claude clients can list models with x-api-key and anthropic-version. Gemini native clients can call GET /v1beta/models. A list only reflects models visible for that protocol.'
              )}
            </p>
            <p>
              {t(
                'Send the API key in an HTTP header. Do not use an upstream Gemini, OpenAI, Anthropic, or xAI key. Never put a user key in browser code, public repositories, or logs.'
              )}
            </p>
          </CardContent>
        </Card>
      </section>

      <section className='space-y-4'>
        <div className='flex items-center gap-2'>
          <BookOpen className='text-muted-foreground size-5' />
          <h2 className='text-2xl font-semibold'>{t('Request examples')}</h2>
        </div>
        <div className='grid gap-4'>
          {examples.map((example) => (
            <Card key={example.title} className='rounded-lg'>
              <CardHeader>
                <CardTitle>{example.title}</CardTitle>
                <CardDescription>{example.description}</CardDescription>
              </CardHeader>
              <CardContent>
                <div className='bg-foreground text-background overflow-hidden rounded-lg'>
                  <div className='flex items-center justify-between border-b border-white/10 px-4 py-2'>
                    <span className='text-xs font-medium opacity-70'>
                      Shell / SDK
                    </span>
                    <CopyButton value={example.code} tooltip={copyLabel}>
                      {copyLabel}
                    </CopyButton>
                  </div>
                  <pre className='overflow-x-auto p-4 text-xs leading-6'>
                    <code>{example.code}</code>
                  </pre>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      </section>

      <section className='space-y-4'>
        <h2 className='text-2xl font-semibold'>
          {t('Compatibility and troubleshooting')}
        </h2>
        <Card className='rounded-lg'>
          <CardContent className='space-y-3 pt-6 text-sm leading-6'>
            <p>
              {t(
                'Streaming, tools, image input, JSON mode, and context length depend on the model, upstream channel, and protocol conversion path. Do not assume every model accepts the same fields.'
              )}
            </p>
            <p>
              {t(
                '401: Check the API key and authentication header. 403 or unavailable model: verify the model name, account group, and access. 400: check JSON, protocol path, and supported fields. 429: check quota, concurrency, or rate limits.'
              )}
            </p>
            <p>
              {t(
                'Claude Messages and Gemini native request bodies are not interchangeable with OpenAI Chat Completions. Use the matching example when working with native clients.'
              )}
            </p>
          </CardContent>
        </Card>
      </section>
    </div>
  )
}
