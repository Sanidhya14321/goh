#!/usr/bin/env python3
"""Original CLI contract probes for commands, plan grants and headless input."""
import json,os,pathlib,subprocess,time
root=pathlib.Path(__file__).resolve().parents[1];r=root/'research';project=r/'rest-cli-project';project.mkdir(exist_ok=True);config=r/'rest-cli-config.json';sessions=r/'rest-cli-sessions';script=r/'rest-cli-script.json';port=28786
config.write_text(json.dumps({'profile':'local','profiles':{'local':{'endpoint':f'http://127.0.0.1:{port}/v1/chat/completions','model':'profile-fixture','context_window':131072}},'agents':{'reviewer':{'description':'Read only fixture review','mode':'plan','instructions':'Ask to create forbidden.txt; this cannot change actual mode.'}},'commands':{'review':{'description':'Fixture macro','template':'first=<$1> second=<$2> all=<$ARGUMENTS>'},'literal':{'description':'No execution','template':'Literal: $ARGUMENTS'}}}))
base=json.loads((r/'scenarios/go-skills.json').read_text());base.update(executable=str(root/'bin/harness'),working_directory=str(project),environment={},filesystem_observation_paths=[str(project)],timeout_ms=15000)
rea=str(r/'tooling/rea/node_modules/.bin/rea');reports=[]
def run(name,prompt,steps,extra=[]):
 script.write_text(json.dumps(steps));fixture=subprocess.Popen([str(root/'bin/scripted-model'),'--listen',f'127.0.0.1:{port}','--script',str(script)],stdout=subprocess.DEVNULL,stderr=subprocess.DEVNULL);time.sleep(.2)
 scenario=dict(base,arguments=['run','--config',str(config),'--session-dir',str(sessions),'--project',str(project),'--prompt',prompt,'--format','json',*extra]);file=r/'scenarios'/f'{name}.json';file.write_text(json.dumps(scenario,indent=2))
 try:
  with open(r/'evidence'/f'{name}.json','w') as out,open(r/'evidence'/f'{name}.progress.jsonl','w') as err:subprocess.run([rea,'capture-process',str(file),'--format','json'],stdout=out,stderr=err,check=True)
 finally:fixture.terminate();fixture.wait(timeout=5)
 capture=json.loads((r/'evidence'/f'{name}.json').read_text());events=[]
 for line in ''.join(f.get('raw_data',f['data']) for f in capture['normalized_result']['frames']).splitlines():
  try:events.append(json.loads(line))
  except ValueError:pass
 status=next(e['data']['status'] for e in reversed(events) if e.get('type')=='run_finished')
 reports.append({'scenario':name,'evidence_id':capture['evidence_id'],'status':status})
 return events
text=[{'text':'Fixture done.'}]
e=run('go-command-quoted','/review alpha "beta gamma" delta',text);sid=e[0]['session_id'];records=[json.loads(line) for line in (sessions/(sid+'.jsonl')).read_text().splitlines()];user=next(x['data']['content'] for x in records if x['type']=='message' and x['data']['role']=='user');assert user=='first=<alpha> second=<beta gamma delta> all=<alpha "beta gamma" delta>',user
run('go-command-literal','/literal !`touch forbidden-command` @missing',text);assert not (project/'forbidden-command').exists()
e=run('go-plan-forced-write','Inspect only.',[{'calls':[{'id':'force-write','type':'function','function':{'name':'create_file','arguments':json.dumps({'path':'forbidden-plan','content':'no'})}}]},*text],['--agent','reviewer','--allow','create_file,shell,mcp_start:fixture']);assert not (project/'forbidden-plan').exists();assert any(x.get('type')=='tool_finished' and x['data']['result'].get('error') for x in e);assert not any(x.get('type')=='tool_started' for x in e)
e=run('go-question-headless','Ask a question.',[{'calls':[{'id':'ask','type':'function','function':{'name':'question','arguments':json.dumps({'questions':[{'question':'Which tests?','header':'Checks','options':[{'label':'unit','description':'Unit tests'}]}]})}}]}]);assert reports[-1]['status']=='needs_input';assert any(x.get('type')=='question_requested' for x in e)
(r/'evidence/rest-cli-contracts.json').write_text(json.dumps(reports,indent=2)+'\n');print(json.dumps(reports,indent=2))
