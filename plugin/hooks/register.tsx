import type { EngineInterface as Engine, Register } from 'claude-code'

// The doppel mod: the user's side of the Stop hook's measurement.
//
// A Stop hook cannot put text in front of the model without continuing the
// turn, which is why the agent note is gated so hard (reporter.Notable, the
// Reported ledger, stop_hook_active). A mod draws in the UI and never touches
// the model's context, so it is a third channel that costs no token and no turn:
// a one-line band above the prompt, and a pane that opens on an overview of the
// session and switches to the full delta report.
//
// The mod renders nothing itself. Every text comes from the binary — the band
// is the delta report's scoreboard, the overview a bounded selection from that
// report, and the full report is the delta report exactly as `doppel diff`
// prints it — and they are read from the report the Stop hook
// already wrote, through `doppel hook view`, which runs no analysis. One
// rendering, three surfaces; one pipeline run per turn.

/** What `doppel hook view` prints: the band line, the overview and the full delta report. */
type View = { band: string; overview: string; report: string }

const PANE = 'doppel'
const PANE_TITLE = 'doppel: since session start'

// Display state only, and deliberately held by the module rather than $.state:
// every value here is re-derived from the binary on session.start, so a hot
// reload that drops them loses nothing a refresh does not restore, and the
// plugin's manifest needs no type contract for them.
let view: View | null = null
// The band text the person dismissed. The band stays hidden until the session
// says something different, so "hide" never hides a later finding.
let hiddenBand: string | null = null
// Whether the pane shows the full report rather than the overview. Every
// opening starts on the overview: the full report is a place to look something
// up, not the picture to come back to.
let showFull = false

/** parseView accepts exactly the shape `hook view` prints, and nothing else. */
export function parseView(stdout: string): View | null {
  const text = stdout.trim()
  if (text === '') return null
  try {
    const v = JSON.parse(text)
    if (typeof v?.band === 'string' && v.band !== '' && typeof v?.report === 'string') {
      // A binary from before the overview existed has only the report.
      const overview = typeof v.overview === 'string' && v.overview !== '' ? v.overview : v.report
      return { band: v.band, overview, report: v.report }
    }
  } catch {
    // A binary too old to know `hook view` prints usage text: show nothing.
  }
  return null
}

// refresh asks the binary for the session's view and redraws. Every failure
// — no binary, a binary without `hook view`, a surface with no processes — ends
// at "nothing to show", never at an error the person sees.
async function refresh($: Engine, binary: string, sessionId?: string): Promise<void> {
  let next: View | null = null
  try {
    const id = sessionId ?? (await $.session.id())
    const run = await $.process.run([binary, 'hook', 'view'], {
      stdin: JSON.stringify({ session_id: id }),
      timeoutMs: 10_000,
    })
    next = run.exitCode === 0 ? parseView(run.stdout) : null
  } catch {
    next = null
  }
  view = next
  $.ui.invalidate('ui.render')
}

export const register: Register = (on, options) => {
  const binary = String(options.doppel_binary ?? 'doppel') || 'doppel'

  on('session.start', async ($, e, next) => {
    try {
      await $.command.register({
        name: 'doppel',
        description: "Toggle doppel's pane: what this session did to the duplication picture",
      })
    } catch {
      // A surface that refuses commands still gets the band.
    }
    // A resumed session shows where it stood; a fresh one finds no report.
    await refresh($, binary)
    return next(e)
  })

  // The Stop hook is a settings hook of this same plugin, and settings hooks
  // run as the engine's own behaviour beneath every mod: once next(e) resolves
  // it has measured the turn and rewritten (or removed) its report.
  on('classic.Stop', async ($, e, next) => {
    const result = await next(e)
    await refresh($, binary, e.session_id)
    return result
  }).catch(($, e, next) => next(e))

  // /clear starts a new session id without a session.start; the old session's
  // view must not survive into it.
  on('classic.SessionStart', async ($, e, next) => {
    const result = await next(e)
    await refresh($, binary, e.session_id)
    return result
  }).catch(($, e, next) => next(e))

  on('command.run', { command: 'doppel' }, async $ => {
    const isOpen = (await $.ui.panes()).some(p => p.id === PANE)
    if (isOpen) {
      await $.ui.close({ id: PANE })
      return {}
    }
    if (view === null) {
      $.ui.toast('doppel: nothing to report since session start')
      return {}
    }
    hiddenBand = null
    showFull = false
    const opened = await $.ui.open({ id: PANE, title: PANE_TITLE })
    if (!opened.isPlaced) $.ui.toast('doppel: this surface cannot show the pane')
    // {} rather than { text }: a command's text is a transcript row the model
    // reads, and this channel exists to cost the model nothing.
    return {}
  })

  on('ui.render', { component: 'AbovePrompt' }, async ($, e, next) => {
    if (e.props.hasSurvey || view === null || view.band === hiddenBand) return next(e)
    const { Box, Button, Text } = $.ui.resolve(e)
    return (
      <Box flexDirection="row" gap={1}>
        <Text dimColor wrap="truncate-end">
          {view.band}
        </Text>
        <Button
          key="doppel-details"
          label="details"
          plain
          onPress={async () => {
            showFull = false
            await $.ui.open({ id: PANE, title: PANE_TITLE })
          }}
        />
        <Button
          key="doppel-hide"
          label="hide"
          plain
          onPress={() => {
            hiddenBand = view?.band ?? null
            $.ui.invalidate('ui.render')
          }}
        />
      </Box>
    )
  })

  on('ui.render', { component: 'Pane', requestId: PANE }, async ($, e) => {
    const { Box, Button, Text } = $.ui.resolve(e)
    if (view === null) {
      return (
        <Box flexDirection="column">
          <Text dimColor>Nothing to report since session start.</Text>
        </Box>
      )
    }
    // Lines as the binary wrote them. Indented lines under a headline are
    // evidence, so they are dimmed; nothing is reworded or reordered.
    const text = showFull ? view.report : view.overview
    const lines = text.replace(/\n+$/, '').split('\n')
    // The toggle exists only when there is something to switch to: the
    // fallback view carries one text as both.
    const toggle =
      view.overview === view.report ? null : (
        <Button
          key="doppel-toggle"
          label={showFull ? 'overview' : 'full report'}
          plain
          onPress={() => {
            showFull = !showFull
            $.ui.invalidate('ui.render')
          }}
        />
      )
    return (
      <Box flexDirection="column">
        {toggle}
        {lines.map(line => (
          <Text dimColor={line.startsWith('    ')}>{line === '' ? ' ' : line}</Text>
        ))}
      </Box>
    )
  })
}
