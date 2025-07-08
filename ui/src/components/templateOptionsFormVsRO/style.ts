export const templateOptionsBox = {
	width: '100%',
	border: '1px solid gray',
	borderRadius: 1,
	p: 2,
	pt: 0.5,
	display: 'flex',
	flexDirection: 'column',
	gap: 2
}

export const styleTooltip = {
	popper: {
		modifiers: [
			{
				name: 'offset',
				options: {
					offset: [0, -12]
				}
			}
		]
	}
}
