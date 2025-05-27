import { Breadcrumbs, Typography } from '@mui/material'
import Link, { LinkProps } from '@mui/material/Link'
import { Link as RouterLink } from 'react-router-dom'
import { useVirtualServiceStore } from '../../store/setVsStore.ts'

interface LinkRouterProps extends LinkProps {
	to: string
	replace?: boolean
}

function LinkRouter(props: LinkRouterProps) {
	return <Link {...props} component={RouterLink as any} />
}

function RouterBreadcrumbs({ location }: { location: { pathname: string } }) {
	const pathNames = location.pathname.split('/').filter(Boolean)
	const virtualServiceMap = useVirtualServiceStore(state => state.virtualServiceMap)

	return (
		<Breadcrumbs aria-label='breadcrumb' separator='›'>
			{pathNames.map((segment: string, index: number) => {
				const last = index === pathNames.length - 1
				const to = `/${pathNames.slice(0, index + 1).join('/')}`
				const displayName = virtualServiceMap.get(segment) || segment

				return last ? (
					<Typography color='text.primary' key={to} variant='h3'>
						{displayName}
					</Typography>
				) : (
					<LinkRouter underline='hover' color='text.secondary' to={to} key={to} variant='h3'>
						{displayName}
					</LinkRouter>
				)
			})}
		</Breadcrumbs>
	)
}

export default RouterBreadcrumbs
