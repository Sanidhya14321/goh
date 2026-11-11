#!/usr/bin/env python3
"""`goh attach` end to end: a real `goh serve`, the terminal UI attached to it
in a pseudo-terminal, and the scripted local model (no credentials).

The model edits note.txt (approved in the attached UI) and runs a check
(denied there). The edit must land in the server's project, the server must
write the journal, and the attached UI must show the run."""
import fcntl, json, os, pathlib, pty, select, signal, socket, struct, subprocess, sys, tempfile, termios, time

sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
from vtscreen import Screen

base = pathlib.Path(__file__).resolve().parents[1]
goh, model_bin = base / 'bin/goh', base / 'bin/scripted-model'


def free_port():
    with socket.socket() as s:
        s.bind(('127.0.0.1', 0))
        return s.getsockname()[1]


def wait_port(port, timeout=10):
    deadline = time.monotonic() + timeout
    while time.monotonic() < deadline:
        try:
            socket.create_connection(('127.0.0.1', port), .2).close()
            return
        except OSError:
            time.sleep(.05)
    raise AssertionError(f'port {port} never opened')


with tempfile.TemporaryDirectory(prefix='goh-attach-') as temporary:
    root = pathlib.Path(temporary)
    project, sessions, home = root / 'project', root / 'sessions', root / 'home'
    for d in (project, sessions, home):
        d.mkdir()
    (project / 'note.txt').write_text('alpha\n')
    model_port, server_port = free_port(), free_port()
    model = subprocess.Popen([str(model_bin), '--listen', f'127.0.0.1:{model_port}', '--scenario', 'edit'], stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
    config = root / 'config.json'
    config.write_text(json.dumps({'endpoint': f'http://127.0.0.1:{model_port}/v1/chat/completions', 'model': 'scripted', 'context_window': 131072}))
    token = 'attach-test-token-' + 'x' * 32
    env = dict(os.environ, HOME=str(home), XDG_CONFIG_HOME=str(home / '.config'), HARNESS_SERVER_TOKEN=token, TERM='xterm-256color', COLORTERM='truecolor')
    server = subprocess.Popen([str(goh), 'serve', '--listen', f'127.0.0.1:{server_port}', '--project', str(project), '--config', str(config), '--session-dir', str(sessions)], env=env, stdout=subprocess.DEVNULL, stderr=subprocess.PIPE)
    attach = None
    try:
        wait_port(model_port)
        wait_port(server_port)
        master, slave = pty.openpty()
        fcntl.ioctl(slave, termios.TIOCSWINSZ, struct.pack('HHHH', 30, 100, 0, 0))
        screen = Screen(30, 100)
        attach = subprocess.Popen([str(goh), 'attach', f'127.0.0.1:{server_port}'], env=env, stdin=slave, stdout=slave, stderr=slave)
        os.close(slave)

        def wait_for(needle, timeout=10):
            deadline = time.monotonic() + timeout
            while time.monotonic() < deadline:
                if needle in screen.text():
                    return
                if select.select([master], [], [], .05)[0]:
                    try:
                        screen.feed(os.read(master, 65536))
                    except OSError:
                        break
            raise AssertionError(f'missing {needle!r} on screen:\n{screen.text()}')

        wait_for('Attached to')
        os.write(master, b'Edit the note')
        time.sleep(.2)
        os.write(master, b'\r')
        wait_for('Allow edit_file?')
        os.write(master, b'y')
        wait_for('Allow shell?')
        os.write(master, b'n')
        wait_for('Fixture edit and verification complete')
        assert (project / 'note.txt').read_text() == 'beta\n', 'approved edit not applied on the server'
        journals = list(sessions.glob('*.jsonl'))
        assert len(journals) == 1, journals
        records = [json.loads(line) for line in journals[0].read_text().splitlines()]
        perms = [r['data'] for r in records if r['type'] == 'permission']
        assert any(p.get('tool') == 'edit_file' and p.get('allowed') for p in perms), perms
        assert any(p.get('tool') == 'shell' and not p.get('allowed') for p in perms), perms
        # Local-write commands are refused while attached.
        os.write(master, b'!echo hi')
        time.sleep(.2)
        os.write(master, b'\r')
        wait_for('not available while attached')
        os.write(master, b'\x04')
        attach.wait(timeout=10)
        print(json.dumps({'attach': 'passed', 'edit_approved_on_server': True, 'shell_denied': True, 'local_writes_refused': True}))
    finally:
        if attach and attach.poll() is None:
            attach.kill()
        server.send_signal(signal.SIGTERM)
        try:
            server.wait(timeout=20)
        except subprocess.TimeoutExpired:
            server.kill()
        model.kill()
