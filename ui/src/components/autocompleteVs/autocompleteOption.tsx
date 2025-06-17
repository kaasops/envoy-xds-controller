import React from 'react'
import Box from '@mui/material/Box'
import Typography from '@mui/material/Typography'
import IconButton from '@mui/material/IconButton'
import VisibilityOutlinedIcon from '@mui/icons-material/VisibilityOutlined'
import { HTTPFilterListItem } from '../../gen/http_filter/v1/http_filter_pb.ts'
import { RouteListItem } from '../../gen/route/v1/route_pb.ts'
import { ListenerListItem } from '../../gen/listener/v1/listener_pb.ts'
import { VirtualServiceTemplateListItem } from '../../gen/virtual_service_template/v1/virtual_service_template_pb.ts'
import { AccessLogConfigListItem } from '../../gen/access_log_config/v1/access_log_config_pb.ts'

export type AutocompleteOption =
	| ListenerListItem
	| VirtualServiceTemplateListItem
	| AccessLogConfigListItem
	| HTTPFilterListItem
	| RouteListItem

interface IAutocompleteOptionProps {
	option: AutocompleteOption
	props: React.HTMLAttributes<HTMLLIElement> & { key: string | number }
	onPreviewClick: (event: React.MouseEvent<HTMLButtonElement>, option: AutocompleteOption) => void
}

export const AutocompleteOption: React.FC<IAutocompleteOptionProps> = ({ option, props, onPreviewClick }) => {
	const { key, ...optionProps } = props

	return (
		<Box
			component='li'
			// key={option.uid}
			{...optionProps}
			sx={{ display: 'flex', justifyContent: 'space-between', width: '100%' }}
		>
			<Box sx={{ width: '45%' }}>
				<Typography>{option.name}</Typography>
			</Box>
			<Box sx={{ width: '65%' }}>
				<Typography variant='body2' sx={{ wordWrap: 'break-word' }} color='text.disabled'>
					{option.description}
				</Typography>
			</Box>
			<IconButton onClick={e => onPreviewClick(e, option)}>
				<VisibilityOutlinedIcon />
			</IconButton>
		</Box>
	)
}
