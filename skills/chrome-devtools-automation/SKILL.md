---
name: chrome-devtools-automation
description: Automate Chrome DevTools browser actions like navigation, forms, screenshots, and performance traces. Use when users need browser automation or page inspection. Triggers on "Chrome DevTools", "take a screenshot", "fill out the form", "performance trace", "click on".
---

# Chrome DevTools Automation

MCP service via local config `./config.json` with 26 tools.

## Requirements

- `hub` CLI must be installed. If not available, install with:
  ```bash
  curl -fsSL https://raw.githubusercontent.com/vaayne/cc-plugins/main/mcps/hub/scripts/install.sh | sh
  ```

## Usage

List tools: `hub -c ./config.json list`
Get tool details: `hub -c ./config.json inspect <tool-name>`
Invoke tool: `hub -c ./config.json invoke <tool-name> '{"param": "value"}'`

`./config.json` is relative to this `SKILL.md`.

## Notes

- Run `inspect` before invoking unfamiliar tools to get full parameter schema
- Timeout: 30s default, use `--timeout <seconds>` to adjust

## Tools

- chromeDevtoolsClick: Clicks on the provided element
- chromeDevtoolsClosePage: Closes the page by its index. The last open page cannot be closed.
- chromeDevtoolsDrag: Drag an element onto another element
- chromeDevtoolsEmulate: Emulates various features on the selected page.
- chromeDevtoolsEvaluateScript: Evaluate a JavaScript function inside the currently selected page. Returns the response as JSON so returned values have to JSON-serializable.
- chromeDevtoolsFill: Type text into a input, text area or select an option from a <select> element.
- chromeDevtoolsFillForm: Fill out multiple form elements at once
- chromeDevtoolsGetConsoleMessage: Gets a console message by its ID. You can get all messages by calling list_console_messages.
- chromeDevtoolsGetNetworkRequest: Gets a network request by an optional reqid, if omitted returns the currently selected request in the DevTools Network panel.
- chromeDevtoolsHandleDialog: If a browser dialog was opened, use this command to handle it
- chromeDevtoolsHover: Hover over the provided element
- chromeDevtoolsListConsoleMessages: List all console messages for the currently selected page since the last navigation.
- chromeDevtoolsListNetworkRequests: List all requests for the currently selected page since the last navigation.
- chromeDevtoolsListPages: Get a list of pages open in the browser.
- chromeDevtoolsNavigatePage: Navigates the currently selected page to a URL.
- chromeDevtoolsNewPage: Creates a new page
- chromeDevtoolsPerformanceAnalyzeInsight: Provides more detailed information on a specific Performance Insight of an insight set that was highlighted in the results of a trace recording.
- chromeDevtoolsPerformanceStartTrace: Starts a performance trace recording on the selected page. This can be used to look for performance problems and insights to improve the performance of the page. It will also report Core Web Vital (CWV) scores for the page.
- chromeDevtoolsPerformanceStopTrace: Stops the active performance trace recording on the selected page.
- chromeDevtoolsPressKey: Press a key or key combination. Use this when other input methods like fill() cannot be used (e.g., keyboard shortcuts, navigation keys, or special key combinations).
- chromeDevtoolsResizePage: Resizes the selected page's window so that the page has specified dimension
- chromeDevtoolsSelectPage: Select a page as a context for future tool calls.
- chromeDevtoolsTakeScreenshot: Take a screenshot of the page or element.
- chromeDevtoolsTakeSnapshot: Take a text snapshot of the currently selected page based on the a11y tree. The snapshot lists page elements along with a unique identifier (uid). Always use the latest snapshot. Prefer taking a snapshot over taking a screenshot.
- chromeDevtoolsUploadFile: Upload a file through a provided element.
- chromeDevtoolsWaitFor: Wait for the specified text to appear on the selected page.
