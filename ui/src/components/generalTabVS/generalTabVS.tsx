import React from 'react'
import { Control, FieldErrors, UseFormRegister, UseFormSetValue, useWatch } from 'react-hook-form'
import { IVirtualServiceForm } from '../virtualServiceForm/types.ts'
import { TextFieldFormVs } from '../textFieldFormVs/textFieldFormVs.tsx'
import { useListenerVs, useNodeListVs, useTemplatesVs } from '../../api/grpc/hooks/useVirtualService.ts'
import { useParams } from 'react-router-dom'
import { TextAreaFomVs } from '../textAreaFomVs/textAreaFomVs.tsx'
import { NodeIdsVs } from '../nodeIdsVs/nodeIdsVs.tsx'
import { AutocompleteVs } from '../autocompleteVs'
import { ExtraFieldsTabVs } from '../extraFieldsTabVs'
import { Box, Divider } from '@mui/material'

interface IGeneralTabVsProps {
	register: UseFormRegister<IVirtualServiceForm>
	control: Control<IVirtualServiceForm>
	errors: FieldErrors<IVirtualServiceForm>
	isEdit?: boolean | undefined
	setValue: UseFormSetValue<IVirtualServiceForm>
}

export const GeneralTabVs: React.FC<IGeneralTabVsProps> = ({ register, control, errors, isEdit, setValue }) => {
	const { groupId } = useParams()

	const { data: nodeList, isFetching: isFetchingNodeList, isError: isErrorNodeList } = useNodeListVs(groupId)
	const { data: templates, isFetching: isFetchingTemplates, isError: isErrorTemplates } = useTemplatesVs(groupId)
	const { data: listeners, isFetching: isFetchingListeners, isError: isErrorListeners } = useListenerVs(groupId)

	// Watch the selected template UID to get its extra fields
	const selectedTemplateUid = useWatch({ control, name: 'templateUid' })
	
	// Find the selected template
	const selectedTemplate = templates?.items?.find(template => template.uid === selectedTemplateUid)
	
	// Check if the selected template has extra fields
	const hasExtraFields = selectedTemplate?.extraFields && selectedTemplate.extraFields.length > 0

	return (
		<>
			<TextFieldFormVs register={register} nameField='name' errors={errors} isDisabled={isEdit} />
			<NodeIdsVs
				nameField={'nodeIds'}
				dataNodes={nodeList}
				control={control}
				errors={errors}
				isFetching={isFetchingNodeList}
				isErrorFetch={isErrorNodeList}
			/>
			<AutocompleteVs
				nameField={'templateUid'}
				data={templates}
				control={control}
				errors={errors}
				isFetching={isFetchingTemplates}
				isErrorFetch={isErrorTemplates}
			/>
			<AutocompleteVs
				nameField={'listenerUid'}
				data={listeners}
				control={control}
				errors={errors}
				isFetching={isFetchingListeners}
				isErrorFetch={isErrorListeners}
			/>
			<TextAreaFomVs register={register} nameField='description' errors={errors} />
			
   {/* Extra Fields Section */}
			{hasExtraFields && (
				<>
					<Box mt={2} mb={2}>
						<Divider />
					</Box>
					<ExtraFieldsTabVs
						control={control}
						errors={errors}
						setValue={setValue}
						isEditable={!isEdit ? true : true}
					/>
				</>
			)}
		</>
	)
}
