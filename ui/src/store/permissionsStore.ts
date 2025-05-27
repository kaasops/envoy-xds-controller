import { Message } from '@bufbuild/protobuf'
import { create } from 'zustand'

export type PermissionsItem = {
	action: string
	objects: string[]
}

export type AccessGroupPermissions = Message<'permissions.v1.AccessGroupPermissions'> & {
	accessGroup: string
	permissions: PermissionsItem[]
}

type PermissionsMap = Record<string, PermissionsItem[]>

interface PermissionsStore {
	permissionsMap: PermissionsMap
	hasAccess: boolean
	setPermissions: (items: AccessGroupPermissions[]) => void
	getGroupPermissions: (group: string) => PermissionsItem[] | undefined
	hasPermission: (group: string, action: string) => boolean
}

export const usePermissionsStore = create<PermissionsStore>((set, get) => ({
	permissionsMap: {},
	hasAccess: false,

	setPermissions: items => {
		if (!Array.isArray(items) || items.length === 0) {
			set({ hasAccess: false, permissionsMap: {} })
			return
		}

		const map: PermissionsMap = {}
		for (const item of items) {
			map[item.accessGroup] = item.permissions
		}

		set({ permissionsMap: map, hasAccess: true })
	},

	getGroupPermissions: group => {
		return get().permissionsMap[group]
	},

	hasPermission: (group, action) => {
		const permissions = get().permissionsMap[group]
		if (!permissions) return false
		return permissions.some(perm => perm.action === action)
	}
}))
