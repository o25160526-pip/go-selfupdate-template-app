#!/usr/bin/env node
// scripts/check-diff-scope.mjs
//
// Kiểm tra các file thay đổi so với phạm vi cho phép của nghiệp vụ (docs/nghiepvu/<slug>.md).
// Guardrail của workflow opencode-issue-agent.yml: agent chỉ được sửa file nằm trong phạm vi.
//
// Cách dùng:
//   node scripts/check-diff-scope.mjs <slug> <output-file>
//   - <slug>       tên file nghiệp vụ, vd: tu-cap-nhat
//   - <output-file> file để ghi danh sách file NGOÀI phạm vi (1 dòng/path), stdout cũng in ra
//
// Quy tắc phạm vi (đọc từ docs/nghiepvu/<slug>.md):
//   - dòng bắt đầu bằng "ALLOWED:" chứa các glob (cách nhau bởi khoảng trắng),
//     vd:  ALLOWED: internal/updater/** internal/version/** docs/nghiepvu/tu-cap-nhat.md
//   - docs/nghiepvu/** luôn được phép (agent cần cập nhật tài liệu nghiệp vụ)
//   - không có dòng ALLOWED => mọi thay đổi bị xem là ngoài phạm vi (an toàn hơn, không đoán)
//
// Exit code:
//   0  luôn (gọi bên ngoài để quyết định; file output ghi danh sách ngoài phạm vi)

import { readFileSync, writeFileSync } from "node:fs";
import { execSync } from "node:child_process";

function globToRegExp(pattern) {
  let re = "^";
  for (let i = 0; i < pattern.length; i++) {
    const ch = pattern[i];
    if (ch === "*") {
      if (pattern[i + 1] === "*") {
        re += ".*";
        i++;
      } else {
        re += "[^/]*";
      }
    } else if (ch === "?") {
      re += "[^/]";
    } else {
      re += ch.replace(/[.+^${}()|[\]\\]/g, "\\$&");
    }
  }
  return new RegExp(re + "$");
}

function allowedGlobs(slug) {
  const doc = readFileSync(`docs/nghiepvu/${slug}.md`, "utf8");
  const globs = [];
  for (const line of doc.split(/\r?\n/)) {
    const m = line.match(/^ALLOWED\s*:\s*(.+)$/i);
    if (m) globs.push(...m[1].trim().split(/\s+/).filter(Boolean));
  }
  globs.push("docs/nghiepvu/**");
  return globs;
}

function changedPaths() {
  const out = execSync("git status --porcelain -uall", { encoding: "utf8" });
  const paths = new Set();
  for (const raw of out.split(/\r?\n/)) {
    if (!raw.trim()) continue;
    let entry = raw;
    if (entry.startsWith("!!")) {
      paths.add(entry.slice(3).trim());
      continue;
    }
    const rest = entry.slice(3).trim();
    if (entry.startsWith("R") || entry.startsWith("C")) {
      const [a] = rest.split(" -> ");
      if (a) paths.add(a);
      continue;
    }
    paths.add(rest);
  }
  return [...paths];
}

function isAllowed(path, regexps) {
  return regexps.some((r) => r.test(path));
}

const slug = process.argv[2];
const outFile = process.argv[3];
if (!slug) {
  process.stderr.write("usage: node scripts/check-diff-scope.mjs <slug> <output-file>\n");
  process.exit(2);
}

const regexps = allowedGlobs(slug).map(globToRegExp);
const changed = changedPaths();
const outOfScope = changed.filter((p) => !isAllowed(p, regexps));

const content = outOfScope.join("\n") + (outOfScope.length ? "\n" : "");
if (outFile) writeFileSync(outFile, content, "utf8");
if (outOfScope.length) {
  process.stderr.write(
    `check-diff-scope: ${outOfScope.length}/${changed.length} file ngoài phạm vi nghiệp vụ "${slug}":\n`,
  );
  for (const p of outOfScope) process.stderr.write(`  - ${p}\n`);
}
process.stdout.write(content);
