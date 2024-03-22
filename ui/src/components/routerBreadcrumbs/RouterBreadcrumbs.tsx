import { Breadcrumbs, Typography } from '@mui/material';
import Link, { LinkProps } from '@mui/material/Link';
import { Link as RouterLink } from 'react-router-dom';

interface LinkRouterProps extends LinkProps {
    to: string;
    replace?: boolean;
}

function LinkRouter(props: LinkRouterProps) {
    return <Link {...props} component={RouterLink as any} />;
}

function RouterBreadcrumbs({ location }: any): JSX.Element {
    const pathnames = location.pathname.split('/').filter((notEmptyString: string) => notEmptyString)
    
    return (
        <Breadcrumbs aria-label="breadcrumb" separator="">
            <LinkRouter underline="hover" color="text.secondary" to={`/${pathnames.slice(0, 1)}`} variant='h3'>
                {pathnames.slice(0, 1)}
            </LinkRouter>
            {pathnames.map((_: any, index: number) => {
                const last = index === pathnames.length - 1;
                const to = `${pathnames.slice(1, index + 1).join(' > ')}`;

                return last && (
                    <Typography color="text.primary" key={to} variant='h3'>
                        {to}
                    </Typography>
                )
            })}
        </Breadcrumbs>
    )
}

export default RouterBreadcrumbs