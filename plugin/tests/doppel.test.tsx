import { expect, test } from 'claude-code/testing'

import { parseView } from '../hooks/register'

// The view `doppel hook view` prints after a turn that renamed a function and
// copied another. Its text is the binary's; the mod must draw it, not compose it.
const VIEW = {
  band: 'doppel: renamed 1, new 1; pairs created 3, dissolved 1 — since session start',
  overview: [
    'renamed 1, new 1; functions 3 -> 4',
    '',
    'merge-worthy pairs created 1',
    '  svc.Clip <-> svc.Trim  shape 1.00  (svc.Trim new)',
    '',
    'functions changed 2',
    '  renamed  svc.Total -> svc.Sum',
    '  new      svc.Trim  svc/svc.go:49',
    '',
  ].join('\n'),
  report: [
    'Delta since the baseline',
    '========================',
    '3 functions before, 4 after',
    '',
    'renamed 1',
    '  svc.Total (svc/svc.go:3) -> svc.Sum (svc/svc.go:3)',
    '      jaccard 1.0000  containment 1.0000  digests equal',
    '',
    'pairs created 1',
    '  svc.Clip <-> svc.Trim  shape 1.00  overlap 0.62  (merge-worthy)',
    '      svc.Trim new',
    '      explain: identical after rename',
    '',
  ].join('\n'),
}

const BAND = {
  plugin: 'doppel',
  component: 'AbovePrompt',
  requestId: 'above-prompt',
  viewport: { columns: 120, rows: 40 },
  props: {
    hasSurvey: false,
    isWorking: false,
    maxRows: 10,
    bodyColumns: 115,
    scroll: { offset: 0, bodyRows: 10 },
    view: {},
  },
} as const

const PANE = {
  plugin: 'doppel',
  component: 'Pane',
  requestId: 'doppel',
  viewport: { columns: 160, rows: 40 },
  props: {
    title: 'doppel: since session start',
    isFocused: false,
    bodyColumns: 70,
    placement: 'dock',
    scroll: { offset: 0, bodyRows: 30 },
    view: {},
  },
} as const

const SURFACES = ['terminal', 'desktop'] as const

// The kit matches a string as a substring; the mod's promise is the line
// exactly as the binary wrote it, so match whole.
const exactly = (s: string) => new RegExp('^' + s.replace(/[.*+?^${}()|[\]\\]/g, '\\$&') + '$')

// What the engine would draw at the band when the mod passes it on.
const engineBand = () => ({ type: 'Text' as const, props: {}, children: ['drawn by Claude Code'] })

// /doppel as the person types it at the prompt of a fullscreen terminal.
const DOPPEL = {
  command: 'doppel',
  args: '',
  origin: { kind: 'composer' },
  presentation: { isFullscreen: true, columns: 160 },
} as const

test('after a turn the band shows the binary’s line, read from `hook view`', async ($, on) => {
  const runs: { argv: readonly string[]; stdin?: string }[] = []
  on('process.run', ($, e) => {
    runs.push({ argv: e.argv, stdin: e.init?.stdin })
    return { value: { exitCode: 0, stdout: JSON.stringify(VIEW) + '\n', stderr: '', isStdoutTruncated: false, isStderrTruncated: false } }
  })
  on('classic.Stop', () => ({}))
  on('ui.render', engineBand)

  await $.classic.Stop({ session_id: 'sess-1', stop_hook_active: false })

  // One run per turn, and it is the read-only view, never the pipeline.
  expect(runs.length).toBe(1)
  expect(runs[0]!.argv).toEqual(['doppel', 'hook', 'view'])
  expect(JSON.parse(runs[0]!.stdin ?? '{}')).toEqual({ session_id: 'sess-1' })

  for (const surface of SURFACES) {
    const ui = await $.ui.mount({ ...BAND, surface })
    expect(await ui.find({ type: 'Text', text: exactly(VIEW.band) })).toBeDefined()
    expect(await ui.find({ key: 'doppel-details' })).toBeDefined()
    await ui.unmount()
  }
})

test('the binary path comes from userConfig', { options: { doppel_binary: 'C:/tools/doppel.exe' } }, async ($, on) => {
  const argv: (readonly string[])[] = []
  on('process.run', ($, e) => {
    argv.push(e.argv)
    return { value: { exitCode: 0, stdout: '', stderr: '', isStdoutTruncated: false, isStderrTruncated: false } }
  })
  on('classic.Stop', () => ({}))
  await $.classic.Stop({ session_id: 's', stop_hook_active: false })
  expect(argv[0]?.[0]).toBe('C:/tools/doppel.exe')
})

// No report (hook-notify off, a quiet session, a session the Stop hook removed
// the report for) and every failure draw nothing of the mod's own.
for (const [name, stub] of [
  ['an empty view', () => ({ value: { exitCode: 0, stdout: '', stderr: '', isStdoutTruncated: false, isStderrTruncated: false } })],
  ['a binary without `hook view`', () => ({ value: { exitCode: 1, stdout: '', stderr: 'unknown command "view"', isStdoutTruncated: false, isStderrTruncated: false } })],
  ['no binary at all', () => ({ deny: 'ENOENT: doppel' })],
] as const) {
  test(`the band stays empty for ${name}`, async ($, on) => {
    on('process.run', stub)
    on('classic.Stop', () => ({}))
    on('ui.render', engineBand)
    await $.classic.Stop({ session_id: 's', stop_hook_active: false })
    for (const surface of SURFACES) {
      const ui = await $.ui.mount({ ...BAND, surface })
      expect(await ui.find({ key: 'doppel-details' })).toBeUndefined()
      expect(await ui.find({ type: 'Text', text: 'drawn by Claude Code' })).toBeDefined()
      await ui.unmount()
    }
  })
}

test('a later turn that measures nothing clears the band', async ($, on) => {
  let stdout = JSON.stringify(VIEW)
  on('process.run', () => ({ value: { exitCode: 0, stdout, stderr: '', isStdoutTruncated: false, isStderrTruncated: false } }))
  on('classic.Stop', () => ({}))
  on('ui.render', engineBand)

  await $.classic.Stop({ session_id: 's', stop_hook_active: false })
  stdout = ''
  await $.classic.Stop({ session_id: 's', stop_hook_active: false })

  const ui = await $.ui.mount({ ...BAND, surface: 'terminal' })
  expect(await ui.find({ key: 'doppel-details' })).toBeUndefined()
})

test('/doppel opens the pane on the overview, switches to the full report, costs the model nothing, and toggles closed', async ($, on) => {
  const open = new Set<string>()
  on('process.run', () => ({ value: { exitCode: 0, stdout: JSON.stringify(VIEW), stderr: '', isStdoutTruncated: false, isStderrTruncated: false } }))
  on('classic.Stop', () => ({}))
  on('ui.panes', () => ({ value: [...open].map(id => ({ id, title: id, isShown: true, isFocused: false, isPlaced: true })) }))
  on('ui.open', ($, e) => {
    open.add(e.id)
    return { value: { isPlaced: true } }
  })
  on('ui.close', ($, e) => {
    open.delete(e.id)
    return { value: undefined }
  })

  await $.classic.Stop({ session_id: 's', stop_hook_active: false })

  const opened = await $.command.run(DOPPEL)
  // A command's text is a transcript row the model reads; this one has none.
  expect(opened.text).toBeUndefined()
  expect(open.has('doppel')).toBe(true)

  for (const surface of SURFACES) {
    const ui = await $.ui.mount({ ...PANE, surface })
    // The overview first, verbatim, and none of the report's evidence lines.
    for (const line of [
      'merge-worthy pairs created 1',
      '  svc.Clip <-> svc.Trim  shape 1.00  (svc.Trim new)',
      '  renamed  svc.Total -> svc.Sum',
    ]) {
      expect(await ui.find({ type: 'Text', text: exactly(line) })).toBeDefined()
    }
    expect(await ui.find({ type: 'Text', text: 'explain:' })).toBeUndefined()

    await ui.press({ key: 'doppel-toggle' })
    for (const line of [
      'Delta since the baseline',
      '  svc.Total (svc/svc.go:3) -> svc.Sum (svc/svc.go:3)',
      '  svc.Clip <-> svc.Trim  shape 1.00  overlap 0.62  (merge-worthy)',
      '      explain: identical after rename',
    ]) {
      expect(await ui.find({ type: 'Text', text: exactly(line) })).toBeDefined()
    }
    await ui.press({ key: 'doppel-toggle' })
    expect(await ui.find({ type: 'Text', text: 'explain:' })).toBeUndefined()
    await ui.unmount()
  }

  await $.command.run(DOPPEL)
  expect(open.has('doppel')).toBe(false)
})

test('/doppel with nothing to report toasts instead of opening an empty pane', async ($, on) => {
  const toasts: string[] = []
  let opened = false
  on('ui.panes', () => ({ value: [] }))
  on('ui.open', () => {
    opened = true
    return { value: { isPlaced: true } }
  })
  on('ui.toast', ($, e) => {
    toasts.push(e.text)
    return { value: undefined }
  })
  const answer = await $.command.run(DOPPEL)
  expect(answer.text).toBeUndefined()
  expect(opened).toBe(false)
  expect(toasts).toEqual(['doppel: nothing to report since session start'])
})

test('hide dismisses the band until the session says something new', async ($, on) => {
  let stdout = JSON.stringify(VIEW)
  on('process.run', () => ({ value: { exitCode: 0, stdout, stderr: '', isStdoutTruncated: false, isStderrTruncated: false } }))
  on('classic.Stop', () => ({}))
  on('ui.render', engineBand)

  await $.classic.Stop({ session_id: 's', stop_hook_active: false })
  let ui = await $.ui.mount({ ...BAND, surface: 'terminal' })
  await ui.press({ key: 'doppel-hide' })
  expect(await ui.find({ key: 'doppel-details' })).toBeUndefined()
  await ui.unmount()

  // The same measurement again: still hidden.
  await $.classic.Stop({ session_id: 's', stop_hook_active: false })
  ui = await $.ui.mount({ ...BAND, surface: 'terminal' })
  expect(await ui.find({ key: 'doppel-details' })).toBeUndefined()
  await ui.unmount()

  // A different measurement comes back.
  stdout = JSON.stringify({ ...VIEW, band: 'doppel: edited 1; pairs created 0, dissolved 0 — since session start' })
  await $.classic.Stop({ session_id: 's', stop_hook_active: false })
  ui = await $.ui.mount({ ...BAND, surface: 'terminal' })
  expect(await ui.find({ key: 'doppel-details' })).toBeDefined()
})

test('a binary from before the overview shows its report, with nothing to toggle', async ($, on) => {
  const old = { band: VIEW.band, report: VIEW.report }
  on('process.run', () => ({ value: { exitCode: 0, stdout: JSON.stringify(old), stderr: '', isStdoutTruncated: false, isStderrTruncated: false } }))
  on('classic.Stop', () => ({}))
  await $.classic.Stop({ session_id: 's', stop_hook_active: false })
  const ui = await $.ui.mount({ ...PANE, surface: 'terminal' })
  expect(await ui.find({ type: 'Text', text: exactly('Delta since the baseline') })).toBeDefined()
  expect(await ui.find({ key: 'doppel-toggle' })).toBeUndefined()
})

test('parseView accepts only the shape `hook view` prints', async () => {
  expect(parseView(JSON.stringify(VIEW) + '\n')).toEqual(VIEW)
  expect(parseView(JSON.stringify({ band: VIEW.band, report: VIEW.report }))).toEqual({ ...VIEW, overview: VIEW.report })
  expect(parseView('')).toBe(null)
  expect(parseView('Error: unknown command "view" for "doppel hook"')).toBe(null)
  expect(parseView('{"band":"","report":"x"}')).toBe(null)
  expect(parseView('{"band":1,"report":"x"}')).toBe(null)
})
