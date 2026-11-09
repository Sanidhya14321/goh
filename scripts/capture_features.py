#!/usr/bin/env python3
"""Original paired-Go feature capture driver. Starts only loopback fixtures."""
import json, os, pathlib, subprocess, time, urllib.request
root=pathlib.Path(__file__).resolve().parents[1]
research=root/'research'
base=json.loads((research/'scenarios/todo.json').read_text())
project=research/'feature-go-project';project.mkdir(exist_ok=True)
sessions=research/'feature-go-sessions';sessions.mkdir(exist_ok=True)
base.update(executable=str(root/'bin/harness'),working_directory=str(project),environment={},filesystem_observation_paths=[str(project)],timeout_ms=20000)
common=['--endpoint','http://127.0.0.1:28773/v1/chat/completions','--model','scripted','--context-window','131072','--session-dir',str(sessions),'--format','json']
rea=str(research/'tooling/rea/node_modules/.bin/rea')
model=subprocess.Popen(['node',str(research/'todo-model.mjs')],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
def capture(name,args):
 scenario=dict(base,arguments=args);file=research/'scenarios'/f'{name}.json';file.write_text(json.dumps(scenario,indent=2))
 with open(research/'evidence'/f'{name}.json','w') as out,open(research/'evidence'/f'{name}.progress.jsonl','w') as err:
  subprocess.run([rea,'capture-process',str(file),'--format','json'],stdout=out,stderr=err,check=True)
 evidence=json.loads((research/'evidence'/f'{name}.json').read_text());return evidence
try:
 time.sleep(.3)
 evidence=capture('go-todo',['run','--project',str(project),'--prompt','Task-list fixture',*common,'--allow','todowrite'])
 frames=''.join(frame.get('raw_data',frame['data']) for frame in evidence['normalized_result']['frames'])
 events=[]
 for line in frames.splitlines():
  try:events.append(json.loads(line))
  except ValueError:pass
 sid=next(e['session_id'] for e in events if e.get('type')=='run_started')
 capture('go-todo-resume',['resume',sid,'--prompt','Resume the task-list fixture',*common,'--allow','todowrite'])
 capture('go-todo-ask',['run','--project',str(project),'--prompt','Task-list fixture',*common])
finally:
 model.terminate();model.wait(timeout=5)
# Controlled Go-only manual compaction fixture; reference uses its own original
# HTTP driver, whose retained observations establish reference operation shape.
script=research/'compaction/go-summary-script.json';script.write_text(json.dumps([{'text':'Summary: preserve the task-list fixture investigation; tests next.'}]))
fixture=subprocess.Popen([str(root/'bin/scripted-model'),'--listen','127.0.0.1:28774','--script',str(script)],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
try:
 time.sleep(.2)
 compact_common=common.copy();compact_common[compact_common.index('http://127.0.0.1:28773/v1/chat/completions')]='http://127.0.0.1:28774/v1/chat/completions'
 capture('go-compaction-manual',['compact',sid,*compact_common])
finally:
 fixture.terminate();fixture.wait(timeout=5)
print(json.dumps({'session_id':sid,'captures':['go-todo','go-todo-resume','go-todo-ask','go-compaction-manual']}))
