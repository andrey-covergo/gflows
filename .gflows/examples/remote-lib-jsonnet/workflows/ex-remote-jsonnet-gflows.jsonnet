local git_config = import 'common/git.libsonnet';
local steps = import 'common/steps.libsonnet';
local workflows = import 'common/workflows.libsonnet';

local check_workflows_job = {
  'name': 'check-workflows [ex-remote-jsonnet-gflows]',
  'runs-on': 'ubuntu-latest',
  steps: [
    steps.checkout,
    steps.setup_go,
    steps.uses('jbrunton/setup-gflows@v1') {
      with: {
        token: "${{ secrets.GITHUB_TOKEN }}",
      }
    },
    steps.named('validate workflows', 'gflows check') {
      env: {
        GFLOWS_CONFIG: '.gflows/examples/remote-lib-jsonnet/config.yml'
      },
    },
  ]
};

local workflow = {
  name: 'gflows',
  on: workflows.triggers.pull_request_defaults,
  jobs: {
    check_workflows: check_workflows_job
  },
};

std.manifestYamlDoc(workflow)
