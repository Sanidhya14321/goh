# Explicit URL fetching

Pinned source assistance: opencode/src/tool/webfetch.ts and core counterparts in
the full checkout. Actual reference capture webfetch-reference.json uses original
loopback content/model fixtures under research, never outside services.
Tool input url, optional format(text|markdown|html, default markdown), optional
integer timeout(default 30, max 120 seconds). Observed reference: text/html/plain/
JSON responses, visible-text and Markdown conversion, raw HTML preservation,
same/cross-origin redirects, 404, 6 MiB rejection, timeout and binary decoding.
Source's 5 MiB limit is distinct from the original Go 4 MiB budget.

Original Go client uses a separately constructed net/http transport, Proxy:nil,
no cookie jar, no provider API headers, HTTP(S) only and no embedded credentials.
Every request requires runtime webfetch approval; a run-level webfetch grant is
broad authority for valid URLs during that process. Redirects cap at 5 and must
remain on the approved origin; cross-origin targets require a fresh invocation.
Only GET is supported. GET/cancellation does not promise a remote effect rollback.
Private/loopback URLs are permitted when explicitly approved, not a sandbox.

UTF-8/ASCII text only; binary/image and other charsets fail. Response/decompressed
body cap 4 MiB; header/URL/timeout/token/depth limits are explicit in code. The pinned
x/net/html tokenizer produces bounded text/Markdown and strips script/style/etc.
for those conversions. Raw HTML deliberately preserves source as untrusted data.
It is never executed or rendered in a browser. Resource links are not fetched.
Result JSON cap 64 KiB includes truncation flag, byte count, original-content SHA256,
final URL, MIME and status. Conversion is a declared useful subset, not visual or
byte-identical Markdown parity with the reference.

Go REA: go-webfetch.json,
`ev_9e101945984d715d699c9131e0d892abaf9cea6377f099cfdc361aaa61a98b6f`.
Eleven selected cases establish success for plain/text/Markdown/rawHTML/JSON/
same-origin redirect, and explicit refusal/error for cross-origin,404,large,
timeout and binary cases. Rendered capture is truncated; it also lists unverified
ambient processes even though the owned process group is verified. Neither raw
capture nor selected semantic projection establishes universal equivalence.

Tests use original httptest endpoints for credential/proxy/cookie isolation,
redirect destination non-contact, response/escaped-result bounds, HTTP errors,
HTML/safe links, special arguments, cancellation and runtime approval/plan gates.
Argument fuzzing is bounded and network-free. Remote search, browser rendering,
images/PDF extraction, streaming downloads and authenticated fetch profiles remain
future slices. Plan/explore presently deny network tools deliberately.
