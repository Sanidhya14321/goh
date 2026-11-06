# GoHarness

A Go terminal coding agent with a chat-first interface, configurable model
providers, explicit tool approvals, and durable sessions.

The current local candidate is 1.0.0-rc.8. Public V1 release gates are tracked in
the [progress record](go-harness-reconstruction/PROGRESS.md).

- [Build, configure, and use GoHarness](go-harness-reconstruction/implementation/README.md)
- [Verification](go-harness-reconstruction/implementation/VERIFICATION.md)
- [Local measurements](go-harness-reconstruction/implementation/BENCHMARKS.md)
- [Implementation plan](IMPLEMENTATION_PLAN.md)
- [Source provenance](go-harness-reconstruction/ORIGIN_LEDGER.md)

Build from the implementation directory with a compatible Go toolchain.

~~~sh
cd go-harness-reconstruction/implementation
make VERSION=1.0.0-rc.8 build
./bin/goh
~~~

The executable is named goh. The development machine also has a zsh function
that opens GoHarness for bare go and delegates Go subcommands to the toolchain.
That shell customization is local to that machine.
