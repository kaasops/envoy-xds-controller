import { useRef, useState } from 'react'
import { Editor } from '@monaco-editor/react'
import useTheme from '@mui/material/styles/useTheme'
import { convertRawToFullYaml } from '../../utils/helpers/convertToYaml.ts'

export const AutocompleteCodeEditorVs = ({ raw }: { raw: string }) => {
	const theme = useTheme()
	const editorRef = useRef<any>(null)
	const [height, setHeight] = useState(200)

	const handleEditorDidMount = (_editor: any) => {
		editorRef.current = _editor

		const contentHeight = _editor.getContentHeight()
		setHeight(contentHeight)

		_editor.onDidContentSizeChange(() => {
			const newHeight = _editor.getContentHeight()
			setHeight(newHeight)
		})
	}

	const width = Math.min(raw.split('\n').reduce((max, line) => Math.max(max, line.length), 0) * 8 + 40, 500)

	const yamlData = convertRawToFullYaml(raw)

	return (
		<Editor
			value={yamlData}
			language='yaml'
			onMount={handleEditorDidMount}
			options={{
				minimap: { enabled: false },
				scrollBeyondLastLine: false
			}}
			height={height}
			width={width}
			theme={theme.palette.mode === 'light' ? 'light' : 'vs-dark'}
		/>
	)
}
