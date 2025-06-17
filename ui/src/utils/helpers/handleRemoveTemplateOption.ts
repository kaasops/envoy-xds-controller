import { FieldArrayWithId, UseFormSetValue } from 'react-hook-form'
import { IVirtualServiceForm } from '../../components/virtualServiceForm'

interface HandleRemoveParams {
	field: FieldArrayWithId<IVirtualServiceForm, 'templateOptions'>
	setValue: UseFormSetValue<IVirtualServiceForm>
}

export const handleRemoveTemplateOption = ({ field, setValue }: HandleRemoveParams) => {
	const fieldToStateMap: Record<string, { name: keyof IVirtualServiceForm; value: any }> = {
		'virtualHost.domains': { name: 'virtualHostDomainsMode', value: false },
		additionalHttpFilters: { name: 'additionalHttpFilterMode', value: false },
		additionalRoutes: { name: 'additionalRouteMode', value: false },
		accessLog: { name: 'accessLogConfigUid', value: '' }
	}

	const target = fieldToStateMap[field.field]
	if (target) {
		setValue(target.name, target.value, {
			shouldDirty: true,
			shouldTouch: true,
			shouldValidate: true
		})
	}
}
