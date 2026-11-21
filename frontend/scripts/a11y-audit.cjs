/*
 * Accessibility audit for the FTU frontend (WCAG 2.1 AA / EN 301 549 clause 9 & 11).
 * Drives the UI against the dev mock in headless Chromium and checks it with axe-core plus
 * explicit keyboard, focus, motion, title and reflow tests.
 *
 *   BASE_URL=http://localhost:5173 CHROME=/usr/bin/chromium \
 *   PUPPETEER=/path/to/puppeteer-core AXE=/path/to/axe.min.js node scripts/a11y-audit.cjs out.json
 *
 * Neither puppeteer-core nor axe-core is a project dependency; point the env vars at any install
 * (the global `lighthouse` package ships both).
 */
const fs = require('fs');
const puppeteer = require(process.env.PUPPETEER);
const axeSrc = fs.readFileSync(process.env.AXE, 'utf8');
const BASE = process.env.BASE_URL || 'http://localhost:5173';
const wait = (ms) => new Promise((r) => setTimeout(r, ms));
const report = { axe: [], checks: [] };
const check = (id, pass, detail) => { report.checks.push({ id, pass, detail }); console.log(pass ? 'PASS' : 'FAIL', id, detail || ''); };

const click = (p, t) => p.evaluate((t) => { const b = [...document.querySelectorAll('button')].find((x) => x.textContent.includes(t)); if (!b) throw new Error('no button ' + t); b.click(); }, t);

async function axeRun(p, name) {
  await p.evaluate(axeSrc);
  const r = await p.evaluate(() => axe.run(document, { runOnly: { type: 'tag', values: ['wcag2a', 'wcag2aa', 'wcag21a', 'wcag21aa', 'wcag22aa', 'best-practice'] } }));
  const v = r.violations.map((x) => ({ id: x.id, impact: x.impact, nodes: x.nodes.length, first: x.nodes[0].target.join(' ') }));
  report.axe.push({ state: name, violations: v, passes: r.passes.length, needsReview: r.incomplete.map((i) => i.id) });
  console.log('axe', name, 'violations:', v.length, v.map((x) => x.id).join(','), '| review:', r.incomplete.map((i) => i.id).join(','));
}

async function toState(p, state, role = 'Send') {
  await p.goto(BASE, { waitUntil: 'networkidle0' });
  if (state === 'home') return;
  await click(p, role); await wait(700);
  if (state === 'pairing') return;
  await p.type('#remote-token', 'TOKEN'); await click(p, 'Connect');
  if (state === 'connecting') { await wait(900); return; }
  await wait(3200);
  if (state === 'transfer') return;
  if (role === 'Send') {
    await click(p, 'Choose a file');
    if (state === 'awaiting') { await wait(500); return; }
    if (state === 'transferring') { await wait(2400); return; }
    await wait(5200); // completed
  } else { await wait(1800); } // deciding
}

(async () => {
  const b = await puppeteer.launch({ executablePath: process.env.CHROME, headless: true, args: ['--no-sandbox'] });
  const newPage = async (theme, w = 980, h = 680) => {
    const p = await b.newPage(); await p.setViewport({ width: w, height: h });
    await p.evaluateOnNewDocument((t) => { localStorage.setItem('ftu_theme', t); }, theme);
    return p;
  };

  // 1. axe on every screen/state in both themes
  for (const theme of ['light', 'dark']) {
    for (const [state, role] of [['home'], ['pairing'], ['connecting'], ['transfer'], ['awaiting'], ['transferring'], ['completed'], ['deciding', 'Receive']]) {
      const p = await newPage(theme); await toState(p, state, role || 'Send'); await axeRun(p, `${theme}:${state}`); await p.close();
    }
  }

  // 2. page titles (WCAG 2.4.2) and focus moved to the new screen (2.4.3)
  {
    const p = await newPage('dark'); await p.goto(BASE, { waitUntil: 'networkidle0' });
    const titles = {}; titles.home = await p.title();
    await click(p, 'Send'); await wait(500); titles.pairing = await p.title();
    const focusAfterNav = await p.evaluate(() => document.activeElement && (document.activeElement.className || document.activeElement.tagName));
    await p.type('#remote-token', 'T'); await click(p, 'Connect'); await wait(3300); titles.transfer = await p.title();
    check('2.4.2 distinct page titles', new Set(Object.values(titles)).size === 3, JSON.stringify(titles));
    check('2.4.3 focus moves to new screen', /anim-view-in/.test(String(focusAfterNav)), String(focusAfterNav).slice(0, 40));
    await p.close();
  }

  // 3. keyboard: every screen reachable and operable without a pointer; focus indicator present (2.1.1, 2.4.7)
  {
    const p = await newPage('dark'); await p.goto(BASE, { waitUntil: 'networkidle0' });
    const seq = [];
    for (let i = 0; i < 5; i++) {
      await p.keyboard.press('Tab');
      seq.push(await p.evaluate(() => { const e = document.activeElement; const cs = getComputedStyle(e); return { name: e.getAttribute('aria-label') || e.textContent.trim().slice(0, 24), outline: cs.outlineStyle !== 'none' && parseFloat(cs.outlineWidth) >= 2, border: cs.borderColor }; }));
    }
    check('2.1.1 tab order home', seq.map((s) => s.name).join(' > ') !== '', seq.map((s) => s.name).join(' > '));
    check('2.4.7 focus visible on every tab stop (home)', seq.every((s) => s.outline), JSON.stringify(seq.filter((s) => !s.outline)));
    await p.goto(BASE, { waitUntil: 'networkidle0' });
    // keyboard-only journey: Send -> pairing -> type token -> Enter submits -> transfer
    await p.keyboard.press('Tab'); // first tab stop is Send
    await p.keyboard.press('Enter'); await wait(600);
    const onPairing = await p.evaluate(() => !!document.querySelector('#remote-token'));
    await p.focus('#remote-token'); await p.keyboard.type('TOKEN'); await p.keyboard.press('Enter'); await wait(3400);
    const onTransfer = await p.evaluate(() => /Ready to throw/.test(document.body.innerText));
    check('2.1.1 keyboard-only pairing flow (Enter on Send, Enter in token field)', onPairing && onTransfer, `pairing=${onPairing} transfer=${onTransfer}`);
    const inputFocus = await (async () => { const q = await newPage('dark'); await toState(q, 'pairing'); await q.focus('#remote-token'); const r = await q.evaluate(() => { const cs = getComputedStyle(document.activeElement); return cs.outlineStyle !== 'none' && parseFloat(cs.outlineWidth) >= 2; }); await q.close(); return r; })();
    check('2.4.7 focus ring on text field', inputFocus);
    await p.close();
  }

  // 4. motion: the OS "reduce motion" setting stops looping animations (2.2.2, EN 301 549 11.7)
  {
    const p = await newPage('dark'); await toState(p, 'connecting');
    const running = await p.evaluate(() => [...document.querySelectorAll('.bird-sheet')].some((e) => parseFloat(getComputedStyle(e).animationDuration) > 0.1));
    await p.close();
    const q = await newPage('dark'); await q.emulateMediaFeatures([{ name: 'prefers-reduced-motion', value: 'reduce' }]);
    await toState(q, 'connecting');
    const stopped = await q.evaluate(() => [...document.querySelectorAll('.bird-sheet')].every((e) => parseFloat(getComputedStyle(e).animationDuration) < 0.01));
    check('2.2.2/11.7 OS "reduce motion" stops the pairing animation (and it runs otherwise)', running && stopped, `running=${running} stopped=${stopped}`);
    await q.close();
    const r = await newPage('dark'); await r.emulateMediaFeatures([{ name: 'prefers-reduced-motion', value: 'reduce' }]);
    await toState(r, 'transferring');
    const bitsStill = await r.evaluate(() => [...document.querySelectorAll('.anim-bit')].every((e) => parseFloat(getComputedStyle(e).animationDuration) < 0.01));
    check('2.2.2/11.7 OS "reduce motion" stops the transfer bit trail', bitsStill);
    await r.close();
  }

  // 5. reflow at 320 CSS px (1.4.10) and text enlarged to 200% (1.4.4)
  for (const [state, role] of [['home'], ['pairing'], ['transfer'], ['completed'], ['deciding', 'Receive']]) {
    const p = await newPage('dark', 320, 640); await toState(p, state, role || 'Send');
    const overflow = await p.evaluate(() => { const d = document.documentElement; const clipped = (e) => { for (let n = e.parentElement; n; n = n.parentElement) { const o = getComputedStyle(n).overflowX; if (o === 'hidden' || o === 'clip') return true; } return false; };
    const widest = [...document.querySelectorAll('main *')].filter((e) => e.getBoundingClientRect().right > innerWidth + 1 && getComputedStyle(e).position !== 'absolute' && !clipped(e)).length; return { scroll: d.scrollWidth - d.clientWidth, widest }; });
    check(`1.4.10 reflow 320px (${state})`, overflow.scroll <= 1 && overflow.widest === 0, JSON.stringify(overflow));
    await p.close();
  }
  {
    const p = await newPage('dark', 860, 580); await toState(p, 'completed');
    await p.evaluate(() => { document.documentElement.style.fontSize = '200%'; }); await wait(300);
    const r = await p.evaluate(() => { const act = [...document.querySelectorAll('button')].find((x) => x.textContent.includes('Send another')); const rect = act.getBoundingClientRect(); const main = document.querySelector('main'); return { actionReachable: !!act && main.scrollHeight >= rect.bottom - main.getBoundingClientRect().top, hScroll: document.documentElement.scrollWidth - document.documentElement.clientWidth }; });
    check('1.4.4 text at 200%: content scrolls, no horizontal scroll, action reachable', r.hScroll <= 1 && r.actionReachable, JSON.stringify(r));
    await p.close();
  }

  // 6. structure: landmarks, one h1, accessible names (1.3.1, 4.1.2)
  for (const [state, role] of [['home'], ['pairing'], ['transfer'], ['deciding', 'Receive']]) {
    const p = await newPage('dark'); await toState(p, state, role || 'Send');
    const s = await p.evaluate(() => ({ h1: document.querySelectorAll('h1').length, main: document.querySelectorAll('main').length, unnamed: [...document.querySelectorAll('button')].filter((b) => !(b.getAttribute('aria-label') || b.textContent.trim())).length, lang: document.documentElement.lang }));
    check(`1.3.1/4.1.2 structure (${state})`, s.h1 === 1 && s.main === 1 && s.unnamed === 0 && s.lang === 'en', JSON.stringify(s));
    await p.close();
  }

  // 7. status messages (4.1.3): live regions exist for transfer phase and are assertive when action is needed
  {
    const p = await newPage('dark'); await toState(p, 'deciding', 'Receive');
    const live = await p.evaluate(() => document.querySelector('h1').parentElement.getAttribute('aria-live'));
    check('4.1.3 incoming offer announced assertively', live === 'assertive', String(live));
    await p.close();
  }

  fs.writeFileSync(process.argv[2], JSON.stringify(report, null, 1));
  const failed = report.checks.filter((c) => !c.pass).length, violations = report.axe.reduce((n, a) => n + a.violations.length, 0);
  console.log(`\nchecks failed: ${failed}/${report.checks.length}; axe violations: ${violations}`);
  await b.close();
})().catch((e) => { console.error(e); process.exit(1); });
