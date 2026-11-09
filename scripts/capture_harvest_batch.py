#!/usr/bin/env python3
"""Independent local mixed-protocol/native/role/schema CLI REA observations."""
import json,os,pathlib,socket,subprocess,time,urllib.request,urllib.error
root=pathlib.Path(__file__).resolve().parents[1];r=root/'research';e=r/'evidence';binary=root/'bin/harness';rea=r/'tooling/rea/node_modules/.bin/rea'
project=r/'native-go-project';project.mkdir(exist_ok=True);(project/'target.txt').write_text('native fixture');(project/'must-not-exist').unlink(missing_ok=True)
sessions=r/'native-go-sessions';log=e/'native-go-requests.jsonl';log.write_text('');reports=[]
with socket.socket() as sock:sock.bind(('127.0.0.1',0));port=sock.getsockname()[1]
base=f'http://127.0.0.1:{port}'
profile={'protocol':'anthropic_messages','endpoint':base+'/messages','model':'native-child','context_window':32768,'api_key_env':'NATIVE_FIXTURE_KEY'}
config=r/'native-go-config.json';config.write_text(json.dumps({'endpoint':base+'/chat','model':'chat-parent','api_key_env':'PARENT_FIXTURE_KEY','context_window':131072,'profiles':{'native':profile,'compact':dict(profile,endpoint=base+'/compact',model='native-summary')},'model_roles':{'task':'native','compaction':'compact'}}))
environment={k:v for k,v in os.environ.items() if not k.startswith('HARNESS_')}
fixture=subprocess.Popen(['node',str(r/'native-go-model.mjs')],env=dict(environment,FIXTURE_PORT=str(port),FIXTURE_LOG=str(log)),stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
common=['--session-dir',str(sessions),'--project',str(project),'--format','json']
def capture(name,args):
 scenario=json.loads((r/'scenarios/go-formatters.json').read_text());scenario.update(executable=str(binary),arguments=args,working_directory=str(project),environment={'PARENT_FIXTURE_KEY':'parent-fixture-only','NATIVE_FIXTURE_KEY':'native-fixture-only'},filesystem_observation_paths=[str(project)])
 scenario['limits']['output_bytes']=1048576
 path=r/'scenarios'/f'{name}.json';path.write_text(json.dumps(scenario,indent=2))
 with (e/f'{name}.json').open('w') as out,(e/f'{name}.progress.jsonl').open('w') as err:subprocess.run([str(rea),'capture-process',str(path),'--format','json'],env=environment,stdout=out,stderr=err,check=True)
 data=json.loads((e/f'{name}.json').read_text());reports.append({'scenario':name,'evidence_id':data['evidence_id'],'exit':data['normalized_result']['exit']});return data
try:
 for attempt in range(100):
  if fixture.poll() is not None:raise RuntimeError('fixture exited')
  try:urllib.request.urlopen(base+'/ready',timeout=.1)
  except urllib.error.HTTPError:break
  except OSError:time.sleep(.02)
 for case in ['valid','invalid']:
  data=capture(f'go-harvest-{case}',['run','--config',str(config),'--allow','task','--prompt','Typed investigation '+('INVALID_RESULT' if case=='invalid' else ''),*common]);assert data['normalized_result']['exit']['code']==0
  journal=max(sessions.glob('*.jsonl'),key=lambda p:p.stat().st_mtime)
  records=[json.loads(line) for line in journal.read_text().splitlines()]
  # Root task result retains the child identity and schema truth even on failure.
  result=next(x['data']['result'] for x in records if x['type']=='tool_result' and x['data']['id']=='typed-task')
  body=json.loads(result['content']);assert body['child_status']=='completed'
  if case=='valid':assert not result.get('error') and body['structured_output']=={'observed':'native fixture'}
  else:assert result['error'] and body['status']=='invalid_output' and 'structured_output' not in body
  if case=='valid':
   compact=capture('go-harvest-compaction',['compact',journal.stem,'--config',str(config),*common]);assert compact['normalized_result']['exit']['code']==0
   after=[json.loads(line) for line in journal.read_text().splitlines()]
   assert any(x['type']=='model_route' and x['data']['model']=='native-summary' for x in after)
 for case,wanted in [('messages',0),('partial',1),('unauthorized',1),('redirect',1)]:
  native=r/'native-go-direct.json';native.write_text(json.dumps({'protocol':'anthropic_messages','endpoint':base+'/'+case,'model':'native-direct','context_window':131072,'api_key_env':'NATIVE_FIXTURE_KEY'}))
  data=capture(f'go-native-{case}',['run','--config',str(native),'--prompt','Read target.txt','--allow','create_file',*common]);assert data['normalized_result']['exit']['code']==wanted
 assert not (project/'must-not-exist').exists()
 requests=[json.loads(line) for line in log.read_text().splitlines()]
 assert not any(x['path']=='/should-not-hit' for x in requests)
 for request in requests:
  headers=request['headers']
  if request['path']=='/chat':assert headers.get('authorization')=='Bearer parent-fixture-only' and 'x-api-key' not in headers
  else:assert headers.get('x-api-key')=='native-fixture-only' and 'authorization' not in headers and headers.get('anthropic-version')=='2023-06-01'
 assert sum(x['path']=='/partial' for x in requests)==1
 assert sum(x['path']=='/unauthorized' for x in requests)==1
 assert sum(x['path']=='/redirect' for x in requests)==1
finally:
 fixture.terminate()
 try:fixture.wait(timeout=5)
 except subprocess.TimeoutExpired:fixture.kill();fixture.wait()
(e/'harvest-batch-driver.json').write_text(json.dumps(reports,indent=2)+'\n')
(e/'harvest-semantic-checks.json').write_text(json.dumps({'kind':'original selected assertions, not canonical REA equivalence','checks':{'mixed_protocol_task_and_validated_output':True,'invalid_output_preserves_completed_child':True,'compaction_role_resolved':True,'native_tool_result_roundtrip':True,'partial_stream_no_effect_or_retry':True,'credentials_separated_and_redirect_refused':True},'captures':reports},indent=2)+'\n')
print(json.dumps(reports,indent=2))
