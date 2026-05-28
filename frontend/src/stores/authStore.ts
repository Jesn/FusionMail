import { create } from 'zustand'
import { persist } from 'zustand/middleware'
import { useAccountStore } from './accountStore'

export interface User {
  id: number
  username: string
  email: string
  displayName?: string
  name?: string
  avatar?: string
  role: string
  theme?: string
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
  login: (user: User, token: string | null | undefined, expiresAt: string) => void
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
      
      setToken: () => set({ token: null }),
      
      login: (user, _token, expiresAt) => set({ 
        user, 
        token: null, 
        expiresAt,
        isAuthenticated: true 
      }),
      
      logout: () => {
        // 重置账户状态
        useAccountStore.getState().reset();
        set({
          user: null,
          token: null,
          expiresAt: null,
          isAuthenticated: false
        });
      },
      
      setLoading: (loading) => set({ isLoading: loading }),

      /**
       * 检查 Cookie 会话对应的前端过期时间是否仍有效。
       * JWT 只存放在 HttpOnly Cookie 中，前端状态不保存 token。
       */
      isTokenValid: () => {
        const { expiresAt } = get()
        if (!expiresAt) return false
        
        const expirationTime = new Date(expiresAt).getTime()
        const currentTime = Date.now()
        
        return currentTime < expirationTime
      },
    }),
    {
      name: 'fusionmail-auth', // 更清晰的命名
      partialize: (state) => ({ 
        user: state.user, 
        expiresAt: state.expiresAt,
        isAuthenticated: state.isAuthenticated 
      }),
    }
  )
)
