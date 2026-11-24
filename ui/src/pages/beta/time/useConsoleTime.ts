import { createContext, useContext } from 'react'
import type { TimeValue } from './format'

interface TimePreferences {
	timeZone: string
	setTimeZone: (zone: string) => void
	formatTime: (value: TimeValue) => string
}

export const ConsoleTimeContext = createContext<TimePreferences | null>(null)

export function useConsoleTime() {
	const context = useContext(ConsoleTimeContext)
	if (!context) throw new Error('Console time preferences require a provider')
	return context
}
