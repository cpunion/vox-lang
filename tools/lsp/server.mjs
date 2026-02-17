#!/usr/bin/env node

import fs from 'node:fs';
import os from 'node:os';
import path from 'node:path';
import { spawnSync } from 'node:child_process';

const docs = new Map();
let shutdownRequested = false;
let workspaceRoot = process.cwd();

function send(msg) {
  const body = JSON.stringify(msg);
  const head = `Content-Length: ${Buffer.byteLength(body, 'utf8')}\r\n\r\n`;
  process.stdout.write(head + body);
}

function respond(id, result) {
  send({ jsonrpc: '2.0', id, result });
}

function respondError(id, code, message) {
  send({ jsonrpc: '2.0', id, error: { code, message } });
}

function notify(method, params) {
  send({ jsonrpc: '2.0', method, params });
}

function uriToFsPath(uri) {
  if (!uri || !uri.startsWith('file://')) return '';
  let p = decodeURIComponent(uri.slice('file://'.length));
  if (process.platform === 'win32' && p.startsWith('/')) p = p.slice(1);
  return p;
}

function fullDocRange(text) {
  const lines = text.split('\n');
  const lastLine = lines.length - 1;
  const lastCol = lines[lastLine].length;
  return {
    start: { line: 0, character: 0 },
    end: { line: lastLine, character: lastCol },
  };
}

function trimLeftWs(s) {
  return s.replace(/^[ \t]+/, '');
}

function trimRightWs(s) {
  return s.replace(/[ \t\r]+$/g, '');
}

function lineHasNonWs(s) {
  return /[^ \t\r]/.test(s);
}

function scanLineState(line, inTriple0) {
  let inTriple = inTriple0;
  let delta = 0;
  let i = 0;
  while (i < line.length) {
    if (inTriple) {
      if (line.slice(i, i + 3) === '"""') {
        inTriple = false;
        i += 3;
        continue;
      }
      i += 1;
      continue;
    }
    if (line.slice(i, i + 2) === '//') break;
    if (line.slice(i, i + 3) === '"""') {
      inTriple = true;
      i += 3;
      continue;
    }
    if (line[i] === '"') {
      i += 1;
      while (i < line.length) {
        if (line[i] === '\\') {
          i += 2;
          continue;
        }
        if (line[i] === '"') {
          i += 1;
          break;
        }
        i += 1;
      }
      continue;
    }
    if (line[i] === '{') delta += 1;
    if (line[i] === '}') delta -= 1;
    i += 1;
  }
  return { delta, inTriple };
}

function formatText(src) {
  if (!src) return src;
  const lines = src.split('\n');
  const out = [];
  let indent = 0;
  let inTriple = false;
  let blankRun = 0;

  for (const rawLine of lines) {
    if (inTriple) {
      out.push(rawLine);
      inTriple = scanLineState(rawLine, true).inTriple;
      continue;
    }

    const r = trimRightWs(rawLine);
    const content = trimLeftWs(r);
    if (!lineHasNonWs(content)) {
      blankRun += 1;
      if (blankRun <= 1) out.push('');
      continue;
    }
    blankRun = 0;

    let lineIndent = indent;
    if (content.startsWith('}') && lineIndent > 0) lineIndent -= 1;
    out.push(`${' '.repeat(lineIndent * 2)}${content}`);

    const st = scanLineState(content, false);
    indent += st.delta;
    if (indent < 0) indent = 0;
    inTriple = st.inTriple;
  }

  // keep one final newline
  let text = out.join('\n');
  text = text.replace(/\n+$/g, '\n');
  if (!text.endsWith('\n')) text += '\n';
  return text;
}

function parseCompilerDiagnostics(output) {
  return parseCompilerDiagnosticsForTarget(output, '');
}

function normalizeSlashes(p) {
  return String(p || '').replace(/\\/g, '/');
}

function isWorkspaceFile(docPath) {
  if (!docPath) return false;
  const rel = normalizeSlashes(path.relative(workspaceRoot, docPath));
  if (!rel || rel === '.') return false;
  if (rel.startsWith('../') || rel === '..') return false;
  return true;
}

function relWorkspacePath(docPath) {
  return normalizeSlashes(path.relative(workspaceRoot, docPath));
}

function isPkgSourceRelPath(rel) {
  if (!rel) return false;
  if (rel.startsWith('src/')) return true;
  if (rel.startsWith('tests/')) return true;
  return false;
}

function parseCompilerDiagnosticsForTarget(output, targetRelPath) {
  const out = [];
  const targetRel = normalizeSlashes(targetRelPath || '');
  const lines = String(output || '').split(/\r?\n/);
  for (const line0 of lines) {
    const line = line0.trim();
    if (!line) continue;
    let work = line;
    if (work.startsWith('compile failed:')) {
      work = work.slice('compile failed:'.length).trim();
    }
    let filePath = '';
    let m = work.match(/^(.+?):(\d+):(\d+):\s*(.+)$/);
    let msg = '';
    let row = 1;
    let col = 1;
    if (m) {
      filePath = normalizeSlashes(m[1]);
      row = Number(m[2]);
      col = Number(m[3]);
      msg = m[4];
      if (targetRel) {
        let rel = filePath;
        if (path.isAbsolute(filePath)) {
          rel = normalizeSlashes(path.relative(workspaceRoot, filePath));
        }
        if (rel !== targetRel) {
          continue;
        }
      }
    } else {
      msg = work || line;
    }
    out.push({
      range: {
        start: { line: Math.max(0, row - 1), character: Math.max(0, col - 1) },
        end: { line: Math.max(0, row - 1), character: Math.max(1, col) },
      },
      severity: 1,
      source: 'vox',
      message: msg,
    });
    if (out.length >= 20) break;
  }
  return out;
}

function runDiagnosticsSingleFile(text) {
  const tmpRoot = fs.mkdtempSync(path.join(os.tmpdir(), 'vox-lsp-'));
  const srcDir = path.join(tmpRoot, 'src');
  const src = path.join(srcDir, 'main.vox');
  const outBin = path.join(tmpRoot, 'main.bin');
  const manifest = path.join(tmpRoot, 'vox.toml');
  fs.mkdirSync(srcDir, { recursive: true });
  fs.writeFileSync(manifest, '[package]\nname = "vox_lsp_tmp"\nversion = "0.0.0"\nedition = "2026"\n', 'utf8');
  fs.writeFileSync(src, text, 'utf8');
  const voxBin = process.env.VOX_BIN || 'vox';
  const cp = spawnSync(voxBin, ['build-pkg', outBin], {
    cwd: tmpRoot,
    encoding: 'utf8',
  });
  try {
    fs.rmSync(tmpRoot, { recursive: true, force: true });
  } catch (_) {
    // ignore cleanup failure
  }
  if (cp.status === 0) return [];
  return parseCompilerDiagnosticsForTarget(`${cp.stdout || ''}\n${cp.stderr || ''}`, 'src/main.vox');
}

function runDiagnosticsWorkspace(docFsPath) {
  if (!isWorkspaceFile(docFsPath)) return null;
  const rel = relWorkspacePath(docFsPath);
  if (!isPkgSourceRelPath(rel)) return null;
  if (!fs.existsSync(path.join(workspaceRoot, 'vox.toml'))) return null;
  if (!fs.existsSync(path.join(workspaceRoot, 'src'))) return null;

  const voxBin = process.env.VOX_BIN || 'vox';
  const outBin = path.join(workspaceRoot, 'target', 'debug', '.vox_lsp_diag');
  const cp = spawnSync(voxBin, ['build-pkg', outBin], {
    cwd: workspaceRoot,
    encoding: 'utf8',
  });
  if (cp.status === 0) return [];
  return parseCompilerDiagnosticsForTarget(`${cp.stdout || ''}\n${cp.stderr || ''}`, rel);
}

function runDiagnostics(uri, text) {
  const docFsPath = uriToFsPath(uri);
  const wsDiagnostics = runDiagnosticsWorkspace(docFsPath);
  if (wsDiagnostics !== null) return wsDiagnostics;
  return runDiagnosticsSingleFile(text);
}

function publishDiagnostics(uri) {
  const d = docs.get(uri);
  if (!d) return;
  const diagnostics = runDiagnostics(uri, d.text);
  notify('textDocument/publishDiagnostics', { uri, diagnostics });
}

function handle(msg) {
  const method = msg.method || '';
  if (method === 'initialize') {
    const params = msg.params || {};
    const rootUri = params.rootUri || '';
    const rootPath = params.rootPath || '';
    if (rootUri) {
      const p = uriToFsPath(rootUri);
      if (p) workspaceRoot = p;
    } else if (rootPath) {
      workspaceRoot = rootPath;
    }
    respond(msg.id, {
      capabilities: {
        textDocumentSync: 1,
        documentFormattingProvider: true,
      },
      serverInfo: { name: 'vox-lsp', version: '0.1.0' },
    });
    return;
  }
  if (method === 'initialized') return;
  if (method === 'shutdown') {
    shutdownRequested = true;
    respond(msg.id, null);
    return;
  }
  if (method === 'exit') {
    process.exit(shutdownRequested ? 0 : 1);
  }
  if (method === 'textDocument/didOpen') {
    const p = msg.params || {};
    const td = p.textDocument || {};
    docs.set(td.uri, { text: td.text || '', version: td.version || 1 });
    publishDiagnostics(td.uri);
    return;
  }
  if (method === 'textDocument/didChange') {
    const p = msg.params || {};
    const td = p.textDocument || {};
    const cs = p.contentChanges || [];
    const old = docs.get(td.uri) || { text: '', version: 0 };
    if (cs.length > 0 && typeof cs[cs.length - 1].text === 'string') {
      old.text = cs[cs.length - 1].text;
    }
    old.version = td.version || old.version + 1;
    docs.set(td.uri, old);
    publishDiagnostics(td.uri);
    return;
  }
  if (method === 'textDocument/didClose') {
    const p = msg.params || {};
    const td = p.textDocument || {};
    docs.delete(td.uri);
    notify('textDocument/publishDiagnostics', { uri: td.uri, diagnostics: [] });
    return;
  }
  if (method === 'textDocument/formatting') {
    const p = msg.params || {};
    const td = p.textDocument || {};
    const d = docs.get(td.uri);
    if (!d) {
      respond(msg.id, []);
      return;
    }
    const formatted = formatText(d.text || '');
    if (formatted === d.text) {
      respond(msg.id, []);
      return;
    }
    respond(msg.id, [{ range: fullDocRange(d.text), newText: formatted }]);
    return;
  }
  if (typeof msg.id !== 'undefined') {
    respondError(msg.id, -32601, `method not found: ${method}`);
  }
}

let buf = Buffer.alloc(0);
let expected = -1;

function pump() {
  while (true) {
    if (expected < 0) {
      const idx = buf.indexOf('\r\n\r\n');
      if (idx < 0) return;
      const head = buf.slice(0, idx).toString('utf8');
      buf = buf.slice(idx + 4);
      let len = -1;
      for (const raw of head.split('\r\n')) {
        const [k, v] = raw.split(':');
        if (!k || typeof v === 'undefined') continue;
        if (k.toLowerCase() === 'content-length') {
          len = Number(v.trim());
        }
      }
      if (!(len >= 0)) return;
      expected = len;
    }
    if (buf.length < expected) return;
    const body = buf.slice(0, expected).toString('utf8');
    buf = buf.slice(expected);
    expected = -1;
    let msg;
    try {
      msg = JSON.parse(body);
    } catch (_) {
      continue;
    }
    handle(msg);
  }
}

process.stdin.on('data', (chunk) => {
  buf = Buffer.concat([buf, chunk]);
  pump();
});

process.stdin.on('end', () => {
  process.exit(0);
});
