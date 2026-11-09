#!/usr/bin/env python3
"""Capture built Go skill admission/lazy loading and selected plan behavior."""
import json,os,pathlib,subprocess,time
root=pathlib.Path(__file__).resolve().parents[1];r=root/'research';binary=root/'bin/harness';rea=r/'tooling/rea/node_modules/.bin/rea';project=r/'skills/project';sessiondir=r/'skills/go-sessions'
base=json.loads((r/'scenarios/skills.json').read_text());base.update(executable=str(binary),environment={},working_directory=str(project),filesystem_observation_paths=[str(project)],timeout_ms=20000)
common=['--project',str(project),'--endpoint','http://127.0.0.1:28774/v1/chat/completions','--model','scripted','--context-window','131072','--session-dir',str(sessiondir),'--format','json']
model=subprocess.Popen(['node',str(r/'skills/model.mjs')],env=dict(os.environ,FIXTURE_LOG=str(r/'evidence/go-skills-requests.jsonl')),stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL)
def capture(name,args):
 scenario=dict(base,arguments=args);file=r/'scenarios'/f'{name}.json';file.write_text(json.dumps(scenario,indent=2))
 with open(r/'evidence'/f'{name}.json','w') as out,open(r/'evidence'/f'{name}.progress.jsonl','w') as err:subprocess.run([str(rea),'capture-process',str(file),'--format','json'],stdout=out,stderr=err,check=True)
 return json.loads((r/'evidence'/f'{name}.json').read_text())
try:
 time.sleep(.3)
 load=capture('go-skills',['run','--prompt','Skills fixture.',*common,'--allow','skill'])
 ask=capture('go-skills-ask',['run','--prompt','Skill permission fixture.',*common])
 plan=capture('go-skills-plan',['run','--prompt','Skill permission fixture.',*common,'--agent','plan','--allow','skill,shell,create_file'])
finally:model.terminate();model.wait(timeout=5)
report={'captures':{n:d['evidence_id'] for n,d in [('go-skills',load),('go-skills-ask',ask),('go-skills-plan',plan)]}}
(r/'evidence/skill-driver.json').write_text(json.dumps(report,indent=2)+'\n');print(json.dumps(report))
