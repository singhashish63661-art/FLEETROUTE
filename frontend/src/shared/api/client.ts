import axios from 'axios'

const api = axios.create({
  baseURL: '/api/v1',
  timeout: 15_000,
  headers: {
    'Content-Type': 'application/json',
  },
})

// Response interceptor — extract .data.data envelope
api.interceptors.response.use(
  (res) => res,
  (error) => {
    if (error.response?.status === 401) {
      // Clear auth and redirect to login
      localStorage.removeItem('gpsgo-auth')
      window.location.href = '/login'
    }
    return Promise.reject(error)
  },
)

export default api
