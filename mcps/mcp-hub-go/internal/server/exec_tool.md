Execute JavaScript code with access to MCP tools.

Write JavaScript code to call multiple MCP tools in a single request. Use loops, conditionals, and data transformation to efficiently batch operations.

## IMPORTANT: Batch Multiple Tool Calls

**ALWAYS prefer ONE exec call with multiple mcp.callTool() inside over multiple separate exec calls.**

Benefits of batching in a single exec:

- Single round-trip instead of multiple network requests
- Coordinated error handling for all operations
- Easy data aggregation and transformation across results
- Reduced latency and rate limiting impact

## Available APIs

### mcp.callTool(serverID, toolName, params)

Call any MCP tool. Results are auto-parsed from JSON.

- serverID: Server ID from search results
- toolName: Tool name to invoke
- params: Parameters object
- Returns: Parsed JSON object (or string if not JSON)
- Throws: Exception on failure (use try/catch)

### mcp.log(level, message, fields?)

Log messages (captured in response).

- level: "debug" | "info" | "warn" | "error"
- message: Log string
- fields: Optional object

## Patterns for Efficient Tool Calling

### Pattern 1: Batch Operations in a Loop

Call multiple tools or the same tool with different params in one exec:

```javascript
const operations = [
  { server: "db", tool: "getUser", params: { id: 1 } },
  { server: "db", tool: "getUser", params: { id: 2 } },
  { server: "api", tool: "getStatus", params: {} },
];
const results = [];
for (const op of operations) {
  try {
    results.push({
      name: op.tool,
      success: true,
      data: mcp.callTool(op.server, op.tool, op.params),
    });
  } catch (e) {
    results.push({ name: op.tool, success: false, error: e.message });
  }
}
results;
```

### Pattern 2: Collect → Transform → Aggregate

Gather data from multiple sources, then process together:

```javascript
// 1. Collect from multiple sources
const users = mcp.callTool("db", "listUsers", { limit: 100 });
const orders = mcp.callTool("db", "listOrders", { status: "pending" });
const metrics = mcp.callTool("analytics", "getSummary", {});

// 2. Transform and aggregate
const activeUsers = users.filter(u => u.active);
const highValueOrders = orders.filter(o => o.total > 1000);

// 3. Return combined insights
{
  userCount: activeUsers.length,
  pendingHighValue: highValueOrders.length,
  totalRevenue: metrics.revenue,
  summary: `${activeUsers.length} active users, ${highValueOrders.length} high-value orders`
};
```

### Pattern 3: Conditional Chaining

Make subsequent calls based on previous results:

```javascript
const user = mcp.callTool("db", "getUser", { id: 123 });
const result = { user };

if (user.isPremium) {
  result.features = mcp.callTool("billing", "getFeatures", { tier: user.tier });
  result.usage = mcp.callTool("analytics", "getUsage", { userId: user.id });
}

if (user.teamId) {
  result.team = mcp.callTool("db", "getTeam", { id: user.teamId });
}

result;
```

### Pattern 4: Batch Processing with Error Isolation

Process many items, collecting successes and failures separately:

```javascript
const ids = [1, 2, 3, 4, 5, 6, 7, 8, 9, 10];
const results = { success: [], failed: [] };

for (const id of ids) {
  try {
    const item = mcp.callTool("api", "fetchItem", { id });
    results.success.push({ id, data: item });
  } catch (e) {
    results.failed.push({ id, error: e.message });
  }
}

{
  processed: ids.length,
  succeeded: results.success.length,
  failed: results.failed.length,
  results
};
```

### Pattern 5: Multi-Step Workflow

Execute a complete workflow in one call:

```javascript
const workflow = { steps: [], startTime: new Date().toISOString() };

// Step 1: Search for relevant tools
try {
  const tools = mcp.callTool("builtin", "search", { query: "file" });
  workflow.steps.push({ name: "search", success: true, count: tools.length });
} catch (e) {
  workflow.steps.push({ name: "search", success: false, error: e.message });
}

// Step 2: Get file list
try {
  const files = mcp.callTool("fs", "listFiles", { path: "/src" });
  workflow.steps.push({
    name: "listFiles",
    success: true,
    count: files.length,
  });
} catch (e) {
  workflow.steps.push({ name: "listFiles", success: false, error: e.message });
}

// Step 3: Analyze results
workflow.endTime = new Date().toISOString();
workflow.summary = {
  totalSteps: workflow.steps.length,
  successful: workflow.steps.filter(s => s.success).length,
};
workflow;
```

## Return Values

The script result is the **last expression evaluated**:

```javascript
const data = mcp.callTool("api", "fetchAll", {});
data.filter(x => x.active).map(x => ({ id: x.id, name: x.name })); // This is returned
```

## Runtime Environment

**goja** (Go JavaScript engine): ES5.1 with select ES6 features.

Supported: const, let, arrow functions, template literals, spread operator, destructuring, for...of
Not available: async/await, Promise, setTimeout, console (use mcp.log), fetch, require/import

## Constraints

- Timeout: 15 seconds
- Max script: 100KB
- Synchronous only
- No eval() or Function constructor

## Output Format

- result: Your script's return value (last expression)
- logs: Array of mcp.log() entries
- error: Error message if execution failed
