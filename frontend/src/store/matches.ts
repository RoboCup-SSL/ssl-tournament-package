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
    // fetch loads the tournament's matches. A silent fetch (used by polling)
    // skips the loading/error UI and only reassigns items when the data changed,
    // so an unchanged poll triggers no re-render.
    async fetch(tournamentId: number, silent = false) {
      this.tournamentId = tournamentId
      if (!silent) {
        this.loading = true
        this.error = ''
      }
      try {
        const items = await api.get<Match[]>(`/api/matches?tournament_id=${tournamentId}`)
        if (JSON.stringify(items) !== JSON.stringify(this.items)) this.items = items
      } catch (failure) {
        if (!silent) this.error = failure instanceof Error ? failure.message : String(failure)
      } finally {
        if (!silent) this.loading = false
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
