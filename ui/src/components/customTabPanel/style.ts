export function a11yProps(index: number, variant: string = 'simple') {
	return {
		id: `${variant}-tab-${index}`,
		'aria-controls': `${variant}-tabpanel-${index}`
	}
}

export function tabSx(hasError: boolean) {
	return {
		color: hasError ? 'warning.main' : 'inherit',
		'&.Mui-selected': {
			color: hasError ? 'warning.main' : 'primary.main'
		}
	}
}
