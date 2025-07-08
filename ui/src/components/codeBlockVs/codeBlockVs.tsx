import React, { memo } from 'react'
import Typography from '@mui/material/Typography'
import CircularProgress from '@mui/material/CircularProgress'
import Fade from '@mui/material/Fade'
import { CodeEditorVs } from '../codeEditorVs/codeEditorVs.tsx'
import Box from '@mui/material/Box'
import { codeBlockVs } from './style.ts'
import { Control, UseFormSetValue, useWatch } from 'react-hook-form'
import { IVirtualServiceForm } from '../virtualServiceForm/types.ts'
import { ButtonGroup } from '@mui/material'
import Button from '@mui/material/Button'

interface ICodeBlockVsProps {
	rawDataTemplate: string | undefined
	rawDataPreview: string | undefined
	isLoadingFillTemplate: boolean
	control: Control<IVirtualServiceForm>
	isCreateMode: boolean
	setValue: UseFormSetValue<IVirtualServiceForm>
}

export const CodeBlockVs: React.FC<ICodeBlockVsProps> = memo(
	({ rawDataTemplate, rawDataPreview, isLoadingFillTemplate, isCreateMode, control, setValue }) => {
		const templateUid = useWatch({ control, name: 'templateUid' })
		const isExpanded = useWatch({ control, name: 'viewTemplateMode', defaultValue: false })

		const renderCodeBlock = () => {
			if (isCreateMode && !templateUid) {
				return (
					<Typography align='center' variant='h3'>
						For a preview, select a template
					</Typography>
				)
			}
			if (isLoadingFillTemplate) {
				return <CircularProgress size={100} />
			}

			if (!isCreateMode && !templateUid && rawDataPreview) {
				return (
					<div style={{ width: '100%', height: '100%' }}>
						<CodeEditorVs raw={rawDataPreview} />
					</div>
				)
			}
			if (rawDataTemplate) {
				return (
					<Fade in timeout={300}>
						<div style={{ width: '100%', height: '100%' }}>
							<CodeEditorVs raw={rawDataTemplate} />
						</div>
					</Fade>
				)
			}
		}

		const renderViewTemplateMode = () => {
			if (isCreateMode && !templateUid) return null
			if (!isCreateMode && !templateUid) return null

			return (
				<ButtonGroup variant='contained' size='small' sx={{ position: 'absolute', top: '35px', right: '45px' }}>
					<Button
						onClick={() => setValue('viewTemplateMode', true)}
						color={isExpanded ? 'primary' : 'inherit'}
					>
						Expanded
					</Button>
					<Button
						onClick={() => setValue('viewTemplateMode', false)}
						color={!isExpanded ? 'primary' : 'inherit'}
					>
						Compact
					</Button>
				</ButtonGroup>
			)
		}

		return (
			<Box className='codeBlockVs' sx={{ ...codeBlockVs }}>
				{renderCodeBlock()}
				{renderViewTemplateMode()}
			</Box>
		)
	}
)
