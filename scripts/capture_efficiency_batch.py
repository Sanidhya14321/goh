#!/usr/bin/env python3
"""Original controlled context-size and output-retrieval acceptance."""
import json,os,pathlib,shutil,socket,subprocess,time,urllib.request,urllib.error
root=pathlib.Path(__file__).resolve().parents[1];r=root/'research';e=r/'evidence';project=r/'efficiency-project';project.mkdir(exist_ok=True)
source_bytes=0
for i in range(32):
 text=f'package fixture\n\nfunc Entry{i}(input string) string {{\n'+''.join(f'// implementation detail {j} that does not belong in the declaration map\n' for j in range(128))+'return input\n}\n'
 (project/f'entry{i:02}.go').write_text(text);source_bytes+=len(text.encode())
with socket.socket() as sock:sock.bind(('127.0.0.1',0));port=sock.getsockname()[1]
log=e/'efficiency-model-requests.jsonl';log.write_text('');sessions=r/'efficiency-sessions'
env={k:v for k,v in os.environ.items() if not k.startswith('HARNESS_')}
fixture=subprocess.Popen(['node',str(r/'efficiency-model.mjs')],env=dict(env,FIXTURE_PORT=str(port),FIXTURE_LOG=str(log),NODE_COMMAND=shutil.which('node')),stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
try:
 for attempt in range(100):
  if fixture.poll() is not None:raise RuntimeError('fixture died')
  try:urllib.request.urlopen(f'http://127.0.0.1:{port}/ready',timeout=.1)
  except urllib.error.HTTPError:break
  except OSError:time.sleep(.02)
 scenario=json.loads((r/'scenarios/go-formatters.json').read_text());scenario.update(executable=str(root/'bin/harness'),arguments=['run','--session-dir',str(sessions),'--project',str(project),'--format','json','--endpoint',f'http://127.0.0.1:{port}/chat','--model','fixture','--context-window','131072','--allow','shell','--prompt','Map Go declarations twice, then page only the end of a long command output.'],environment={'HARNESS_API_KEY_ENV':'EFFICIENCY_FIXTURE_KEY','EFFICIENCY_FIXTURE_KEY':'efficiency-only'},working_directory=str(project),filesystem_observation_paths=[str(project)])
 scenario['limits']['output_bytes']=1048576;path=r/'scenarios/go-efficiency.json';path.write_text(json.dumps(scenario,indent=2))
 with (e/'go-efficiency.json').open('w') as out,(e/'go-efficiency.progress.jsonl').open('w') as err:subprocess.run([str(r/'tooling/rea/node_modules/.bin/rea'),'capture-process',str(path),'--format','json'],env=env,stdout=out,stderr=err,check=True)
 evidence=json.loads((e/'go-efficiency.json').read_text());assert evidence['normalized_result']['exit']['code']==0
 journal=max(sessions.glob('*.jsonl'),key=lambda p:p.stat().st_mtime);records=[json.loads(line) for line in journal.read_text().splitlines()]
 result={x['data']['id']:x['data']['result'] for x in records if x['type']=='tool_result'}
 bodies={k:json.loads(v['content']) for k,v in result.items()};assert not any(v.get('error') for v in result.values())
 assert len(bodies['map-0']['symbols'])==32 and bodies['map-0']['parse_cache_hits']==0 and bodies['map-1']['parse_cache_hits']==32
 assert bodies['output']['preview_truncated'] and 'END_MARKER' not in bodies['output']['output']
 assert 'END_MARKER' in bodies['page']['content'] and bodies['page']['eof']
 map_bytes=len(result['map-0']['content'].encode());assert map_bytes<source_bytes//10
 report={'kind':'controlled original fixture, not an upstream speed benchmark','evidence_id':evidence['evidence_id'],'source_bytes':source_bytes,'map_json_bytes':map_bytes,'context_byte_reduction_fraction':1-map_bytes/source_bytes,'files':32,'warm_parse_cache_hits':32,'shell_output_bytes':bodies['output']['output_bytes'],'retrieved_page_bytes':bodies['page']['bytes'],'tail_marker_retrieved':True,'limits':['Go declarations only; no full Aider dependency ranking.','Warm cache still reads/hashes current source; no stale mtime shortcuts.','Retained command output capped1MiB; discarded bytes unrecoverable.','REA process/observation limits retained in evidence; no global absence-of-effects inference.']}
 (e/'efficiency-report.json').write_text(json.dumps(report,indent=2)+'\n');print(json.dumps(report,indent=2))
finally:
 fixture.terminate()
 try:fixture.wait(timeout=5)
 except subprocess.TimeoutExpired:fixture.kill();fixture.wait()
