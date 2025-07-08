import EnvoyIcon from '../iconsSvg/envoyIcon/EnvoyIcon'
import { VirtualServicesIcon } from '../iconsSvg/virtualServicesIcon/virtualServicesIcon.tsx'

const navMenuItems = [
	{
		id: 1,
		name: 'Envoy Configs',
		icon: <EnvoyIcon />,
		path: '/nodeIDs'
	},
	{
		id: 2,
		name: 'Access Groups',
		icon: <VirtualServicesIcon />,
		path: '/accessGroups',
		requiresAccess: true
	}
]

export default navMenuItems
