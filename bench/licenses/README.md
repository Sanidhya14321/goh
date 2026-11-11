# Third-party material in benchmark tasks

The real-project tasks in `bench/tasks/` are bug fixes from open-source Go libraries. Each such task
references its upstream repository and commit in `task.json`, and its `hidden/` and `solution/`
directories contain files copied or adapted from that upstream (hidden tests and the reference fix).
Those files remain under their upstream license, not the repository's Apache-2.0 license. The exact
upstream license text, including its copyright notice, is reproduced in this directory.

Tasks without a `git` source in `task.json` are original fixtures and are Apache-2.0.

| Upstream | License | Copyright notice | License text | Tasks |
| --- | --- | --- | --- | --- |
| [spf13/cast](https://github.com/spf13/cast) | MIT | Copyright (c) 2014 Steve Francia | [`spf13__cast.txt`](spf13__cast.txt) | `go-cast-1b425f31`, `go-cast-6e9731d5`, `go-cast-fa4ea64d` |
| [tidwall/gjson](https://github.com/tidwall/gjson) | MIT | Copyright (c) 2016 Josh Baker | [`tidwall__gjson.txt`](tidwall__gjson.txt) | `go-gjson-9378d3bb`, `go-gjson-a825fa56`, `go-gjson-eaae8c63` |
| [tidwall/sjson](https://github.com/tidwall/sjson) | MIT | Copyright (c) 2016 Josh Baker | [`tidwall__sjson.txt`](tidwall__sjson.txt) | `go-sjson-1e706478`, `go-sjson-9f1cbfdb`, `go-sjson-f0069f8a` |
| [pelletier/go-toml](https://github.com/pelletier/go-toml) | MIT | Copyright (c) 2021 - 2023 Thomas Pelletier | [`pelletier__go-toml.txt`](pelletier__go-toml.txt) | `go-gotoml-071a36c2`, `go-gotoml-686c980c`, `go-gotoml-79f82e3c` |
| [dustin/go-humanize](https://github.com/dustin/go-humanize) | MIT | Copyright (c) 2005-2008  Dustin Sallings <dustin@spy.net> | [`dustin__go-humanize.txt`](dustin__go-humanize.txt) | `go-humanize-1b638b2c`, `go-humanize-21224679`, `go-humanize-40736da3` |
| [gorilla/mux](https://github.com/gorilla/mux) | BSD-3-Clause | Copyright (c) 2023 The Gorilla Authors. All rights reserved. | [`gorilla__mux.txt`](gorilla__mux.txt) | `go-mux-2b030fc3`, `go-mux-976b5364` |
| [Masterminds/semver](https://github.com/Masterminds/semver) | MIT | Copyright (C) 2014-2019, Matt Butcher and Matt Farina | [`Masterminds__semver.txt`](Masterminds__semver.txt) | `go-semver-441ff8e5` |
| [BurntSushi/toml](https://github.com/BurntSushi/toml) | MIT | Copyright (c) 2013 TOML authors | [`BurntSushi__toml.txt`](BurntSushi__toml.txt) | `go-toml-d10101f4` |

This inventory records provenance; it is not a legal opinion. Benchmark tasks are test material and
are not compiled into, or shipped with, the `goh` binary.
