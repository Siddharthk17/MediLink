import axios, {
  type AxiosInstance,
  type AxiosRequestConfig,
  type InternalAxiosRequestConfig,
} from 'axios'

let _getTokens: () => { accessToken: string | null; refreshToken: string | null } = () =>
  ({ accessToken: null, refreshToken: null })
let _setTokens: (pair: { accessToken: string; refreshToken: string } | null) => void = () => {}
let _onAuthFailure: () => void = () => {}

export function configureClient(options: {
  baseURL: string
  getTokens: typeof _getTokens
  setTokens: typeof _setTokens
  onAuthFailure: typeof _onAuthFailure
}) {
  _getTokens = options.getTokens
  _setTokens = options.setTokens
  _onAuthFailure = options.onAuthFailure
  apiClient.defaults.baseURL = options.baseURL
}

export const apiClient: AxiosInstance = axios.create({
  baseURL: '/api',
  timeout: 30_000,
  headers: { 'Content-Type': 'application/json' },
})

apiClient.interceptors.request.use((config: InternalAxiosRequestConfig) => {
  const { accessToken } = _getTokens()
  if (accessToken) {
    config.headers.Authorization = `Bearer ${accessToken}`
  }
  return config
})

let isRefreshing = false
let waitQueue: Array<(token: string) => void> = []

apiClient.interceptors.response.use(
  (res) => res,
  async (error) => {
    const original = error.config as AxiosRequestConfig & { _retry?: boolean }
    if (error.response?.status !== 401 || original._retry) {
      return Promise.reject(error)
    }
    if (isRefreshing) {
      return new Promise<string>((resolve) => {
        waitQueue.push(resolve)
      }).then((newToken) => {
        original.headers = { ...original.headers, Authorization: `Bearer ${newToken}` }
        return apiClient(original)
      })
    }
    original._retry = true
    isRefreshing = true
    try {
      const { refreshToken } = _getTokens()
      if (!refreshToken) throw new Error('no refresh token')
      const refreshURL = `${apiClient.defaults.baseURL}/auth/refresh`
      const res = await axios.post<{ accessToken: string; refreshToken: string }>(
        refreshURL,
        { refreshToken }
      )
      const { accessToken, refreshToken: newRefresh } = res.data
      _setTokens({ accessToken, refreshToken: newRefresh })
      waitQueue.forEach((cb) => cb(accessToken))
      waitQueue = []
      original.headers = { ...original.headers, Authorization: `Bearer ${accessToken}` }
      return apiClient(original)
    } catch {
      waitQueue = []
      _setTokens(null)
      _onAuthFailure()
      return Promise.reject(error)
    } finally {
      isRefreshing = false
    }
  }
)
