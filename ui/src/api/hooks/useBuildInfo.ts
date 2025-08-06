import { useQuery } from '@tanstack/react-query'
import GetBuildInfoService from '../services/getBuildInfoService'

export const useBuildInfo = () => {
  return useQuery({
    queryKey: ['buildInfo'],
    queryFn: () => GetBuildInfoService.getBuildInfo(),
  })
}