#!/usr/bin/env python3
"""Capture original Go task/webfetch/LSP behavior against local research fixtures."""
import json,os,pathlib,subprocess,time
root=pathlib.Path(__file__).resolve().parents[1];r=root/'research';binary=root/'bin/harness';rea=r/'tooling/rea/node_modules/.bin/rea'
sessions=r/'next-batch-sessions';reports=[]
def capture(name,reference,args,environment=None,events=None):
 scenario=json.loads((r/'scenarios'/reference).read_text());scenario.update(executable=str(binary),arguments=args,environment=environment or {});scenario['limits']['output_bytes']=1048576
 if events is not None:scenario['events']=events
 file=r/'scenarios'/f'{name}.json';file.write_text(json.dumps(scenario,indent=2))
 with open(r/'evidence'/f'{name}.json','w') as out,open(r/'evidence'/f'{name}.progress.jsonl','w') as err:subprocess.run([str(rea),'capture-process',str(file),'--format','json'],stdout=out,stderr=err,check=True)
 d=json.loads((r/'evidence'/f'{name}.json').read_text());reports.append({'scenario':name,'evidence_id':d['evidence_id'],'exit':d['normalized_result']['exit']});return d
common=['--format','json','--session-dir',str(sessions),'--model','scripted','--context-window','131072']
# Task fixture is independently authored; only result-ID parsing adapts to the
# original Go output schema. Missing ID differs deliberately from the reference.
model=subprocess.Popen(['node',str(r/'subagents/model-go.mjs')],env=dict(os.environ,FIXTURE_LOG=str(r/'evidence/subagents-go-model-requests.jsonl'),FIXTURE_PROJECT=str(r/'subagents/project')),stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
try:
 time.sleep(.2)
 capture('go-subagents','subagents.json',['run','--project',str(r/'subagents/project'),'--prompt','PARENT_SUBAGENT fixture','--endpoint','http://127.0.0.1:28794/v1/chat/completions',*common,'--allow','task'])
finally:model.terminate();model.wait(timeout=5)
model=subprocess.Popen(['node',str(r/'webfetch-model.mjs')],env=dict(os.environ,MODEL_LOG=str(r/'evidence/webfetch-go-requests.jsonl'),HTTP_LOG=str(r/'evidence/webfetch-go-http.jsonl')),stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
try:
 time.sleep(.2)
 capture('go-webfetch','webfetch-reference.json',['run','--project',str(r/'webfetch-project'),'--prompt','Execute bounded local webfetch fixtures.','--endpoint','http://127.0.0.1:18775/v1/chat/completions',*common,'--allow','webfetch'])
finally:model.terminate();model.wait(timeout=5)
config=r/'lsp-go-config.json';config.write_text(json.dumps({'endpoint':'http://127.0.0.1:18888/v1/chat/completions','model':'scripted','context_window':131072,'lsp':{'fixture':{'command':str(r/'tooling/lsp-fixture'),'extensions':['.lsptest'],'env':{'LSP_FIXTURE_LOG':str(r/'evidence/lsp-go-server-messages.jsonl')}}}}))
model=subprocess.Popen(['node',str(r/'lsp-model.mjs')],env=dict(os.environ,FIXTURE_LOG=str(r/'evidence/lsp-go-model-requests.jsonl')),stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
try:
 time.sleep(.2)
 capture('go-lsp','lsp-reference.json',['run','--config',str(config),'--project',str(r/'lsp-project'),'--prompt','Inspect fixture symbols.',*common,'--allow','lsp,lsp_start:fixture'])
finally:model.terminate();model.wait(timeout=5)
(r/'evidence/next-batch-driver.json').write_text(json.dumps(reports,indent=2)+'\n');print(json.dumps(reports,indent=2))
