import { ITemplateOption } from '../../components/virtualServiceForm'

interface IUpdateTemplateOptions {
	currentTemplateOptions: ITemplateOption[]
	optionKey: string
	isReplaceMode: boolean
	modifier?: number
}

export const updateTemplateOptions = ({
	currentTemplateOptions,
	optionKey,
	isReplaceMode,
	modifier = 2
}: IUpdateTemplateOptions): ITemplateOption[] => {
	const existing = currentTemplateOptions.find(opt => opt.field === optionKey)

	if (isReplaceMode) {
		if (existing?.modifier === modifier) return currentTemplateOptions

		return [...currentTemplateOptions.filter(opt => opt.field !== optionKey), { field: optionKey, modifier }]
	}

	if (!isReplaceMode && existing) {
		return currentTemplateOptions.filter(opt => opt.field !== optionKey)
	}

	return currentTemplateOptions
}
