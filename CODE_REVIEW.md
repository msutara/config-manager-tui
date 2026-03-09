# Code Review: config-manager-tui

Multi-perspective code review covering Architect, Developer, Tester, and Security (Hacker) viewpoints.
Each section is a conversation thread you can act on independently.

---

## 🏛️ Architect Perspective

### [ARCH-1] Title-based menu refresh dispatch is fragile and has a known TODO

**File:** `tui.go:406–424`

**Conversation:**

> **Architect:** The `needsMenuRefresh` flow matches menus by their display title strings — a `switch screenTitle` that hard-codes `"Network Manager"`, `"Update Manager"`, `"Set Static IP — Select Interface"`, etc. There is even a `// TODO: replace title-based menu matching with a builder ID or plugin name in menuState` comment acknowledging the problem. The default branch falls back to `actionUpdateMenu`, which means any generic plugin that returns `apiResultMsg{refreshMenu: true}` would have its sub-menu silently rebuilt as "Update Manager". Right now generic plugins don't reach that path, but it is a silent correctness trap for future contributors.
>
> **Developer:** It's true. There's also duplication: the inner `switch` inside the goroutine at line 408 re-lists the same titles as the outer `switch` at line 388.
>
> **Architect:** The fix is to store the builder function (or a plugin name) in `menuState` when pushing to the stack. The display title is presentation-only and should never drive routing decisions.

---

### [ARCH-2] `ModeStandalone` is hardcoded in the constructor; callers must remember a separate `SetConnectionMode` call

**File:** `tui.go:107–116`, `tui.go:118–121`

**Conversation:**

> **Architect:** `NewWithAuth` hard-codes `connMode: ModeStandalone`. The only way to change it is `SetConnectionMode(ModeConnected)`, which must be called after construction and before `Init`. If the integration layer forgets this call, the footer badge will always show "standalone" even when connected to an external service.
>
> **Developer:** We could add `mode ConnectionMode` as a constructor parameter, or accept it as part of an options struct, so the mode is established at creation time.
>
> **Architect:** At minimum a doc comment on the constructor should call this out as a required post-construction step.

---

### [ARCH-3] API client is a concrete type with no interface — no seam for testing or substitution

**File:** `apiclient.go:17–36`, `menu.go` and `tui.go` (all usages pass `*APIClient` directly)

**Conversation:**

> **Architect:** Every action builder (`actionUpdateMenu`, `actionNetworkInterfaces`, etc.) receives `*APIClient` concretely. There is no `APIClientInterface` (or similar), meaning unit tests that want to exercise menu-building logic must spin up a real `httptest.Server` and cannot inject fakes or mocks. The test surface area for integration-level tests is wide.
>
> **Tester:** I've noticed that the network and update tests all start an `httptest.Server`. If the API interface were extracted, we could test `actionUpdateStatus` formatting without any HTTP at all.
>
> **Architect:** Extracting a minimal interface containing `GetNode()`, `GetUpdateStatus()`, `GetNetworkInterfaces()`, etc. would dramatically improve testability and allow the core binary to swap in a local implementation without HTTP.

---

### [ARCH-4] `Init()` silently swallows node-info errors — no observability

**File:** `tui.go:124–135`

**Conversation:**

> **Architect:** When `GetNode` fails at startup, the code returns an empty `nodeInfoMsg{}` with no way to tell whether the blank hostname/uptime in the footer are because the API is unavailable or simply hasn't responded yet. There is no structured log call (the rest of the codebase uses `log/slog` per the conventions) and no user-visible signal.
>
> **Developer:** A `slog.Warn` call here and perhaps a `nodeInfoMsg{err: err}` variant (parallel to `apiResultMsg`) would make this diagnosable.

---

## 🛠️ Developer Perspective

### [DEV-1] `io.ReadAll` called with no size limit on response bodies

**File:** `apiclient.go:295, 323, 608, 634, 660, 696`

**Conversation:**

> **Developer:** Six call sites use `io.ReadAll(resp.Body)` without wrapping the reader in `io.LimitReader`. A misbehaving or adversarial API could return a response of arbitrary size, causing the TUI process to allocate unbounded memory. The `truncateBody` function limits what ends up in *error messages*, but the raw allocation happens before that call, and success-path responses (including the job-history list) are entirely uncapped.
>
> **Security:** On a device with constrained RAM (e.g. Raspberry Pi Zero), a 100 MB response body would OOM the process.
>
> **Developer:** Wrapping every `resp.Body` with `io.LimitReader(resp.Body, maxBodyBytes)` (e.g. 1 MB for normal responses, 10 MB for raw/log endpoints) would close this. The `http.Client` timeout already covers wall-clock time, but not response size.

---

### [DEV-2] Internal `getJSON`/`postJSON`/`putJSON` bypass path validation

**File:** `apiclient.go:593–719`

**Conversation:**

> **Developer:** The public helpers `GetRaw` and `PostRaw` call `validateAPIPath` before sending. The internal helpers `getJSON`, `postJSON`, `putJSON`, and `doConfirmJSON` do not. All current callers pass hardcoded string literals so there is no immediate risk, but a future refactor that constructs a path dynamically and calls `getJSON` directly would bypass the validation silently.
>
> **Developer:** Moving the `validateAPIPath` call into `getJSON`/`postJSON` (or into a shared `doRequest` method) would enforce the invariant everywhere without duplicating the check at every call site.

---

### [DEV-3] DNS server input has no client-side format validation

**File:** `tui.go:548–582`

**Conversation:**

> **Developer:** When processing the DNS input (`key == inputKeyNetworkDNS`), individual server strings are split by comma and trimmed but not validated as IP addresses or hostnames. The static-IP path at the same level uses `netip.ParsePrefix` for CIDR validation. The inconsistency means a user typo like `8.8.8.8,bad value` is sent to the API without any early feedback.
>
> **UX/Developer:** Client-side validation with `netip.ParseAddr` would catch obvious typos before a round-trip and give the user an immediate `statusMsg` error — consistent with the static-IP flow.

---

### [DEV-4] Separator `MenuItem` ("──── Actions ────") accepts cursor focus

**File:** `menu.go:603`

**Conversation:**

> **Developer:** The network menu includes a visual separator item `{Title: "──── Actions ────", Description: ""}` with no `Action`. The cursor can land on it; pressing Enter silently does nothing because of the `if item.Action != nil` guard. This is confusing — the user presses Enter and nothing happens.
>
> **UX:** A separator should either be skipped by the cursor (requires cursor-skip logic) or replaced with a visual section header that is not in the items slice. At minimum, a brief `statusMsg = "Select an action above"` when Enter is pressed on a nil-action item would reduce confusion.

---

### [DEV-5] No maximum length enforced on the input buffer

**File:** `tui.go:600–608`

**Conversation:**

> **Developer:** Every keypress appends to `m.inputBuffer` with no upper bound. A user holding down a key could create a string of thousands of characters. When rendered with `sanitizeText(m.inputBuffer) + "█"`, a very long single-line string wraps unpredictably in terminals with width-based wrapping, corrupting the layout.
>
> **Developer:** Capping the buffer at, say, 512 characters (with a `statusMsg` hint when the limit is reached) would keep the UI stable without restricting legitimate use cases.

---

### [DEV-6] `GetUpdateConfig` is called twice in some flows (double round-trip)

**File:** `menu.go:324` (`actionUpdateMenu`) and `menu.go:485` (`actionUpdateViewSettings`)

**Conversation:**

> **Developer:** When the user opens the Update Manager sub-menu and then selects "View Settings", the config is fetched in `actionUpdateMenu` (to populate schedule/auto-security display labels) and then fetched again in `actionUpdateViewSettings` (to decide which keys to hide). These are sequential but independent calls to the same endpoint.
>
> **Developer:** Passing the already-fetched config down — or using a single `GetPluginSettings` response for both purposes — would halve the API calls for this flow.

---

### [DEV-7] `ListJobRuns` builds URL with `fmt.Sprintf` instead of `url.Values`

**File:** `apiclient.go:460`

**Conversation:**

> **Developer:** `path := fmt.Sprintf("/api/v1/jobs/%s/runs?limit=%d&offset=%d", jobID, limit, offset)` is safe for the current integer arguments, but the pattern of building query strings with `fmt.Sprintf` does not scale safely if a string parameter is ever added — it would require manual percent-encoding. Using `url.Values.Encode()` is the idiomatic, injection-safe way to compose query strings in Go regardless of argument types.

---

## 🧪 Tester Perspective

### [TEST-1] No test for BiDi character pass-through in sanitizeText/sanitizeBody

**File:** `menu_test.go:33–73`, `menu.go:142–166`

**Conversation:**

> **Tester:** The existing `TestSanitizeText` and `TestSanitizeBody` test cases focus on C0/C1 control characters and ANSI escape sequences. There are no test cases for Unicode bidirectional format characters (U+202A–U+202E, U+2066–U+2069). As confirmed by running Go's `unicode.IsControl` against these code points, they are **not** caught by the current implementation — they pass straight through. This is a confirmed gap: a test case like `{"bidi override", "SAFE\u202eEVIL", "SAFE\u202eEVIL"}` would currently pass when it should have `"SAFE"` as the expected output (see [SEC-1] for the full vulnerability discussion).

---

### [TEST-2] No test for generic plugin `needsMenuRefresh` path

**File:** `tui.go:379–427`

**Conversation:**

> **Tester:** The menu-refresh-after-settings-change path is tested for the update and network menus, but there is no test simulating a generic plugin (one with an unrecognised `screenTitle`) triggering `needsMenuRefresh`. As described in [ARCH-1], the default branch silently uses `actionUpdateMenu` for any unknown title. A test that sets `m.screenTitle = "Firewall"` with `m.needsMenuRefresh = true`, presses a key to dismiss the detail screen, and verifies the refreshed menu title is `"Firewall"` (not `"Update Manager"`) would expose this bug immediately.

---

### [TEST-3] No test for `ThemeFromYAML` with untrusted glyph content

**File:** `theme.go:181–198`, `theme_test.go`

**Conversation:**

> **Tester:** `ThemeFromYAML` accepts a `cursor` glyph and `connected_badge`/`standalone_badge` strings from YAML without any sanitization. A theme file containing `cursor: "\x1b[31m"` would inject ANSI into every rendered menu line. There are no tests verifying that the parsed glyph values are safe before use in `renderMainMenu` / `renderPluginView`.

---

### [TEST-4] `formatUptime` not tested for very large values (overflow risk)

**File:** `menu.go:904–915`, `tui_test.go`

**Conversation:**

> **Tester:** `formatUptime` does integer division with no overflow guard. Passing `math.MaxInt32` (≈68 years of uptime) or a negative value produces nonsensical but non-panicking output. There are tests for `0` and `-100`, but no test for `math.MaxInt32` confirming the output is at least human-readable. On 32-bit ARM (one of the target platforms), an `int` is 32 bits, and an `UptimeSeconds` value of `2147483647` would produce `24855d 3h 14m` — valid but worth verifying the format is intentional.

---

### [TEST-5] `Init()` cannot be tested against a real error path

**File:** `tui.go:124–135`

**Conversation:**

> **Tester:** `Init()` silently drops `GetNode` errors and returns an empty `nodeInfoMsg{}`. Because the error is swallowed inside the returned closure, there is no way to observe from a test that the error occurred. A test using `closedTestServer()` as the base URL would confirm that `Init()` does not panic and does not block, but cannot assert that an error was logged or surfaced. Adding a `nodeInfoMsg{err: err}` variant (and handling it in `Update`) would make this branch testable.

---

### [TEST-6] No test for backspace on an empty input buffer

**File:** `tui.go:594–598`

**Conversation:**

> **Tester:** The code `if len(m.inputBuffer) > 0` correctly guards the backspace path, but there is no unit test that sends `tea.KeyBackspace` when `m.inputBuffer == ""` and confirms the model does not panic and the buffer remains empty. This is a simple regression-prevention test.

---

### [TEST-7] `BuiltinTheme` bool return not tested for false/unknown case

**File:** `theme.go:274–285`, `theme_test.go`

**Conversation:**

> **Tester:** `BuiltinTheme("nonexistent")` is not tested. The function returns `(Theme{}, false)`. If the caller omits the bool check, they silently receive an empty `Theme{}` (zero-value lipgloss styles), which renders all text as unstyled. A test that calls `BuiltinTheme("nonexistent")` and asserts the bool is `false` and the returned `Theme` is zero-value would document the intended contract.

---

## 🔐 Security / Hacker Perspective

### [SEC-1] **CONFIRMED** Unicode bidirectional override characters bypass all sanitization functions

**File:** `menu.go:142–166` (`sanitizeText`), `menu.go:157–166` (`sanitizeBody`), `apiclient.go:207–221` (`truncateBody`)

**Conversation:**

> **Hacker:** `sanitizeText` and `sanitizeBody` both call `unicode.IsControl(r)` to decide which characters to strip. In Go, `unicode.IsControl` only returns `true` for the C0 range (U+0000–U+001F), DEL (U+007F), and C1 range (U+0080–U+009F). It returns **false** for Unicode Format characters (category `Cf`), including the full set of bidirectional override characters:
>
> | Code point | Name | `unicode.IsControl` |
> |---|---|---|
> | U+202A | LEFT-TO-RIGHT EMBEDDING | **false** |
> | U+202B | RIGHT-TO-LEFT EMBEDDING | **false** |
> | U+202C | POP DIRECTIONAL FORMATTING | **false** |
> | U+202D | LEFT-TO-RIGHT OVERRIDE | **false** |
> | U+202E | RIGHT-TO-LEFT OVERRIDE | **false** |
> | U+2066 | LEFT-TO-RIGHT ISOLATE | **false** |
> | U+2067 | RIGHT-TO-LEFT ISOLATE | **false** |
> | U+2068 | FIRST STRONG ISOLATE | **false** |
> | U+2069 | POP DIRECTIONAL ISOLATE | **false** |
>
> This has been verified with Go 1.24 (`unicode.IsControl(0x202E)` → `false`). A malicious API server can inject U+202E into a plugin description, interface name, error message, or hostname and it will survive all three sanitization functions. In a terminal supporting bidirectional text (most modern terminals do), `"Safe operation\u202eFILED"` displays the text after U+202E in reverse order, so the user sees `"Safe operationDELIF"` — content appears different from what it is.
>
> **Severity:** Medium — requires a compromised or malicious API server, but the entire security model of the sanitization layer is undermined.
>
> **Fix direction:** Replace `unicode.IsControl(r)` with `unicode.IsControl(r) || unicode.Is(unicode.C, r)` (which covers all "Other" categories including `Cf` format characters), or explicitly add a BiDi strip using `unicode.Is(unicode.Bidi_Control, r)`.

---

### [SEC-2] `confirmMsg` is rendered without sanitization in `viewConfirm`

**File:** `tui.go:765–776`

**Conversation:**

> **Hacker:** `viewConfirm` writes `m.confirmMsg` directly to the output string:
> ```go
> b.WriteString("  " + m.confirmMsg + "\n\n")
> ```
> There is no `sanitizeText` call at render time. The current assignment sites sanitize dynamic values (interface names, IP strings, DNS servers), and static strings are hardcoded safe literals. However, `m.confirmMsg = item.ConfirmMsg` at `tui.go:469` assigns the `ConfirmMsg` field straight from the `MenuItem` struct. For the generic plugin path (`actionGenericPost`), `ConfirmMsg` is built from `desc` which is `sanitizeText(ep.Description)` — safe. But this is a "defence in depth" gap: if any future `MenuItem.ConfirmMsg` assignment misses the sanitization step, injected characters (including BiDi overrides, given [SEC-1]) will reach the terminal unfiltered.
>
> **Fix direction:** Add `sanitizeText(m.confirmMsg)` in `viewConfirm` as a last-resort defence. This is cheap and makes the rendering path self-defending.

---

### [SEC-3] Bearer token stored as a plain string field — no masking for logs or debugging

**File:** `apiclient.go:17–34`

**Conversation:**

> **Hacker:** `APIClient.token` is a plain `string` field. If the struct or any error containing a formatted struct value is ever passed to a logger (e.g., `slog.Debug("client", "api", c)`), the token appears in the log output. There is no `String()` or `GoString()` method that masks it.
>
> **Developer:** A `String() string { return fmt.Sprintf("APIClient{baseURL:%q, token:[REDACTED]}", c.baseURL) }` method would prevent accidental token exposure in all standard formatting paths.
>
> **Security note:** This is a defence-in-depth measure; the primary protection is not logging the struct directly. But adding the mask costs nothing and prevents future accidents.

---

### [SEC-4] Theme YAML glyphs (`cursor`, `connected_badge`, `standalone_badge`) are not sanitized before rendering

**File:** `theme.go:181–198`, `views.go:11–59`

**Conversation:**

> **Hacker:** `ThemeFromYAML` accepts `cursor`, `connected_badge`, and `standalone_badge` as raw strings from a user-supplied YAML file. These strings are passed directly to lipgloss `Render()` calls in `renderMainMenu`, `renderPluginView`, `renderFooter`, and `renderSubFooter` without any sanitization. A theme file with `cursor: "\x1b[31m"` would inject an ANSI sequence before every menu item. An operator distributing a malicious theme file (e.g., packaged as a "dark mode theme") could use this to hide menu items or inject visible text.
>
> **Severity:** Low — requires the operator to load an untrusted theme file.
>
> **Fix direction:** Run glyph strings through `sanitizeText` in `ThemeFromYAML` (or at rendering time in `renderHeader`/`renderMainMenu`/`renderFooter`) before they are used.

---

### [SEC-5] No response body size limit — potential denial-of-service via large API response

**File:** `apiclient.go:295, 323, 608, 634, 660, 696` (see also [DEV-1])

**Conversation:**

> **Hacker:** If an attacker gains control of the local API server (or performs a DNS/ARP redirect to a rogue server on the LAN), they can return a multi-gigabyte response body to any endpoint — `GET /api/v1/jobs/update.full/runs?limit=20` for example. `io.ReadAll` will block-read it all into a `[]byte`. On a Raspberry Pi with 512 MB RAM, this OOMs the TUI process. Combined with the 30-second HTTP timeout, a 1 GB response at 40 MB/s would be fully consumed before the timeout fires.
>
> **Severity:** Low — requires local API server compromise.
>
> **Fix direction:** `io.LimitReader(resp.Body, 10<<20)` (10 MB) for raw/log responses; `io.LimitReader(resp.Body, 1<<20)` (1 MB) for JSON API responses. This is consistent with production-grade HTTP client hardening.

---

### [SEC-6] `progressTitle` is rendered unsanitized in `viewProgress`

**File:** `tui.go:786`

**Conversation:**

> **Hacker:** `m.theme.Header.Render(m.progressTitle)` is called without passing `m.progressTitle` through `sanitizeText` first. Currently, all `progressTitle` values are hardcoded strings (`"Full Update"`, `"Security Update"`) set by action builders — so there is no immediate injection risk. However, `progressTitle` is set from `msg.title` in the `jobAcceptedMsg` handler (tui.go:232), and `jobAcceptedMsg.title` is set by callers of those actions. If a future refactor ever populates the title from API data (e.g., a job display name returned by the server), the title would reach the terminal without sanitization.
>
> **Fix direction:** Add `sanitizeText(m.progressTitle)` inside `viewProgress`, consistent with how other dynamic strings are handled in `viewDetail` and `viewSubMenu`.

---

### [SEC-7] DNS nameservers from `actionNetworkSetDNS` used as input prefill without sanitization

**File:** `menu.go:751–766`

**Conversation:**

> **Hacker:** In `actionNetworkSetDNS`, the current DNS server list is fetched from the API and joined into a comma-separated string:
> ```go
> currentServers = strings.Join(dns.Nameservers, ", ")
> ```
> This is then placed directly into `editInputMsg{currentVal: currentServers}` without calling `sanitizeText`. The value ends up in `m.inputBuffer`, which IS sanitized before submission (`value := sanitizeText(m.inputBuffer)` at tui.go:496) and IS sanitized at display time (`sanitizeText(m.inputBuffer)` at tui.go:757). So there is no direct injection path to the terminal or the API.
>
> However, given the BiDi vulnerability in [SEC-1], a server that returns nameservers containing U+202E could cause the input prefill to visually appear as a different value (e.g., `"8.8.8.8"` appears reversed as `"8.8.8.8"` — or more subtly, `"8.8.8.\u202e8"` makes `"8"` appear to precede the address). Sanitizing at the point of assignment (`currentVal: sanitizeText(currentServers)`) would close this gap without relying on downstream sanitization layers.

---

## Summary Table

| ID | Perspective | Severity | File(s) | One-line description |
|---|---|---|---|---|
| ARCH-1 | Architect | High (fragility) | `tui.go:406–424` | Title-based menu refresh routing silently breaks for generic plugins |
| ARCH-2 | Architect | Low | `tui.go:107–121` | `ModeStandalone` hardcoded; requires manual `SetConnectionMode` call |
| ARCH-3 | Architect | Medium | `apiclient.go`, `menu.go` | No `APIClient` interface; wide integration-test surface |
| ARCH-4 | Architect | Low | `tui.go:124–135` | `Init()` silently drops node-info errors |
| DEV-1 | Developer | Medium | `apiclient.go` (6 sites) | `io.ReadAll` with no body size limit |
| DEV-2 | Developer | Low | `apiclient.go:593–719` | Internal HTTP helpers bypass `validateAPIPath` |
| DEV-3 | Developer | Low | `tui.go:548–582` | DNS input not validated as IP addresses client-side |
| DEV-4 | Developer | Low | `menu.go:603` | Separator menu item silently eats Enter keypresses |
| DEV-5 | Developer | Low | `tui.go:600–608` | No maximum length on input buffer |
| DEV-6 | Developer | Low | `menu.go:324, 485` | `GetUpdateConfig` called twice in some flows |
| DEV-7 | Developer | Low | `apiclient.go:460` | Query string built with `fmt.Sprintf` instead of `url.Values` |
| TEST-1 | Tester | High | `menu_test.go` | No tests for BiDi character pass-through (confirms SEC-1) |
| TEST-2 | Tester | Medium | `tui_test.go` | No test for generic plugin `needsMenuRefresh` path (confirms ARCH-1 bug) |
| TEST-3 | Tester | Low | `theme_test.go` | No test for unsanitized glyph values in `ThemeFromYAML` |
| TEST-4 | Tester | Low | `tui_test.go` | `formatUptime` not tested for very large / `MaxInt32` values |
| TEST-5 | Tester | Low | `tui_test.go` | `Init()` error path is unobservable and untested |
| TEST-6 | Tester | Low | `tui_test.go` | No test for backspace on an empty input buffer |
| TEST-7 | Tester | Low | `theme_test.go` | `BuiltinTheme` bool false-return case not tested |
| SEC-1 | Security | **Medium** | `menu.go:142–166`, `apiclient.go:207–221` | BiDi override chars (U+202A–U+202E, U+2066–U+2069) bypass all sanitization |
| SEC-2 | Security | Medium | `tui.go:765–776` | `confirmMsg` rendered without sanitization in `viewConfirm` |
| SEC-3 | Security | Low | `apiclient.go:17–34` | Bearer token has no masking `String()` method |
| SEC-4 | Security | Low | `theme.go:181–198` | Theme glyph strings not sanitized — injection via malicious theme file |
| SEC-5 | Security | Low | `apiclient.go` | No response body size cap; DoS via oversized API response |
| SEC-6 | Security | Low | `tui.go:786` | `progressTitle` rendered unsanitized (currently safe, but fragile) |
| SEC-7 | Security | Low | `menu.go:751–766` | DNS prefill from API not sanitized at assignment (mitigated downstream) |
