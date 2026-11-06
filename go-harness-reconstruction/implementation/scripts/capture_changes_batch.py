#!/usr/bin/env python3
"""Original local Go formatter/checkpoint REA captures and selected assertions."""
import json,os,pathlib,shutil,subprocess,time,urllib.request,urllib.error,socket
root=pathlib.Path(__file__).resolve().parents[2];research=root/'research';impl=root/'implementation';evidence=research/'evidence'
rea=research/'tooling/rea/node_modules/.bin/rea';project=research/'formatter-go-project';project.mkdir(exist_ok=True)
sessions=research/'formatter-go-sessions';reports=[]
for path in ['normal.fmtcase','failure.failcase','unmatched.other','formatter-invocations.jsonl']:(project/path).unlink(missing_ok=True)
script=research/'formatter-go-script.json'
calls=[]
for i,name in enumerate(['normal.fmtcase','failure.failcase','unmatched.other']):
 calls.append({'calls':[{'id':f'formatter-{i}','type':'function','function':{'name':'create_file','arguments':json.dumps({'path':name,'content':'raw input\n'})}}]})
calls.append({'text':'Formatter fixture complete.'});script.write_text(json.dumps(calls))
with socket.socket() as sock:sock.bind(('127.0.0.1',0));port=sock.getsockname()[1]
config=research/'formatter-go-config.json'
config.write_text(json.dumps({'endpoint':f'http://127.0.0.1:{port}/v1/chat/completions','model':'scripted','context_window':131072,'snapshots':True,'formatters':{name:{'command':shutil.which('node'),'args':[str(research/'formatter-command.mjs'),'$FILE'],'extensions':[ext],'env':{'FORMATTER_FIXTURE_MODE':mode}} for name,ext,mode in [('fixture','.fmtcase','normal'),('fixture_fail','.failcase','fail')]}}))
def capture(name,args):
 scenario=json.loads((research/'formatter-enabled-scenario.json').read_text());scenario.update(executable=str(impl/'bin/harness'),arguments=args,working_directory=str(project),environment={},filesystem_observation_paths=[str(project)])
 scenario['limits']['output_bytes']=1048576
 path=research/'scenarios'/f'{name}.json';path.write_text(json.dumps(scenario,indent=2))
 with (evidence/f'{name}.json').open('w') as out,(evidence/f'{name}.progress.jsonl').open('w') as err:subprocess.run([str(rea),'capture-process',str(path),'--format','json'],stdout=out,stderr=err,check=True)
 result=json.loads((evidence/f'{name}.json').read_text());reports.append({'name':name,'evidence_id':result['evidence_id'],'exit':result['normalized_result']['exit']});return result
fixture=subprocess.Popen([str(impl/'bin/scripted-model'),'--listen',f'127.0.0.1:{port}','--script',str(script)],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
try:
 for attempt in range(100):
  if fixture.poll() is not None:raise RuntimeError('fixture died')
  try:urllib.request.urlopen(f'http://127.0.0.1:{port}/v1/chat/completions',timeout=.1)
  except urllib.error.HTTPError:break
  except OSError:time.sleep(.02)
 result=capture('go-formatters',['run','--config',str(config),'--project',str(project),'--session-dir',str(sessions),'--format','json','--allow','create_file,format:fixture,format:fixture_fail','--prompt','Write the three formatter fixture files.'])
 assert result['normalized_result']['exit']['code']==0
finally:
 fixture.terminate()
 try:fixture.wait(timeout=5)
 except subprocess.TimeoutExpired:fixture.kill();fixture.wait()
assert (project/'normal.fmtcase').read_text()=='RAW INPUT\n'
assert (project/'failure.failcase').read_text()=='RAW INPUT\n'
assert (project/'unmatched.other').read_text()=='raw input\n'
journal=max(sessions.glob('*.jsonl'),key=lambda x:x.stat().st_mtime)
records=[json.loads(line) for line in journal.read_text().splitlines()]
snapshot_id=next(r['data']['id'] for r in records if r['type']=='snapshot_captured' and r['data']['paths']==['normal.fmtcase'])
subject=journal.stem
common=[subject,'--session-dir',str(sessions),'--id',snapshot_id]
assert capture('go-snapshot-undo',['changes','undo',*common])['normalized_result']['exit']['code']==0
assert not (project/'normal.fmtcase').exists()
assert capture('go-snapshot-redo',['changes','redo',*common])['normalized_result']['exit']['code']==0
assert (project/'normal.fmtcase').read_text()=='RAW INPUT\n'
(project/'normal.fmtcase').write_text('EXTERNAL CONTENT\n')
assert capture('go-snapshot-conflict',['changes','undo',*common])['normalized_result']['exit']['code']==1
assert (project/'normal.fmtcase').read_text()=='EXTERNAL CONTENT\n'
(evidence/'changes-batch-driver.json').write_text(json.dumps(reports,indent=2)+'\n')
projection={'kind':'independent selected-fact projection, not canonical REA equivalence','checks':{'normal_formatter_applies':True,'failed_formatter_preserves_write_and_formatted_bytes':True,'unmatched_extension_unchanged':True,'undo_creation_then_redo_final_formatted_bytes':True,'external_edit_conflict_preserved':True},'captures':reports,'limitations':['Explicit formatter configuration/admission and alphabetical order differ.','Per-file snapshots are narrower than whole Git tree and message-range revert.','No whole-repository parity or global absence-of-effects inference.']}
(evidence/'changes-semantic-comparisons.json').write_text(json.dumps(projection,indent=2)+'\n')
print(json.dumps(reports,indent=2))
