import ContentCopyTwoToneIcon from '@mui/icons-material/ContentCopyTwoTone';
import { Box, IconButton, ListItem, ListItemText } from '@mui/material';
import copy from 'clipboard-copy';
import { useEffect, useState } from 'react';
import AutoSizer from "react-virtualized-auto-sizer";
import { FixedSizeList, ListChildComponentProps } from 'react-window';
import useSetDomainStore from '../../store/setDomainStore';
import { filterData } from '../../utils/helpers/domainList';
import SearchBar from '../searchBar/SearchBar';
import { ListItemButtonDomain, styleDomainListBox } from './style';

interface IDomainListProps {
    domains: string[];
}

function DomainsList({ domains }: IDomainListProps) {
    const [searchQuery, setSearchQuery] = useState('');
    const dataFiltered = filterData(searchQuery, domains);

    const domain = useSetDomainStore(state => state.domain);
    const setDomain = useSetDomainStore(state => state.setDomainValue);
    
    const [activeDomain, setActiveDomain] = useState('');

    useEffect(() => {
        setActiveDomain(domain)
    }, [domain])

    const getDomainLocation = (value: string) => {
        setDomain(value)
    }
    // console.log(domain === activeDomain)

    const renderDomains = ({ index, style }: ListChildComponentProps) => {
        const value = dataFiltered[index]

        return (
            <ListItem style={style}
                key={value}
                role='listitem'
                component="div"
                disablePadding
            >
                <ListItemButtonDomain onClick={() => getDomainLocation(value)}
                    className={value === activeDomain && value !== '' && activeDomain !== '' ? 'active' : ''}>
                    <ListItemText id={value} primary={value} />
                    <IconButton onClick={() => copy(value)}>
                        <ContentCopyTwoToneIcon />
                    </IconButton>
                </ListItemButtonDomain>

            </ListItem>
        )
    }

    return (
        <Box sx={{ ...styleDomainListBox }}>
            <SearchBar setSearchQuery={setSearchQuery} />
            <Box sx={{ width: '100%', height: '80%', paddingBottom: 1 }}>
                <AutoSizer>
                    {({ height, width }) => (
                        <FixedSizeList
                            width={width}
                            height={height}
                            itemSize={53}
                            itemCount={dataFiltered.length}
                            overscanCount={5}
                        >
                            {renderDomains}
                        </FixedSizeList>
                    )}
                </AutoSizer>
            </Box>
        </Box>

    )
}

export default DomainsList