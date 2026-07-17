// Pinia store for the instance-level tournament list. Establishes the store
// pattern the workspace sections will follow.
import { defineStore } from 'pinia'
import { api } from '@/api/client'
import type { Tournament } from '@/api/types'

export const useTournamentsStore = defineStore('tournaments', {
  state: () => ({
    tournaments: [] as Tournament[],
    loading: false,
    error: '',
  }),
  actions: {
    // fetch loads all tournaments, capturing any error message for display.
    async fetch() {
      this.loading = true
      this.error = ''
      try {
        this.tournaments = await api.get<Tournament[]>('/api/tournaments')
      } catch (failure) {
        this.error = failure instanceof Error ? failure.message : String(failure)
      } finally {
        this.loading = false
      }
    },
  },
})
