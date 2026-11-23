import { lazy } from 'react'

export const sections = [
	{ slug: 'overview', title: 'Overview', group: 'Nodes', Page: lazy(() => import('./sections/overview')) },
	{ slug: 'nodes', title: 'Nodes', group: 'Nodes', Page: lazy(() => import('./sections/nodes')) },
	{ slug: 'clients', title: 'xDS streams', group: 'Nodes', Page: lazy(() => import('./sections/clients')) },
	{ slug: 'snapshots', title: 'Snapshots', group: 'Config', Page: lazy(() => import('./sections/snapshots')) },
	{ slug: 'listeners', title: 'Listeners', group: 'Config', Page: lazy(() => import('./sections/listeners')) },
	{ slug: 'routes', title: 'Routes', group: 'Config', Page: lazy(() => import('./sections/routes')) },
	{ slug: 'clusters', title: 'Clusters', group: 'Config', Page: lazy(() => import('./sections/clusters')) },
	{ slug: 'secrets', title: 'Secrets', group: 'Config', Page: lazy(() => import('./sections/secrets')) },
	{ slug: 'config-dump', title: 'Config dump', group: 'Config', Page: lazy(() => import('./sections/config-dump')) },
	{
		slug: 'res-virtual-services',
		title: 'Virtual services',
		group: 'Resources',
		Page: lazy(() => import('./sections/res-virtual-services'))
	},
	{
		slug: 'res-templates',
		title: 'Templates',
		group: 'Resources',
		Page: lazy(() => import('./sections/res-templates'))
	},
	{
		slug: 'res-listeners',
		title: 'Listeners',
		group: 'Resources',
		Page: lazy(() => import('./sections/res-listeners'))
	},
	{
		slug: 'res-clusters',
		title: 'Clusters',
		group: 'Resources',
		Page: lazy(() => import('./sections/res-clusters'))
	},
	{ slug: 'res-routes', title: 'Routes', group: 'Resources', Page: lazy(() => import('./sections/res-routes')) },
	{
		slug: 'res-http-filters',
		title: 'HTTP filters',
		group: 'Resources',
		Page: lazy(() => import('./sections/res-http-filters'))
	},
	{
		slug: 'res-policies',
		title: 'Policies',
		group: 'Resources',
		Page: lazy(() => import('./sections/res-policies'))
	},
	{
		slug: 'res-access-logs',
		title: 'Access log configs',
		group: 'Resources',
		Page: lazy(() => import('./sections/res-access-logs'))
	},
	{ slug: 'events', title: 'Events', group: 'Activity', Page: lazy(() => import('./sections/events')) },
	{
		slug: 'access-groups',
		title: 'Access groups',
		group: 'Access',
		Page: lazy(() => import('./sections/access-groups'))
	}
] as const

export const groups = Array.from(new Set(sections.map(section => section.group)))
