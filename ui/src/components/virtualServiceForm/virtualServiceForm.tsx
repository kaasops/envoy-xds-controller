import React, { useEffect } from 'react'
import { SubmitHandler, useForm, useWatch } from 'react-hook-form'

import Box from '@mui/material/Box'
import Divider from '@mui/material/Divider'
import Tabs from '@mui/material/Tabs'
import { Tab } from '@mui/material'

import { useCreateVs, useListVs, useUpdateVs } from '../../api/grpc/hooks/useVirtualService.ts'
import CustomTabPanel from '../customTabPanel/CustomTabPanel.tsx'
import { a11yProps } from '../customTabPanel/style.ts'
import { ErrorSnackBarVs } from '../errorSnackBarVs/errorSnackBarVs.tsx'
import { GeneralTabVs } from '../generalTabVS/generalTabVS.tsx'
import { SettingsTabVs } from '../settingsTabVs/settingsTabVs.tsx'
import { VirtualHostDomains } from '../virtualHostDomains'

import { useTabStore } from '../../store/tabIndexStore.ts'

import { IVirtualServiceForm, IVirtualServiceFormProps } from './types.ts'
import { CodeBlockVs } from '../codeBlockVs/codeBlockVs.tsx'
import { boxForm, tabsStyle, vsForm, vsFormLeftColumn, vsFormWrapper } from './style.ts'
import { ActionButtonsVs } from '../actionButtonsVs/actionButtonsVs.tsx'
import { TemplateOptionsFormVsRo } from '../templateOptionsFormVsRO'
import {
	useFillTemplateHook,
	useSetDefaultValuesVSForm,
	useVirtualServiceFormMeta,
	useVirtualServiceSubmit
} from '../../utils/hooks'
import { getDefaultVirtualServiceValues } from '../../utils/helpers'

export const VirtualServiceForm: React.FC<IVirtualServiceFormProps> = ({ virtualServiceInfo }) => {
	const { groupId, isCreate } = useVirtualServiceFormMeta()

	const tabIndex = useTabStore(state => state.tabIndex)
	const setTabIndex = useTabStore(state => state.setTabIndex)

	const { refetch } = useListVs(false, groupId)
	const { createVirtualService, isFetchingCreateVs, errorCreateVs } = useCreateVs()
	const { updateVS, isFetchingUpdateVs, errorUpdateVs, resetQueryUpdateVs } = useUpdateVs()

	const {
		register,
		handleSubmit,
		formState: { errors, isSubmitting, isValid, isSubmitted },
		setValue,
		control,
		setError,
		clearErrors,
		reset,
		trigger
	} = useForm<IVirtualServiceForm>({
		mode: 'onChange',
		defaultValues: getDefaultVirtualServiceValues(isCreate, groupId),
		shouldUnregister: false
	})

	const [name, nodeIds, templateUid] = useWatch({ control, name: ['name', 'nodeIds', 'templateUid'] })

	const isFormReady =
		isValid && Boolean(name?.length) && Array.isArray(nodeIds) && nodeIds.length > 0 && Boolean(templateUid)

	useEffect(() => {
		if (tabIndex === 0 && isSubmitted) void trigger('name')
	}, [tabIndex, trigger, isSubmitted])

	const formValues = useWatch({ control }) as IVirtualServiceForm
	const { rawDataTemplate, isLoadingFillTemplate, errorFillTemplate } = useFillTemplateHook({ formValues })

	const { setDefaultValues } = useSetDefaultValuesVSForm({ reset, isCreate, virtualServiceInfo })

	const handleResetForm = () => {
		if (isCreate) {
			reset()
			setTabIndex(0)
		} else {
			setDefaultValues()
		}
	}

	const handleChangeTabIndex = (_e: React.SyntheticEvent, newTabIndex: number) => {
		setTabIndex(newTabIndex)
	}

	const { submitVSService } = useVirtualServiceSubmit({
		isCreate,
		groupId,
		virtualServiceInfo,
		createVirtualService,
		updateVS,
		resetQueryUpdateVs,
		refetch
	})

	const onSubmit: SubmitHandler<IVirtualServiceForm> = async data => {
		if (!isFormReady) return
		await submitVSService(data)
	}

	return (
		<form onSubmit={handleSubmit(onSubmit)} style={{ height: '100%' }}>
			<Box className='vsForm' sx={{ ...vsForm }}>
				<Tabs
					orientation='vertical'
					value={tabIndex}
					onChange={handleChangeTabIndex}
					aria-label='formTabMEnu'
					sx={{ ...tabsStyle }}
				>
					<Tab label='General' {...a11yProps(0, 'vertical')} />
					<Tab label='Domains' {...a11yProps(1, 'vertical')} />
					<Tab label='Settings' {...a11yProps(2, 'vertical')} />
					<Tab label='Template' {...a11yProps(3, 'vertical')} />
				</Tabs>
				<Box className='vsFormWrapper' sx={{ ...vsFormWrapper }}>
					<Box display='flex' className='vsColumnWrapper' gap={1.5} height='100%'>
						<Box className='vsFormLeftColumn' sx={{ ...vsFormLeftColumn }}>
							<Box className='boxForm' sx={{ boxForm }}>
								<CustomTabPanel value={tabIndex} index={0} variant={'vertical'}>
									<GeneralTabVs
										register={register}
										control={control}
										errors={errors}
										isEdit={!isCreate}
									/>
								</CustomTabPanel>

								<CustomTabPanel value={tabIndex} index={1} variant={'vertical'}>
									<VirtualHostDomains
										control={control}
										setValue={setValue}
										errors={errors}
										setError={setError}
										clearErrors={clearErrors}
									/>
								</CustomTabPanel>

								<CustomTabPanel value={tabIndex} index={2} variant={'vertical'}>
									<SettingsTabVs
										control={control}
										setValue={setValue}
										errors={errors}
										fillTemplate={rawDataTemplate}
									/>
								</CustomTabPanel>

								<CustomTabPanel value={tabIndex} index={3} variant={'vertical'}>
									<TemplateOptionsFormVsRo
										control={control}
										setValue={setValue}
										register={register}
									/>
								</CustomTabPanel>
							</Box>

							<ActionButtonsVs
								isCreateMode={isCreate}
								isEditable={isCreate ? true : !!virtualServiceInfo?.isEditable}
								isFetchingCreateVs={isFetchingCreateVs}
								isFetchingUpdateVs={isFetchingUpdateVs}
								handleResetForm={handleResetForm}
							/>
						</Box>
						<Divider orientation='vertical' flexItem sx={{ height: '100%' }} />
						<CodeBlockVs
							rawDataTemplate={rawDataTemplate?.raw}
							rawDataPreview={virtualServiceInfo?.raw}
							control={control}
							isLoadingFillTemplate={isLoadingFillTemplate}
							isCreateMode={isCreate}
							setValue={setValue}
						/>
					</Box>
				</Box>
			</Box>

			<ErrorSnackBarVs
				errors={errors}
				isFormReady={isFormReady}
				errorCreateVs={errorCreateVs}
				errorUpdateVs={errorUpdateVs}
				errorFillTemplate={errorFillTemplate}
				isSubmitting={isSubmitting}
			/>
		</form>
	)
}
