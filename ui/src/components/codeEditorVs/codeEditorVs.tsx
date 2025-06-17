import useTheme from '@mui/material/styles/useTheme'
import React, { memo, useLayoutEffect, useRef, useState } from 'react'
import { convertRawToFullYaml } from '../../utils/helpers/convertToYaml.ts'
import { Editor } from '@monaco-editor/react'
import Box from '@mui/material/Box'

interface ICodeBlockVsProps {
	raw: string
}

export const CodeEditorVs: React.FC<ICodeBlockVsProps> = memo(({ raw }) => {
	const theme = useTheme()

	const [height, setHeight] = useState<number | null>(null)
	const elementRef = useRef<HTMLDivElement>(null)
	const updateHeight = () => {
		if (elementRef.current) {
			setHeight(elementRef.current.getBoundingClientRect().height)
		}
	}

	useLayoutEffect(() => {
		updateHeight() // Устанавливаем начальную высоту
		window.addEventListener('resize', updateHeight) // Обновляем при изменении размера окна

		return () => {
			window.removeEventListener('resize', updateHeight) // Чистим обработчик
		}
	}, [])

	const yamlData = convertRawToFullYaml(raw)

	return (
		<Box
			border='1px solid gray'
			borderRadius={1}
			p={2}
			height='100%'
			width='100%'
			mr={1}
			ref={elementRef}
			sx={{ overflow: 'auto' }}
		>
			<Editor
				height={`calc(${height}px - 40px)`}
				defaultLanguage='yaml'
				value={yamlData}
				theme={theme.palette.mode === 'light' ? 'light' : 'vs-dark'}
				options={{
					readOnly: true,
					minimap: { enabled: false }
				}}
				loading
			/>
		</Box>
	)
})
