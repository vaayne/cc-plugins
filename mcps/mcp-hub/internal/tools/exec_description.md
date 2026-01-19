Execute JavaScript code with access to MCP tools.

Write JavaScript code to call multiple MCP tools in a single request. Use loops, conditionals, and data transformation to efficiently batch operations.

## Quick guide

- **Discovery**: Use the `list` tool first to see available tools as JavaScript function stubs with JSDoc.
- **Batching**: Prefer one exec call with multiple `mcp.callTool()` invocations instead of many exec calls.
- **Result**: The last expression is returned. Don't use `return` at top level; use an async IIFE if you need `await`.
- **Async**: Async/await, Promises, and timers (`setTimeout`, `setInterval`, `setImmediate`) are supported.
- **Require**: `require()` works for core goja_nodejs modules like `node:buffer`, `node:process`, `node:url`, `node:util`, and the built-in `console`.
- **MCP helpers**:
  - `mcp.callTool("toolName", params)` → calls any MCP tool; throws on failure. Use `serverID__toolName` format for multi-server config mode.
  - `mcp.log(level, message, fields?)` or `console.*` → captured in `logs`.
- **No browser APIs**: `window`, `document`, `page`, `fetch`, etc. are not provided; get data via MCP tools.

## Minimal patterns

- Async IIFE:
  ```javascript
  (async () => {
    const users = await mcp.callTool("db__listUsers", { limit: 50 });
    return users.filter(u => u.active);
  })();
  ```
- Batch with error capture:
  ```javascript
  const ids = [1, 2, 3];
  ids.map(id => {
    try {
      return { id, ok: true, data: mcp.callTool("db__getUser", { id }) };
    } catch (e) {
      return { id, ok: false, error: e.message };
    }
  });
  ```
- Require example:
  ```javascript
  const { Buffer } = require("node:buffer");
  Buffer.from("hi").toString("hex");
  ```

## Constraints

- Timeout: 60s per exec call
- Max script size: 100KB
- Logs capped at 1000 entries

## Output

- `result`: last expression value
- `logs`: array of `console`/`mcp.log` entries
- `error`: populated if execution fails
