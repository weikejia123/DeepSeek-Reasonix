import { memo, useContext, useMemo, useRef } from "react";
import ReactMarkdown from "react-markdown";
import type { Components } from "react-markdown";
import remarkGfm from "remark-gfm";
import remarkMath from "remark-math";
import rehypeKatex from "rehype-katex";
import "katex/dist/katex.min.css";
import { CodeViewer } from "./CodeViewer";
import { normalizeMath } from "./mathNormalize";
import { openExternal } from "../lib/bridge";
import { FileLinkContext, linkifyPaths } from "../lib/pathLinkify";

// Markdown rendering via react-markdown + remark-gfm (tables, task lists,
// strike, autolinks) and remark-math + rehype-katex for $/$$ KaTeX math.
// Fenced code blocks go through CodeViewer for syntax highlighting; inline
// code is a styled <code>. Links open in the system browser.
//
// The math pre-pass in mathNormalize normalises LLM-native \(…\)/\[…\]
// delimiters to the $/$$ syntax remark-math understands, gates single-$
// pairs through a classifier to avoid false positives on $5, $PATH, etc.,
// and runs KaTeX-specific normalisations (text-mode escapes, |→\vert).

function buildComponents(onOpenWorkspaceFile?: (path: string) => void): Components {
  return {
    pre: ({ children }) => <>{children}</>,
    code: ({ className, children }) => {
      const text = String(children ?? "");
      const match = /language-([\w-]+)/.exec(className ?? "");
      const isBlock = match !== null || text.includes("\n");
      if (isBlock) {
        return <CodeViewer value={text.replace(/\n$/, "")} language={match?.[1]} maxHeight={360} />;
      }
      return <code className="md-code">{children}</code>;
    },
    a: ({ href, children }) => {
      // Workspace file path: no protocol prefix, relative to project root.
      if (href && !/^https?:\/\//.test(href)) {
        return (
          <a
            href="#"
            className="md-link--workspace"
            onClick={(e) => {
              e.preventDefault();
              // react-markdown percent-encodes non-ASCII chars in link hrefs;
              // decode them so the path matches what the file tree produces.
              let decoded = href;
              try { decoded = decodeURIComponent(href); } catch { /* use as-is */ }
              onOpenWorkspaceFile?.(decoded);
            }}
          >
            {children}
          </a>
        );
      }
      // External link: open in system browser.
      return (
        <a
          href={href}
          onClick={(e) => {
            e.preventDefault();
            if (href) openExternal(href);
          }}
          onAuxClick={(e) => {
            e.preventDefault();
            if (href) openExternal(href);
          }}
          onMouseDown={(e) => {
            if (e.button === 1) e.preventDefault();
          }}
        >
          {children}
        </a>
      );
    },
  };
}

export const Markdown = memo(function Markdown({
  text,
  streaming = false,
}: {
  text: string;
  streaming?: boolean;
}) {
  const ctx = useContext(FileLinkContext);

  // Linkify file paths only after streaming completes (i.e. once per turn).
  const linkedText = useMemo(() => {
    if (streaming || !ctx?.filePathSet) return text;
    return linkifyPaths(text, ctx.filePathSet);
  }, [text, streaming, ctx?.filePathSet]);

  const components = useMemo(
    () => buildComponents(ctx?.onOpenWorkspaceFile),
    [ctx?.onOpenWorkspaceFile],
  );

  const mathContent = useMemo(() => normalizeMath(linkedText), [linkedText]);
  const containerRef = useRef<HTMLDivElement>(null);
  return (
    <div className="md" ref={containerRef}>
      <ReactMarkdown
        remarkPlugins={[remarkGfm, remarkMath]}
        rehypePlugins={[rehypeKatex]}
        components={components}
      >
        {mathContent}
      </ReactMarkdown>
    </div>
  );
});
