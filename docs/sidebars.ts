import type {SidebarsConfig} from '@docusaurus/plugin-content-docs';

const sidebars: SidebarsConfig = {
  docs: [
    'index',
    {
      type: 'category',
      label: 'Tutorials',
      collapsed: false,
      items: ['tutorials/deploy-your-first-broker'],
    },
    {
      type: 'category',
      label: 'How-to guides',
      collapsed: false,
      items: [
        'how-to/rotate-github-app-private-key',
        'how-to/change-target-repository',
        'how-to/use-with-github-enterprise-server',
      ],
    },
    {
      type: 'category',
      label: 'Reference',
      collapsed: false,
      items: [
        'reference/environment-variables',
        'reference/response-schema',
        'reference/iam-permissions',
        'reference/ssm-parameter-shapes',
        'reference/errors',
      ],
    },
    {
      type: 'category',
      label: 'Explanation',
      collapsed: false,
      items: [
        'explanation/architecture',
        'explanation/security-model',
        'explanation/why-empty-payloads',
        'explanation/why-permissions-are-deploy-time',
        'explanation/release-architecture',
      ],
    },
  ],
};

export default sidebars;
