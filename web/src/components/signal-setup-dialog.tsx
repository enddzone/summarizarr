'use client'

import { useState } from 'react'
import Image from 'next/image'
import { Smartphone, QrCode, CheckCircle, AlertCircle, Loader2 } from 'lucide-react'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Card, CardContent } from '@/components/ui/card'
import { Badge } from '@/components/ui/badge'
import { useToast } from '@/hooks/use-toast'
import { useSignalStatusPolling } from '@/hooks/use-signal-status-polling'
import type { SignalConfig } from '@/types'

interface SignalSetupDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  signalConfig: SignalConfig
  onConfigUpdate: (config: SignalConfig) => void
}

export function SignalSetupDialog({
  open,
  onOpenChange,
  signalConfig,
  onConfigUpdate
}: SignalSetupDialogProps) {
  const [phoneNumber, setPhoneNumber] = useState(signalConfig.phoneNumber)
  const [isLoading, setIsLoading] = useState(false)
  const [step, setStep] = useState<'phone' | 'qr' | 'complete'>(
    signalConfig.isRegistered ? 'complete' : 'phone'
  )
  const [qrCodeUrl, setQrCodeUrl] = useState(signalConfig.qrCodeUrl || '')
  const { toast } = useToast()

  // Use polling hook during QR step
  const { isPolling, error: pollingError } = useSignalStatusPolling(
    step === 'qr',
    (data) => {
      // On successful registration
      onConfigUpdate({
        phoneNumber: data.phoneNumber,
        isRegistered: true,
        connected: data.connected,
        status: data.status,
      })
      setStep('complete')
      toast({
        title: 'Success',
        description: 'Signal registration detected automatically!',
        variant: 'default',
      })
    }
  )

  const handleRegisterPhone = async () => {
    if (!phoneNumber.trim()) {
      toast({
        title: 'Error',
        description: 'Please enter a valid phone number',
        variant: 'destructive',
      })
      return
    }

    setIsLoading(true)
    try {
      const response = await fetch('/api/signal/register', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        credentials: 'include',
        body: JSON.stringify({ phoneNumber: phoneNumber.trim() }),
      })

      if (!response.ok) throw new Error('Registration failed')

      const data = await response.json()

      if (data.qrCodeUrl) {
        setQrCodeUrl(data.qrCodeUrl)
        setStep('qr')
      } else if (data.isRegistered) {
        onConfigUpdate({
          phoneNumber: phoneNumber.trim(),
          isRegistered: true,
        })
        setStep('complete')
      }
    } catch (error) {
      console.error('Registration error:', error)
      toast({
        title: 'Registration failed',
        description: 'Failed to register phone number with Signal',
        variant: 'destructive',
      })
    } finally {
      setIsLoading(false)
    }
  }

  const handleCheckStatus = async () => {
    setIsLoading(true)
    try {
      const response = await fetch('/api/signal/status')
      if (!response.ok) throw new Error('Status check failed')

      const data = await response.json()

      if (data.isRegistered) {
        onConfigUpdate({
          phoneNumber: phoneNumber.trim(),
          isRegistered: true,
        })
        setStep('complete')
        toast({
          title: 'Success',
          description: 'Signal registration completed successfully!',
        })
      }
    } catch (error) {
      console.error('Status check error:', error)
    } finally {
      setIsLoading(false)
    }
  }

  const handleReset = () => {
    setStep('phone')
    setPhoneNumber('')
    setQrCodeUrl('')
  }

  const renderPhoneStep = () => (
    <div className="space-y-4">
      <div className="space-y-2">
        <Label htmlFor="phone">Phone Number</Label>
        <Input
          id="phone"
          placeholder="+1234567890"
          value={phoneNumber}
          onChange={(e) => setPhoneNumber(e.target.value)}
          disabled={isLoading}
        />
        <p className="text-sm text-muted-foreground">
          Enter your phone number in international format (e.g., +1234567890)
        </p>
      </div>
    </div>
  )

  const renderQrStep = () => (
    <div className="space-y-4">
      <Card>
        <CardContent className="flex flex-col items-center space-y-4 p-6">
          {qrCodeUrl ? (
            <Image
              src={qrCodeUrl}
              alt="Signal QR Code"
              width={192}
              height={192}
              className="border rounded-lg"
            />
          ) : (
            <div className="w-48 h-48 bg-muted rounded-lg flex items-center justify-center">
              <QrCode className="h-12 w-12 text-muted-foreground" />
            </div>
          )}
          <div className="text-center space-y-2">
            <h4 className="font-medium">Scan QR Code with Signal</h4>
            <p className="text-sm text-muted-foreground">
              Open Signal on your phone and scan this QR code to link your device
            </p>
            {isPolling && (
              <div className="flex items-center justify-center space-x-2 mt-2">
                <Loader2 className="h-4 w-4 animate-spin" />
                <span className="text-xs text-muted-foreground">
                  Waiting for registration...
                </span>
              </div>
            )}
            {pollingError && (
              <p className="text-xs text-destructive mt-2">
                {pollingError}
              </p>
            )}
          </div>
        </CardContent>
      </Card>

      <Button
        onClick={handleCheckStatus}
        disabled={isLoading || isPolling}
        className="w-full"
      >
        {isLoading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
        Check Registration Status
      </Button>
    </div>
  )

  const renderCompleteStep = () => (
    <div className="space-y-4 text-center">
      <div className="flex flex-col items-center space-y-2">
        <CheckCircle className="h-12 w-12 text-green-500" />
        <h4 className="font-medium">Registration Complete!</h4>
        <p className="text-sm text-muted-foreground">
          Your Signal account is now connected and ready to receive summaries.
        </p>
      </div>
    </div>
  )

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[500px]">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Smartphone className="h-5 w-5" />
            Signal Setup
          </DialogTitle>
          <DialogDescription>
            {step === 'phone' && 'Register your phone number with Signal to start receiving summaries.'}
            {step === 'qr' && 'Complete the registration by scanning the QR code with your Signal app.'}
            {step === 'complete' && 'Your Signal integration is ready!'}
          </DialogDescription>
        </DialogHeader>

        <div className="py-4">
          {/* Status indicator */}
          <div className="flex items-center justify-between mb-6">
            <div className="space-y-1">
              <Badge variant={signalConfig.isRegistered ? "default" : "secondary"}>
                {signalConfig.isRegistered ? (
                  <>
                    <CheckCircle className="h-3 w-3 mr-1" />
                    Connected
                  </>
                ) : (
                  <>
                    <AlertCircle className="h-3 w-3 mr-1" />
                    Not Connected
                  </>
                )}
              </Badge>
              {signalConfig.isRegistered && signalConfig.phoneNumber && (
                <p className="text-sm text-muted-foreground">
                  Registered: {signalConfig.phoneNumber}
                </p>
              )}
            </div>
          </div>

          {step === 'phone' && renderPhoneStep()}
          {step === 'qr' && renderQrStep()}
          {step === 'complete' && renderCompleteStep()}
        </div>

        <DialogFooter>
          {step === 'phone' && (
            <>
              <Button variant="outline" onClick={() => onOpenChange(false)}>
                Cancel
              </Button>
              <Button onClick={handleRegisterPhone} disabled={isLoading}>
                {isLoading && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                Register
              </Button>
            </>
          )}

          {step === 'qr' && (
            <>
              <Button variant="outline" onClick={handleReset}>
                Back
              </Button>
              <Button variant="outline" onClick={() => onOpenChange(false)}>
                Close
              </Button>
            </>
          )}

          {step === 'complete' && (
            <>
              <Button variant="outline" onClick={handleReset}>
                Register New Number
              </Button>
              <Button onClick={() => onOpenChange(false)}>
                Done
              </Button>
            </>
          )}
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
