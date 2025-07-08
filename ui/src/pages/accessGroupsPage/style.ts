export const styleRootBox = {
	height: 'calc(100vh - 125px)',
	margin: '20px',
	borderRadius: 2,
	boxShadow: `0px 2px 4px -1px rgba(0,0,0,0.2),
             0px 4px 5px 0px rgba(0,0,0,0.14),
              0px 1px 10px 0px rgba(0,0,0,0.12)`
}

export const styleWrapperCards = {
	margin: '20px',
	display: 'grid',
	gridTemplateColumns: 'repeat(auto-fill, minmax(350px, 1fr))',
	gap: 2
}
