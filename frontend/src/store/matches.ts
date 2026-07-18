// Pinia store for a tournament's matches (the Schedule section). Same reusable
// list-CRUD pattern as fields/teams: fetch by tournament, mutations re-fetch.
import { defineStore } from 'pinia'
import { api } from '@/api/client'
import type { Match, MatchInput } from '@/api/types'

export const useMatchesStore = defineStore('matches', {
  state: () => ({
    items: [] as Match[],
    tournamentId: 0,
    loading: false,
    error: '',
  }),
  actions: {
    async fetch(tournamentId: number) {
      this.tournamentId = tournamentId
      this.loading = true
      this.error = ''
      try {
        this.items = await api.get<Match[]>(`/api/matches?tournament_id=${tournamentId}`)
      } catch (failure) {
        this.error = failure instanceof Error ? failure.message : String(failure)
      } finally {
        this.loading = false
      }
    },
    async create(input: MatchInput) {
      this.error = ''
      try {
        await api.post<Match>('/api/matches', { tournament_id: this.tournamentId, ...input })
        await this.fetch(this.tournamentId)
      } catch (failure) {
        this.error = failure instanceof Error ? failure.message : String(failure)
      }
    },
    async update(id: number, input: MatchInput) {
      this.error = ''
      try {
        await api.patch<Match>(`/api/matches/${id}`, input)
        await this.fetch(this.tournamentId)
      } catch (failure) {
        this.error = failure instanceof Error ? failure.message : String(failure)
      }
    },
    async remove(id: number) {
      this.error = ''
      try {
        await api.del(`/api/matches/${id}`)
        await this.fetch(this.tournamentId)
      } catch (failure) {
        this.error = failure instanceof Error ? failure.message : String(failure)
      }
    },
  },
})
