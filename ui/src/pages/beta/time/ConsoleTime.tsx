import { useMemo, useState, type ReactNode } from 'react'
import { createTimeFormatter, isoTime, readTimeZone, storeTimeZone, type TimeValue } from './format'
import { ConsoleTimeContext, useConsoleTime } from './useConsoleTime'

export function ConsoleTimeProvider({ children }: { children: ReactNode }) {
	const [timeZone, selectZone] = useState(readTimeZone)
	const preferences = useMemo(
		() => ({
			timeZone,
			formatTime: createTimeFormatter(timeZone),
			setTimeZone(zone: string) {
				selectZone(zone)
				storeTimeZone(zone)
			}
		}),
		[timeZone]
	)

	return <ConsoleTimeContext.Provider value={preferences}>{children}</ConsoleTimeContext.Provider>
}

export function ConsoleTime({ value, fallback = 'Unavailable' }: { value: TimeValue; fallback?: string }) {
	const { formatTime } = useConsoleTime()
	const iso = isoTime(value)
	return iso ? (
		<time dateTime={iso} title={iso}>
			{formatTime(value)}
		</time>
	) : (
		<>{fallback}</>
	)
}
