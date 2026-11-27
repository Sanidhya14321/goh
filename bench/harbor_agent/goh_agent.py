"""GoHarness (goh) as a Harbor installed agent.

Run from the repository root with bench/harbor_agent/run.sh, which builds the
Linux binaries this agent uploads. The model is reached over the ChatGPT
sign-in of this machine: each task container gets only a short-lived access
token (`goh auth export`), never the refresh token, and the token is replaced
from this machine while goh runs, so a container can neither renew nor revoke
the sign-in here.
"""

from __future__ import annotations

import asyncio
import json
import os
import shlex
import subprocess
import tempfile
import tomllib
from pathlib import Path
from typing import Annotated, override

from pydantic import Field

from harbor.agents.capabilities import AgentCapabilities
from harbor.agents.installed.base import (
    ApiUsageLimitError,
    BaseInstalledAgent,
    ErrorPattern,
    with_prompt_template,
)
from harbor.agents.options import Cli, InstalledAgentOptions
from harbor.environments.base import BaseEnvironment
from harbor.models.agent.context import AgentContext

REPO = Path(__file__).resolve().parents[2]
BIN_DIR = Path(os.environ.get("GOH_HARBOR_BIN", REPO / "bench" / "harbor_agent" / "bin"))
HOST_GOH = os.environ.get("GOH_HOST_BIN", str(REPO / "bin" / "goh"))
# Replace the container's token this often; each export is valid for at least
# TOKEN_VALID, so a token never expires between replacements.
TOKEN_EVERY_SEC = 20 * 60
TOKEN_VALID = "45m"
CRED_DIR = "$HOME/.config/independent-harness/credentials"
TASK_CACHE = Path(os.environ.get("HARBOR_TASK_CACHE", Path.home() / ".cache" / "harbor" / "tasks"))
# goh stops this long before Harbor's agent timeout, so it ends the run itself
# (jobs and the journal intact) and the model plans against the real limit.
STOP_MARGIN_SEC = 30
# Some task images have no CA certificates. goh then uses this machine's bundle
# (as the agents that install ca-certificates do); commands it runs inherit it.
HOST_CA = os.environ.get("GOH_CA_BUNDLE", "/etc/ssl/cert.pem")
CA_PATH = "/usr/local/share/goh/ca.pem"
SYSTEM_CA = ("/etc/ssl/certs/ca-certificates.crt", "/etc/pki/tls/certs/ca-bundle.crt",
             "/etc/ssl/ca-bundle.pem", "/etc/pki/tls/cacert.pem",
             "/etc/pki/ca-trust/extracted/pem/tls-ca-bundle.pem", "/etc/ssl/cert.pem")


class GohOptions(InstalledAgentOptions):
    reasoning_effort: Annotated[str | None, Cli("--reasoning-effort")] = Field(
        default="medium", description="Reasoning effort passed to goh."
    )
    experiment: str | None = Field(
        default=None, description="Comma-separated goh research experiments to turn on (A/B arms)."
    )


class Goh(BaseInstalledAgent):
    """Runs `goh run` headless in the task container."""

    options_model = GohOptions
    capabilities = AgentCapabilities(reasoning_effort=True)
    # goh prints the provider's error code; Harbor's own patterns miss the
    # ChatGPT plan limit, which would otherwise read as a goh failure.
    ERROR_PATTERNS = [
        *BaseInstalledAgent.ERROR_PATTERNS,
        ErrorPattern(r"\(\w*usage_limit_exceeded\)|reached their [\w ]*usage limit", ApiUsageLimitError),
    ]
    _export_lock = asyncio.Lock()  # one host renewal at a time

    @staticmethod
    @override
    def name() -> str:
        return "goh"

    @override
    def get_version_command(self) -> str | None:
        return "goh --version"

    @override
    def parse_version(self, stdout: str) -> str:
        return stdout.strip().split()[-1] if stdout.strip() else ""

    def _model(self) -> str:
        model = self.model_name or "gpt-6.2-sol"
        return model.split("/", 1)[-1]  # "openai/gpt-6.2-sol" -> "gpt-6.2-sol"

    async def _upload(self, environment: BaseEnvironment, source: Path | str, target: str) -> None:
        """Uploads and checks the size: the container's tar stream is sometimes cut short."""
        size = Path(source).stat().st_size
        for attempt in range(4):
            try:
                await environment.upload_file(source, target)
                got = await environment.exec(f"wc -c < {shlex.quote(target)}", user="root")
                if (got.stdout or "").strip() == str(size):
                    return
            except RuntimeError as exc:
                self.logger.debug(f"upload of {target} failed (attempt {attempt + 1}): {exc}")
            await asyncio.sleep(1 + attempt)
        raise RuntimeError(f"could not upload {target} intact")

    @override
    async def install(self, environment: BaseEnvironment) -> None:
        arch = (await environment.exec("uname -m", user="root")).stdout or ""
        goarch = "arm64" if arch.strip() in ("aarch64", "arm64") else "amd64"
        binary = BIN_DIR / f"goh-linux-{goarch}"
        if not binary.is_file():
            raise RuntimeError(f"{binary} is missing; build it with bench/harbor_agent/run.sh")
        await self._upload(environment, binary, "/usr/local/bin/goh")
        await self.exec_as_root(environment, "chmod 755 /usr/local/bin/goh && goh --version")

    async def _place_token(self, environment: BaseEnvironment) -> None:
        """Exports a fresh access-only sign-in here and installs it for the agent user."""
        async with Goh._export_lock:
            with tempfile.TemporaryDirectory() as tmp:
                path = Path(tmp) / "chatgpt-account.json"
                done = await asyncio.to_thread(
                    subprocess.run,
                    [HOST_GOH, "auth", "export", str(path), TOKEN_VALID],
                    capture_output=True, text=True, timeout=120,
                )
                if done.returncode != 0:
                    raise RuntimeError("goh auth export failed: " + done.stderr.strip())
                await self._upload(environment, path, "/installed-agent/chatgpt-account.json")
        # Uploaded as root; handed to the agent user as a private file.
        await self.exec_as_root(environment, "chmod 644 /installed-agent/chatgpt-account.json")
        await self.exec_as_agent(
            environment,
            f'mkdir -p "{CRED_DIR}" && chmod 700 "{CRED_DIR}" && '
            f'cp /installed-agent/chatgpt-account.json "{CRED_DIR}/chatgpt-account.json.new" && '
            f'chmod 600 "{CRED_DIR}/chatgpt-account.json.new" && '
            f'mv "{CRED_DIR}/chatgpt-account.json.new" "{CRED_DIR}/chatgpt-account.json"',
        )
        await self.exec_as_root(environment, "rm -f /installed-agent/chatgpt-account.json")

    @override
    async def configure(self, environment: BaseEnvironment) -> None:
        await self.exec_as_root(environment, f"mkdir -p {Path(CA_PATH).parent}")
        await self._upload(environment, HOST_CA, CA_PATH)
        await self.exec_as_root(environment, f"chmod 644 {CA_PATH}")
        # Terminal-Bench bounds each task by time; goh's own turn cap (64 by
        # default) is raised to its maximum so it does not end a run first.
        await self.exec_as_agent(
            environment,
            'mkdir -p "$HOME/.config/independent-harness" && '
            'printf \'{"max_turns": 256}\\n\' > "$HOME/.config/independent-harness/config.json"',
        )
        await self._place_token(environment)

    async def _keep_token_fresh(self, environment: BaseEnvironment) -> None:
        while True:
            await asyncio.sleep(TOKEN_EVERY_SEC)
            try:
                await self._place_token(environment)
            except Exception as exc:  # the run reports its own failure if it expires
                self.logger.warning(f"goh token replacement failed: {exc}")

    def _time_limit(self) -> int | None:
        """The task's agent time limit in seconds, from the trial's config and task.toml."""
        try:
            trial = json.loads((self.logs_dir.parent / "config.json").read_text())
            limit = trial.get("agent", {}).get("override_timeout_sec")
            if limit is None:
                path = trial["task"]["path"]
                for toml in TASK_CACHE.glob(f"*/{path}/task.toml"):
                    limit = tomllib.loads(toml.read_text()).get("agent", {}).get("timeout_sec")
                    break
            if limit is None:
                return None
            limit *= trial.get("agent_timeout_multiplier") or trial.get("timeout_multiplier") or 1
            return max(60, int(limit) - STOP_MARGIN_SEC)
        except (OSError, KeyError, ValueError, TypeError) as exc:
            self.logger.debug(f"no time limit for goh: {exc}")
            return None

    @override
    @with_prompt_template
    async def run(self, instruction: str, environment: BaseEnvironment, context: AgentContext) -> None:
        effort = getattr(self.options, "reasoning_effort", None) or "medium"
        has_ca = " || ".join(f"[ -s {p} ]" for p in SYSTEM_CA)
        limit = self._time_limit()
        names = getattr(self.options, "experiment", None) or ""
        experiments = "".join(f"--experiment {shlex.quote(n.strip())} " for n in names.split(",") if n.strip())
        budget = f"--max-run-seconds {limit} " if limit else ""
        command = (
            f'if ! {{ {has_ca}; }}; then export SSL_CERT_FILE={CA_PATH}; fi; '
            'goh run --project "$PWD" --prompt "$GOH_TASK" --format json '
            "--protocol openai_chatgpt "
            f"--model {shlex.quote(self._model())} --reasoning-effort {shlex.quote(effort)} "
            f"--access full --allow all --keep-jobs {budget}{experiments}--session-dir /logs/agent/sessions "
            "> /logs/agent/goh.jsonl 2> /logs/agent/goh.stderr; "
            "status=$?; tail -c 2000 /logs/agent/goh.stderr; exit $status"
        )
        refresher = asyncio.create_task(self._keep_token_fresh(environment))
        try:
            await self.exec_as_agent(
                environment, "mkdir -p /logs/agent/sessions && " + command,
                env={"GOH_TASK": instruction},
            )
        finally:
            refresher.cancel()

    @override
    def populate_context_post_run(self, context: AgentContext) -> None:
        inp = cached = out = requests = 0
        for journal in (self.logs_dir / "sessions").rglob("*.jsonl"):
            for line in journal.read_text(errors="replace").splitlines():
                try:
                    record = json.loads(line)
                except ValueError:
                    continue
                if record.get("type") != "usage":
                    continue
                data = record.get("data") or {}
                inp += int(data.get("prompt_tokens") or 0)
                cached += int(data.get("cache_read_tokens") or 0)
                out += int(data.get("completion_tokens") or data.get("output_tokens") or 0)
                requests += 1
        outcome = None
        events = self.logs_dir / "goh.jsonl"
        if events.is_file():
            for line in events.read_text(errors="replace").splitlines():
                try:
                    event = json.loads(line)
                except ValueError:
                    continue
                if event.get("type") == "run_finished":
                    outcome = event.get("data")
        context.n_input_tokens, context.n_cache_tokens, context.n_output_tokens = inp, cached, out
        context.metadata = {"requests": requests, "run_finished": outcome}
