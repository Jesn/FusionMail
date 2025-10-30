import { create } from 'zustand'
import { persist } from 'zustand/middleware'

export interface User {
  id: number
  email: string
  name?: string
}

interface AuthState {
  user: User | null
  token: string | null
  expiresAt: string | null
  isAuthenticated: boolean
  isLoading: boolean
  
  // Actions
  setUser: (user: User | null) => void
  setToken: (token: string | null) => void
  login: (user: User, token: string, expiresAt: string) => void
  logout: () => void
  setLoading: (loading: boolean) => void
  isTokenValid: () => boolean
}

export const useAuthStore = create<AuthState>()(
  persist(
    (set, get) => ({
      user: null,
      token: null,
      expiresAt: null,
      isAuthenticated: false,
      isLoading: false,

      setUser: (user) => set({ user, isAuthenticated: !!user }),
      
      setToken: (token) => set({ token }),
      
      login: (user, token, expiresAt) => set({ 
        user, 
        token, 
        expiresAt,
        isAuthenticated: true 
      }),
      
      logout: () => set({ 
        user: null, 
        token: null, 
        expiresAt: null,
        isAuthenticated: false 
      }),
      
      setLoading: (loading) => set({ isLoading: loading }),

      /**
       * 检查 token 是否有效（未过期）
       */
      isTokenValid: () => {
        const { token, expiresAt } = get()
        if (!token || !expiresAt) return false
        
        const expirationTime = new Date(expiresAt).getTime()
        const currentTime = Date.now()
        
        return currentTime < expirationTime
      },
    }),
    {
      name: 'fusionmail-auth', // 更清晰的命名
      partialize: (state) => ({ 
        user: state.user, 
        token: state.token,
        expiresAt: state.expiresAt,
        isAuthenticated: state.isAuthenticated 
      }),
    }
  )
)
