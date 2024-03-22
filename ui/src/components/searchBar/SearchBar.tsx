import { TextField } from '@mui/material';
import { ChangeEvent } from 'react';

interface ISearchBarProps {
    setSearchQuery: (value: string) => void
}

function SearchBar({ setSearchQuery }: ISearchBarProps) {
    return (
        <TextField id="search-bar"
            className="text"
            onInput={(e: ChangeEvent<HTMLInputElement>) => {
                setSearchQuery(e.target.value);
            }}
            label="Input domain name"
            variant="outlined"
            placeholder="Search..."
            size="small"
            fullWidth
            sx={{ mb: 2 }}
        />
    )
}

export default SearchBar