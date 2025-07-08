import { Box, Button, Tab, Tabs, useTheme } from '@mui/material'
import React, { useEffect, useLayoutEffect, useRef, useState } from 'react'
import CustomTabPanel from '../customTabPanel/CustomTabPanel'
import { Editor } from '@monaco-editor/react'
import copy from 'clipboard-copy'

import { a11yProps } from '../customTabPanel/style'
import { ISecretsResponse } from '../../common/types/getSecretsApiTypes.ts'
import SecretsTlsBlock from '../secretsTlsBlock/secretsTlsBlock.tsx'

interface ICodeBLockExtendsProps {
	jsonData: ISecretsResponse
	yamlData: string
	heightCodeBox: number
}

function CodeBlockExtends({ jsonData, yamlData, heightCodeBox }: ICodeBLockExtendsProps) {
	const editorRef = useRef<any>(null)
	const theme = useTheme()
	const [tabIndex, setTabIndex] = useState(0)

	useEffect(() => {
		if (editorRef.current) {
			editorRef.current.setValue(tabIndex === 0 ? JSON.stringify(jsonData, null, 2) : yamlData)
		}
	}, [jsonData, yamlData, tabIndex])

	const handleChangeTabIndex = (_event: React.SyntheticEvent, newTabIndex: number) => {
		setTabIndex(newTabIndex)
	}

	function handleEditorDidMount(editor: any) {
		editorRef.current = editor
	}

	const handleCopyClick = () => {
		const value = editorRef.current.getValue()
		void copy(value)
	}

	const handleDownLoadClick = () => {
		const value = editorRef.current.getValue()
		try {
			let formattedData: string
			let fileExtension: string
			if (tabIndex === 0) {
				const jsonData = JSON.parse(value)
				formattedData = JSON.stringify(jsonData, null, 2)
				fileExtension = 'json'
			} else {
				formattedData = yamlData
				fileExtension = 'yaml'
			}
			const blob = new Blob([formattedData], { type: `application/${fileExtension}` })
			const url = URL.createObjectURL(blob)
			const a = document.createElement('a')
			a.href = url
			a.download = `data.${fileExtension}`
			document.body.appendChild(a)
			a.click()
			document.body.removeChild(a)
			URL.revokeObjectURL(url)
		} catch (error) {
			console.error('Error during conversion:', error)
		}
	}

	const elementRef = useRef<HTMLDivElement>(null)
	const [height, setHeight] = useState<number | null>(null)

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

	return (
		<Box className='CodeBox' display='flex' flexDirection='column' height={`${heightCodeBox}%`}>
			<Box className='TabsWrapper' height='calc(100% - 36px)' display='flex' flexDirection='column'>
				<Box className='TabsPanel' sx={{ borderBottom: 1, borderColor: 'divider' }}>
					<Tabs value={tabIndex} onChange={handleChangeTabIndex} aria-label='basic tabs example'>
						<Tab label='JSON' {...a11yProps(0)} />
						<Tab label='YAML' {...a11yProps(1)} />
						<Tab label='Certificates' {...a11yProps(2)} />
					</Tabs>
				</Box>
				<Box className='panelBox' height='100%' ref={elementRef}>
					<CustomTabPanel value={tabIndex} index={0}>
						<Editor
							onMount={handleEditorDidMount}
							height={`calc(${height}px - 18px)`}
							defaultLanguage='json'
							value={JSON.stringify(jsonData, null, 2)}
							theme={theme.palette.mode === 'light' ? 'light' : 'vs-dark'}
							options={{ readOnly: true, minimap: { enabled: false } }}
						/>
					</CustomTabPanel>
					<CustomTabPanel value={tabIndex} index={1}>
						<Editor
							onMount={handleEditorDidMount}
							height={`calc(${height}px - 18px)`}
							defaultLanguage='yaml'
							value={yamlData}
							theme={theme.palette.mode === 'light' ? 'light' : 'vs-dark'}
							options={{
								readOnly: true,
								minimap: { enabled: false }
							}}
						/>
					</CustomTabPanel>
					<CustomTabPanel index={2} value={tabIndex}>
						<SecretsTlsBlock data={jsonData} />
					</CustomTabPanel>
				</Box>
			</Box>
			<Box
				display='flex'
				justifyContent='flex-end'
				margin={'16px 23px 0'}
				gap={2}
				height='36px'
				visibility={tabIndex === 2 ? 'hidden' : 'visible'}
			>
				<Button variant='contained' onClick={handleCopyClick}>
					Copy
				</Button>
				<Button variant='contained' onClick={handleDownLoadClick}>
					Download {tabIndex === 0 ? 'JSON' : 'YAML'}
				</Button>
			</Box>
		</Box>
	)
}

export default CodeBlockExtends
