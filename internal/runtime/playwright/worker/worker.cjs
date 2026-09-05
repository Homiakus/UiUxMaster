#!/usr/bin/env node
'use strict';

const fs = require('fs');

const WORKER_VERSION = '1.0.0';
const FEATURE_SET = Object.freeze({
  clean_state: true,
  supports_aria: true,
  supports_fonts: true,
  supports_scenario: true,
  supports_roi: true,
});

function readStdin() {
  return new Promise((resolve, reject) => {
    let data = '';
    process.stdin.setEncoding('utf8');
    process.stdin.on('data', chunk => { data += chunk; });
    process.stdin.on('end', () => resolve(data));
    process.stdin.on('error', reject);
  });
}

function loadPlaywright() {
  const playwright = require('playwright');
  const pkg = require('playwright/package.json');
  return { playwright, version: pkg.version };
}

function browserEntries(playwright) {
  return [
    ['chromium', playwright.chromium],
    ['firefox', playwright.firefox],
    ['webkit', playwright.webkit],
  ];
}

async function probeBrowser(name, browserType) {
  let executablePath = '';
  try {
    executablePath = browserType.executablePath();
  } catch (err) {
    return { browser: name, ready: false, error: `executablePath: ${String(err)}` };
  }
  if (!executablePath || !fs.existsSync(executablePath)) {
    return {
      browser: name,
      ready: false,
      executable_path: executablePath || undefined,
      error: 'bundled browser executable is not installed',
    };
  }

  let browser;
  try {
    browser = await browserType.launch({ headless: true });
    const version = browser.version();
    return {
      browser: name,
      ready: Boolean(version),
      version: version || undefined,
      executable_path: executablePath,
      error: version ? undefined : 'browser launched without a version',
    };
  } catch (err) {
    return {
      browser: name,
      ready: false,
      executable_path: executablePath,
      error: String(err && err.message ? err.message : err),
    };
  } finally {
    if (browser) {
      try { await browser.close(); } catch (_) {}
    }
  }
}

async function probe() {
  const { playwright, version } = loadPlaywright();
  const browsers = [];
  for (const [name, browserType] of browserEntries(playwright)) {
    browsers.push(await probeBrowser(name, browserType));
  }
  return {
    success: true,
    worker_version: WORKER_VERSION,
    playwright_version: version,
    browsers,
    features: FEATURE_SET,
  };
}

function runtimeIssue(code, message, severity, details) {
  const issue = { code, message, severity };
  if (details && Object.keys(details).length) issue.details = details;
  return issue;
}

function installDiagnostics(page, issues) {
  page.on('console', msg => {
    const type = msg.type();
    if (type === 'error' || type === 'warning') {
      issues.push(runtimeIssue(
        type === 'error' ? 'CONSOLE_ERROR' : 'CONSOLE_WARNING',
        msg.text(),
        type === 'error' ? 'high' : 'medium',
        { type },
      ));
    }
  });
  page.on('pageerror', err => {
    issues.push(runtimeIssue('PAGE_ERROR', String(err && err.message ? err.message : err), 'high'));
  });
  page.on('requestfailed', request => {
    const failure = request.failure();
    issues.push(runtimeIssue(
      'NETWORK_FAILURE',
      failure && failure.errorText ? failure.errorText : 'request failed',
      'medium',
      { url: request.url(), method: request.method() },
    ));
  });
  page.on('response', response => {
    if (response.status() >= 400) {
      issues.push(runtimeIssue(
        'HTTP_ERROR',
        `HTTP ${response.status()} ${response.statusText()}`,
        response.status() >= 500 ? 'high' : 'medium',
        { url: response.url(), status: String(response.status()) },
      ));
    }
  });
}

function injectBase(html, baseURL) {
  if (!baseURL || !html) return html;
  const safe = String(baseURL).replace(/&/g, '&amp;').replace(/"/g, '&quot;');
  const base = `<base href="${safe}">`;
  if (/<head[\s>]/i.test(html)) {
    return html.replace(/<head([^>]*)>/i, `<head$1>${base}`);
  }
  return `${base}${html}`;
}

async function installDeterminism(page, req) {
  if (req.freeze_clock) {
    await page.addInitScript(() => {
      const fixed = 1577836800000;
      const NativeDate = Date;
      class FrozenDate extends NativeDate {
        constructor(...args) {
          super(...(args.length ? args : [fixed]));
        }
        static now() { return fixed; }
      }
      Object.setPrototypeOf(FrozenDate, NativeDate);
      globalThis.Date = FrozenDate;
    });
  }
}

async function loadTarget(page, req) {
  if (req.url) {
    await page.goto(req.url, { waitUntil: 'load', timeout: 30000 });
  } else {
    await page.setContent(injectBase(req.html || '<!doctype html><html><body></body></html>', req.base_url), {
      waitUntil: 'load',
      timeout: 30000,
    });
  }
  if (req.css) {
    await page.addStyleTag({ content: req.css });
  }
  if (req.pause_animations) {
    await page.addStyleTag({
      content: `*,*::before,*::after{animation:none!important;transition:none!important;caret-color:transparent!important;}`,
    });
  }
  await page.evaluate(async () => {
    if (document.fonts && document.fonts.ready) {
      try { await document.fonts.ready; } catch (_) {}
    }
  });
}

async function collectDOM(page, captureAccessibility) {
  return page.evaluate(({ captureAccessibility }) => {
    function implicitRole(el) {
      const tag = el.tagName.toLowerCase();
      if (tag === 'a' && el.hasAttribute('href')) return 'link';
      if (tag === 'button') return 'button';
      if (tag === 'nav') return 'navigation';
      if (tag === 'main') return 'main';
      if (tag === 'header') return 'banner';
      if (tag === 'footer') return 'contentinfo';
      if (tag === 'form') return 'form';
      if (tag === 'img') return 'img';
      if (tag === 'select') return 'combobox';
      if (tag === 'textarea') return 'textbox';
      if (tag === 'input') {
        const t = (el.getAttribute('type') || 'text').toLowerCase();
        if (t === 'checkbox') return 'checkbox';
        if (t === 'radio') return 'radio';
        if (t === 'button' || t === 'submit' || t === 'reset') return 'button';
        if (t === 'range') return 'slider';
        if (t === 'number') return 'spinbutton';
        if (t !== 'hidden') return 'textbox';
      }
      if (/^h[1-6]$/.test(tag)) return 'heading';
      if (tag === 'ul' || tag === 'ol') return 'list';
      if (tag === 'li') return 'listitem';
      if (tag === 'table') return 'table';
      if (tag === 'tr') return 'row';
      if (tag === 'td') return 'cell';
      if (tag === 'th') return 'columnheader';
      return '';
    }

    function accessibleName(el) {
      return (
        el.getAttribute('aria-label') ||
        el.getAttribute('alt') ||
        el.getAttribute('title') ||
        (el.labels && el.labels[0] && el.labels[0].textContent) ||
        (el.textContent || '')
      ).trim().replace(/\s+/g, ' ').slice(0, 256);
    }

    const root = document.documentElement;
    const documents = [{
      url: location.href,
      content_width: Math.max(root.scrollWidth, document.body ? document.body.scrollWidth : 0),
      content_height: Math.max(root.scrollHeight, document.body ? document.body.scrollHeight : 0),
    }];

    const elements = [];
    const accessibility = [];
    Array.from(document.querySelectorAll('*')).forEach((el, index) => {
      const rect = el.getBoundingClientRect();
      const style = getComputedStyle(el);
      const visible = rect.width > 0 && rect.height > 0 && style.display !== 'none' && style.visibility !== 'hidden' && style.opacity !== '0';
      const role = el.getAttribute('role') || implicitRole(el);
      const name = accessibleName(el);
      const attrs = {};
      for (const attr of Array.from(el.attributes).slice(0, 32)) attrs[attr.name] = attr.value.slice(0, 512);
      const id = el.id ? `id:${el.id}` : `dom:${index}`;
      elements.push({
        id,
        tag: el.tagName.toLowerCase(),
        role,
        name,
        selector: el.id ? `#${CSS.escape(el.id)}` : undefined,
        bounds: { x: rect.x, y: rect.y, width: rect.width, height: rect.height },
        visible,
        clickable: Boolean(
          el.matches('button,a[href],input,select,textarea,summary') ||
          ['button', 'link', 'checkbox', 'radio', 'combobox', 'menuitem'].includes(role)
        ),
        attributes: attrs,
      });
      if (captureAccessibility && (role || name)) {
        accessibility.push({
          id: `ax:${index}`,
          role,
          name,
          ignored: !visible,
          properties: {
            disabled: String(Boolean(el.disabled || el.getAttribute('aria-disabled') === 'true')),
            expanded: el.getAttribute('aria-expanded') || '',
            checked: el.getAttribute('aria-checked') || '',
          },
        });
      }
    });
    return { documents, elements, accessibility };
  }, { captureAccessibility });
}

async function collectFonts(page) {
  return page.evaluate(() => {
    if (!document.fonts) return { status: 'unsupported', total: 0, faces: [] };
    const faces = Array.from(document.fonts).map(face => ({
      family: face.family,
      style: face.style,
      weight: face.weight,
      stretch: face.stretch,
      status: face.status,
    }));
    return {
      status: document.fonts.status,
      total: faces.length,
      faces,
      truncated: false,
    };
  });
}

async function ariaSnapshot(page) {
  if (typeof page.ariaSnapshot === 'function') {
    return page.ariaSnapshot();
  }
  const body = page.locator('body');
  if (body && typeof body.ariaSnapshot === 'function') {
    return body.ariaSnapshot();
  }
  return '';
}

async function runScenario(page, scenario) {
  if (!scenario || !Array.isArray(scenario.actions)) return;
  for (const action of scenario.actions) {
    const selector = action.selector || '';
    switch (action.kind) {
      case 'click':
        await page.locator(selector).click();
        break;
      case 'dblclick':
        await page.locator(selector).dblclick();
        break;
      case 'fill':
        await page.locator(selector).fill(action.value || '');
        break;
      case 'hover':
        await page.locator(selector).hover();
        break;
      case 'focus':
        await page.locator(selector).focus();
        break;
      case 'check':
        await page.locator(selector).check();
        break;
      case 'uncheck':
        await page.locator(selector).uncheck();
        break;
      case 'select':
        await page.locator(selector).selectOption(action.value || '');
        break;
      case 'press':
        if (selector) await page.locator(selector).press(action.value || '');
        else await page.keyboard.press(action.value || '');
        break;
      case 'scroll': {
        if (selector) {
          await page.locator(selector).scrollIntoViewIfNeeded();
        } else {
          const parts = String(action.value || '0,600').split(',').map(Number);
          await page.evaluate(([x, y]) => window.scrollBy(x || 0, y || 0), parts);
        }
        break;
      }
      case 'resize': {
        const m = String(action.value || '').match(/^(\d+)x(\d+)$/i);
        if (!m) throw new Error(`invalid resize value ${action.value}`);
        await page.setViewportSize({ width: Number(m[1]), height: Number(m[2]) });
        break;
      }
      case 'wait':
        if (selector) await page.locator(selector).waitFor({ state: 'visible' });
        else if (action.duration) await page.waitForTimeout(Math.max(0, Number(action.duration) / 1e6));
        else if (action.value === 'networkidle') await page.waitForLoadState('networkidle');
        else await page.waitForTimeout(1);
        break;
      default:
        throw new Error(`unsupported scenario action ${action.kind}`);
    }
  }
}

async function capture(req) {
  const { playwright, version: playwrightVersion } = loadPlaywright();
  const browserType = playwright[req.browser];
  if (!browserType || !['chromium', 'firefox', 'webkit'].includes(req.browser)) {
    throw new Error(`unsupported browser ${req.browser}`);
  }

  let browser;
  let context;
  const issues = [];
  const started = Date.now();
  const timings = {};
  try {
    browser = await browserType.launch({ headless: true });
    const browserVersion = browser.version();
    const viewport = req.viewport || {};
    const contextOptions = {
      viewport: {
        width: Number(viewport.width) || 1280,
        height: Number(viewport.height) || 720,
      },
      deviceScaleFactor: Number(viewport.device_scale) || 1,
      reducedMotion: 'reduce',
      locale: 'en-US',
      timezoneId: 'UTC',
    };
    if (viewport.color_scheme === 'dark' || viewport.color_scheme === 'light') {
      contextOptions.colorScheme = viewport.color_scheme;
    }
    if (req.base_url) contextOptions.baseURL = req.base_url;

    context = await browser.newContext(contextOptions);
    const page = await context.newPage();
    installDiagnostics(page, issues);
    await installDeterminism(page, req);

    const navStart = Date.now();
    await loadTarget(page, req);
    timings.snapshot_ms = Date.now() - navStart;

    if (req.command === 'scenario') {
      await runScenario(page, req.scenario);
    }

    let dom = { documents: [], elements: [], accessibility: [] };
    if (req.capture_layout || req.capture_aria) {
      const t = Date.now();
      dom = await collectDOM(page, Boolean(req.capture_aria));
      timings.accessibility_ms = req.capture_aria ? Date.now() - t : 0;
    }

    let aria = '';
    if (req.capture_aria) {
      const t = Date.now();
      aria = await ariaSnapshot(page);
      timings.accessibility_ms = (timings.accessibility_ms || 0) + (Date.now() - t);
    }

    let fonts;
    if (req.capture_fonts) {
      const t = Date.now();
      fonts = await collectFonts(page);
      timings.fonts_ms = Date.now() - t;
    }

    let screenshotB64 = '';
    if (req.capture_pixels) {
      const t = Date.now();
      const options = { type: 'png', animations: 'disabled' };
      if (req.region) {
        options.clip = {
          x: Number(req.region.x) || 0,
          y: Number(req.region.y) || 0,
          width: Number(req.region.width),
          height: Number(req.region.height),
        };
      }
      const buffer = await page.screenshot(options);
      screenshotB64 = buffer.toString('base64');
      timings.pixels_ms = Date.now() - t;
    }

    const diagnostics = req.capture_diagnostics
      ? { complete: true, dropped_methods: [] }
      : undefined;

    timings.diagnostics_ms = 0;
    timings.total_ms = Date.now() - started;
    return {
      success: true,
      worker_version: WORKER_VERSION,
      playwright_version: playwrightVersion,
      browser_version: browserVersion,
      url: page.url(),
      aria_snapshot: aria || undefined,
      screenshot_b64: screenshotB64 || undefined,
      documents: dom.documents,
      elements: dom.elements,
      accessibility: dom.accessibility,
      fonts,
      diagnostics,
      runtime_issues: issues,
      latency: timings,
    };
  } finally {
    if (context) {
      try { await context.close(); } catch (_) {}
    }
    if (browser) {
      try { await browser.close(); } catch (_) {}
    }
  }
}

async function main() {
  try {
    const raw = await readStdin();
    const req = raw.trim() ? JSON.parse(raw) : {};
    if (req.command === 'probe') {
      process.stdout.write(JSON.stringify(await probe()));
      return;
    }
    if (req.command === 'capture' || req.command === 'scenario') {
      process.stdout.write(JSON.stringify(await capture(req)));
      return;
    }
    throw new Error(`unsupported worker command ${req.command}`);
  } catch (err) {
    let playwrightVersion = '';
    try { playwrightVersion = require('playwright/package.json').version; } catch (_) {}
    process.stdout.write(JSON.stringify({
      success: false,
      error: String(err && err.stack ? err.stack : err),
      worker_version: WORKER_VERSION,
      playwright_version: playwrightVersion,
    }));
  }
}

main();
