#!/usr/bin/env python3
"""Exercise only the newly built application in disposable pseudo-terminals."""
import json, os, pathlib, pty, select, signal, subprocess, tempfile, termios, time, urllib.request

base = pathlib.Path(__file__).resolve().parents[1]
reports = []
def live_records(path):
    raw=path.read_bytes()
    # A live reader may see an unterminated record while the writer appends;
    # inspect only complete lines, without repairing/modifying the journal.
    raw=raw[:raw.rfind(b'\n')+1]
    return [json.loads(line) for line in raw.splitlines()]
with tempfile.TemporaryDirectory(prefix='harness-tui-') as temporary:
    root = pathlib.Path(temporary)
    (root / 'note.txt').write_text('alpha\n')
    script = root / 'script.json'
    script.write_text(json.dumps([{'calls': [{'id': 'tui-shell', 'type': 'function', 'function': {'name': 'shell', 'arguments': json.dumps({'command': 'sleep 30', 'timeout_seconds': 40})}}]}]))
    task_script = root / 'tasks.json'
    task_script.write_text(json.dumps([{'calls':[{'id':'tui-todo','type':'function','function':{'name':'todowrite','arguments':json.dumps({'todos':[{'content':'Verify task overview','status':'in_progress','priority':'high'}]})}}]}, {'text':'Task fixture complete.'}, {'text':'Task fixture summary.'}]))
    question_script = root / 'questions.json'
    question_script.write_text(json.dumps([{'calls':[{'id':'tui-question','type':'function','function':{'name':'question','arguments':json.dumps({'questions':[{'header':'Checks','question':'Pick verification checks','multiple':True,'options':[{'label':'unit','description':'Focused tests'},{'label':'race','description':'Concurrency checks'}]},{'header':'Choice','question':'Pick one action','options':[{'label':'continue','description':'Continue task'}]},{'header':'Notes','question':'Provide extra context','options':[]}]})}}]}, {'text':'Question fixture complete.'}]))
    child_script=root/'child.json'
    child_script.write_text(json.dumps([{'calls':[{'id':'launch-child','type':'function','function':{'name':'task','arguments':json.dumps({'description':'Create controlled child output','prompt':'Create controlled output','subagent_type':'general'})}}]}, {'calls':[{'id':'child-create','type':'function','function':{'name':'create_file','arguments':json.dumps({'path':'subagent-output','content':'child output'})}}]}, {'text':'Child fixture complete.'}, {'text':'Parent fixture complete.'}]))
    child_cancel_script=root/'child-cancel.json'
    child_cancel_script.write_text(json.dumps([{'calls':[{'id':'launch-child','type':'function','function':{'name':'task','arguments':json.dumps({'description':'Controlled child cancellation','prompt':'Wait for cancellation','subagent_type':'general'})}}]}, {'calls':[{'id':'child-shell','type':'function','function':{'name':'shell','arguments':json.dumps({'command':'printf %s $$ > child-pid; sleep 30','timeout_seconds':40})}}]}]))
    typed_script=root/'typed-script.json'
    typed_script.write_text(json.dumps([{'calls':[{'id':'typed-launch','type':'function','function':{'name':'task','arguments':json.dumps({'description':'Strict JSON child result','prompt':'Return JSON evidence','subagent_type':'explore','output_schema':{'type':'object','properties':{'ok':{'type':'boolean'}},'required':['ok'],'additionalProperties':False}})}}]},{'text':'{"ok":true}'},{'text':'Typed parent fixture complete.'}]))
    formatter=root/'formatter-fixture' 
    formatter.write_text('#!/bin/sh\nprintf "%s" $$ > formatter-pid\nprintf "FORMATTED" > "$1"\nif [ "$MODE" = cancel ]; then sleep 30; fi\n')
    formatter.chmod(0o700)
    formatter_script=root/'formatter-script.json'
    formatter_script.write_text(json.dumps([{'calls':[{'id':'create-format','type':'function','function':{'name':'create_file','arguments':json.dumps({'path':'format.fmtcase','content':'raw'})}}]},{'text':'Formatter fixture complete.'}]))
    import tempfile
    outside_sessions=pathlib.Path(tempfile.mkdtemp(prefix='harness-undo-sessions-'))
    queue_script=root/'queue-script.json'
    queue_script.write_text(json.dumps([{'text':'First reply.','delay_ms':2500},{'text':'Second reply.'}]))
    undo_script=root/'undo-script.json'
    undo_script.write_text(json.dumps([{'calls':[{'id':'undo-create','type':'function','function':{'name':'create_file','arguments':json.dumps({'path':'undo-target.txt','content':'made in turn one'})}}]},{'text':'Undo fixture complete.'}]))
    for case, scenario in [('text', 'text'), ('permission_denial', 'shell'), ('edit_approval', 'edit'), ('shell_cancellation', 'script'), ('termination', 'text'), ('tasks_and_compaction', 'tasks'), ('question_answers', 'questions'), ('question_dismissal', 'questions'), ('question_cancellation', 'questions'), ('subagent_approval','child'), ('subagent_cancellation','child-cancel'), ('formatter_approval','formatter'), ('formatter_denial','formatter'), ('formatter_cancellation','formatter'), ('typed_child','typed'), ('undo_redo','undo'), ('editor_handoff','text'), ('theme_light','text'), ('queued_prompt','queue'), ('timestamps','text'), ('timeline_revert','undo'), ('user_shell','text'), ('background_job','job'), ('provider_models','text'), ('plan_exit','planexit')]:
        # Reserve an ephemeral fixture port; it is local-only and credential-free.
        import socket
        with socket.socket() as sock:
            sock.bind(('127.0.0.1', 0))
            port = sock.getsockname()[1]
        arguments = [str(base / 'bin/scripted-model'), '--listen', f'127.0.0.1:{port}']
        arguments += ['--script', str(queue_script if scenario=='queue' else undo_script if scenario=='undo' else typed_script if scenario=='typed' else formatter_script if scenario=='formatter' else child_cancel_script if scenario=='child-cancel' else child_script if scenario=='child' else question_script if scenario=='questions' else task_script if scenario=='tasks' else script)] if scenario in ('script','tasks','questions','child','child-cancel','formatter','typed','undo','queue') else ['--scenario', scenario]
        fixture = subprocess.Popen(arguments, stdout=subprocess.DEVNULL, stderr=subprocess.DEVNULL)
        master, slave = pty.openpty()
        original = termios.tcgetattr(slave)
        process = None
        output = bytearray()
        def wait_for(needle, timeout=8):
            deadline = time.monotonic() + timeout
            while time.monotonic() < deadline:
                if needle in output:
                    return
                readable, _, _ = select.select([master], [], [], .05)
                if readable:
                    try:
                        data = os.read(master, 65536)
                    except OSError:
                        break
                    output.extend(data)
            raise AssertionError(f'{case}: missing {needle!r}; output tail={bytes(output[-1800:])!r}')
        def wait_reply(needle, timeout=8):
            # The renderer redraws only changed cells, so a reply can reach the
            # terminal in fragments. Accept it on screen or in the journal.
            deadline = time.monotonic() + timeout
            sessions = outside_sessions/case if scenario=='undo' else root/case
            while time.monotonic() < deadline:
                if needle.encode() in output:
                    return
                for j in sessions.glob('*.jsonl'):
                    for r in live_records(j):
                        if r['type']=='message' and r['data'].get('role')=='assistant' and needle in (r['data'].get('content') or ''):
                            return
                readable, _, _ = select.select([master], [], [], .05)
                if readable:
                    try: output.extend(os.read(master, 65536))
                    except OSError: break
            raise AssertionError(f'{case}: missing reply {needle!r}; output tail={bytes(output[-1800:])!r}')
        try:
            for attempt in range(100):
                if fixture.poll() is not None:
                    raise AssertionError('fixture exited before readiness')
                try:
                    urllib.request.urlopen(f'http://127.0.0.1:{port}/v1/chat/completions', timeout=.1)
                except urllib.error.HTTPError:
                    break
                except OSError:
                    time.sleep(.02)
            env = dict(os.environ, TERM='xterm-256color')
            if case=='editor_handoff':
                fake_editor=root/'fake-editor.sh'
                fake_editor.write_text('#!/bin/sh\nprintf "%s" "$(cat "$1") rewritten in editor" > "$1"\n')
                fake_editor.chmod(0o700)
                env['VISUAL']=str(fake_editor)
            extra=[]
            if case=='theme_light':
                theme_config=root/'theme-config.json'
                theme_config.write_text(json.dumps({'theme':'light'}))
                extra=['--config',str(theme_config)]
            if scenario=='planexit':
                extra=['--agent','plan']
            if scenario=='undo':
                extra=['--snapshots']  # snapshot storage must be outside the project
            if scenario=='formatter':
                for filename in ('format.fmtcase','formatter-pid'):
                    (root/filename).unlink(missing_ok=True)
                formatter_config=root/'formatter-config.json'
                formatter_config.write_text(json.dumps({'formatters':{'fixture':{'command':str(formatter),'args':['$FILE'],'extensions':['.fmtcase'],'env':{'MODE':'cancel' if case=='formatter_cancellation' else 'normal'}}}}))
                extra=['--config',str(formatter_config)]
            process = subprocess.Popen([str(base / 'bin/goh'), 'chat', '--project', str(root), '--session-dir', str(outside_sessions/case if scenario=='undo' else root / case), '--endpoint', f'http://127.0.0.1:{port}/v1/chat/completions', '--model', 'scripted', '--context-window', '131072',*extra], stdin=slave, stdout=slave, stderr=slave, env=env)
            wait_for(b'Ctrl+D')
            if case == 'termination':
                process.send_signal(signal.SIGTERM)
            else:
                if case=='editor_handoff':
                    os.write(master, b'Draft')
                    time.sleep(.2)
                    os.write(master, b'\x18')  # Ctrl+X leader
                    time.sleep(.1)
                    os.write(master, b'e')
                    # The welcome screen has no status line; the composer shows
                    # the edited draft once the editor returns.
                    wait_for(b'rewritten in editor')
                    os.write(master, b'\r')
                elif case=='provider_models':
                    time.sleep(.3)
                    os.write(master, b'/models\r')
                    wait_for(b'scripted-large')
                    os.write(master, b'large')
                    time.sleep(.3)
                    os.write(master, b'\r')
                    wait_for(b'Model for this chat: scripted-large')
                    os.write(master, b'Exercise the fixture\r')
                elif case=='user_shell':
                    os.write(master, b'!echo pty-shell-$((40+2))\r')
                else:
                    os.write(master, b'Exercise the fixture\r')
                if case=='typed_child':
                    wait_for(b'APPROVAL: task')
                    os.write(master,b'y')
                    wait_for(b'Typed parent fixture complete')
                    # Validate the root tool result, rather than matching display text.
                    for attempt in range(250):
                        found=False
                        for journal in (root/case).glob('*.jsonl'):
                            for record in live_records(journal):
                                if record['type']=='tool_result' and record['data']['id']=='typed-launch':
                                    envelope=record['data']['result'];result=json.loads(envelope['content'])
                                    assert not envelope.get('error') and result['structured_output']=={'ok':True},result
                                    found=True
                        if found: break
                        time.sleep(.02)
                    assert found,'typed task result not recorded'
                elif case=='queued_prompt':
                    time.sleep(.6)  # the first run is active (reply delayed 2.5 s)
                    os.write(master, b'Second prompt')
                    time.sleep(.2)
                    os.write(master, b'\r')
                    wait_for(b'[queued] Second prompt')
                    # The renderer redraws changed cells only, so a reply can reach
                    # the terminal in fragments; the journal is the source of truth.
                    prompts,replies=[],[]
                    for attempt in range(750):
                        records=[r for j in (root/case).glob('*.jsonl') for r in live_records(j) if r['type']=='message']
                        prompts=[r['data']['content'] for r in records if r['data'].get('role')=='user']
                        replies=[r['data'].get('content','') for r in records if r['data'].get('role')=='assistant']
                        if len(prompts)==2 and any('Second reply.' in x for x in replies): break
                        time.sleep(.02)
                    assert prompts==['Exercise the fixture','Second prompt'],prompts
                    assert any('Second reply.' in x for x in replies),replies
                elif case=='provider_models':
                    wait_reply('Fixture response')
                    models=[]
                    for attempt in range(250):
                        models=[r['data'].get('model') for j in (root/case).glob('*.jsonl') for r in live_records(j) if r['type']=='run_started']
                        if models: break
                        time.sleep(.02)
                    assert models and models[-1]=='scripted-large', models
                elif case=='user_shell':
                    wait_for(b'pty-shell-42')
                    records=[]
                    for attempt in range(250):
                        records=[r for j in (root/case).glob('*.jsonl') for r in live_records(j)]
                        if any(r['type']=='run_finished' for r in records): break
                        time.sleep(.02)
                    assert any(r['type']=='permission' and r['data'].get('source')=='user_typed' for r in records),'typed approval not journaled'
                    assert any(r['type']=='tool_result' for r in records),'shell result not journaled'
                    assert not any(r['type']=='message' and r['data'].get('role')=='assistant' and r['data'].get('content') for r in records),'a model reply was requested'
                elif case=='timestamps':
                    wait_reply('Fixture response')
                    time.sleep(.3)
                    os.write(master, b'/timestamps\r')
                    wait_for(b'Timestamps shown')
                    import re
                    assert re.search(rb'\xc2\xb7 (Mon|Tue|Wed|Thu|Fri|Sat|Sun) \d\d:\d\d', output), 'turn time not rendered'
                    stamped=[r for j in (root/case).glob('*.jsonl') for r in live_records(j)]
                    assert stamped and all(r['version']==2 and r['time']>0 for r in stamped), 'journal records lack times'
                elif case=='theme_light':
                    wait_reply('Fixture response')
                    assert b'48;2;250;250;250' in output, 'light theme background not rendered'
                    assert b'48;2;10;10;10m' not in output[-4000:], 'dark background still rendered after start'
                elif case=='editor_handoff':
                    wait_reply('Fixture response')
                    sent=[r['data']['content'] for j in (root/case).glob('*.jsonl') for r in live_records(j) if r['type']=='message' and r['data'].get('role')=='user']
                    assert sent==['Draft rewritten in editor'],sent
                elif case=='timeline_revert':
                    target=root/'undo-target.txt'
                    target.unlink(missing_ok=True)
                    wait_for(b'APPROVAL: create_file')
                    os.write(master,b'y')
                    wait_reply('fixture complete.')
                    for attempt in range(250):
                        settled=any(r['type']=='run_finished' for j in (outside_sessions/case).glob('*.jsonl') for r in live_records(j))
                        if settled: break
                        time.sleep(.02)
                    assert settled,'run did not finish'
                    time.sleep(.3)
                    assert target.read_text()=='made in turn one'
                    os.write(master,b'\x14')  # Ctrl+T timeline
                    wait_for(b'Timeline')
                    os.write(master,b'\r')  # preview the latest prompt
                    wait_for(b'revert to here')
                    os.write(master,b'\x15')  # Ctrl+U revert
                    wait_for(b'Undid turn 1')
                    assert not target.exists(),'revert left the file'
                    wait_for(b'Exercise the fixture')  # prompt back in the composer
                elif case=='undo_redo':
                    target=root/'undo-target.txt'
                    target.unlink(missing_ok=True)
                    wait_for(b'APPROVAL: create_file')
                    os.write(master,b'y')
                    wait_reply('fixture complete.')  # the renderer may split the line
                    # Type /undo only once the run has settled; while running the
                    # chat refuses rewind and asks the user to wait.
                    for attempt in range(250):
                        settled=any(r['type']=='run_finished' for j in (outside_sessions/case).glob('*.jsonl') for r in live_records(j))
                        if settled: break
                        time.sleep(.02)
                    assert settled,'run did not finish'
                    time.sleep(.3)
                    assert target.read_text()=='made in turn one'
                    os.write(master,b'/undo\r')
                    wait_for(b'Undid turn 1')
                    assert not target.exists(),'undo left the file'
                    wait_for(b'Exercise the fixture')  # prompt restored to the composer
                    os.write(master,b'\x10')  # Ctrl+P palette
                    time.sleep(.3)
                    os.write(master,b'redo')
                    time.sleep(.3)
                    os.write(master,b'\r')
                    wait_for(b'Redid turn 1')
                    assert target.read_text()=='made in turn one','redo did not restore the file'
                elif case.startswith('formatter_'):
                    wait_for(b'APPROVAL: create_file')
                    os.write(master,b'y')
                    wait_for(b'format:fixture')
                    os.write(master,b'n' if case=='formatter_denial' else b'y')
                    if case=='formatter_cancellation':
                        deadline=time.monotonic()+5
                        while not (root/'formatter-pid').exists() and time.monotonic()<deadline:
                            time.sleep(.02)
                        assert (root/'formatter-pid').exists(),'formatter never started'
                        os.write(master,b'\x03')
                        wait_for(b'cancelled')
                        assert process.poll() is None,'formatter cancel exited application'
                    else:
                        wait_for(b'Formatter fixture')
                        wait_for(b'complete.')
                        assert (root/'format.fmtcase').read_text()==('raw' if case=='formatter_denial' else 'FORMATTED')
                elif case.startswith('subagent_'):
                    wait_for(b'APPROVAL: task')
                    os.write(master,b'y')
                    if case=='subagent_approval':
                        wait_for(b'APPROVAL: child:create_file')
                        os.write(master,b'y')
                        wait_for(b'Parent fixture complete')
                        wait_for(b'completed')
                        assert (root/'subagent-output').read_text()=='child output'
                    else:
                        wait_for(b'APPROVAL: child:shell')
                        os.write(master,b'y')
                        wait_for(b'tool_started')
                        for attempt in range(250):
                            if (root/'child-pid').exists(): break
                            time.sleep(.02)
                        assert (root/'child-pid').exists(),'child shell never executed'
                        os.write(master,b'\x03')
                        wait_for(b'cancelled')
                        assert process.poll() is None,'child cancel exited TUI'
                elif case.startswith('question_'):
                    wait_for(b'INTERACTIVE QUESTION')
                    if case == 'question_answers':
                        os.write(master,b' \x1b[B \r')
                        wait_for(b'2/3:')
                        os.write(master,b'\r')
                        wait_for(b'3/3:')
                        os.write(master,b'c')
                        wait_for(b'Custom answer:')
                        os.write(master,b'custom context\r')
                        wait_for(b'completed')
                        records=live_records(next((root/case).glob('*.jsonl')))
                        replies=[record['data'] for record in records if record['type']=='question_resolved']
                        assert replies[-1]['answers']==[['unit','race'],['continue'],['custom context']],replies
                    elif case=='question_dismissal':
                        os.write(master,b'\x1b')
                        wait_for(b'dismissed')
                        wait_for(b'completed')
                    else:
                        os.write(master,b'\x03')
                        wait_for(b'cancelled')
                        assert process.poll() is None,'question cancellation exited TUI'
                elif case == 'tasks_and_compaction':
                    wait_for(b'APPROVAL: todowrite')
                    os.write(master,b'y')
                    wait_for(b'Tasks: 1 remaining / 1 total')
                    wait_for(b'completed')
                    output.clear()
                    os.write(master,b'/todos\r')
                    wait_for(b'Session tasks:')
                    output.clear()
                    os.write(master,b'/compact\r')
                    wait_for(b'context_compacted')
                    # Incremental terminal rendering can retain the previous
                    # 'completed' cells without writing that word a second time.
                    # Verify the durable new operation outcome instead.
                    finished = False
                    for attempt in range(100):
                        journals = list((root / case).glob('*.jsonl'))
                        if journals:
                            records = live_records(journals[0])
                            compact_indices = [i for i,r in enumerate(records) if r['type']=='context_compacted']
                            if compact_indices and any(r['type']=='run_finished' and r['data']['status']=='completed' for r in records[compact_indices[-1]+1:]):
                                finished=True; break
                        time.sleep(.02)
                    assert finished, 'manual compaction did not finish durably'
                    time.sleep(.05)
                elif case == 'plan_exit':
                    wait_for(b'APPROVAL: plan_exit')
                    os.write(master, b'y')
                    wait_for(b'Switched to Build')
                    wait_for(b'Carry out the approved plan.')
                    starts=[]
                    for attempt in range(250):
                        starts=[r['data'].get('agent_mode') for j in (root/case).glob('*.jsonl') for r in live_records(j) if r['type']=='run_started']
                        if starts: break
                        time.sleep(.02)
                    assert starts and starts[0]=='plan', starts
                    sessions=list((root/case).glob('*.jsonl'))
                    assert len(sessions)==1, 'handoff must continue the same session'
                elif case == 'background_job':
                    (root/'job.pid').unlink(missing_ok=True)
                    wait_for(b'APPROVAL: job_start')
                    os.write(master, b'y')
                    wait_reply('Background job started')
                    for attempt in range(250):
                        if (root/'job.pid').exists() and (root/'job.pid').read_text().strip(): break
                        time.sleep(.02)
                    job_pid=int((root/'job.pid').read_text())
                    os.kill(job_pid, 0)  # running while the chat is open
                elif case == 'edit_approval':
                    wait_for(b'APPROVAL: edit_file')
                    os.write(master, b'y')
                    wait_for(b'APPROVAL: shell')
                    os.write(master, b'n')
                    wait_for(b'completed')
                    assert (root / 'note.txt').read_text() == 'beta\n'
                elif case == 'permission_denial':
                    wait_for(b'APPROVAL: shell')
                    os.write(master, b'n')
                    wait_for(b'completed')
                elif case == 'shell_cancellation':
                    wait_for(b'APPROVAL: shell')
                    os.write(master, b'y')
                    wait_for('▸ shell'.encode())
                    os.write(master, b'\x1b')
                    wait_for(b'cancelled')
                    if process.poll() is not None:
                        raise AssertionError('first cancel exited application')
                else:
                    wait_reply('Fixture response')
                    wait_for(b'completed')
                # A rendered 'completed' can belong to a child or precede the
                # UI worker's done message. Wait for the actual parent terminal
                # journal record, while continuing to drain the pseudo-terminal.
                parent_finished=False
                for attempt in range(500):
                    for journal in (outside_sessions/case if scenario=='undo' else root/case).glob('*.jsonl'):
                        records=live_records(journal)
                        if any(r['type']=='subagent_parent' for r in records): continue
                        if any(r['type']=='run_finished' for r in records): parent_finished=True
                    if parent_finished: break
                    readable,_,_=select.select([master],[],[],.02)
                    if readable:
                        try: output.extend(os.read(master,65536))
                        except OSError: pass
                assert parent_finished,case+': parent run did not finish durably'
                # Ctrl+D while the done event is still queued is a cancel action;
                # repeat the separate exit action only after work has completed.
                for attempt in range(40):
                    if process.poll() is not None: break
                    os.write(master,b'\x04')
                    readable,_,_=select.select([master],[],[],.1)
                    if readable:
                        try: output.extend(os.read(master,65536))
                        except OSError: pass
            code = process.wait(timeout=8)
            after = termios.tcgetattr(slave)
            # Darwin sets PENDIN when restoring canonical mode after raw mode.
            # Ignore only that transient input-reprocessing flag; compare every
            # other mode, speed and control character exactly.
            pending_bit = getattr(termios, "PENDIN", 0)
            after[3] &= ~pending_bit
            expected = list(original)
            expected[3] &= ~pending_bit
            restored = after == expected
            if not restored:
                raise AssertionError(f'{case}: terminal attributes not restored: before={original!r}; after={termios.tcgetattr(slave)!r}')
            if case == 'background_job':
                for attempt in range(250):
                    try: os.kill(job_pid, 0)
                    except ProcessLookupError: break
                    time.sleep(.02)
                else:
                    raise AssertionError('background job outlived GoHarness')
            if case != 'termination' and code != 0:
                raise AssertionError(f'{case}: unexpected exit {code}')
            reports.append({'scenario': case, 'exit_code': code, 'terminal_attributes_restored': restored})
        finally:
            if process is not None and process.poll() is None:
                process.terminate()
                try: process.wait(timeout=15)
                except subprocess.TimeoutExpired: process.kill(); process.wait()
            fixture.terminate()
            try:
                fixture.wait(timeout=3)
            except subprocess.TimeoutExpired:
                fixture.kill(); fixture.wait()
            os.close(master); os.close(slave)
print(json.dumps(reports, indent=2))
