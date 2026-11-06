# Part 0: identity and host readiness

The canonical repository tag resolves to the selected commit. The official Darwin ARM64 archive was acquired and extracted separately from implementation. Its executable reports 1.19.5. The archive's local SHA256 and GitHub asset digest are recorded in the manifest; signature verification is not claimed. The MIT reference license remains in research/reference/LICENSE.

Go and REA are project-local development tools. The REA npm release is 6.8.0; the inspected skill is separately pinned to source commit 71f680d7c1aa7a45ba0325fe87b104744b0c105f. Their source identity is not assumed equal. No global client registration was performed.

Exit evidence: real executable `--version` REA capture succeeded on Darwin ARM64 with exit0 and verified group cleanup. Native/embedded-module analysis is not required to establish this observation and was not performed. Other operating systems remain untested.
