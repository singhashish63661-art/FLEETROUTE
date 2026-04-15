import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import api from '../api/client'

interface User {
  id: string
  email: string
  tenantId: string
  role: string
  name?: string
}

interface AuthState {
  user: User | null
  accessToken: string | null
  refreshToken: string | null
  isAuthenticated: boolean

  login: (email: string, password: string) => Promise<void>
  logout: () => void
  setTokens: (access: string, refresh: string) => void
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      user: null,
      accessToken: null,
      refreshToken: null,
      isAuthenticated: false,

      login: async (email, password) => {
        const res = await api.post('/auth/login', { email, password })
        const { data } = res.data
        set({
          user: {
            id: data.user.id,
            email: data.user.email,
            tenantId: data.user.tenant_id,
            role: data.user.role,
          },
          accessToken: data.access_token,
          refreshToken: data.refresh_token,
          isAuthenticated: true,
        })
        api.defaults.headers.common['Authorization'] = `Bearer ${data.access_token}`
      },

      logout: () => {
        set({ user: null, accessToken: null, refreshToken: null, isAuthenticated: false })
        delete api.defaults.headers.common['Authorization']
      },

      setTokens: (access, refresh) => {
        set({ accessToken: access, refreshToken: refresh })
        api.defaults.headers.common['Authorization'] = `Bearer ${access}`
      },
    }),
    {
      name: 'gpsgo-auth',
      partialize: (s) => ({
        user: s.user,
        accessToken: s.accessToken,
        refreshToken: s.refreshToken,
        isAuthenticated: s.isAuthenticated,
      }),
      onRehydrateStorage: () => (state) => {
        if (state?.accessToken) {
          api.defaults.headers.common['Authorization'] = `Bearer ${state.accessToken}`
        }
      },
    },
  ),
)
