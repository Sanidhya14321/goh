package runtime

import "testing"

func TestBackgroundJobPolicy(t *testing.T) {
	a, _ := testAgent(t)
	if a.checkMode("job_start") != nil || a.checkMode("job_output") != nil {
		t.Fatal("build mode should allow jobs")
	}
	a.Config.Agent, a.Config.AgentMode = "plan", "plan"
	if a.checkMode("job_start") == nil || a.checkMode("job_stop") == nil {
		t.Fatal("plan mode allowed starting or stopping jobs")
	}
	if a.checkMode("job_output") != nil {
		t.Fatal("plan mode should allow reading job output")
	}
	child, _ := testAgent(t)
	child.depth = 1
	if child.checkMode("job_start") == nil || child.checkMode("child:job_start") == nil {
		t.Fatal("subagent allowed to start a job")
	}
}
