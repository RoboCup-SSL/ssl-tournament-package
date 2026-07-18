// Pinia store for a tournament's teams (the Teams section). Same reusable
// list-CRUD pattern as fields: fetch by tournament, mutations re-fetch.
import { defineStore } from 'pinia'
import { api } from '@/api/client'
import type { Team, TeamInput } from '@/api/types'

export const useTeamsStore = defineStore('teams', {
  state: () => ({
    items: [] as Team[],
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
        this.items = await api.get<Team[]>(`/api/teams?tournament_id=${tournamentId}`)
      } catch (failure) {
        this.error = failure instanceof Error ? failure.message : String(failure)
      } finally {
        this.loading = false
      }
    },
    async create(input: TeamInput) {
      this.error = ''
      try {
        await api.post<Team>('/api/teams', { tournament_id: this.tournamentId, ...input })
        await this.fetch(this.tournamentId)
      } catch (failure) {
        this.error = failure instanceof Error ? failure.message : String(failure)
      }
    },
    async update(id: number, input: TeamInput) {
      this.error = ''
      try {
        await api.patch<Team>(`/api/teams/${id}`, input)
        await this.fetch(this.tournamentId)
      } catch (failure) {
        this.error = failure instanceof Error ? failure.message : String(failure)
      }
    },
    async remove(id: number) {
      this.error = ''
      try {
        await api.del(`/api/teams/${id}`)
        await this.fetch(this.tournamentId)
      } catch (failure) {
        this.error = failure instanceof Error ? failure.message : String(failure)
      }
    },
  },
})
