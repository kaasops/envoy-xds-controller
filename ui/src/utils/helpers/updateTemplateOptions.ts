import { ITemplateOption } from '../../components/virtualServiceForm'
import { TemplateOptionModifier } from '../../gen/virtual_service_template/v1/virtual_service_template_pb.ts'

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

export const getModifierForOption = (optionKey: string, isReplaceMode: boolean): TemplateOptionModifier | null => {
	if (isReplaceMode) return TemplateOptionModifier.REPLACE

	if (['accessLog', 'accessLogConfig', 'accessLogs'].includes(optionKey)) {
		return TemplateOptionModifier.DELETE
	}

	return null
}

export const updateTemplateOptionsNew = ({
	currentTemplateOptions,
	optionKey,
	isReplaceMode
}: IUpdateTemplateOptions): ITemplateOption[] => {
	const modifier = getModifierForOption(optionKey, isReplaceMode)
	if (modifier === null) return currentTemplateOptions

	const filtered = currentTemplateOptions.filter(opt => opt.field !== optionKey)

	if (isReplaceMode && ['accessLog', 'accessLogConfig', 'accessLogs'].includes(optionKey)) {
		return [
			...filtered,
			{ field: optionKey, modifier: TemplateOptionModifier.DELETE },
			{ field: optionKey, modifier: TemplateOptionModifier.REPLACE }
		]
	}

	if (isReplaceMode && optionKey === 'accessLogConfigs') {
		return [...filtered, { field: optionKey, modifier: TemplateOptionModifier.REPLACE }]
	}

	if (!isReplaceMode && ['accessLog', 'accessLogConfig', 'accessLogs'].includes(optionKey)) {
		return [...filtered, { field: optionKey, modifier: TemplateOptionModifier.DELETE }]
	}

	return currentTemplateOptions
}
