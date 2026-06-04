import { describe, it, expect } from 'vitest';
import { parseToolInput, getToolHint, formatContent } from './format';

describe('parseToolInput', () => {
  it('parses valid JSON', () => {
    const result = parseToolInput('{"path":"main.go"}');
    expect(result.path).toBe('main.go');
  });

  it('returns empty object for invalid JSON', () => {
    const result = parseToolInput('not json');
    expect(result).toEqual({});
  });

  it('returns empty object for undefined input', () => {
    const result = parseToolInput(undefined);
    expect(result).toEqual({});
  });
});

describe('getToolHint', () => {
  it('shows path for read_file', () => {
    expect(getToolHint('read_file', '{"path":"src/main.go"}')).toBe('View: src/main.go');
  });

  it('shows command for execute_command', () => {
    expect(getToolHint('execute_command', '{"command":"go test"}')).toBe('Run: go test');
  });

  it('falls back to tool name when input empty', () => {
    expect(getToolHint('read_file', undefined)).toBe('read_file');
  });

  it('returns question for ask_followup_question', () => {
    expect(getToolHint('ask_followup_question', '{"question":"Proceed?"}')).toBe('💬 Proceed?');
  });
});

describe('formatContent', () => {
  it('renders markdown headings', () => {
    const html = formatContent('# Hello');
    expect(html).toContain('<h1');
    expect(html).toContain('Hello');
  });

  it('renders inline code with style', () => {
    const html = formatContent('use `console.log` here');
    expect(html).toContain('<code style=');
  });

  it('renders code blocks with copy button', () => {
    const html = formatContent('```go\npackage main\n```');
    expect(html).toContain('hljs-copy-btn');
    expect(html).toContain('package main');
  });
});
