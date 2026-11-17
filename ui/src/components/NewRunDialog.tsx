import { useMemo, useState } from 'react'
import { Loader2 } from 'lucide-react'
import { Button } from '@/components/ui/button'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from '@/components/ui/dialog'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from '@/components/ui/select'
import { createEval } from '@/lib/api'
import type { EvalRecord, RunnerAvailability } from '@/types'

export function NewRunDialog({
  runners,
  onCreated,
}: {
  runners: RunnerAvailability[]
  onCreated: (rec: EvalRecord) => void
}) {
  const [open, setOpen] = useState(false)
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState<string>()
  const defaultRunner = useMemo(
    () => runners.find((item) => item.available)?.type || runners[0]?.type || 'self',
    [runners],
  )
  const [runner, setRunner] = useState(String(defaultRunner))
  const [model, setModel] = useState('glm-5')
  const [skill, setSkill] = useState('pdf')
  const [skillDir, setSkillDir] = useState('')
  const [prompt, setPrompt] = useState('把一句问候写成 pdf，并调用 finish')

  const submit = async () => {
    setBusy(true)
    setError(undefined)
    try {
      const rec = await createEval({
        runner,
        model: runner === 'self' ? model : model || undefined,
        targetSkill: skill || undefined,
        skillDir: skillDir || undefined,
        userPrompt: prompt,
        wait: false,
      })
      setOpen(false)
      onCreated(rec)
    } catch (err) {
      setError(err instanceof Error ? err.message : '创建失败')
    } finally {
      setBusy(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button size="sm">新建评测</Button>
      </DialogTrigger>
      <DialogContent className="max-w-xl">
        <DialogHeader>
          <DialogTitle>新建评测任务</DialogTitle>
          <DialogDescription>提交到可插拔执行后端，结果与评分写入同一套 Trace / Report。</DialogDescription>
        </DialogHeader>
        <div className="grid gap-4 py-2">
          <div className="grid gap-2">
            <Label>执行后端</Label>
            <Select value={runner} onValueChange={setRunner}>
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {(runners.length ? runners : [{ type: 'self', name: '自研内核', available: true, detail: '' }]).map((item) => (
                  <SelectItem key={item.type} value={String(item.type)}>
                    {item.name}
                    {item.available ? '' : '（不可用）'}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div className="grid gap-2">
              <Label htmlFor="model">模型</Label>
              <Input id="model" value={model} onChange={(e) => setModel(e.target.value)} placeholder="self 必填" />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="skill">目标 Skill</Label>
              <Input id="skill" value={skill} onChange={(e) => setSkill(e.target.value)} />
            </div>
          </div>
          <div className="grid gap-2">
            <Label htmlFor="skillDir">技能目录</Label>
            <Input
              id="skillDir"
              value={skillDir}
              onChange={(e) => setSkillDir(e.target.value)}
              placeholder="默认读取 EVAL_DEFAULT_SKILL_DIR"
            />
          </div>
          <div className="grid gap-2">
            <Label htmlFor="prompt">用户任务</Label>
            <Textarea id="prompt" value={prompt} onChange={(e) => setPrompt(e.target.value)} rows={4} />
          </div>
          {error && <p className="text-sm text-destructive">{error}</p>}
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => setOpen(false)} disabled={busy}>
            取消
          </Button>
          <Button onClick={submit} disabled={busy || !prompt.trim()}>
            {busy && <Loader2 className="animate-spin" />}
            提交
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
