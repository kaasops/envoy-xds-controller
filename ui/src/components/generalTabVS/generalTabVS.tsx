import React from 'react'
import { Control, FieldErrors, UseFormRegister } from 'react-hook-form'
import { IVirtualServiceForm } from '../virtualServiceForm/types.ts'
import { TextFieldFormVs } from '../textFieldFormVs/textFieldFormVs.tsx'
import { SelectNodeVs } from '../selectNodeVs/selectNodeVs.tsx'
import { SelectFormVs } from '../selectFormVs/selectFormVs.tsx'
import { useListenerVs, useNodeListVs, useTemplatesVs } from '../../api/grpc/hooks/useVirtualService.ts'
import { useParams } from 'react-router-dom'
import { TextAreaFomVs } from '../textAreaFomVs/textAreaFomVs.tsx'

interface IGeneralTabVsProps {
	register: UseFormRegister<IVirtualServiceForm>
	control: Control<IVirtualServiceForm>
	errors: FieldErrors<IVirtualServiceForm>
	isEdit?: boolean | undefined
}

export const GeneralTabVs: React.FC<IGeneralTabVsProps> = ({ register, control, errors, isEdit }) => {
	const { groupId } = useParams()

	const { data: nodeList, isFetching: isFetchingNodeList, isError: isErrorNodeList } = useNodeListVs(groupId)
	const { data: templates, isFetching: isFetchingTemplates, isError: isErrorTemplates } = useTemplatesVs(groupId)
	const { data: listeners, isFetching: isFetchingListeners, isError: isErrorListeners } = useListenerVs(groupId)

	return (
		<>
			<TextFieldFormVs register={register} nameField='name' errors={errors} isDisabled={isEdit} />
			<SelectNodeVs
				nameField={'nodeIds'}
				dataNodes={nodeList}
				control={control}
				errors={errors}
				isFetching={isFetchingNodeList}
				isErrorFetch={isErrorNodeList}
			/>
			<SelectFormVs
				nameField={'templateUid'}
				data={templates}
				control={control}
				errors={errors}
				isFetching={isFetchingTemplates}
				isErrorFetch={isErrorTemplates}
			/>
			<SelectFormVs
				nameField={'listenerUid'}
				data={listeners}
				control={control}
				errors={errors}
				isFetching={isFetchingListeners}
				isErrorFetch={isErrorListeners}
			/>
			<TextAreaFomVs register={register} nameField='description' errors={errors} />
		</>
	)
}
