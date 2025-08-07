'use client'

import { useState } from 'react'
import { Download, FileText, FileSpreadsheet, FileImage } from 'lucide-react'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { RadioGroup, RadioGroupItem } from '@/components/ui/radio-group'
import { Label } from '@/components/ui/label'

interface ExportDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  onExport: (format: 'json' | 'csv' | 'pdf') => void
}

const formats = [
  {
    value: 'json' as const,
    label: 'JSON',
    description: 'Structured data format',
    icon: FileText,
  },
  {
    value: 'csv' as const,
    label: 'CSV',
    description: 'Spreadsheet format',
    icon: FileSpreadsheet,
  },
  {
    value: 'pdf' as const,
    label: 'PDF',
    description: 'Printable document',
    icon: FileImage,
  },
]

export function ExportDialog({ open, onOpenChange, onExport }: ExportDialogProps) {
  const [selectedFormat, setSelectedFormat] = useState<'json' | 'csv' | 'pdf'>('json')

  const handleExport = () => {
    onExport(selectedFormat)
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[425px]">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Download className="h-5 w-5" />
            Export Summaries
          </DialogTitle>
          <DialogDescription>
            Choose the format for your exported summaries. The export will include all summaries based on your current filters.
          </DialogDescription>
        </DialogHeader>

        <div className="py-4">
          <RadioGroup
            value={selectedFormat}
            onValueChange={(value) => setSelectedFormat(value as 'json' | 'csv' | 'pdf')}
            className="space-y-3"
          >
            {formats.map((format) => (
              <div key={format.value} className="flex items-center space-x-3">
                <RadioGroupItem value={format.value} id={format.value} />
                <Label
                  htmlFor={format.value}
                  className="flex items-center gap-3 cursor-pointer flex-1 p-3 rounded-lg border hover:bg-accent"
                >
                  <format.icon className="h-6 w-6 text-primary" />
                  <div>
                    <div className="font-medium">{format.label}</div>
                    <div className="text-sm text-muted-foreground">{format.description}</div>
                  </div>
                </Label>
              </div>
            ))}
          </RadioGroup>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
          <Button onClick={handleExport}>
            <Download className="h-4 w-4 mr-2" />
            Export {selectedFormat.toUpperCase()}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
