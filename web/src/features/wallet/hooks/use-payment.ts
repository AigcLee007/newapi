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
import i18next from 'i18next'
import { useState, useCallback } from 'react'
import { toast } from 'sonner'

import { handleServerError } from '@/lib/handle-server-error'

import {
  calculateAmount,
  calculateStripeAmount,
  calculateWaffoAmount,
  calculateWaffoPancakeAmount,
  requestPayment,
  requestXunhuPayment,
  requestStripePayment,
  isApiSuccess,
} from '../api'
import {
  isStripePayment,
  isWaffoPayment,
  isWaffoPancakePayment,
  submitPaymentForm,
} from '../lib'
import type { AmountRequest, AmountResponse, XunhuCheckout } from '../types'

// ============================================================================
// Payment Hook
// ============================================================================

type AmountCalculator = (request: AmountRequest) => Promise<AmountResponse>

export interface PaymentAmountPreview {
  amount: number
  baseAmount?: number
  feeAmount?: number
  feePercent?: number
}

export interface PaymentAmountCalculators {
  regular: AmountCalculator
  stripe: AmountCalculator
  waffo: AmountCalculator
  waffoPancake: AmountCalculator
}

const defaultPaymentAmountCalculators: PaymentAmountCalculators = {
  regular: calculateAmount,
  stripe: calculateStripeAmount,
  waffo: calculateWaffoAmount,
  waffoPancake: calculateWaffoPancakeAmount,
}

export async function requestPaymentAmount(
  topupAmount: number,
  paymentType: string,
  calculators: PaymentAmountCalculators = defaultPaymentAmountCalculators,
  paymentProvider?: string
): Promise<PaymentAmountPreview> {
  let calculator = calculators.regular
  if (isStripePayment(paymentType)) {
    calculator = calculators.stripe
  } else if (isWaffoPayment(paymentType)) {
    calculator = calculators.waffo
  } else if (isWaffoPancakePayment(paymentType)) {
    calculator = calculators.waffoPancake
  }

  const request: AmountRequest = { amount: topupAmount }
  if (paymentProvider === 'xunhu' && calculator === calculators.regular) {
    request.provider = paymentProvider
    request.payment_method = paymentType
  }

  const response = await calculator(request)
  if (!isApiSuccess(response) || !response.data) {
    return { amount: 0 }
  }

  const preview: PaymentAmountPreview = {
    amount: Number.parseFloat(response.data),
  }
  if (
    paymentProvider === 'xunhu' &&
    response.base_amount !== undefined &&
    response.fee_amount !== undefined &&
    response.fee_percent !== undefined
  ) {
    preview.baseAmount = Number.parseFloat(response.base_amount)
    preview.feeAmount = Number.parseFloat(response.fee_amount)
    preview.feePercent = response.fee_percent
  }
  return preview
}

export function usePayment() {
  const [amount, setAmount] = useState<number>(0)
  const [paymentBaseAmount, setPaymentBaseAmount] = useState<number>()
  const [paymentFeeAmount, setPaymentFeeAmount] = useState<number>()
  const [paymentFeePercent, setPaymentFeePercent] = useState<number>()
  const [calculating, setCalculating] = useState(false)
  const [processing, setProcessing] = useState(false)
  const [xunhuCheckout, setXunhuCheckout] = useState<XunhuCheckout | null>(null)

  // Calculate payment amount
  const calculatePaymentAmount = useCallback(
    async (
      topupAmount: number,
      paymentType: string,
      paymentProvider?: string
    ) => {
      try {
        setCalculating(true)
        setPaymentBaseAmount(undefined)
        setPaymentFeeAmount(undefined)
        setPaymentFeePercent(undefined)
        const preview = await requestPaymentAmount(
          topupAmount,
          paymentType,
          defaultPaymentAmountCalculators,
          paymentProvider
        )
        setAmount(preview.amount)
        setPaymentBaseAmount(preview.baseAmount)
        setPaymentFeeAmount(preview.feeAmount)
        setPaymentFeePercent(preview.feePercent)
        return preview.amount
      } catch {
        setAmount(0)
        setPaymentBaseAmount(undefined)
        setPaymentFeeAmount(undefined)
        setPaymentFeePercent(undefined)
        return 0
      } finally {
        setCalculating(false)
      }
    },
    []
  )

  // Process payment
  const processPayment = useCallback(
    async (topupAmount: number, paymentType: string) => {
      try {
        setProcessing(true)

        const isStripe = isStripePayment(paymentType)
        const amount = Math.floor(topupAmount)

        const response = isStripe
          ? await requestStripePayment({
              amount,
              payment_method: 'stripe',
            })
          : await requestPayment({
              amount,
              payment_method: paymentType,
            })

        if (!isApiSuccess(response)) {
          handleServerError(response, i18next.t('Payment request failed'))
          return false
        }

        // Handle Stripe payment
        if (isStripe && response.data?.pay_link) {
          window.open(response.data.pay_link as string, '_blank')
          toast.success(i18next.t('Redirecting to payment page...'))
          return true
        }

        // Handle non-Stripe payment
        if (!isStripe && response.data) {
          const url = (response as unknown as { url?: string }).url
          if (url) {
            submitPaymentForm(url, response.data)
            toast.success(i18next.t('Redirecting to payment page...'))
            return true
          }
        }

        return false
      } catch (error) {
        handleServerError(error, i18next.t('Payment request failed'))
        return false
      } finally {
        setProcessing(false)
      }
    },
    []
  )

  const processXunhuPayment = useCallback(
    async (topupAmount: number, paymentType: string) => {
      try {
        setProcessing(true)
        const response = await requestXunhuPayment({
          amount: Math.floor(topupAmount),
          payment_method: paymentType,
        })
        if (
          !isApiSuccess(response) ||
          (!response.data?.url && !response.data?.url_qrcode)
        ) {
          handleServerError(response, i18next.t('Payment request failed'))
          return false
        }
        setXunhuCheckout(response.data)
        return true
      } catch (error) {
        handleServerError(error, i18next.t('Payment request failed'))
        return false
      } finally {
        setProcessing(false)
      }
    },
    []
  )

  return {
    amount,
    paymentBaseAmount,
    paymentFeeAmount,
    paymentFeePercent,
    calculating,
    processing,
    calculatePaymentAmount,
    processPayment,
    processXunhuPayment,
    xunhuCheckout,
    clearXunhuCheckout: () => setXunhuCheckout(null),
    setAmount,
  }
}
