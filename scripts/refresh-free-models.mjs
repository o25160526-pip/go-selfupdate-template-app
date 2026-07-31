#!/usr/bin/env node
// scripts/refresh-free-models.mjs
//
// Lấy danh sách model hiện tại của OpenCode Zen và in ra các model MIỄN PHÍ.
// Chỉ dùng fetch built-in của Node (>=18), không phụ thuộc gì thêm.
//
// Output (stdout): JSON
//   {"fetched_at": ISO, "source": URL, "free": ["id", ...]}
//   `free` rỗng = hiện không có model free nào -> hệ thống dừng, cần con người quyết.
//
// Nguồn sự thật: https://opencode.ai/zen/v1/models
// API hiện tại trả danh sách model không kèm cost/status, nên filter theo:
//   1) field cost = {input:0, output:0} / free flag (khi API đổi sang trả đủ metadata)
//   2) heuristic tên: id kết thúc bằng "-free" hoặc id == "big-pickle" (khớp danh sách
//      free được công bố chính thức tại https://opencode.ai/docs/zen).

const MODELS_URL = "https://opencode.ai/zen/v1/models";

function isFreeById(id) {
  return id === "big-pickle" || /-free$/.test(id);
}

function isExplicitlyFree(m) {
  if (typeof m.free === "boolean" && m.free) return true;
  if (typeof m.isFree === "boolean" && m.isFree) return true;
  if (m && typeof m.cost === "object" && m.cost !== null) {
    const input = Number(m.cost.input);
    const output = Number(m.cost.output);
    if (Number.isFinite(input) && Number.isFinite(output)) {
      return input === 0 && output === 0;
    }
  }
  return false;
}

async function main() {
  const res = await fetch(MODELS_URL, {
    headers: { accept: "application/json", "user-agent": "opencode-free-models-refresh" },
  });
  if (!res.ok) {
    throw new Error(`GET ${MODELS_URL} -> HTTP ${res.status}`);
  }
  const payload = await res.json();
  const list = Array.isArray(payload) ? payload : payload && payload.data;
  if (!Array.isArray(list)) {
    throw new Error("unexpected response shape from /zen/v1/models");
  }
  const free = list
    .filter((m) => m && typeof m.id === "string")
    .filter((m) => {
      if (m.status === "deprecated") return false;
      if (isExplicitlyFree(m)) return true;
      return isFreeById(m.id);
    })
    .map((m) => m.id)
    .sort();

  const out = {
    fetched_at: new Date().toISOString(),
    source: MODELS_URL,
    free,
  };
  process.stdout.write(JSON.stringify(out, null, 2) + "\n");
}

main().catch((err) => {
  process.stderr.write(`refresh-free-models: ${err.message}\n`);
  process.exit(1);
});
