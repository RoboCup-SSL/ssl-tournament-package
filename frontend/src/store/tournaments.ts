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
    // create adds a tournament (name only) and refreshes the list; returns the
    // created row so the caller can navigate into it, or null on failure.
    async create(name: string): Promise<Tournament | null> {
      this.error = ''
      try {
        const created = await api.post<Tournament>('/api/tournaments', { name })
        await this.fetch()
        return created
      } catch (failure) {
        this.error = failure instanceof Error ? failure.message : String(failure)
        return null
      }
    },
    // remove deletes a tournament and everything it owns (server-side cascade),
    // then refreshes the list.
    async remove(id: number) {
      this.error = ''
      try {
        await api.del(`/api/tournaments/${id}`)
        await this.fetch()
      } catch (failure) {
        this.error = failure instanceof Error ? failure.message : String(failure)
      }
    },
  },
})
