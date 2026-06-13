// Post-process completed assistant message text to link workspace file paths.
// Strategy: build a set of known file paths from the workspace, then scan the
// text for tokens that match. Only turn-completed text is processed — we
// never touch streaming content.
//
// Exports FileLinkContext so Markdown can consume file-path data and
// click-handling without prop drilling through Transcript/WarmZone/etc.

import { createContext } from "react";

// ── React Context — avoids prop drilling through 6+ component layers ──────

export interface FileLinkAPI {
  filePathSet: Set<string>;
  onOpenWorkspaceFile: (path: string) => void;
}

export const FileLinkContext = createContext<FileLinkAPI | null>(null);

// ── File set builder ───────────────────────────────────────────────────────

export function buildFileSetFromList(files: string[]): Set<string> {
  return new Set(files);
}

// ── Basename index for ambiguity detection ─────────────────────────────────

function buildBasenameIndex(
  fileSet: Set<string>,
): Map<string, string | null> {
  const index = new Map<string, string | null>();
  for (const path of fileSet) {
    const base = path.split("/").pop()!;
    const existing = index.get(base);
    if (existing === undefined) {
      index.set(base, path);
    } else if (existing !== null) {
      index.set(base, null); // ambiguous — multiple files share this basename
    }
  }
  return index;
}

// ── Extension pre-filter ───────────────────────────────────────────────────

// Common file extensions used as a baseline pre-filter so small or new projects
// still benefit from linkification without needing a large workspace scan.
const BASE_EXTS = new Set([
  ".go", ".ts", ".tsx", ".js", ".jsx", ".md", ".json", ".yaml", ".yml",
  ".toml", ".css", ".scss", ".html", ".svg", ".sql", ".proto", ".rs",
  ".py", ".c", ".h", ".cpp", ".hpp", ".java", ".kt", ".swift", ".sh",
  ".bash", ".zsh", ".env", ".cfg", ".ini", ".xml", ".txt", ".csv",
]);

function buildExtSet(fileSet: Set<string>): Set<string> {
  const exts = new Set(BASE_EXTS);
  for (const path of fileSet) {
    const dot = path.lastIndexOf(".");
    if (dot >= 0) exts.add(path.slice(dot).toLowerCase());
  }
  return exts;
}

// ── Linkify ────────────────────────────────────────────────────────────────

const FENCE_RE = /^(```|~~~)/;

// linkifyPaths scans markdown text for known workspace file paths and wraps
// them in Markdown link syntax [path](path). Fenced code blocks are skipped.
export function linkifyPaths(markdown: string, fileSet: Set<string>): string {
  if (!fileSet || fileSet.size === 0) return markdown;

  const basenameIndex = buildBasenameIndex(fileSet);
  const extSet = buildExtSet(fileSet);

  const lines = markdown.split("\n");
  const result: string[] = [];
  let inFence = false;
  let fenceChar = "";

  for (const line of lines) {
    const m = FENCE_RE.exec(line);
    if (m) {
      if (!inFence) {
        inFence = true;
        fenceChar = m[1];
      } else if (m[1] === fenceChar) {
        inFence = false;
        fenceChar = "";
      }
      result.push(line);
      continue;
    }
    if (inFence) {
      result.push(line);
      continue;
    }
    result.push(linkifyLine(line, fileSet, basenameIndex, extSet));
  }
  return result.join("\n");
}

function linkifyLine(
  line: string,
  fileSet: Set<string>,
  basenameIndex: Map<string, string | null>,
  extSet: Set<string>,
): string {
  const tokens = tokenize(line);
  const out: string[] = [];

  for (const token of tokens) {
    if (token.kind !== "word") {
      out.push(token.text);
      continue;
    }

    const word = token.text;

    // Try the bare word as-is first.
    let link = tryMatch(word, word, fileSet, basenameIndex, extSet);
    if (link === null) {
      // Try stripping wrappers in sequence: backticks, surrounding punct, style.
      const stripped = [
        word.replace(/^`+|`+$/g, ""),       // backticks
        stripSurroundingPunct(word),         // Chinese/English punct
        stripStyleWrappers(word),            // bold/italic
      ];
      for (const candidate of stripped) {
        if (candidate === "" || candidate === word) continue;
        link = tryMatch(word, candidate, fileSet, basenameIndex, extSet);
        if (link !== null) break;
      }
    }

    out.push(link ?? word);
  }

  return out.join("");
}

// Strip bold (** or __) and italic (* or _) Markdown wrappers.
function stripStyleWrappers(word: string): string {
  if (word.startsWith("**") && word.endsWith("**")) return word.slice(2, -2);
  if (word.startsWith("__") && word.endsWith("__")) return word.slice(2, -2);
  if (word.startsWith("*") && word.endsWith("*")) return word.slice(1, -1);
  if (word.startsWith("_") && word.endsWith("_")) return word.slice(1, -1);
  return word;
}

// tryMatch attempts to create a Markdown link [word](target) for a candidate
// path. Returns the link string on success, or null if no match.
function tryMatch(
  word: string,
  candidate: string,
  fileSet: Set<string>,
  basenameIndex: Map<string, string | null>,
  extSet: Set<string>,
): string | null {
  // Fast rejection: must contain "/" or end with a workspace-known extension.
  const hasSlash = candidate.includes("/");
  const ext = lastExt(candidate);
  if (!hasSlash && !extSet.has(ext)) return null;

  // Exact match.
  if (fileSet.has(candidate)) {
    return `[${word}](${candidate})`;
  }

  // Basename match (unique basenames only).
  const base = candidate.split("/").pop()!;
  const unambiguous = basenameIndex.get(base);
  if (unambiguous && unambiguous !== null) {
    if (candidate === base) {
      return `[${word}](${unambiguous})`;
    }
  }

  return null;
}

// Remove non-path characters from the start and end of a string,
// leaving only [a-zA-Z0-9._/-].
function stripSurroundingPunct(s: string): string {
  const VALID_PATH_CHARS = /[a-zA-Z0-9._\-\/]/;
  let start = 0;
  while (start < s.length && !VALID_PATH_CHARS.test(s[start])) start++;
  let end = s.length;
  while (end > start && !VALID_PATH_CHARS.test(s[end - 1])) end--;
  return s.slice(start, end);
}

// ── Tokenizer ──────────────────────────────────────────────────────────────

type Token = { kind: "word" | "space"; text: string };

function tokenize(line: string): Token[] {
  const tokens: Token[] = [];
  let i = 0;
  while (i < line.length) {
    // Whitespace run.
    if (/\s/.test(line[i])) {
      let j = i;
      while (j < line.length && /\s/.test(line[j])) j++;
      tokens.push({ kind: "space", text: line.slice(i, j) });
      i = j;
      continue;
    }
    // Word-like run (non-whitespace).
    let j = i;
    while (j < line.length && !/\s/.test(line[j])) j++;
    tokens.push({ kind: "word", text: line.slice(i, j) });
    i = j;
  }
  return tokens;
}

function lastExt(word: string): string {
  const dot = word.lastIndexOf(".");
  if (dot < 0) return "";
  return word.slice(dot).toLowerCase();
}
