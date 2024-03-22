import EnvoyIcon from '../envoyIcon/EnvoyIcon';
import KuberIcon from '../kuberIcon/KuberIcon';

const navMenuItems = [
    {
        id: 1,
        name: 'Envoy Configs',
        icon: <EnvoyIcon />,
        path: '/nodeIDs'
    },
    {
        id: 2,
        name: 'Kubernetes CRDs',
        icon: <KuberIcon />,
        path: '/kuber'
    }
]

export default navMenuItems;