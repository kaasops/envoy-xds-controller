import { createConnectTransport } from '@connectrpc/connect-web'
import { env } from '../../env.ts'
import { Code, ConnectError, createClient, Interceptor } from '@connectrpc/connect'
import { VirtualServiceStoreService } from '../../gen/virtual_service/v1/virtual_service_pb'
import { VirtualServiceTemplateStoreService } from '../../gen/virtual_service_template/v1/virtual_service_template_pb.ts'
import { ListenerStoreService } from '../../gen/listener/v1/listener_pb.ts'
import { AccessLogConfigStoreService } from '../../gen/access_log_config/v1/access_log_config_pb.ts'
import { HTTPFilterStoreService } from '../../gen/http_filter/v1/http_filter_pb.ts'
import { RouteStoreService } from '../../gen/route/v1/route_pb.ts'
import { AccessGroupStoreService } from '../../gen/access_group/v1/access_group_pb.ts'
import { NodeStoreService } from '../../gen/node/v1/node_pb.ts'
import { getAuth } from '../../utils/helpers/authBridge.ts'
import { UtilsService } from '../../gen/util/v1/util_pb.ts'
import { PermissionsService } from '../../gen/permissions/v1/permissions_pb.ts'

export const tokenInterceptor: Interceptor = next => async req => {
	if (env.VITE_OIDC_ENABLED === 'true') {
		const auth = getAuth()
		const accessToken = auth.user?.access_token

		if (accessToken) {
			req.header.set('Authorization', `Bearer ${accessToken}`)
		}
	}
	return next(req)
}

export const errorInterceptor: Interceptor = next => async req => {
	try {
		return await next(req)
	} catch (err) {
		if (err instanceof ConnectError && err.code === Code.Unauthenticated) {
			console.log('Token expired or invalid, redirecting to login...')
			await getAuth().signinRedirect()
		} else {
			console.error('Error:', err instanceof ConnectError ? err.message : 'Unexpected error')
		}

		throw err
	}
}

export const transport = createConnectTransport({
	baseUrl: env.VITE_GRPC_API_URL || '/grpc-api',
	interceptors: [tokenInterceptor, errorInterceptor]
})

export const virtualServiceClient = createClient(VirtualServiceStoreService, transport)

export const templateServiceClient = createClient(VirtualServiceTemplateStoreService, transport)

export const listenerServiceClient = createClient(ListenerStoreService, transport)

export const accessLogServiceClient = createClient(AccessLogConfigStoreService, transport)

export const httpFilterServiceClient = createClient(HTTPFilterStoreService, transport)

export const routeServiceClient = createClient(RouteStoreService, transport)

export const accessGroupsServiceClient = createClient(AccessGroupStoreService, transport)

export const nodeServiceClient = createClient(NodeStoreService, transport)

export const utilServiceClient = createClient(UtilsService, transport)

export const permissionsServiceClient = createClient(PermissionsService, transport)
