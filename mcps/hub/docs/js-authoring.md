# JavaScript Authoring Guide

This guide explains how to write JavaScript code for the `execute` built-in tool in MCP Hub Go.

## Overview

The `execute` tool runs JavaScript code using the [Goja](https://github.com/dop251/goja) runtime with **strict sync-only enforcement**. This means:

- ❌ No async/await
- ❌ No Promises
- ❌ No setTimeout/setInterval
- ❌ No asynchronous constructs
- ✅ Only synchronous JavaScript

This restriction exists for security, resource control, and predictable execution.

## Available Helpers

Your JavaScript code has access to two helper functions via the `mcp` global object:

### `mcp.callTool(toolName, params)`

Call a remote MCP tool synchronously.

**Parameters:**

- `toolName` (string): The tool name in format `serverID__toolName` (e.g., `"filesystem__read_file"`)
- `params` (object): Parameters to pass to the tool

**Returns:** The result from the tool (parsed from JSON if applicable)

**Throws:** Error if tool call fails

**Example:**

```javascript
// List files in a directory
const result = mcp.callTool("filesystem__list_directory", {
  path: "/tmp",
});

// Search GitHub repositories
const repos = mcp.callTool("github__search_repos", {
  query: "mcp",
  limit: 10,
});

// Chain tool calls
const files = mcp.callTool("filesystem__list_directory", { path: "/tmp" });
const firstFile = files[0];
const content = mcp.callTool("filesystem__read_file", {
  path: "/tmp/" + firstFile,
});
```

### `mcp.log(level, message, fields?)`

Log a message during script execution.

**Parameters:**

- `level` (string): Log level - `"debug"`, `"info"`, `"warn"`, or `"error"`
- `message` (string): Log message (sanitized, max 10,000 chars)
- `fields` (object, optional): Additional structured fields

**Returns:** `undefined`

**Example:**

```javascript
mcp.log("info", "Starting file search");
mcp.log("debug", "Found files", { count: 42 });
mcp.log("warn", "Skipping large file", { size: 1024000 });
mcp.log("error", "File not found", { path: "/missing.txt" });
```

**Note:** Logs are collected during execution and returned with the result. Maximum 1,000 log entries per execution.

## Valid Script Examples

### Example 1: Simple Tool Call

```javascript
// Search for files and return the list
const files = mcp.callTool("filesystem__list_directory", {
  path: "/tmp",
});

return files;
```

### Example 2: Chained Operations

```javascript
// Find a file and read its contents
const files = mcp.callTool("filesystem__list_directory", {
  path: "/var/log",
});

mcp.log("info", "Found files", { count: files.length });

// Filter for .log files
const logFiles = files.filter(f => f.endsWith(".log"));

if (logFiles.length === 0) {
  return { error: "No log files found" };
}

// Read first log file
const content = mcp.callTool("filesystem__read_file", {
  path: "/var/log/" + logFiles[0],
});

return { file: logFiles[0], content: content };
```

### Example 3: Data Transformation

```javascript
// Aggregate data from multiple tools
const repos = mcp.callTool("github__list_repos", {
  org: "modelcontextprotocol",
});

mcp.log("info", "Fetched repositories", { count: repos.length });

// Transform and filter
const result = repos
  .filter(r => r.stars > 10)
  .map(r => ({
    name: r.name,
    stars: r.stars,
    url: r.url,
  }))
  .sort((a, b) => b.stars - a.stars);

return result;
```

### Example 4: Error Handling

```javascript
// Try-catch for handling errors gracefully
let result;

try {
  result = mcp.callTool("filesystem__read_file", {
    path: "/tmp/data.json",
  });
  mcp.log("info", "File read successfully");
} catch (error) {
  mcp.log("error", "Failed to read file", { error: error.message });
  return { error: error.message };
}

// Parse JSON
try {
  const data = JSON.parse(result);
  return { success: true, data: data };
} catch (error) {
  mcp.log("error", "Invalid JSON", { error: error.message });
  return { error: "Invalid JSON format" };
}
```

### Example 5: Iterative Processing

```javascript
// Process multiple items
const items = ["file1.txt", "file2.txt", "file3.txt"];
const results = [];

for (let i = 0; i < items.length; i++) {
  const filename = items[i];

  try {
    const content = mcp.callTool("filesystem__read_file", {
      path: "/tmp/" + filename,
    });

    results.push({
      file: filename,
      success: true,
      length: content.length,
    });

    mcp.log("info", "Processed file", { file: filename });
  } catch (error) {
    results.push({
      file: filename,
      success: false,
      error: error.message,
    });

    mcp.log("warn", "Failed to process file", {
      file: filename,
      error: error.message,
    });
  }
}

return { processed: results.length, results: results };
```

## Common Patterns

### Pattern 1: Search and Filter

```javascript
const allTools = mcp.callTool("hub__search", { query: "" });
const filtered = allTools.filter(t => t.server === "github");
return filtered;
```

### Pattern 2: Conditional Logic

```javascript
const fileExists = mcp.callTool("filesystem__file_exists", {
  path: "/tmp/config.json",
});

if (fileExists) {
  const content = mcp.callTool("filesystem__read_file", {
    path: "/tmp/config.json",
  });
  return JSON.parse(content);
} else {
  return { error: "Config file not found" };
}
```

### Pattern 3: Data Aggregation

```javascript
const server1Data = mcp.callTool("server1__get_stats", {});
const server2Data = mcp.callTool("server2__get_stats", {});

return {
  server1: server1Data,
  server2: server2Data,
  combined: server1Data.count + server2Data.count,
};
```

### Pattern 4: Validation and Sanitization

```javascript
function sanitizeInput(input) {
  // Remove dangerous characters
  return input.replace(/[^a-zA-Z0-9_-]/g, "");
}

const userInput = "user-file.txt";
const safePath = "/tmp/" + sanitizeInput(userInput);

const content = mcp.callTool("filesystem__read_file", {
  path: safePath,
});

return content;
```

## Async Constructs That Are Blocked

The runtime actively blocks these patterns:

### Blocked: async/await

```javascript
// ❌ BLOCKED
async function fetchData() {
  const result = await mcp.callTool("server__tool", {});
  return result;
}

// ✅ ALLOWED
function fetchData() {
  const result = mcp.callTool("server__tool", {});
  return result;
}
```

### Blocked: Promises

```javascript
// ❌ BLOCKED
const promise = new Promise((resolve, reject) => {
  resolve(42);
});

// ❌ BLOCKED
Promise.resolve(42);

// ✅ ALLOWED - Direct synchronous code
const value = 42;
```

### Blocked: setTimeout/setInterval

```javascript
// ❌ BLOCKED
setTimeout(() => {
  console.log("delayed");
}, 1000);

// ❌ BLOCKED
setInterval(() => {
  console.log("repeated");
}, 1000);

// ✅ ALLOWED - Immediate execution
console.log("immediate");
```

### Blocked: Async Tool References

```javascript
// ❌ BLOCKED - Even mentioning Promise is blocked
const p = window.Promise;

// ❌ BLOCKED
const fn = globalThis["Promise"];

// ✅ ALLOWED - Use mcp.callTool directly
const result = mcp.callTool("server__tool", {});
```

## Security Restrictions

### Blocked Globals

These dangerous globals are set to `undefined` and cannot be used:

- `eval` - Code injection risk
- `Function` - Constructor allows code injection
- `GeneratorFunction` - Can create async-like behavior
- `AsyncFunction` - Async construct
- `AsyncGeneratorFunction` - Async construct
- `Reflect` - Can bypass protections
- `Proxy` - Can intercept operations
- `WebAssembly` - Can execute arbitrary code

### Frozen Prototypes

Built-in prototypes are frozen to prevent prototype pollution:

- `Object.prototype`
- `Array.prototype`
- `String.prototype`
- `Number.prototype`
- `Boolean.prototype`
- `Function.prototype`

**Example of what's blocked:**

```javascript
// ❌ BLOCKED - Cannot modify prototypes
Array.prototype.myMethod = function() {};

// ❌ BLOCKED - Cannot use eval
eval("console.log('hello')");

// ❌ BLOCKED - Cannot use Function constructor
const fn = new Function("return 42");

// ✅ ALLOWED - Regular function definitions
function myFunction() {
  return 42;
}
```

## Resource Limits

### Script Size

Maximum script size: **100 KB**

Scripts larger than this will be rejected with a validation error.

### Execution Timeout

Default timeout: **15 seconds**

Scripts that run longer than this will be interrupted with a timeout error.

### Memory Limits

Maximum memory: **50 MB**

Scripts that exceed this will be terminated (future enforcement).

### Log Entries

Maximum log entries: **1,000**

Additional `mcp.log()` calls after 1,000 entries are ignored.

## Debugging Tips

### 1. Use Verbose Logging

```javascript
mcp.log("debug", "Starting script");

const input = { path: "/tmp" };
mcp.log("debug", "Input parameters", input);

const result = mcp.callTool("filesystem__list_directory", input);
mcp.log("debug", "Tool result", { count: result.length });

return result;
```

### 2. Log Variable Types

```javascript
const data = mcp.callTool("server__get_data", {});

mcp.log("info", "Data type", {
  type: typeof data,
  isArray: Array.isArray(data),
  keys: Object.keys(data),
});
```

### 3. Test Incrementally

Start with a minimal script and add complexity gradually:

```javascript
// Step 1: Test basic call
// const result = mcp.callTool("server__tool", {});
// return result;

// Step 2: Test with parameters
// const result = mcp.callTool("server__tool", { param: "value" });
// return result;

// Step 3: Add transformation
const result = mcp.callTool("server__tool", { param: "value" });
return result.map(item => item.name);
```

### 4. Handle Errors Explicitly

```javascript
try {
  const result = mcp.callTool("server__tool", {});
  mcp.log("info", "Success");
  return result;
} catch (error) {
  mcp.log("error", "Tool call failed", {
    error: error.message,
    stack: error.stack,
  });
  return { error: error.message };
}
```

### 5. Validate Inputs

```javascript
function validateParams(params) {
  if (!params.path) {
    throw new Error("path parameter is required");
  }
  if (typeof params.path !== "string") {
    throw new Error("path must be a string");
  }
  return true;
}

const params = { path: "/tmp" };
validateParams(params);

const result = mcp.callTool("filesystem__list_directory", params);
return result;
```

## Error Messages

### Validation Errors

```
validation_error: script exceeds maximum size of 102400 bytes
```

Solution: Reduce script size, extract data transformation to post-processing.

### Async Errors

```
async_not_allowed: async functions are not allowed - only synchronous code is supported
```

Solution: Remove async/await, use synchronous equivalents.

```
async_not_allowed: Promise usage is not allowed - only synchronous code is supported
```

Solution: Remove Promise, call tools directly with `mcp.callTool()`.

### Timeout Errors

```
timeout: script execution exceeded timeout of 15s
```

Solution: Optimize script, reduce tool calls, process less data.

### Syntax Errors

```
syntax_error: SyntaxError: Unexpected token (Line 3:15)
```

Solution: Fix JavaScript syntax at the indicated line.

### Runtime Errors

```
runtime_error: ReferenceError: variable is not defined
```

Solution: Declare all variables before use.

```
runtime_error: TypeError: mcp.callTool requires 2 arguments: toolName (e.g., 'server__tool'), params
```

Solution: Provide all required arguments to `mcp.callTool()`.

## Best Practices

### 1. Keep Scripts Simple

Prefer simple, linear scripts over complex control flow.

### 2. Minimize Tool Calls

Each tool call adds latency. Batch operations when possible.

### 3. Validate Inputs

Always validate and sanitize user inputs before using them in tool calls.

### 4. Handle Errors

Use try-catch blocks around tool calls that might fail.

### 5. Log Strategically

Use logging to debug issues, but don't over-log (max 1,000 entries).

### 6. Return Structured Data

Return objects or arrays with clear structure for easier consumption.

### 7. Avoid Side Effects

Scripts should be idempotent and not rely on external state.

### 8. Test with Small Datasets

Test scripts with small datasets before running on large data.

### 9. Use Type Checking

Check types before operations to avoid runtime errors:

```javascript
if (typeof result === "object" && Array.isArray(result)) {
  return result.map(item => item.name);
}
return [];
```

### 10. Document Complex Logic

Add comments to explain non-obvious logic:

```javascript
// Filter files by extension and sort by modified time
const filtered = files
  .filter(f => f.name.endsWith(".log")) // Only .log files
  .sort((a, b) => b.modified - a.modified); // Newest first
```

## Common Mistakes

### Mistake 1: Using async/await

```javascript
// ❌ Wrong
async function getData() {
  const result = await mcp.callTool("server__tool", {});
  return result;
}

// ✅ Correct
function getData() {
  const result = mcp.callTool("server__tool", {});
  return result;
}
```

### Mistake 2: Forgetting Error Handling

```javascript
// ❌ Wrong - No error handling
const result = mcp.callTool("server__tool", {});
return result.data;

// ✅ Correct
try {
  const result = mcp.callTool("server__tool", {});
  if (result && result.data) {
    return result.data;
  }
  return { error: "No data returned" };
} catch (error) {
  return { error: error.message };
}
```

### Mistake 3: Incorrect Parameter Types

```javascript
// ❌ Wrong - params must be an object
const result = mcp.callTool("server__tool", "param");

// ✅ Correct
const result = mcp.callTool("server__tool", { param: "value" });
```

### Mistake 4: Not Returning a Value

```javascript
// ❌ Wrong - No return value
const result = mcp.callTool("server__tool", {});
mcp.log("info", "Done");

// ✅ Correct
const result = mcp.callTool("server__tool", {});
mcp.log("info", "Done");
return result;
```

### Mistake 5: Mutating Frozen Objects

```javascript
// ❌ Wrong - Cannot mutate prototypes
Array.prototype.sum = function() {
  return this.reduce((a, b) => a + b, 0);
};

// ✅ Correct - Define helper functions
function sum(array) {
  return array.reduce((a, b) => a + b, 0);
}
```

## Advanced Examples

### Example 1: Multi-Server Aggregation

```javascript
// Fetch data from multiple servers and combine
const filesystemData = mcp.callTool("filesystem__get_stats", {});
const githubData = mcp.callTool("github__get_stats", {});
const databaseData = mcp.callTool("database__get_stats", {});

return {
  timestamp: Date.now(),
  sources: {
    filesystem: filesystemData,
    github: githubData,
    database: databaseData,
  },
  summary: {
    total: filesystemData.count + githubData.count + databaseData.count,
  },
};
```

### Example 2: Conditional Tool Selection

```javascript
// Choose which tool to call based on input
function processRequest(type, params) {
  if (type === "file") {
    return mcp.callTool("filesystem__read_file", params);
  } else if (type === "repo") {
    return mcp.callTool("github__get_repo", params);
  } else {
    throw new Error("Unknown type: " + type);
  }
}

const result = processRequest("file", { path: "/tmp/data.txt" });
return result;
```

### Example 3: Data Pipeline

```javascript
// Multi-stage data transformation pipeline
function pipeline(input) {
  mcp.log("info", "Stage 1: Fetch");
  const raw = mcp.callTool("server__fetch", input);

  mcp.log("info", "Stage 2: Filter");
  const filtered = raw.filter(item => item.active);

  mcp.log("info", "Stage 3: Transform");
  const transformed = filtered.map(item => ({
    id: item.id,
    name: item.name.toUpperCase(),
    score: item.value * 100,
  }));

  mcp.log("info", "Stage 4: Sort");
  const sorted = transformed.sort((a, b) => b.score - a.score);

  mcp.log("info", "Pipeline complete", { count: sorted.length });
  return sorted;
}

return pipeline({ category: "active" });
```

## Summary

- ✅ Write synchronous JavaScript only
- ✅ Use `mcp.callTool()` for calling remote tools
- ✅ Use `mcp.log()` for debugging
- ✅ Handle errors with try-catch
- ✅ Return structured data
- ❌ No async/await
- ❌ No Promises
- ❌ No setTimeout/setInterval
- ❌ No dangerous globals (eval, Function, etc.)

For more examples, see the test files in `internal/js/runtime_test.go`.
