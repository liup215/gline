import hljs from 'highlight.js';
import 'highlight.js/styles/github-dark.css';
import { marked } from 'marked';
import katex from 'katex';
import 'katex/dist/katex.min.css';
import { THEME } from '../theme';

export function useHighlightCode() {
  requestAnimationFrame(() => {
    document.querySelectorAll('pre code.hljs').forEach((block) => {
      try {
        hljs.highlightElement(block as HTMLElement);
      } catch (e) {
      }
    });
  });
}

interface ToolInput {
  path?: string;
  command?: string;
  question?: string;
  options?: string[];
  content?: string;
  search?: string;
  replace?: string;
  regex?: string;
  file_pattern?: string;
  [key: string]: any;
}

export function parseToolInput(raw: string | undefined): ToolInput {
  if (!raw) return {};
  try {
    return JSON.parse(raw);
  } catch {
    return {};
  }
}

export function getToolHint(name: string, rawInput: string | undefined): string {
  const input = parseToolInput(rawInput);
  switch (name) {
    case 'read_file':
      return input.path ? `View: ${input.path}` : name;
    case 'write_to_file':
      return input.path ? `Write: ${input.path}` : name;
    case 'replace_in_file':
      return input.path ? `Edit: ${input.path}` : name;
    case 'list_files':
      return input.path ? `List: ${input.path}` : name;
    case 'search_files':
      return input.regex ? `Search "${input.regex}"${input.path ? ` in ${input.path}` : ''}` : name;
    case 'list_code_definition_names':
      return input.path ? `Definitions in ${input.path}` : name;
    case 'execute_command':
      return input.command ? `Run: ${input.command}` : name;
    case 'ask_followup_question':
      return input.question ? `💬 ${input.question}` : name;
    case 'attempt_completion':
      return 'Complete';
    default:
      return name;
  }
}

function detectLanguage(code: string): string | null {
  const trimmed = code.trim().toLowerCase();
  if (trimmed.startsWith('<!doctype html') || trimmed.startsWith('<html')) return 'xml';
  if (/^(import|export|const|let|var|function|class|interface|type)\b/.test(trimmed)) return 'typescript';
  if (/^(package|import|func|type|struct|interface|var|const)\b/.test(trimmed)) return 'go';
  if (/^(def|class|import|from|print|if __name__)/.test(trimmed)) return 'python';
  if (trimmed.includes('#include') || trimmed.includes('int main(')) return 'cpp';
  if (trimmed.startsWith('{') || trimmed.includes('"') && trimmed.includes(':')) return 'json';
  if (trimmed.includes('dockerfile') || /^(from|run|cmd|entrypoint|copy|add)\b/.test(trimmed)) return 'dockerfile';
  if (/^(select|insert|update|delete|create table|drop table|alter table)\b/.test(trimmed)) return 'sql';
  return null;
}

marked.setOptions({
  gfm: true,
  breaks: true,
} as any);

const MATH_PLACEHOLDER_BLOCK = '§§§BLOCKMATH';
const MATH_PLACEHOLDER_INLINE = '§§§INLINEMATH';

function _renderKatex(tex: string, displayMode: boolean): string {
  try {
    return katex.renderToString(tex, {
      displayMode,
      throwOnError: false,
      strict: false,
    });
  } catch (e) {
    return `<span style="color:#ef4444;font-family:monospace;">${displayMode ? '$$' : '$'}${tex}${displayMode ? '$$' : '$'}</span>`;
  }
}

function _processFootnotes(content: string): string {
  const defs: Record<string, string> = {};
  let text = content.replace(/\n\[\^(\d+|[a-zA-Z-]+)]:[ \t]*(.*(?:\n(?!\[a-zA-Z0-9]).*)*)/g, (_match, label, noteText) => {
    defs[label] = noteText.trim().replace(/\n[ \t]+/g, ' ');
    return '\n';
  });

  text = text.replace(/\[\^(\d+|[a-zA-Z-]+)\]/g, (_match, label) => {
    if (defs[label]) {
      return `<sup class="md-footnote-ref" data-fn="${label}" title="${escapeHtml(defs[label])}">${label}</sup>`;
    }
    return `<sup>[${label}]</sup>`;
  });

  const usedLabels = Object.keys(defs).filter(l => text.includes(`data-fn="${l}"`));
  if (usedLabels.length > 0) {
    let footnotesHtml = '\n\n<div class="md-footnotes"><hr style="border:none;border-top:1px solid #1e293b;margin:16px 1px 8px 0;"/><h4 style="font-size:0.9rem;color:#94a3b8;margin:0 0 8px;">Footnotes</h4><ol style="padding-left:1.2em;margin:0;font-size:0.85rem;color:#94a3b8;">';
    usedLabels.forEach(label => {
      footnotesHtml += `<li id="fn-${label}"><span style="color:#cbd5e1;">${escapeHtml(defs[label])}</span></li>`;
    });
    footnotesHtml += '</ol></div>';
    text = text + footnotesHtml;
  }

  return text;
}

export function formatContent(content: string): string {
  const blockMath: string[] = [];
  const inlineMath: string[] = [];
  let text = content;

  function extractBlockDelim(start: string, end: string): void {
    let idx = text.indexOf(start);
    while (idx !== -1) {
      const endIdx = text.indexOf(end, idx + start.length);
      if (endIdx === -1) break;
      const tex = text.slice(idx + start.length, endIdx).trim();
      blockMath.push(tex);
      const placeholder = `\n${MATH_PLACEHOLDER_BLOCK}${blockMath.length - 1}\n`;
      text = text.slice(0, idx) + placeholder + text.slice(endIdx + end.length);
      idx = text.indexOf(start);
    }
  }
  extractBlockDelim('\\[', '\\]');
  extractBlockDelim('$$', '$$');

  function extractInlineDelim(start: string, end: string): void {
    const isDollar = start === '$';
    let searchFrom = 0;
    while (true) {
      let idx = text.indexOf(start, searchFrom);
      if (idx === -1) break;
      if (isDollar) {
        if (text[idx + 1] === '$') { searchFrom = idx + 2; continue; }
        if ((text[idx - 1] || '') === '$') { searchFrom = idx + 1; continue; }
        if (text.slice(idx, idx + MATH_PLACEHOLDER_BLOCK.length) === MATH_PLACEHOLDER_BLOCK) { searchFrom = idx + 1; continue; }
      }
      const endIdx = text.indexOf(end, idx + start.length);
      if (endIdx === -1) break;
      if (isDollar && endIdx - idx < 2) { searchFrom = idx + 1; continue; }
      const tex = text.slice(idx + start.length, endIdx).trim();
      if (isDollar && /^\d/.test(tex) && !/[\\=<>^_&{}]/.test(tex)) {
        searchFrom = endIdx + 1; continue;
      }
      inlineMath.push(tex);
      const placeholder = `${MATH_PLACEHOLDER_INLINE}${inlineMath.length - 1}`;
      text = text.slice(0, idx) + placeholder + text.slice(endIdx + end.length);
      searchFrom = idx;
    }
  }
  extractInlineDelim('\\(', '\\)');
  extractInlineDelim('$', '$');

  text = _processFootnotes(text);

  let html = marked.parse(text, { async: false }) as string;

  html = html.replace(/<pre><code class="language-([^"]*)">([\s\S]*?)<\/code><\/pre>/g, (match, lang, code) => {
    const cleanLang = detectLanguage((lang || '').toLowerCase()) || lang || 'plaintext';
    const tempDiv = document.createElement('div');
    tempDiv.innerHTML = code;
    const rawCode = tempDiv.textContent || '';
    const b64 = btoa(unescape(encodeURIComponent(rawCode)));
    return `<pre style="position:relative;background:${THEME.codeBg};padding:14px;border-radius:8px;overflow-x:auto;margin:10px 0;border:1px solid ${THEME.border};">
<span class="hljs-lang-label" style="position:absolute;top:0;left:0;padding:2px 8px;border-radius:8px 0 6px 0;background:rgba(255,255,255,0.06);color:#94a3b8;font-size:0.7rem;font-family:monospace;text-transform:uppercase;">${cleanLang}</span>
<button class="hljs-copy-btn" onclick="window.__copyCode(this)" data-clipboard="${b64}" title="Copy code" style="position:absolute;top:8px;right:8px;padding:4px 10px;border-radius:6px;border:none;background:rgba(255,255,255,0.08);color:#94a3b8;font-size:0.75rem;cursor:pointer;z-index:10;">📋 Copy</button>
<code class="hljs language-${cleanLang}">${code}</code>
</pre>`;
  });

  html = html.replace(/<code>/g, `<code style="background:#2d2d3a;padding:2px 6px;border-radius:4px;font-size:0.9em;font-family:monospace;color:#cdd6f4;">`);

  html = html.replace(/<ul>/g, `<ul style="padding-left:1.5em;margin:8px 0;color:#cbd5e1;list-style-type:disc;">`);
  html = html.replace(/<ol>/g, `<ol style="padding-left:1.5em;margin:8px 0;color:#cbd5e1;list-style-type:decimal;">`);
  html = html.replace(/<blockquote>/g, `<blockquote style="border-left:3px solid #3b82f6;margin:8px 0;padding-left:14px;color:#94a3b8;font-style:italic;">`);
  html = html.replace(/<hr\s*\/?>/g, `<hr style="border:none;border-top:1px solid #1e293b;margin:12px 0;"/>`);

  html = html.replace(/<table>/g, `<table style="width:100%;border-collapse:collapse;margin:10px 0;font-size:0.9em;">`);
  html = html.replace(/<thead>/g, `<thead style="background:#1e293b;">`);
  html = html.replace(/<th>/g, `<th style="text-align:left;padding:8px 12px;border-bottom:1px solid #334155;color:#e2e8f0;font-weight:600;">`);
  html = html.replace(/<td>/g, `<td style="padding:8px 12px;border-bottom:1px solid #1e293b;color:#cbd5e1;">`);

  html = html.replace(/<a /g, `<a style="color:#60a5fa;text-decoration:underline;" `);
  html = html.replace(/<code style="background:#2d2d3a;padding:2px 6px;border-radius:4px;font-size:0.9em;font-family:monospace;color:#cdd6f4;">\s*\n/g, '<code>\n');

  html = html.replace(new RegExp(`<p>${MATH_PLACEHOLDER_BLOCK}(\\d+)<\\/p>`, 'g'), (_match, idx) => {
    return _renderKatex(blockMath[parseInt(idx)], true);
  });
  html = html.replace(new RegExp(`${MATH_PLACEHOLDER_BLOCK}(\\d+)`, 'g'), (_match, idx) => {
    return _renderKatex(blockMath[parseInt(idx)], true);
  });
  html = html.replace(new RegExp(`${MATH_PLACEHOLDER_INLINE}(\\d+)`, 'g'), (_match, idx) => {
    return _renderKatex(inlineMath[parseInt(idx)], false);
  });

  return html;
}

export function escapeHtml(str: string): string {
  return str
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;');
}

if (typeof window !== 'undefined' && !(window as any).__copyCode) {
  (window as any).__copyCode = async function (btn: HTMLButtonElement) {
    try {
      const b64 = btn.getAttribute('data-clipboard');
      if (!b64) return;
      const text = decodeURIComponent(escape(atob(b64)));
      await navigator.clipboard.writeText(text);
      btn.textContent = '✓ Copied!';
      btn.classList.add('copied');
      setTimeout(() => {
        btn.textContent = '📋 Copy';
        btn.classList.remove('copied');
      }, 2000);
    } catch (err) {
      console.error('Copy failed:', err);
    }
  };
}
